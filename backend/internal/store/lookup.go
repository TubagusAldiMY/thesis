package store

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// File ini berisi query bantu yang dibutuhkan service untuk memeriksa aturan
// bisnis sebelum menulis.

// GetMustahikByIDs mengambil banyak mustahik sekaligus (satu query, bukan N).
// ID yang tidak ada atau sudah terhapus diam-diam TIDAK ikut dalam hasil —
// pemanggil membandingkan sendiri untuk menemukan yang hilang. Urutan hasil
// tidak dijamin sama dengan input.
func (s *Store) GetMustahikByIDs(ctx context.Context, ids []string) ([]domain.Mustahik, error) {
	if len(ids) == 0 {
		return nil, nil // sqlx.In menolak slice kosong; tidak ada yang perlu dicari
	}
	query, args, err := sqlx.In(
		`SELECT `+kolomMustahik+` FROM mustahik WHERE deleted_at IS NULL AND id IN (?)`, ids)
	if err != nil {
		return nil, fmt.Errorf("store.GetMustahikByIDs: %w", err)
	}
	var rows []mustahikRow
	if err := sqlx.SelectContext(ctx, s.q, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("store.GetMustahikByIDs: %w", mapErr(err))
	}
	hasil := make([]domain.Mustahik, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}

// MustahikDalamPeriodeBerjalan: true bila mustahik menjadi kandidat pada
// periode berstatus dihitung/disetujui — saat itu ranking/penerimanya sedang
// diputuskan, sehingga mustahik tidak boleh dinonaktifkan di tengah jalan.
func (s *Store) MustahikDalamPeriodeBerjalan(ctx context.Context, mustahikID string) (bool, error) {
	var ada bool
	err := sqlx.GetContext(ctx, s.q, &ada,
		`SELECT EXISTS(
		   SELECT 1 FROM periode_kandidat pk
		     JOIN periode p ON p.id = pk.periode_id
		    WHERE pk.mustahik_id = ?
		      AND p.status IN ('dihitung', 'disetujui')
		      AND p.deleted_at IS NULL)`, mustahikID)
	if err != nil {
		return false, fmt.Errorf("store.MustahikDalamPeriodeBerjalan: %w", mapErr(err))
	}
	return ada, nil
}

// GetPeriodeForUpdate seperti GetPeriode tetapi MENGUNCI barisnya
// (SELECT ... FOR UPDATE) sampai transaksi selesai, sehingga status periode
// tidak bisa berubah di antara "cek status" dan "tulis" milik transaksi yang
// sama. Hanya bermakna di dalam WithTx; di luar transaksi kuncinya langsung
// dilepas begitu query selesai.
func (s *Store) GetPeriodeForUpdate(ctx context.Context, id string) (domain.Periode, error) {
	var r periodeRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomPeriode+` FROM periode WHERE id = ? AND deleted_at IS NULL FOR UPDATE`, id)
	if err != nil {
		return domain.Periode{}, fmt.Errorf("store.GetPeriodeForUpdate: %w", mapErr(err))
	}
	return r.domain(), nil
}
