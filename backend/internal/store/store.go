// Package store adalah lapisan akses data MySQL (repository). Isinya SQL
// berparameter saja: tanpa logika bisnis, tanpa enkripsi (NIK/HP sudah
// terenkripsi dari service), tanpa logging data pribadi.
//
// Satu tipe Store dipakai untuk koneksi biasa dan untuk transaksi:
//
//	err := s.WithTx(ctx, func(tx *store.Store) error {
//	    // semua pemanggilan lewat tx ikut satu transaksi
//	})
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// Store membungkus koneksi (atau transaksi). db nil berarti Store ini milik
// sebuah transaksi.
type Store struct {
	q  sqlx.ExtContext
	db *sqlx.DB
}

// Open membuka pool koneksi dan memastikan server menjawab. Pengaturan penting:
//   - parseTime + UTC: DATETIME dibaca sebagai time.Time UTC; time_zone sesi
//     dipaksa UTC supaya CURRENT_TIMESTAMP(3) konsisten di mana pun server berjalan.
//   - clientFoundRows: RowsAffected = baris yang COCOK, bukan yang berubah.
//     Tanpa ini UPDATE dengan nilai sama dikira "tidak ditemukan".
//   - multiStatements MATI: koneksi aplikasi tidak boleh menjalankan beberapa
//     statement sekaligus (memperkecil dampak SQL injection). Hanya migrasi
//     yang membukanya.
func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		// err tidak disertakan: pesannya bisa memuat kredensial.
		return nil, errors.New("store: DSN tidak valid")
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.ClientFoundRows = true
	cfg.MultiStatements = false
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["time_zone"] = "'+00:00'"

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("store: buat connector: %w", err)
	}
	db := sqlx.NewDb(sql.OpenDB(connector), "mysql")
	// Angka dari docs/db-patterns (pool 25/5, umur koneksi 5 menit).
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping MySQL: %w", err)
	}
	return &Store{q: db, db: db}, nil
}

// Close menutup pool. Aman dipanggil pada Store milik transaksi (tidak berbuat apa-apa).
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Ping untuk health check (`/healthz`).
func (s *Store) Ping(ctx context.Context) error {
	if s.db == nil {
		return errors.New("store: Ping hanya untuk Store utama")
	}
	return s.db.PingContext(ctx)
}

// WithTx menjalankan fn dalam satu transaksi: commit bila fn mengembalikan nil,
// rollback bila error atau panic. Transaksi bersarang tidak didukung.
func (s *Store) WithTx(ctx context.Context, fn func(tx *Store) error) error {
	if s.db == nil {
		return errors.New("store: transaksi bersarang tidak didukung")
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: mulai transaksi: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(&Store{q: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit: %w", err)
	}
	return nil
}

// inTx dipakai operasi multi-langkah: ikut transaksi pemanggil bila sudah ada,
// kalau belum membuat sendiri — jadi hasilnya selalu atomik.
func (s *Store) inTx(ctx context.Context, fn func(tx *Store) error) error {
	if s.db == nil {
		return fn(s)
	}
	return s.WithTx(ctx, fn)
}

// mapErr menerjemahkan error database ke error domain supaya lapisan atas
// tidak perlu tahu nomor error MySQL. Error asli tetap bisa dicek (errors.As).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %w", domain.ErrNotFound, err)
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		switch me.Number {
		case 1062: // duplicate entry (UNIQUE)
			return fmt.Errorf("%w: %w", domain.ErrConflict, err)
		case 1451: // tidak bisa menghapus/mengubah: masih dirujuk baris lain
			return fmt.Errorf("%w: %w", domain.ErrConflict, err)
		case 1452: // baris yang dirujuk (FK) tidak ada
			return fmt.Errorf("%w: %w", domain.ErrNotFound, err)
		}
	}
	return err
}

// newID membuat UUID v4 (acak). Dibuat di aplikasi, bukan DEFAULT (UUID())
// di database, supaya ID sudah diketahui sebelum INSERT dan tidak perlu
// SELECT ulang untuk mengambilnya.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("store: crypto/rand gagal: " + err.Error()) // tidak mungkin pulih; jangan lanjut dengan ID lemah
	}
	b[6] = (b[6] & 0x0f) | 0x40 // versi 4
	b[8] = (b[8] & 0x3f) | 0x80 // varian RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
