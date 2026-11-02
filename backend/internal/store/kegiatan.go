package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// Konten kegiatan keagamaan: kegiatan (artikel/acara), jadwal rutin mingguan,
// dan pengumuman. SQL berparameter saja; aturan bisnis ada di service.

// Kolom gambar (BLOB) tidak ikut: hanya penandanya; isinya lewat GetGambarKegiatan.
const kolomKegiatan = `id, slug, judul, kategori, ringkasan, isi, tanggal_acara, jam_acara, lokasi, status,
	gambar IS NOT NULL AS ada_gambar, dibuat_oleh, created_at, updated_at`

type kegiatanRow struct {
	ID           string         `db:"id"`
	Slug         string         `db:"slug"`
	Judul        string         `db:"judul"`
	Kategori     string         `db:"kategori"`
	Ringkasan    string         `db:"ringkasan"`
	Isi          string         `db:"isi"`
	TanggalAcara *time.Time     `db:"tanggal_acara"`
	JamAcara     sql.NullString `db:"jam_acara"`
	Lokasi       sql.NullString `db:"lokasi"`
	Status       string         `db:"status"`
	AdaGambar    bool           `db:"ada_gambar"`
	DibuatOleh   sql.NullString `db:"dibuat_oleh"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func (r kegiatanRow) domain() domain.Kegiatan {
	return domain.Kegiatan{
		ID: r.ID, Slug: r.Slug, Judul: r.Judul, Kategori: r.Kategori, Ringkasan: r.Ringkasan, Isi: r.Isi,
		TanggalAcara: r.TanggalAcara, JamAcara: r.JamAcara.String, Lokasi: r.Lokasi.String,
		Status: domain.StatusKegiatan(r.Status), AdaGambar: r.AdaGambar, DibuatOleh: r.DibuatOleh.String,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// CreateKegiatan menyimpan kegiatan baru dan mengisi ID serta waktu pada k.
// Slug ganda → domain.ErrConflict (service mencoba slug lain).
func (s *Store) CreateKegiatan(ctx context.Context, k *domain.Kegiatan) error {
	id := newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO kegiatan (id, slug, judul, kategori, ringkasan, isi, tanggal_acara, jam_acara, lokasi, status, dibuat_oleh)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, k.Slug, k.Judul, k.Kategori, k.Ringkasan, k.Isi, tanggal(k.TanggalAcara), nullStr(k.JamAcara),
		nullStr(k.Lokasi), string(k.Status), nullStr(k.DibuatOleh))
	if err != nil {
		return fmt.Errorf("store.CreateKegiatan: %w", mapErr(err))
	}
	dibuat, err := s.GetKegiatan(ctx, id)
	if err != nil {
		return err
	}
	*k = dibuat
	return nil
}

// UpdateKegiatan menulis isian yang boleh diubah. Slug tidak ikut berubah
// supaya tautan yang sudah dibagikan tetap hidup.
func (s *Store) UpdateKegiatan(ctx context.Context, k domain.Kegiatan) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE kegiatan
		    SET judul = ?, kategori = ?, ringkasan = ?, isi = ?, tanggal_acara = ?, jam_acara = ?, lokasi = ?, status = ?
		  WHERE id = ? AND deleted_at IS NULL`,
		k.Judul, k.Kategori, k.Ringkasan, k.Isi, tanggal(k.TanggalAcara), nullStr(k.JamAcara), nullStr(k.Lokasi),
		string(k.Status), k.ID)
	return cekSatu(res, err, "store.UpdateKegiatan", k.ID)
}

