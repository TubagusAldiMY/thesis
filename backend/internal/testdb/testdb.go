// Package testdb menyediakan database MySQL sementara untuk test integrasi.
//
// Tiap pemanggilan membuat database baru `spkzakat_test_<acak>` di server yang
// ditunjuk TEST_DATABASE_URL dan menghapusnya saat test selesai, sehingga test
// terisolasi satu sama lain dan tidak pernah menyentuh database lain di server.
// Bila TEST_DATABASE_URL kosong, test dilewati (t.Skip) — jadi `go test ./...`
// tetap jalan di mesin tanpa MySQL.
package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"spkzakat/internal/migrate"
	"spkzakat/migrations"
)

// New membuat database kosong dan mengembalikan DSN-nya.
func New(t testing.TB) string {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL kosong: test integrasi MySQL dilewati (lihat how-to-run.md)")
	}
	cfg, err := mysql.ParseDSN(raw)
	if err != nil {
		t.Fatal("TEST_DATABASE_URL tidak valid") // err tidak dicetak: bisa memuat password
	}
	cfg.DBName = ""

	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatalf("buka koneksi admin: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	acak := make([]byte, 6)
	if _, err := rand.Read(acak); err != nil {
		t.Fatal(err)
	}
	// Nama hanya berisi huruf kecil, angka, dan underscore (hasil hex), jadi
	// aman disisipkan ke SQL — identifier memang tidak bisa diparameterisasi.
	nama := "spkzakat_test_" + hex.EncodeToString(acak)
	if _, err := admin.ExecContext(ctx,
		"CREATE DATABASE `"+nama+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		admin.Close()
		t.Fatalf("buat database uji (server MySQL hidup?): %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE `"+nama+"`")
		admin.Close()
	})

	cfg.DBName = nama
	return cfg.FormatDSN()
}

// Migrated sama seperti New, lalu menerapkan semua migrasi asli
// (skema + seed kriteria).
func Migrated(t testing.TB) string {
	t.Helper()
	dsn := New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if _, err := migrate.Up(ctx, dsn, migrations.FS); err != nil {
		t.Fatalf("migrasi gagal: %v", err)
	}
	return dsn
}
