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

// Terkumpul dan jumlah donatur dihitung dari donasi diterima lewat subquery
// (idx_donasi_program), bukan disimpan di tabel program, supaya tidak pernah
// selisih dengan kuitansi. Kolom gambar (BLOB) sengaja tidak ikut: hanya
// penandanya, isinya dibaca terpisah lewat GetGambarProgram.
const kolomProgram = `id, slug, judul, ringkasan, cerita, CAST(target AS SIGNED) AS target, batas_waktu, status,
	gambar IS NOT NULL AS ada_gambar, dibuat_oleh, created_at, updated_at,
	(SELECT COALESCE(SUM(CAST(d.jumlah AS SIGNED)), 0) FROM donasi d
	  WHERE d.program_id = program_donasi.id AND d.status = 'diterima' AND d.deleted_at IS NULL) AS terkumpul,
	(SELECT COUNT(*) FROM donasi d
	  WHERE d.program_id = program_donasi.id AND d.status = 'diterima' AND d.deleted_at IS NULL) AS donatur`

type programRow struct {
	ID         string         `db:"id"`
	Slug       string         `db:"slug"`
	Judul      string         `db:"judul"`
	Ringkasan  string         `db:"ringkasan"`
	Cerita     string         `db:"cerita"`
	Target     int64          `db:"target"`
	BatasWaktu *time.Time     `db:"batas_waktu"`
	Status     string         `db:"status"`
	AdaGambar  bool           `db:"ada_gambar"`
	DibuatOleh sql.NullString `db:"dibuat_oleh"`
	CreatedAt  time.Time      `db:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at"`
	Terkumpul  int64          `db:"terkumpul"`
	Donatur    int            `db:"donatur"`
}

func (r programRow) domain() domain.Program {
	return domain.Program{
		ID: r.ID, Slug: r.Slug, Judul: r.Judul, Ringkasan: r.Ringkasan, Cerita: r.Cerita, Target: r.Target,
		BatasWaktu: r.BatasWaktu, Status: domain.StatusProgram(r.Status), AdaGambar: r.AdaGambar,
		DibuatOleh: r.DibuatOleh.String, Terkumpul: r.Terkumpul, Donatur: r.Donatur,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// tanggal mengubah batas waktu menjadi teks DATE (atau NULL) supaya tidak
// bergeser oleh zona waktu koneksi.
func tanggal(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

// CreateProgram menyimpan program baru dan mengisi ID, angka, serta waktu pada p.
// Slug ganda → domain.ErrConflict (service mencoba slug lain).
func (s *Store) CreateProgram(ctx context.Context, p *domain.Program) error {
	id := newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO program_donasi (id, slug, judul, ringkasan, cerita, target, batas_waktu, status, dibuat_oleh)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.Slug, p.Judul, p.Ringkasan, p.Cerita, p.Target, tanggal(p.BatasWaktu), string(p.Status), nullStr(p.DibuatOleh))
	if err != nil {
		return fmt.Errorf("store.CreateProgram: %w", mapErr(err))
	}
	dibuat, err := s.GetProgram(ctx, id)
	if err != nil {
		return err
	}
	*p = dibuat
	return nil
}

// UpdateProgram menulis isian yang boleh diubah pengurus. Slug tidak ikut
// berubah supaya tautan yang sudah dibagikan tetap hidup.
func (s *Store) UpdateProgram(ctx context.Context, p domain.Program) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE program_donasi
		    SET judul = ?, ringkasan = ?, cerita = ?, target = ?, batas_waktu = ?, status = ?
		  WHERE id = ? AND deleted_at IS NULL`,
		p.Judul, p.Ringkasan, p.Cerita, p.Target, tanggal(p.BatasWaktu), string(p.Status), p.ID)
	return cekSatu(res, err, "store.UpdateProgram", p.ID)
}

// SetGambarProgram mengganti gambar sampul (isi sudah divalidasi service).
func (s *Store) SetGambarProgram(ctx context.Context, id string, isi []byte, tipe string) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE program_donasi SET gambar = ?, gambar_tipe = ? WHERE id = ? AND deleted_at IS NULL`, isi, tipe, id)
	return cekSatu(res, err, "store.SetGambarProgram", id)
}

// GetGambarProgram mengembalikan isi dan media type gambar sampul; ErrNotFound
// bila program tidak ada atau belum bergambar.
func (s *Store) GetGambarProgram(ctx context.Context, id string) ([]byte, string, error) {
	var r struct {
		Isi  []byte         `db:"gambar"`
		Tipe sql.NullString `db:"gambar_tipe"`
	}
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT gambar, gambar_tipe FROM program_donasi WHERE id = ? AND gambar IS NOT NULL AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, "", fmt.Errorf("store.GetGambarProgram: %w", mapErr(err))
	}
	return r.Isi, r.Tipe.String, nil
}

// GetProgram mengambil satu program lewat ID; ErrNotFound bila tidak ada.
func (s *Store) GetProgram(ctx context.Context, id string) (domain.Program, error) {
	return s.ambilProgram(ctx, `id = ?`, id)
}

// GetProgramBySlug = GetProgram lewat slug (halaman publik).
func (s *Store) GetProgramBySlug(ctx context.Context, slug string) (domain.Program, error) {
	return s.ambilProgram(ctx, `slug = ?`, slug)
}

