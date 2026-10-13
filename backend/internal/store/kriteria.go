package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

const kolomKriteria = `id, kode, nama, tipe, bobot, skala_keterangan, urutan, is_active`

type kriteriaRow struct {
	ID     string  `db:"id"`
	Kode   string  `db:"kode"`
	Nama   string  `db:"nama"`
	Tipe   string  `db:"tipe"`
	Bobot  float64 `db:"bobot"`
	Skala  []byte  `db:"skala_keterangan"`
	Urutan int     `db:"urutan"`
	Aktif  bool    `db:"is_active"`
}

func (r kriteriaRow) domain() (domain.Kriteria, error) {
	k := domain.Kriteria{
		ID: r.ID, Kode: r.Kode, Nama: r.Nama, Tipe: domain.TipeKriteria(r.Tipe),
		Bobot: r.Bobot, Urutan: r.Urutan, Aktif: r.Aktif,
	}
	if len(r.Skala) > 0 {
		if err := json.Unmarshal(r.Skala, &k.Skala); err != nil {
			return domain.Kriteria{}, fmt.Errorf("skala_keterangan %s tidak valid: %w", r.Kode, err)
		}
	}
	return k, nil
}

// ListKriteria mengembalikan kriteria menurut kolom `urutan`.
// hanyaAktif=true dipakai saat menghitung ARAS (kriteria nonaktif diabaikan).
func (s *Store) ListKriteria(ctx context.Context, hanyaAktif bool) ([]domain.Kriteria, error) {
	query := `SELECT ` + kolomKriteria + ` FROM kriteria WHERE deleted_at IS NULL`
	if hanyaAktif {
		query += ` AND is_active = TRUE`
	}
	query += ` ORDER BY urutan, kode`

	var rows []kriteriaRow
	if err := sqlx.SelectContext(ctx, s.q, &rows, query); err != nil {
		return nil, fmt.Errorf("store.ListKriteria: %w", mapErr(err))
	}
	hasil := make([]domain.Kriteria, 0, len(rows))
	for _, r := range rows {
		k, err := r.domain()
		if err != nil {
			return nil, fmt.Errorf("store.ListKriteria: %w", err)
		}
		hasil = append(hasil, k)
	}
	return hasil, nil
}

// GetKriteria mengambil satu kriteria; ErrNotFound bila tidak ada.
func (s *Store) GetKriteria(ctx context.Context, id string) (domain.Kriteria, error) {
	var r kriteriaRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomKriteria+` FROM kriteria WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return domain.Kriteria{}, fmt.Errorf("store.GetKriteria: %w", mapErr(err))
	}
	k, err := r.domain()
	if err != nil {
		return domain.Kriteria{}, fmt.Errorf("store.GetKriteria: %w", err)
	}
	return k, nil
}

// UpdateKriteria mengubah tipe dan bobot. Aturan "Σ bobot = 100%" ada di
// service (butuh melihat semua kriteria); panggil di dalam WithTx supaya
// pembacaan dan penulisan konsisten.
func (s *Store) UpdateKriteria(ctx context.Context, id string, tipe domain.TipeKriteria, bobot float64) error {
	// Bobot dikirim sebagai teks 4 desimal agar pembulatan ke DECIMAL(5,4)
	// dilakukan di sini (eksplisit), bukan bergantung pada perilaku server.
	// Catatan hasil uji: MySQL sendiri juga membulatkan 0.30000000000000004
	// ke 0.3000, jadi ini bukan penambal bug — hanya membuat maksudnya jelas.
	res, err := s.q.ExecContext(ctx,
		`UPDATE kriteria SET tipe = ?, bobot = ? WHERE id = ? AND deleted_at IS NULL`,
		string(tipe), strconv.FormatFloat(bobot, 'f', 4, 64), id)
	if err != nil {
		return fmt.Errorf("store.UpdateKriteria: %w", mapErr(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store.UpdateKriteria: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("store.UpdateKriteria %s: %w", id, domain.ErrNotFound)
	}
	return nil
}
