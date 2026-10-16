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
	"strings"
	"syscall"
	"time"

	"spkzakat/internal/crypto"
	"spkzakat/internal/httpapi"
	"spkzakat/internal/payment"
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

	// Donasi web (opsional). provider kosong = halaman donasi publik nonaktif,
	// pencatatan donasi oleh petugas tetap berfungsi.
	provider    string // "" | "midtrans" | "tiruan"
	midtransKey string
	midtransURL string // kosong = sandbox
	webOrigin   string // alamat publik frontend, tujuan pengalihan setelah bayar
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

	c.provider = os.Getenv("PAYMENT_PROVIDER")
	c.midtransKey = os.Getenv("MIDTRANS_SERVER_KEY")
	c.midtransURL = os.Getenv("MIDTRANS_SNAP_URL")
	c.webOrigin = os.Getenv("WEB_ORIGIN")
	switch c.provider {
	case "":
	case "midtrans", "tiruan":
		if !strings.HasPrefix(c.webOrigin, "http://") && !strings.HasPrefix(c.webOrigin, "https://") {
			return c, errors.New("WEB_ORIGIN wajib diset (mis. https://zakat.masjid.id) bila PAYMENT_PROVIDER aktif")
		}
		if c.provider == "midtrans" && c.midtransKey == "" {
			return c, errors.New("MIDTRANS_SERVER_KEY wajib diset bila PAYMENT_PROVIDER=midtrans")
		}
		// Penyedia tiruan membuka endpoint "tandai lunas" tanpa pembayaran. COOKIE_SECURE
		// bawaan true menandai server sungguhan di belakang HTTPS: tolak di sana.
		if c.provider == "tiruan" && c.cookieSecure {
			return c, errors.New("PAYMENT_PROVIDER=tiruan hanya untuk development: set COOKIE_SECURE=false, atau pakai midtrans")
		}
	default:
		return c, errors.New("PAYMENT_PROVIDER harus kosong, midtrans, atau tiruan")
	}
	return c, nil
}

// pasangGerbang memasang payment gateway sesuai konfigurasi dan mengembalikan
// opsi HTTP yang menyertainya (simulasi bayar, khusus penyedia tiruan).
func pasangGerbang(cfg config, svc *service.Service, log *slog.Logger) ([]httpapi.Opsi, error) {
	switch cfg.provider {
	case "midtrans":
		m, err := payment.NewMidtrans(cfg.midtransKey, cfg.midtransURL)
		if err != nil {
			return nil, err
		}
		svc.AturGerbang(m, cfg.webOrigin)
		log.Info("donasi web aktif", "provider", "midtrans")
	case "tiruan":
		t := payment.NewTiruan()
		svc.AturGerbang(t, cfg.webOrigin)
		log.Warn("PENYEDIA PEMBAYARAN TIRUAN MENYALA: hanya untuk development/demo, bukan pembayaran sungguhan")
		return []httpapi.Opsi{httpapi.DenganSimulasiBayar(t)}, nil
	default:
		log.Info("donasi web nonaktif (PAYMENT_PROVIDER kosong); pencatatan oleh petugas tetap berfungsi")
	}
	return nil, nil
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
	opsi, err := pasangGerbang(cfg, svc, log)
	if err != nil {
		log.Error("payment gateway", "error", err.Error())
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           httpapi.New(svc, log, cfg.cookieSecure, st.Ping, opsi...),
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
