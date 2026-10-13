package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// CreatePersetujuan mencatat keputusan Ketua atas satu run. Satu run hanya
// bisa diputuskan sekali (UNIQUE run_id) → domain.ErrConflict bila diulang.
func (s *Store) CreatePersetujuan(ctx context.Context, p domain.Persetujuan) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO persetujuan (id, run_id, keputusan, catatan, oleh) VALUES (?, ?, ?, ?, ?)`,
		newID(), p.RunID, string(p.Keputusan), nullStr(p.Catatan), p.Oleh)
	if err != nil {
		return fmt.Errorf("store.CreatePersetujuan: %w", mapErr(err))
	}
	return nil
}

type persetujuanRow struct {
	RunID     string         `db:"run_id"`
	Versi     int            `db:"versi"`
	Keputusan string         `db:"keputusan"`
	Catatan   sql.NullString `db:"catatan"`
	Oleh      string         `db:"oleh"`
	OlehNama  string         `db:"oleh_nama"`
	CreatedAt time.Time      `db:"created_at"`
}

// ListPersetujuan mengembalikan semua keputusan pada periode, terbaru dulu
// (untuk "catatan penolakan" dan nama Ketua di laporan).
func (s *Store) ListPersetujuan(ctx context.Context, periodeID string) ([]domain.Persetujuan, error) {
	var rows []persetujuanRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT p.run_id, r.versi, p.keputusan, p.catatan, p.oleh, u.nama AS oleh_nama, p.created_at
		   FROM persetujuan p
		   JOIN aras_run r ON r.id = p.run_id
		   JOIN users u ON u.id = p.oleh
		  WHERE r.periode_id = ?
		  ORDER BY p.created_at DESC, r.versi DESC`, periodeID)
	if err != nil {
		return nil, fmt.Errorf("store.ListPersetujuan: %w", mapErr(err))
	}
	hasil := make([]domain.Persetujuan, len(rows))
	for i, r := range rows {
		hasil[i] = domain.Persetujuan{
			RunID: r.RunID, Versi: r.Versi, Keputusan: domain.Keputusan(r.Keputusan), Catatan: r.Catatan.String,
			Oleh: r.Oleh, OlehNama: r.OlehNama, CreatedAt: r.CreatedAt,
		}
	}
	return hasil, nil
}

// CreatePenyaluran mencatat realisasi untuk satu penerima dan mengisi p.ID.
// Penerima yang sudah tercatat pada periode yang sama → domain.ErrConflict
// (UNIQUE periode_id, mustahik_id).
func (s *Store) CreatePenyaluran(ctx context.Context, p *domain.Penyaluran) error {
	id := newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO penyaluran (id, periode_id, mustahik_id, nominal, tanggal, dicatat_oleh) VALUES (?, ?, ?, ?, ?, ?)`,
		id, p.PeriodeID, p.MustahikID, p.Nominal, p.Tanggal.Format(time.DateOnly), p.DicatatOleh)
	if err != nil {
		return fmt.Errorf("store.CreatePenyaluran: %w", mapErr(err))
	}
	p.ID = id
	return nil
}

type penyaluranRow struct {
	ID          string    `db:"id"`
	PeriodeID   string    `db:"periode_id"`
	MustahikID  string    `db:"mustahik_id"`
	Nominal     int64     `db:"nominal"`
	Tanggal     time.Time `db:"tanggal"`
	DicatatOleh string    `db:"dicatat_oleh"`
	CreatedAt   time.Time `db:"created_at"`
}

// TotalPenyaluran menjumlah seluruh realisasi dari semua periode yang tidak
// dihapus — angka "Total tersalur" di Beranda saat tidak ada periode aktif.
func (s *Store) TotalPenyaluran(ctx context.Context) (int64, error) {
	var total int64
	err := sqlx.GetContext(ctx, s.q, &total,
		`SELECT CAST(COALESCE(SUM(py.nominal), 0) AS SIGNED)
		   FROM penyaluran py JOIN periode p ON p.id = py.periode_id
		  WHERE p.deleted_at IS NULL`)
	if err != nil {
		return 0, fmt.Errorf("store.TotalPenyaluran: %w", mapErr(err))
	}
	return total, nil
}

// ListPenyaluran mengembalikan realisasi periode menurut urutan pencatatan.
func (s *Store) ListPenyaluran(ctx context.Context, periodeID string) ([]domain.Penyaluran, error) {
	var rows []penyaluranRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT id, periode_id, mustahik_id, CAST(nominal AS SIGNED) AS nominal, tanggal, dicatat_oleh, created_at
		   FROM penyaluran WHERE periode_id = ? ORDER BY created_at, id`, periodeID)
	if err != nil {
		return nil, fmt.Errorf("store.ListPenyaluran: %w", mapErr(err))
	}
	hasil := make([]domain.Penyaluran, len(rows))
	for i, r := range rows {
		hasil[i] = domain.Penyaluran(r)
	}
	return hasil, nil
}
