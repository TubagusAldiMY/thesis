// Command server menjalankan REST API SPK Zakat.
//
//	go run ./cmd/server
//
// Konfigurasi lewat environment variable (lihat .env.example). Migrasi TIDAK
// dijalankan otomatis; jalankan `go run ./cmd/migrate` lebih dulu.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"spkzakat/internal/crypto"
	"spkzakat/internal/httpapi"
	"spkzakat/internal/service"
	"spkzakat/internal/store"
)

// config = semua pengaturan dari env (12-factor). Rahasia tidak pernah dilog.
type config struct {
	addr         string
	dsn          string
	kunciEnc     []byte
	kunciHMAC    []byte
	cookieSecure bool
	logLevel     slog.Level
}

func muatConfig() (config, error) {
	c := config{addr: os.Getenv("ADDR"), dsn: os.Getenv("DATABASE_URL"), cookieSecure: os.Getenv("COOKIE_SECURE") != "false"}
	if c.addr == "" {
		c.addr = "127.0.0.1:8090" // hanya lokal: API dipanggil SvelteKit, tidak diekspos publik
	}
	if c.dsn == "" {
		return c, errors.New("DATABASE_URL wajib diset")
	}
	var err error
	if c.kunciEnc, err = crypto.ParseKey(os.Getenv("NIK_ENCRYPTION_KEY")); err != nil {
		return c, errors.New("NIK_ENCRYPTION_KEY: " + err.Error())
	}
	if c.kunciHMAC, err = crypto.ParseKey(os.Getenv("NIK_HMAC_KEY")); err != nil {
		return c, errors.New("NIK_HMAC_KEY: " + err.Error())
	}
	if err := c.logLevel.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))); err != nil {
		c.logLevel = slog.LevelInfo // kosong / tidak dikenal → info
	}
	return c, nil
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := muatConfig()
	if err != nil {
		log.Error("konfigurasi tidak valid", "error", err.Error())
		os.Exit(1)
	}
	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.logLevel}))
	slog.SetDefault(log)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	st, err := store.Open(ctx, cfg.dsn)
	cancel()
	if err != nil {
		log.Error("koneksi database gagal", "error", err.Error())
		os.Exit(1)
	}
	defer st.Close()

	enc, err := crypto.NewCipher(cfg.kunciEnc)
	if err != nil {
		log.Error("kunci enkripsi", "error", err.Error())
		os.Exit(1)
	}
	idx, err := crypto.NewHasher(cfg.kunciHMAC)
	if err != nil {
		log.Error("kunci HMAC", "error", err.Error())
		os.Exit(1)
	}
	svc := service.New(st, enc, idx)

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           httpapi.New(svc, log, cfg.cookieSecure, st.Ping),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Info("server berjalan", "addr", cfg.addr, "cookie_secure", cfg.cookieSecure)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server berhenti", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Graceful shutdown: tunggu permintaan yang sedang berjalan (maks 30 detik).
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Info("mematikan server...")
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown paksa", "error", err.Error())
	}
	log.Info("server berhenti")
}