// SetGambarKegiatan mengganti gambar sampul (isi sudah divalidasi service).
func (s *Store) SetGambarKegiatan(ctx context.Context, id string, isi []byte, tipe string) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE kegiatan SET gambar = ?, gambar_tipe = ? WHERE id = ? AND deleted_at IS NULL`, isi, tipe, id)
	return cekSatu(res, err, "store.SetGambarKegiatan", id)
}

// GetGambarKegiatan mengembalikan isi dan media type gambar sampul; ErrNotFound
// bila kegiatan tidak ada atau belum bergambar.
func (s *Store) GetGambarKegiatan(ctx context.Context, id string) ([]byte, string, error) {
	var r struct {
		Isi  []byte         `db:"gambar"`
		Tipe sql.NullString `db:"gambar_tipe"`
	}
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT gambar, gambar_tipe FROM kegiatan WHERE id = ? AND gambar IS NOT NULL AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, "", fmt.Errorf("store.GetGambarKegiatan: %w", mapErr(err))
	}
	return r.Isi, r.Tipe.String, nil
}

// GetKegiatan mengambil satu kegiatan lewat ID; ErrNotFound bila tidak ada.
func (s *Store) GetKegiatan(ctx context.Context, id string) (domain.Kegiatan, error) {
	return s.ambilKegiatan(ctx, `id = ?`, id)
}

// GetKegiatanBySlug = GetKegiatan lewat slug (halaman publik).
func (s *Store) GetKegiatanBySlug(ctx context.Context, slug string) (domain.Kegiatan, error) {
	return s.ambilKegiatan(ctx, `slug = ?`, slug)
}

func (s *Store) ambilKegiatan(ctx context.Context, kondisi, arg string) (domain.Kegiatan, error) {
	var r kegiatanRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomKegiatan+` FROM kegiatan WHERE `+kondisi+` AND deleted_at IS NULL`, arg)
	if err != nil {
		return domain.Kegiatan{}, fmt.Errorf("store.ambilKegiatan: %w", mapErr(err))
	}
	return r.domain(), nil
}

// KegiatanFilter untuk ListKegiatan. Nilai nol = tanpa filter.
type KegiatanFilter struct {
	Status   domain.StatusKegiatan
	Kategori string
	Batas    int // 0 = 200
}

// ListKegiatan mengembalikan kegiatan menurut tanggal acara (atau tanggal dibuat
// bila tanpa tanggal acara), terbaru dulu.
func (s *Store) ListKegiatan(ctx context.Context, f KegiatanFilter) ([]domain.Kegiatan, error) {
	if f.Batas < 1 || f.Batas > 500 {
		f.Batas = 200
	}
	q := `SELECT ` + kolomKegiatan + ` FROM kegiatan WHERE deleted_at IS NULL`
	var args []any
	if f.Status != "" {
		q += ` AND status = ?`
		args = append(args, string(f.Status))
	}
	if f.Kategori != "" {
		q += ` AND kategori = ?`
		args = append(args, f.Kategori)
	}
	q += ` ORDER BY COALESCE(tanggal_acara, DATE(created_at)) DESC, created_at DESC, id LIMIT ?`
	args = append(args, f.Batas)
	var rows []kegiatanRow
	if err := sqlx.SelectContext(ctx, s.q, &rows, q, args...); err != nil {
		return nil, fmt.Errorf("store.ListKegiatan: %w", mapErr(err))
	}
	hasil := make([]domain.Kegiatan, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}

// DeleteKegiatan menghapus (soft delete) satu kegiatan.
func (s *Store) DeleteKegiatan(ctx context.Context, id string) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE kegiatan SET deleted_at = CURRENT_TIMESTAMP(3) WHERE id = ? AND deleted_at IS NULL`, id)
	return cekSatu(res, err, "store.DeleteKegiatan", id)
}

// ---------- jadwal rutin ----------

type jadwalRow struct {
	ID         string         `db:"id"`
	Judul      string         `db:"judul"`
	Pemateri   sql.NullString `db:"pemateri"`
	Hari       int            `db:"hari"`
	Jam        string         `db:"jam"`
	Lokasi     sql.NullString `db:"lokasi"`
	Keterangan sql.NullString `db:"keterangan"`
	Aktif      bool           `db:"aktif"`
	CreatedAt  time.Time      `db:"created_at"`
}

func (r jadwalRow) domain() domain.JadwalRutin {
	return domain.JadwalRutin{
		ID: r.ID, Judul: r.Judul, Pemateri: r.Pemateri.String, Hari: r.Hari, Jam: r.Jam,
		Lokasi: r.Lokasi.String, Keterangan: r.Keterangan.String, Aktif: r.Aktif, CreatedAt: r.CreatedAt,
	}
}

// ListJadwal mengembalikan jadwal menurut hari (Ahad dulu) lalu jam.
// hanyaAktif=true untuk halaman publik.
func (s *Store) ListJadwal(ctx context.Context, hanyaAktif bool) ([]domain.JadwalRutin, error) {
	q := `SELECT id, judul, pemateri, hari, jam, lokasi, keterangan, aktif, created_at FROM jadwal_rutin WHERE deleted_at IS NULL`
	if hanyaAktif {
		q += ` AND aktif = TRUE`
	}
	q += ` ORDER BY hari, jam, judul, id`
	var rows []jadwalRow
	if err := sqlx.SelectContext(ctx, s.q, &rows, q); err != nil {
		return nil, fmt.Errorf("store.ListJadwal: %w", mapErr(err))
	}
	hasil := make([]domain.JadwalRutin, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}

// GetJadwal mengambil satu jadwal; ErrNotFound bila tidak ada.
func (s *Store) GetJadwal(ctx context.Context, id string) (domain.JadwalRutin, error) {
	var r jadwalRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT id, judul, pemateri, hari, jam, lokasi, keterangan, aktif, created_at
		   FROM jadwal_rutin WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return domain.JadwalRutin{}, fmt.Errorf("store.GetJadwal: %w", mapErr(err))
	}
	return r.domain(), nil
}

