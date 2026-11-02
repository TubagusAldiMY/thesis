package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"spkzakat/internal/crypto"
	"spkzakat/internal/service"
	"spkzakat/internal/store"
	"spkzakat/internal/testdb"
)

// Sanity setup awal lewat HTTP pada instalasi KOSONG: halaman terbuka hanya untuk
// pemegang kode, akun pertama adalah Ketua yang bisa langsung masuk dan mengelola
// pengguna, lalu setup mati dan kode lama tidak berguna.
func TestSetup_AlurInstalasiBaru(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, testdb.Migrated(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	enc, _ := crypto.NewCipher(bytes.Repeat([]byte{1}, crypto.KeySize))
	idx, _ := crypto.NewHasher(bytes.Repeat([]byte{2}, crypto.KeySize))
	svc := service.New(st, enc, idx)
	kode, err := svc.SiapkanSetup(ctx) // yang dicetak main.go ke log server
	if err != nil || kode == "" {
		t.Fatalf("SiapkanSetup = %q, %v", kode, err)
	}
	log := &bufAman{}
	srv := httptest.NewServer(New(svc, slog.New(slog.NewJSONHandler(log, nil)), false, st.Ping))
	t.Cleanup(srv.Close)
	anon := klienBaru(t, srv)

	var status struct {
		Aktif bool `json:"aktif"`
	}
	anon.minta("GET", "/api/publik/setup", nil, 200, &status)
	if !status.Aktif {
		t.Fatal("setup tidak aktif pada instalasi kosong")
	}

	isi := map[string]any{"token": "kode-ngawur", "username": "ketua", "nama": "H. Ketua", "password": "password-awal-12345"}
	anon.minta("POST", "/api/publik/setup", isi, 422, nil) // kode salah
	anon.minta("POST", "/api/auth/login", map[string]string{"username": "ketua", "password": "password-awal-12345"}, 401, nil)

	isi["token"] = kode
	isi["password"] = "pendek"
	anon.minta("POST", "/api/publik/setup", isi, 422, nil) // isian tidak sah, kode tidak hangus

	isi["password"] = "password-awal-12345"
	var u struct {
		Role  string `json:"role"`
		Aktif bool   `json:"aktif"`
	}
	anon.minta("POST", "/api/publik/setup", isi, 201, &u)
	if u.Role != "ketua" || !u.Aktif {
		t.Fatalf("akun pertama = %+v", u)
	}

	// Ketua pertama bisa masuk dan langsung mengelola pengguna (membuat Admin).
	ketua := klienBaru(t, srv)
	ketua.minta("POST", "/api/auth/login", map[string]string{"username": "ketua", "password": "password-awal-12345"}, 200, nil)
	ketua.minta("POST", "/api/pengguna", map[string]any{"username": "petugas", "nama": "Petugas", "role": "admin", "password": "password-petugas-1"}, 201, nil)

	// Setup mati: status false, dan POST ulang (kode benar sekalipun) = 404.
	anon.minta("GET", "/api/publik/setup", nil, 200, &status)
	if status.Aktif {
		t.Error("setup masih aktif setelah akun pertama dibuat")
	}
	anon.minta("POST", "/api/publik/setup", map[string]any{"token": kode, "username": "penyusup", "nama": "Penyusup", "password": "password-penyusup-1"}, 404, nil)

	for _, rahasia := range []string{kode, "password-awal-12345", "password-petugas-1"} {
		if strings.Contains(log.String(), rahasia) {
			t.Errorf("rahasia %q tercatat di log server", rahasia)
		}
	}
}
