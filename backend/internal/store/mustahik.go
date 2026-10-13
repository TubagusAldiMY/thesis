package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

const kolomMustahik = `id, nama, nik_enc, nik_hash, alamat, no_hp_enc, asnaf, is_active, created_at, updated_at`

type mustahikRow struct {
	ID        string    `db:"id"`
	Nama      string    `db:"nama"`
	NIKEnc    []byte    `db:"nik_enc"`
	NIKHash   string    `db:"nik_hash"`
	Alamat    string    `db:"alamat"`
	NoHPEnc   []byte    `db:"no_hp_enc"`
	Asnaf     string    `db:"asnaf"`
	Aktif     bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r mustahikRow) domain() domain.Mustahik {
	return domain.Mustahik{
		ID: r.ID, Nama: r.Nama, NIKEnc: r.NIKEnc, NIKHash: r.NIKHash, Alamat: r.Alamat,
		NoHPEnc: r.NoHPEnc, Asnaf: domain.Asnaf(r.Asnaf), Aktif: r.Aktif,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// MustahikFilter untuk ListMustahik. Nilai nol = tanpa filter.
type MustahikFilter struct {
	Q       string       // cari pada nama (potongan, tak peka huruf besar/kecil)
	Asnaf   domain.Asnaf // "" = semua golongan
	Aktif   *bool        // nil = aktif dan nonaktif
	Page    int          // mulai 1; <1 dianggap 1
	PerPage int          // bawaan 20, maksimal 100
}

// MustahikList = satu halaman hasil + total + jumlah mustahik AKTIF per
// golongan (untuk chip filter dinamis; tidak terpengaruh filter di atas).
type MustahikList struct {
	Items       []domain.Mustahik
	Total       int
	AsnafCounts map[domain.Asnaf]int
}

// escapeLike membuat %, _, dan \ dari input pengguna dibaca harfiah oleh LIKE
// (kalau tidak, mencari "%" akan cocok dengan semua baris).
var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// where merakit kondisi dari bagian SQL TETAP; nilai dari pengguna hanya
// lewat parameter `?`, tidak pernah disambung ke string query.
func (f MustahikFilter) where() (string, []any) {
	cond := []string{"deleted_at IS NULL"}
	var args []any
	if f.Q != "" {
		cond = append(cond, "nama LIKE ?")
		args = append(args, "%"+escapeLike.Replace(f.Q)+"%")
	}
	if f.Asnaf != "" {
		cond = append(cond, "asnaf = ?")
		args = append(args, string(f.Asnaf))
	}
	if f.Aktif != nil {
		cond = append(cond, "is_active = ?")
		args = append(args, *f.Aktif)
	}
	return strings.Join(cond, " AND "), args
}

// CreateMustahik menyimpan mustahik baru, mengisi ID dan waktu pada m.
// NIK ganda → domain.ErrConflict (lewat UNIQUE nik_hash).
func (s *Store) CreateMustahik(ctx context.Context, m *domain.Mustahik) error {
	id := newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO mustahik (id, nama, nik_enc, nik_hash, alamat, no_hp_enc, asnaf, is_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, m.Nama, m.NIKEnc, m.NIKHash, m.Alamat, m.NoHPEnc, string(m.Asnaf), m.Aktif)
	if err != nil {
		return fmt.Errorf("store.CreateMustahik: %w", mapErr(err))
	}
	dibuat, err := s.GetMustahik(ctx, id)
	if err != nil {
		return err
	}
	*m = dibuat
	return nil
}

// GetMustahik mengambil satu mustahik; ErrNotFound bila tidak ada/terhapus.
func (s *Store) GetMustahik(ctx context.Context, id string) (domain.Mustahik, error) {
	var r mustahikRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomMustahik+` FROM mustahik WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return domain.Mustahik{}, fmt.Errorf("store.GetMustahik: %w", mapErr(err))
	}
	return r.domain(), nil
}

// FindMustahikByNIKHash untuk pengecekan duplikat dengan pesan yang ramah;
// ErrNotFound berarti NIK belum terdaftar.
func (s *Store) FindMustahikByNIKHash(ctx context.Context, nikHash string) (domain.Mustahik, error) {
	var r mustahikRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomMustahik+` FROM mustahik WHERE nik_hash = ? AND deleted_at IS NULL`, nikHash)
	if err != nil {
		return domain.Mustahik{}, fmt.Errorf("store.FindMustahikByNIKHash: %w", mapErr(err))
	}
	return r.domain(), nil
}

// UpdateMustahik mengganti semua kolom yang bisa diubah (termasuk Aktif —
// "menghapus" mustahik di UI = menonaktifkan, tidak ada hapus).
func (s *Store) UpdateMustahik(ctx context.Context, m domain.Mustahik) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE mustahik
		    SET nama = ?, nik_enc = ?, nik_hash = ?, alamat = ?, no_hp_enc = ?, asnaf = ?, is_active = ?
		  WHERE id = ? AND deleted_at IS NULL`,
		m.Nama, m.NIKEnc, m.NIKHash, m.Alamat, m.NoHPEnc, string(m.Asnaf), m.Aktif, m.ID)
	if err != nil {
		return fmt.Errorf("store.UpdateMustahik: %w", mapErr(err))
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("store.UpdateMustahik: %w", err)
	} else if n == 0 {
		return fmt.Errorf("store.UpdateMustahik %s: %w", m.ID, domain.ErrNotFound)
	}
	return nil
}

// ListMustahik mengembalikan satu halaman (urut nama), total yang cocok
// filter, dan jumlah mustahik aktif per golongan.
func (s *Store) ListMustahik(ctx context.Context, f MustahikFilter) (MustahikList, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}
	kondisi, args := f.where()

	var total int
	if err := sqlx.GetContext(ctx, s.q, &total, `SELECT COUNT(*) FROM mustahik WHERE `+kondisi, args...); err != nil {
		return MustahikList{}, fmt.Errorf("store.ListMustahik: hitung: %w", mapErr(err))
	}

	var rows []mustahikRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT `+kolomMustahik+` FROM mustahik WHERE `+kondisi+` ORDER BY nama, id LIMIT ? OFFSET ?`,
		append(args, f.PerPage, (f.Page-1)*f.PerPage)...)
	if err != nil {
		return MustahikList{}, fmt.Errorf("store.ListMustahik: %w", mapErr(err))
	}

	var counts []struct {
		Asnaf string `db:"asnaf"`
		N     int    `db:"n"`
	}
	if err := sqlx.SelectContext(ctx, s.q, &counts,
		`SELECT asnaf, COUNT(*) AS n FROM mustahik
		  WHERE deleted_at IS NULL AND is_active = TRUE GROUP BY asnaf`); err != nil {
		return MustahikList{}, fmt.Errorf("store.ListMustahik: jumlah per asnaf: %w", mapErr(err))
	}

	hasil := MustahikList{
		Items:       make([]domain.Mustahik, 0, len(rows)),
		Total:       total,
		AsnafCounts: make(map[domain.Asnaf]int, len(counts)),
	}
	for _, r := range rows {
		hasil.Items = append(hasil.Items, r.domain())
	}
	for _, c := range counts {
		hasil.AsnafCounts[domain.Asnaf(c.Asnaf)] = c.N
	}
	return hasil, nil
}
