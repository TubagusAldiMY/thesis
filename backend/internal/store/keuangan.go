package store

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// Agregat keuangan untuk Beranda. Semua hanya MEMBACA dan menjumlah; rupiah utuh
// (int64) seperti bagian lain. ponytail: GROUP BY di atas fungsi tanggal tidak
// memakai indeks; cukup untuk ribuan baris per tahun, tambahkan indeks bila lambat.

// jumlahPer menjalankan query "label, jumlah" lalu mengubahnya menjadi peta.
func jumlahPer(ctx context.Context, q sqlx.QueryerContext, nama, query string, args ...any) (map[string]int64, error) {
	var rows []struct {
		Label  string `db:"label"`
		Jumlah int64  `db:"jumlah"`
	}
	if err := sqlx.SelectContext(ctx, q, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("store.%s: %w", nama, mapErr(err))
	}
	hasil := make(map[string]int64, len(rows))
	for _, r := range rows {
		hasil[r.Label] = r.Jumlah
	}
	return hasil, nil
}

// PenerimaanPerJenis = total donasi berstatus diterima per jenis (semua waktu).
// Donasi menunggu/gagal/kedaluwarsa/dibatalkan tidak dihitung.
func (s *Store) PenerimaanPerJenis(ctx context.Context) (map[domain.JenisDonasi]int64, error) {
	m, err := jumlahPer(ctx, s.q, "PenerimaanPerJenis",
		`SELECT jenis AS label, CAST(SUM(jumlah) AS SIGNED) AS jumlah
		   FROM donasi WHERE status = 'diterima' AND deleted_at IS NULL GROUP BY jenis`)
	if err != nil {
		return nil, err
	}
	hasil := make(map[domain.JenisDonasi]int64, len(m))
	for k, v := range m {
		hasil[domain.JenisDonasi(k)] = v
	}
	return hasil, nil
}

// PenyaluranPerJenis = total realisasi per jenis dana periode (zakat/infak),
// supaya dana zakat dan infak dibukukan terpisah.
func (s *Store) PenyaluranPerJenis(ctx context.Context) (map[domain.JenisPeriode]int64, error) {
	m, err := jumlahPer(ctx, s.q, "PenyaluranPerJenis",
		`SELECT p.jenis AS label, CAST(SUM(py.nominal) AS SIGNED) AS jumlah
		   FROM penyaluran py JOIN periode p ON p.id = py.periode_id
		  WHERE p.deleted_at IS NULL GROUP BY p.jenis`)
	if err != nil {
		return nil, err
	}
	hasil := make(map[domain.JenisPeriode]int64, len(m))
	for k, v := range m {
		hasil[domain.JenisPeriode(k)] = v
	}
	return hasil, nil
}

// JumlahPenerimaManfaat = banyaknya mustahik berbeda yang pernah menerima
// penyaluran (satu orang yang menerima di dua periode dihitung sekali).
func (s *Store) JumlahPenerimaManfaat(ctx context.Context) (int, error) {
	var n int
	err := sqlx.GetContext(ctx, s.q, &n,
		`SELECT COUNT(DISTINCT py.mustahik_id)
		   FROM penyaluran py JOIN periode p ON p.id = py.periode_id
		  WHERE p.deleted_at IS NULL`)
	if err != nil {
		return 0, fmt.Errorf("store.JumlahPenerimaManfaat: %w", mapErr(err))
	}
	return n, nil
}

// PenerimaanPerBulan = donasi diterima per bulan kalender WIB ("2026-10"),
// mulai bulan `dari` ("2026-01") sampai sekarang. dibayar_pada disimpan UTC,
// jadi digeser +7 jam supaya donasi pukul 23.30 WIB masuk bulan yang benar.
func (s *Store) PenerimaanPerBulan(ctx context.Context, dari string) (map[string]int64, error) {
	return jumlahPer(ctx, s.q, "PenerimaanPerBulan",
		`SELECT DATE_FORMAT(DATE_ADD(dibayar_pada, INTERVAL 7 HOUR), '%Y-%m') AS label,
		        CAST(SUM(jumlah) AS SIGNED) AS jumlah
		   FROM donasi
		  WHERE status = 'diterima' AND deleted_at IS NULL AND dibayar_pada IS NOT NULL
		    AND DATE_FORMAT(DATE_ADD(dibayar_pada, INTERVAL 7 HOUR), '%Y-%m') >= ?
		  GROUP BY label`, dari)
}

// PenyaluranPerBulan = realisasi per bulan menurut tanggal penyaluran (DATE, tanpa zona).
func (s *Store) PenyaluranPerBulan(ctx context.Context, dari string) (map[string]int64, error) {
	return jumlahPer(ctx, s.q, "PenyaluranPerBulan",
		`SELECT DATE_FORMAT(py.tanggal, '%Y-%m') AS label, CAST(SUM(py.nominal) AS SIGNED) AS jumlah
		   FROM penyaluran py JOIN periode p ON p.id = py.periode_id
		  WHERE p.deleted_at IS NULL AND DATE_FORMAT(py.tanggal, '%Y-%m') >= ?
		  GROUP BY label`, dari)
}
