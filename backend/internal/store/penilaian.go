package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

type penilaianRow struct {
	MustahikID string  `db:"mustahik_id"`
	KriteriaID string  `db:"kriteria_id"`
	Nilai      float64 `db:"nilai"`
}

// UpsertPenilaian menyimpan nilai-nilai sekaligus (atomik): sel yang belum ada
// ditambah, yang sudah ada ditimpa. Sel lain yang tidak disebut dibiarkan,
// jadi penyimpanan sebagian (formulir belum lengkap) aman.
// Mustahik yang bukan kandidat periode → domain.ErrNotFound (FK komposit).
func (s *Store) UpsertPenilaian(ctx context.Context, periodeID string, sel []domain.Penilaian) error {
	if len(sel) == 0 {
		return nil
	}
	return s.inTx(ctx, func(tx *Store) error {
		for _, p := range sel {
			// Nilai dikirim 4 desimal agar pembulatan ke DECIMAL(15,4) eksplisit.
			// Sintaks "AS baru" (MySQL ≥ 8.0.19) menggantikan VALUES() yang usang.
			_, err := tx.q.ExecContext(ctx,
				`INSERT INTO penilaian (id, periode_id, mustahik_id, kriteria_id, nilai)
				 VALUES (?, ?, ?, ?, ?) AS baru
				 ON DUPLICATE KEY UPDATE nilai = baru.nilai`,
				newID(), periodeID, p.MustahikID, p.KriteriaID, strconv.FormatFloat(p.Nilai, 'f', 4, 64))
			if err != nil {
				return fmt.Errorf("store.UpsertPenilaian: %w", mapErr(err))
			}
		}
		return nil
	})
}

// ListPenilaian mengembalikan semua nilai yang sudah tersimpan untuk periode.
func (s *Store) ListPenilaian(ctx context.Context, periodeID string) ([]domain.Penilaian, error) {
	var rows []penilaianRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT mustahik_id, kriteria_id, nilai FROM penilaian
		  WHERE periode_id = ? ORDER BY mustahik_id, kriteria_id`, periodeID)
	if err != nil {
		return nil, fmt.Errorf("store.ListPenilaian: %w", mapErr(err))
	}
	hasil := make([]domain.Penilaian, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, domain.Penilaian{MustahikID: r.MustahikID, KriteriaID: r.KriteriaID, Nilai: r.Nilai})
	}
	return hasil, nil
}
