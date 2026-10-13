package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// CAST ... AS SIGNED: DECIMAL(15,2) dibaca sebagai int64 rupiah utuh.
const kolomPeriode = `id, nama, jenis,
	CAST(dana_tersedia AS SIGNED) AS dana_tersedia,
	CAST(nominal_per_penerima AS SIGNED) AS nominal_per_penerima,
	asnaf_diizinkan, status, created_by, created_at, updated_at`

type periodeRow struct {
	ID        string         `db:"id"`
	Nama      string         `db:"nama"`
	Jenis     string         `db:"jenis"`
	Dana      int64          `db:"dana_tersedia"`
	Nominal   int64          `db:"nominal_per_penerima"`
	Asnaf     sql.NullString `db:"asnaf_diizinkan"`
	Status    string         `db:"status"`
	CreatedBy string         `db:"created_by"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func (r periodeRow) domain() domain.Periode {
	p := domain.Periode{
		ID: r.ID, Nama: r.Nama, Jenis: domain.JenisPeriode(r.Jenis),
		DanaTersedia: r.Dana, NominalPerPenerima: r.Nominal,
		Status: domain.StatusPeriode(r.Status), CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.Asnaf.Valid && r.Asnaf.String != "" {
		for _, a := range strings.Split(r.Asnaf.String, ",") {
			p.AsnafDiizinkan = append(p.AsnafDiizinkan, domain.Asnaf(a))
		}
	}
	return p
}

// CreatePeriode menyimpan periode baru berstatus draft dan mengisi ID, status,
// dan waktu pada p. Kolom SET asnaf_diizinkan: kosong → NULL (infak).
func (s *Store) CreatePeriode(ctx context.Context, p *domain.Periode) error {
	var asnaf any // nil → NULL
	if len(p.AsnafDiizinkan) > 0 {
		nama := make([]string, len(p.AsnafDiizinkan))
		for i, a := range p.AsnafDiizinkan {
			nama[i] = string(a)
		}
		asnaf = strings.Join(nama, ",")
	}
	id := newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO periode (id, nama, jenis, dana_tersedia, nominal_per_penerima, asnaf_diizinkan, status, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.Nama, string(p.Jenis), p.DanaTersedia, p.NominalPerPenerima, asnaf,
		string(domain.StatusDraft), p.CreatedBy)
	if err != nil {
		return fmt.Errorf("store.CreatePeriode: %w", mapErr(err))
	}
	dibuat, err := s.GetPeriode(ctx, id)
	if err != nil {
		return err
	}
	*p = dibuat
	return nil
}

// GetPeriode mengambil satu periode; ErrNotFound bila tidak ada.
func (s *Store) GetPeriode(ctx context.Context, id string) (domain.Periode, error) {
	var r periodeRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomPeriode+` FROM periode WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return domain.Periode{}, fmt.Errorf("store.GetPeriode: %w", mapErr(err))
	}
	return r.domain(), nil
}

// ListPeriode mengembalikan periode terbaru dulu. status "" = semua status.
func (s *Store) ListPeriode(ctx context.Context, status domain.StatusPeriode) ([]domain.Periode, error) {
	query := `SELECT ` + kolomPeriode + ` FROM periode WHERE deleted_at IS NULL`
	var args []any
	if status != "" {
		query += ` AND status = ?`
		args = append(args, string(status))
	}
	query += ` ORDER BY created_at DESC, id`

	var rows []periodeRow
	if err := sqlx.SelectContext(ctx, s.q, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("store.ListPeriode: %w", mapErr(err))
	}
	hasil := make([]domain.Periode, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}

// SetStatusPeriode memindahkan status dari `dari` ke `ke` secara atomik
// (compare-and-set): hanya berhasil bila status SAAT INI masih `dari`.
// Dua pengguna yang menekan tombol bersamaan tidak bisa sama-sama menang.
//   - periode tidak ada      → domain.ErrNotFound
//   - status bukan `dari`    → domain.ErrConflict
//
// Boleh/tidaknya transisi dari→ke diputuskan service, bukan di sini.
func (s *Store) SetStatusPeriode(ctx context.Context, id string, dari, ke domain.StatusPeriode) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE periode SET status = ? WHERE id = ? AND status = ? AND deleted_at IS NULL`,
		string(ke), id, string(dari))
	if err != nil {
		return fmt.Errorf("store.SetStatusPeriode: %w", mapErr(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store.SetStatusPeriode: %w", err)
	}
	if n == 1 {
		return nil
	}
	if _, err := s.GetPeriode(ctx, id); err != nil {
		return err // ErrNotFound
	}
	return fmt.Errorf("store.SetStatusPeriode %s: status bukan %q: %w", id, dari, domain.ErrConflict)
}

// ReplaceKandidat membuat daftar kandidat periode PERSIS sama dengan
// mustahikIDs, secara atomik. Hanya selisihnya yang disentuh: kandidat yang
// tetap dipertahankan TIDAK dihapus-lalu-dimasukkan ulang, karena menghapusnya
// akan ikut menghapus penilaiannya (ON DELETE CASCADE).
// ID mustahik/periode yang tidak ada → domain.ErrNotFound (FK), dan seluruh
// perubahan dibatalkan.
func (s *Store) ReplaceKandidat(ctx context.Context, periodeID string, mustahikIDs []string) error {
	return s.inTx(ctx, func(tx *Store) error {
		var lama []string
		if err := sqlx.SelectContext(ctx, tx.q, &lama,
			`SELECT mustahik_id FROM periode_kandidat WHERE periode_id = ?`, periodeID); err != nil {
			return fmt.Errorf("store.ReplaceKandidat: baca: %w", mapErr(err))
		}
		ada := make(map[string]bool, len(lama))
		for _, id := range lama {
			ada[id] = true
		}
		mau := make(map[string]bool, len(mustahikIDs))
		for _, id := range mustahikIDs {
			mau[id] = true
		}

		for _, id := range lama {
			if mau[id] {
				continue
			}
			if _, err := tx.q.ExecContext(ctx,
				`DELETE FROM periode_kandidat WHERE periode_id = ? AND mustahik_id = ?`, periodeID, id); err != nil {
				return fmt.Errorf("store.ReplaceKandidat: hapus: %w", mapErr(err))
			}
		}
		for _, id := range mustahikIDs { // urutan input → hasil deterministik
			if ada[id] {
				continue
			}
			ada[id] = true // jaga dari ID dobel di input
			if _, err := tx.q.ExecContext(ctx,
				`INSERT INTO periode_kandidat (periode_id, mustahik_id) VALUES (?, ?)`, periodeID, id); err != nil {
				return fmt.Errorf("store.ReplaceKandidat: tambah: %w", mapErr(err))
			}
		}
		return nil
	})
}

// ListKandidat mengembalikan mustahik yang menjadi kandidat periode, urut nama.
func (s *Store) ListKandidat(ctx context.Context, periodeID string) ([]domain.Mustahik, error) {
	var rows []mustahikRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT `+kolomMustahik+`
		   FROM mustahik
		  WHERE deleted_at IS NULL
		    AND id IN (SELECT mustahik_id FROM periode_kandidat WHERE periode_id = ?)
		  ORDER BY nama, id`, periodeID)
	if err != nil {
		return nil, fmt.Errorf("store.ListKandidat: %w", mapErr(err))
	}
	hasil := make([]domain.Mustahik, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}
