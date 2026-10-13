// Package migrate menjalankan migrasi SQL berversi ke MySQL.
//
// Sengaja kecil (tanpa library migrasi): baca file V<nomor>__<nama>.sql dari
// sebuah fs.FS, jalankan yang belum tercatat di tabel schema_migrations, dan
// catat checksum SHA-256 tiap file. Bila isi file yang sudah diterapkan
// berubah, Up menolak — aturan "migrasi tidak boleh diedit setelah deploy".
//
// Batas yang diketahui: DDL MySQL tidak transaksional (implicit commit), jadi
// migrasi yang gagal di tengah file bisa meninggalkan sebagian skema. Tulis
// satu perubahan per file dan perbaiki dengan file migrasi baru.
package migrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"github.com/go-sql-driver/mysql"
)

var (
	// ErrChecksum: file migrasi yang sudah diterapkan isinya berubah.
	ErrChecksum = errors.New("migrate: isi migrasi yang sudah diterapkan berubah")
	// ErrNamaFile: nama file tidak sesuai V<nomor>__<nama>.sql atau nomor dobel.
	ErrNamaFile = errors.New("migrate: nama file migrasi tidak valid")
)

// polaFile: V001__initial_schema.sql → versi "V001", nomor 1.
var polaFile = regexp.MustCompile(`^(V(\d+))__[a-z0-9_]+\.sql$`)

// kunciLock mencegah dua proses menjalankan migrasi bersamaan.
const kunciLock = "spkzakat_migrate"

type berkas struct {
	versi string
	nomor int
	path  string
}

// Up menjalankan semua migrasi di fsys yang belum diterapkan, berurutan
// menurut nomor, dan mengembalikan daftar versi yang BARU diterapkan.
//
// DSN dibuka dengan multiStatements=true khusus untuk migrasi (satu file
// berisi banyak statement). Koneksi aplikasi (store.Open) tidak memakainya,
// karena multi-statement memperbesar dampak SQL injection.
func Up(ctx context.Context, dsn string, fsys fs.FS) ([]string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		// err tidak disertakan: pesan DSN bisa memuat kredensial.
		return nil, errors.New("migrate: DSN tidak valid")
	}
	cfg.MultiStatements = true
	cfg.ParseTime = true
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("migrate: buat connector: %w", err)
	}
	db := sql.OpenDB(connector)
	defer db.Close()

	// Satu koneksi tetap: GET_LOCK terikat ke koneksi yang memintanya.
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate: konek ke MySQL: %w", err)
	}
	defer conn.Close()

	var got sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 30)", kunciLock).Scan(&got); err != nil {
		return nil, fmt.Errorf("migrate: ambil lock: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		return nil, errors.New("migrate: migrasi lain sedang berjalan (lock tidak didapat)")
	}
	defer func() {
		var lepas sql.NullInt64
		_ = conn.QueryRowContext(context.Background(), "SELECT RELEASE_LOCK(?)", kunciLock).Scan(&lepas)
	}()

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    VARCHAR(50) NOT NULL PRIMARY KEY,
		checksum   CHAR(64)    NOT NULL,
		applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
	)`); err != nil {
		return nil, fmt.Errorf("migrate: buat schema_migrations: %w", err)
	}

	terapkan, err := bacaTerapkan(ctx, conn)
	if err != nil {
		return nil, err
	}
	daftar, err := daftarBerkas(fsys)
	if err != nil {
		return nil, err
	}

	var baru []string
	for _, b := range daftar {
		isi, err := fs.ReadFile(fsys, b.path)
		if err != nil {
			return baru, fmt.Errorf("migrate: baca %s: %w", b.path, err)
		}
		sum := sha256.Sum256(isi)
		checksum := hex.EncodeToString(sum[:])

		if lama, ada := terapkan[b.versi]; ada {
			if lama != checksum {
				return baru, fmt.Errorf("%w: %s", ErrChecksum, b.versi)
			}
			continue
		}
		if _, err := conn.ExecContext(ctx, string(isi)); err != nil {
			return baru, fmt.Errorf("migrate: jalankan %s: %w", b.versi, err)
		}
		if _, err := conn.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, checksum) VALUES (?, ?)", b.versi, checksum); err != nil {
			return baru, fmt.Errorf("migrate: catat %s: %w", b.versi, err)
		}
		baru = append(baru, b.versi)
	}
	return baru, nil
}

// bacaTerapkan memuat versi → checksum yang sudah tercatat.
func bacaTerapkan(ctx context.Context, conn *sql.Conn) (map[string]string, error) {
	rows, err := conn.QueryContext(ctx, "SELECT version, checksum FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: baca schema_migrations: %w", err)
	}
	defer rows.Close()
	hasil := map[string]string{}
	for rows.Next() {
		var v, c string
		if err := rows.Scan(&v, &c); err != nil {
			return nil, fmt.Errorf("migrate: scan schema_migrations: %w", err)
		}
		hasil[v] = c
	}
	return hasil, rows.Err()
}

// daftarBerkas mengurutkan file V*.sql menurut nomor (bukan abjad) dan
// menolak nama yang salah atau nomor kembar, supaya salah ketik ketahuan
// sejak awal, bukan diam-diam terlewat.
func daftarBerkas(fsys fs.FS) ([]berkas, error) {
	entri, err := fs.Glob(fsys, "V*.sql")
	if err != nil {
		return nil, fmt.Errorf("migrate: baca daftar file: %w", err)
	}
	var daftar []berkas
	pakai := map[int]string{}
	for _, p := range entri {
		m := polaFile.FindStringSubmatch(p)
		if m == nil {
			return nil, fmt.Errorf("%w: %q (mau V<nomor>__<nama>.sql)", ErrNamaFile, p)
		}
		n, _ := strconv.Atoi(m[2])
		if lain, kembar := pakai[n]; kembar {
			return nil, fmt.Errorf("%w: nomor %d dipakai %q dan %q", ErrNamaFile, n, lain, p)
		}
		pakai[n] = p
		daftar = append(daftar, berkas{versi: m[1], nomor: n, path: p})
	}
	sort.Slice(daftar, func(i, j int) bool { return daftar[i].nomor < daftar[j].nomor })
	return daftar, nil
}