func (s *Store) ambilProgram(ctx context.Context, kondisi, arg string) (domain.Program, error) {
	var r programRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomProgram+` FROM program_donasi WHERE `+kondisi+` AND deleted_at IS NULL`, arg)
	if err != nil {
		return domain.Program{}, fmt.Errorf("store.ambilProgram: %w", mapErr(err))
	}
	return r.domain(), nil
}

// ListProgram mengembalikan program berstatus tertentu (kosong = semua):
// aktif dulu, lalu draf, lalu selesai; terbaru di atas.
func (s *Store) ListProgram(ctx context.Context, status ...domain.StatusProgram) ([]domain.Program, error) {
	q := `SELECT ` + kolomProgram + ` FROM program_donasi WHERE deleted_at IS NULL`
	var args []any
	if len(status) > 0 {
		q += ` AND status IN (?` + strings.Repeat(`, ?`, len(status)-1) + `)`
		for _, st := range status {
			args = append(args, string(st))
		}
	}
	q += ` ORDER BY FIELD(status, 'aktif', 'draf', 'selesai'), created_at DESC, id`
	var rows []programRow
	if err := sqlx.SelectContext(ctx, s.q, &rows, q, args...); err != nil {
		return nil, fmt.Errorf("store.ListProgram: %w", mapErr(err))
	}
	hasil := make([]domain.Program, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}

// ListDonaturProgram = donasi diterima untuk satu program, terbaru dulu.
// Hanya kolom yang aman untuk publik (nama, anonim, jumlah, waktu).
func (s *Store) ListDonaturProgram(ctx context.Context, programID string, batas int) ([]domain.DonaturProgram, error) {
	var rows []struct {
		Nama        string    `db:"nama"`
		Anonim      bool      `db:"anonim"`
		Jumlah      int64     `db:"jumlah"`
		DibayarPada time.Time `db:"dibayar_pada"`
	}
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT nama, anonim, CAST(jumlah AS SIGNED) AS jumlah, COALESCE(dibayar_pada, created_at) AS dibayar_pada
		   FROM donasi
		  WHERE program_id = ? AND status = 'diterima' AND deleted_at IS NULL
		  ORDER BY dibayar_pada DESC, id LIMIT ?`, programID, batas)
	if err != nil {
		return nil, fmt.Errorf("store.ListDonaturProgram: %w", mapErr(err))
	}
	hasil := make([]domain.DonaturProgram, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, domain.DonaturProgram{Nama: r.Nama, Anonim: r.Anonim, Jumlah: r.Jumlah, DibayarPada: r.DibayarPada})
	}
	return hasil, nil
}

// CreateKabar menyimpan satu kabar terbaru dan mengisi ID serta waktunya.
func (s *Store) CreateKabar(ctx context.Context, k *domain.KabarProgram) error {
	k.ID = newID()
	if _, err := s.q.ExecContext(ctx,
		`INSERT INTO program_kabar (id, program_id, judul, isi, dibuat_oleh) VALUES (?, ?, ?, ?, ?)`,
		k.ID, k.ProgramID, k.Judul, k.Isi, nullStr(k.DibuatOleh)); err != nil {
		return fmt.Errorf("store.CreateKabar: %w", mapErr(err))
	}
	if err := sqlx.GetContext(ctx, s.q, &k.CreatedAt, `SELECT created_at FROM program_kabar WHERE id = ?`, k.ID); err != nil {
		return fmt.Errorf("store.CreateKabar: %w", mapErr(err))
	}
	return nil
}

// ListKabar = kabar sebuah program, terbaru dulu.
func (s *Store) ListKabar(ctx context.Context, programID string) ([]domain.KabarProgram, error) {
	var rows []struct {
		ID         string         `db:"id"`
		ProgramID  string         `db:"program_id"`
		Judul      string         `db:"judul"`
		Isi        string         `db:"isi"`
		DibuatOleh sql.NullString `db:"dibuat_oleh"`
		CreatedAt  time.Time      `db:"created_at"`
	}
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT id, program_id, judul, isi, dibuat_oleh, created_at FROM program_kabar
		  WHERE program_id = ? AND deleted_at IS NULL ORDER BY created_at DESC, id`, programID)
	if err != nil {
		return nil, fmt.Errorf("store.ListKabar: %w", mapErr(err))
	}
	hasil := make([]domain.KabarProgram, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, domain.KabarProgram{
			ID: r.ID, ProgramID: r.ProgramID, Judul: r.Judul, Isi: r.Isi, DibuatOleh: r.DibuatOleh.String, CreatedAt: r.CreatedAt,
		})
	}
	return hasil, nil
}

// DeleteKabar menghapus (soft delete) satu kabar milik program tertentu.
// Kabar program lain dengan ID itu → ErrNotFound (tidak bisa saling hapus).
func (s *Store) DeleteKabar(ctx context.Context, programID, kabarID string) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE program_kabar SET deleted_at = CURRENT_TIMESTAMP(3)
		  WHERE id = ? AND program_id = ? AND deleted_at IS NULL`, kabarID, programID)
	return cekSatu(res, err, "store.DeleteKabar", kabarID)
}

// cekSatu memastikan UPDATE mengenai satu baris; nol baris → ErrNotFound.
func cekSatu(res sql.Result, err error, op, id string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", op, mapErr(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if n == 0 {
		return fmt.Errorf("%s %s: %w", op, id, domain.ErrNotFound)
	}
	return nil
}