// CreateJadwal menyimpan jadwal baru dan mengisi ID pada j.
func (s *Store) CreateJadwal(ctx context.Context, j *domain.JadwalRutin) error {
	j.ID = newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO jadwal_rutin (id, judul, pemateri, hari, jam, lokasi, keterangan, aktif) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		j.ID, j.Judul, nullStr(j.Pemateri), j.Hari, j.Jam, nullStr(j.Lokasi), nullStr(j.Keterangan), j.Aktif)
	if err != nil {
		return fmt.Errorf("store.CreateJadwal: %w", mapErr(err))
	}
	return nil
}

// UpdateJadwal menulis seluruh isian jadwal.
func (s *Store) UpdateJadwal(ctx context.Context, j domain.JadwalRutin) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE jadwal_rutin SET judul = ?, pemateri = ?, hari = ?, jam = ?, lokasi = ?, keterangan = ?, aktif = ?
		  WHERE id = ? AND deleted_at IS NULL`,
		j.Judul, nullStr(j.Pemateri), j.Hari, j.Jam, nullStr(j.Lokasi), nullStr(j.Keterangan), j.Aktif, j.ID)
	return cekSatu(res, err, "store.UpdateJadwal", j.ID)
}

// DeleteJadwal menghapus (soft delete) satu jadwal.
func (s *Store) DeleteJadwal(ctx context.Context, id string) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE jadwal_rutin SET deleted_at = CURRENT_TIMESTAMP(3) WHERE id = ? AND deleted_at IS NULL`, id)
	return cekSatu(res, err, "store.DeleteJadwal", id)
}

// ---------- pengumuman ----------

type pengumumanRow struct {
	ID           string     `db:"id"`
	Judul        string     `db:"judul"`
	Isi          string     `db:"isi"`
	TampilSampai *time.Time `db:"tampil_sampai"`
	Terbit       bool       `db:"terbit"`
	CreatedAt    time.Time  `db:"created_at"`
}

func (r pengumumanRow) domain() domain.Pengumuman {
	return domain.Pengumuman{ID: r.ID, Judul: r.Judul, Isi: r.Isi, TampilSampai: r.TampilSampai, Terbit: r.Terbit, CreatedAt: r.CreatedAt}
}

// ListPengumuman mengembalikan pengumuman terbaru dulu. tayangPada != nil =
// hanya yang terbit dan masih dalam masa tayang pada tanggal itu (publik);
// nil = semuanya (pengurus).
func (s *Store) ListPengumuman(ctx context.Context, tayangPada *time.Time) ([]domain.Pengumuman, error) {
	q := `SELECT id, judul, isi, tampil_sampai, terbit, created_at FROM pengumuman WHERE deleted_at IS NULL`
	var args []any
	if tayangPada != nil {
		q += ` AND terbit = TRUE AND (tampil_sampai IS NULL OR tampil_sampai >= ?)`
		args = append(args, tayangPada.Format("2006-01-02"))
	}
	q += ` ORDER BY created_at DESC, id`
	var rows []pengumumanRow
	if err := sqlx.SelectContext(ctx, s.q, &rows, q, args...); err != nil {
		return nil, fmt.Errorf("store.ListPengumuman: %w", mapErr(err))
	}
	hasil := make([]domain.Pengumuman, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}

// GetPengumuman mengambil satu pengumuman; ErrNotFound bila tidak ada.
func (s *Store) GetPengumuman(ctx context.Context, id string) (domain.Pengumuman, error) {
	var r pengumumanRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT id, judul, isi, tampil_sampai, terbit, created_at FROM pengumuman WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return domain.Pengumuman{}, fmt.Errorf("store.GetPengumuman: %w", mapErr(err))
	}
	return r.domain(), nil
}

// CreatePengumuman menyimpan pengumuman baru dan mengisi ID pada p.
func (s *Store) CreatePengumuman(ctx context.Context, p *domain.Pengumuman) error {
	p.ID = newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO pengumuman (id, judul, isi, tampil_sampai, terbit) VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.Judul, p.Isi, tanggal(p.TampilSampai), p.Terbit)
	if err != nil {
		return fmt.Errorf("store.CreatePengumuman: %w", mapErr(err))
	}
	return nil
}

// UpdatePengumuman menulis seluruh isian pengumuman.
func (s *Store) UpdatePengumuman(ctx context.Context, p domain.Pengumuman) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE pengumuman SET judul = ?, isi = ?, tampil_sampai = ?, terbit = ? WHERE id = ? AND deleted_at IS NULL`,
		p.Judul, p.Isi, tanggal(p.TampilSampai), p.Terbit, p.ID)
	return cekSatu(res, err, "store.UpdatePengumuman", p.ID)
}

// DeletePengumuman menghapus (soft delete) satu pengumuman.
func (s *Store) DeletePengumuman(ctx context.Context, id string) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE pengumuman SET deleted_at = CURRENT_TIMESTAMP(3) WHERE id = ? AND deleted_at IS NULL`, id)
	return cekSatu(res, err, "store.DeletePengumuman", id)
}
