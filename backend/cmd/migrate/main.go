// Command migrate menerapkan migrasi SQL ke database di DATABASE_URL.
//
//	go run ./cmd/migrate
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"spkzakat/internal/migrate"
	"spkzakat/migrations"
)

func main() {
	// Log JSON ke stdout (aturan logging project). DSN sengaja tidak pernah dicatat.
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL wajib diset (lihat .env.example)")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	baru, err := migrate.Up(ctx, dsn, migrations.FS)
	if err != nil {
		log.Error("migrasi gagal", "error", err.Error(), "diterapkan_sebelum_gagal", baru)
		os.Exit(1)
	}
	if baru == nil {
		baru = []string{} // tercatat sebagai [] di log JSON, bukan null
	}
	log.Info("migrasi selesai", "diterapkan", baru, "jumlah", len(baru))
}
