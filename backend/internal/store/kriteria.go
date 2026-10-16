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

// skalaJSON mengubah skala menjadi JSON untuk kolom skala_keterangan; skala
// kosong/nil disimpan sebagai NULL (= kriteria bernilai asli, bukan berskala).
func skalaJSON(skala map[string]string) (any, error) {
	if len(skala) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(skala)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// UpdateKriteriaPenuh mengganti nama, tipe, bobot, skala, dan status aktif satu
// kriteria (kode dan urutan tidak berubah: kode dipakai sebagai kunci penilaian
// dan potret run). Seperti UpdateKriteria, aturan lintas-kriteria (Σ bobot,
// jumlah aktif, nama unik) ada di service, dalam transaksi yang sama.
func (s *Store) UpdateKriteriaPenuh(ctx context.Context, id, nama string, tipe domain.TipeKriteria, bobot float64, skala map[string]string, aktif bool) error {
	sk, err := skalaJSON(skala)
	if err != nil {
		return fmt.Errorf("store.UpdateKriteriaPenuh: skala: %w", err)
	}
	res, err := s.q.ExecContext(ctx,
		`UPDATE kriteria SET nama = ?, tipe = ?, bobot = ?, skala_keterangan = ?, is_active = ?
		  WHERE id = ? AND deleted_at IS NULL`,
		nama, string(tipe), strconv.FormatFloat(bobot, 'f', 4, 64), sk, aktif, id)
	if err != nil {
		return fmt.Errorf("store.UpdateKriteriaPenuh: %w", mapErr(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store.UpdateKriteriaPenuh: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("store.UpdateKriteriaPenuh %s: %w", id, domain.ErrNotFound)
	}
	return nil
}

// CreateKriteria menambah kriteria aktif di urutan terakhir. Kode berikutnya
// ("C6", "C7", ...) dihitung dari kode terbesar, TERMASUK baris terhapus, karena
// UNIQUE(kode) berlaku atas semua baris. Panggil di dalam WithTx.
func (s *Store) CreateKriteria(ctx context.Context, nama string, tipe domain.TipeKriteria, bobot float64, skala map[string]string) (domain.Kriteria, error) {
	var nx struct {
		Nomor  int `db:"nomor"`
		Urutan int `db:"urutan"`
	}
	err := sqlx.GetContext(ctx, s.q, &nx,
		`SELECT COALESCE(MAX(CAST(SUBSTRING(kode, 2) AS UNSIGNED)), 0) + 1 AS nomor,
		        COALESCE(MAX(urutan), 0) + 1 AS urutan
		   FROM kriteria`)
	if err != nil {
		return domain.Kriteria{}, fmt.Errorf("store.CreateKriteria: kode berikutnya: %w", mapErr(err))
	}
	sk, err := skalaJSON(skala)
	if err != nil {
		return domain.Kriteria{}, fmt.Errorf("store.CreateKriteria: skala: %w", err)
	}
	id := newID()
	_, err = s.q.ExecContext(ctx,
		`INSERT INTO kriteria (id, kode, nama, tipe, bobot, skala_keterangan, urutan, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, TRUE)`,
		id, "C"+strconv.Itoa(nx.Nomor), nama, string(tipe), strconv.FormatFloat(bobot, 'f', 4, 64), sk, nx.Urutan)
	if err != nil {
		return domain.Kriteria{}, fmt.Errorf("store.CreateKriteria: %w", mapErr(err))
	}
	return s.GetKriteria(ctx, id)
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
