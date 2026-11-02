package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// CAST ... AS SIGNED: DECIMAL(15,2) dibaca sebagai int64 rupiah utuh (sama seperti periode).
// Judul program diambil lewat subquery (bukan JOIN) supaya semua query tetap
// `FROM donasi` dan kunci FOR UPDATE cukup dibatasi ke tabel donasi.
const kolomDonasi = `id, kode_publik, nama, anonim, no_hp_enc, jenis, program_id,
	(SELECT p.judul FROM program_donasi p WHERE p.id = donasi.program_id) AS program_judul,
	CAST(jumlah AS SIGNED) AS jumlah, sumber,
	metode, status, nomor_kuitansi, keterangan, bayar_url, dibayar_pada, dilihat_pada, dicatat_oleh,
	alasan_batal, created_at, updated_at`

type donasiRow struct {
	ID            string         `db:"id"`
	KodePublik    string         `db:"kode_publik"`
	Nama          string         `db:"nama"`
	Anonim        bool           `db:"anonim"`
	NoHPEnc       []byte         `db:"no_hp_enc"`
	Jenis         string         `db:"jenis"`
	ProgramID     sql.NullString `db:"program_id"`
	ProgramJudul  sql.NullString `db:"program_judul"`
	Jumlah        int64          `db:"jumlah"`
	Sumber        string         `db:"sumber"`
	Metode        sql.NullString `db:"metode"`
	Status        string         `db:"status"`
	NomorKuitansi sql.NullString `db:"nomor_kuitansi"`
	Keterangan    sql.NullString `db:"keterangan"`
	BayarURL      sql.NullString `db:"bayar_url"`
	DibayarPada   *time.Time     `db:"dibayar_pada"`
	DilihatPada   *time.Time     `db:"dilihat_pada"`
	DicatatOleh   sql.NullString `db:"dicatat_oleh"`
	AlasanBatal   sql.NullString `db:"alasan_batal"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}

func (r donasiRow) domain() domain.Donasi {
	return domain.Donasi{
		ID: r.ID, KodePublik: r.KodePublik, Nama: r.Nama, Anonim: r.Anonim, NoHPEnc: r.NoHPEnc,
		Jenis: domain.JenisDonasi(r.Jenis), ProgramID: r.ProgramID.String, ProgramJudul: r.ProgramJudul.String,
		Jumlah: r.Jumlah, Sumber: domain.SumberDonasi(r.Sumber),
		Metode: r.Metode.String, Status: domain.StatusDonasi(r.Status), NomorKuitansi: r.NomorKuitansi.String,
		Keterangan: r.Keterangan.String, BayarURL: r.BayarURL.String, DibayarPada: r.DibayarPada,
		DilihatPada: r.DilihatPada, DicatatOleh: r.DicatatOleh.String, AlasanBatal: r.AlasanBatal.String,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

// DonasiFilter untuk ListDonasi. Nilai nol = tanpa filter.
type DonasiFilter struct {
	Status  domain.StatusDonasi
	Jenis   domain.JenisDonasi
	Program string // ID program
	Q       string // potongan nama donatur
	Page    int    // mulai 1
	PerPage int    // bawaan 20, maksimal 100
}

// DonasiList = satu halaman + total baris yang cocok filter.
type DonasiList struct {
	Items []domain.Donasi
	Total int
}

// DonasiRingkas = angka untuk penanda notifikasi dan kartu ringkasan.
type DonasiRingkas struct {
	Baru           int   // donasi web diterima yang belum dilihat pengurus
	JumlahDiterima int   // banyak donasi berstatus diterima
	TotalDiterima  int64 // jumlah rupiahnya
}

// where merakit kondisi dari bagian SQL TETAP; nilai dari pengguna hanya lewat `?`.
func (f DonasiFilter) where() (string, []any) {
	cond := []string{"deleted_at IS NULL"}
	var args []any
	if f.Status != "" {
		cond = append(cond, "status = ?")
		args = append(args, string(f.Status))
	}
	if f.Jenis != "" {
		cond = append(cond, "jenis = ?")
		args = append(args, string(f.Jenis))
	}
	if f.Program != "" {
		cond = append(cond, "program_id = ?")
		args = append(args, f.Program)
	}
	if f.Q != "" {
		cond = append(cond, "nama LIKE ?")
		args = append(args, "%"+escapeLike.Replace(f.Q)+"%")
	}
	return strings.Join(cond, " AND "), args
}

// CreateDonasi menyimpan donasi baru dan mengisi ID serta waktu pada d.
// kode_publik atau nomor kuitansi ganda → domain.ErrConflict.
func (s *Store) CreateDonasi(ctx context.Context, d *domain.Donasi) error {
	id := newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO donasi (id, kode_publik, nama, anonim, no_hp_enc, jenis, program_id, jumlah, sumber, metode, status,
		                     nomor_kuitansi, keterangan, bayar_url, dibayar_pada, dilihat_pada, dicatat_oleh)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, d.KodePublik, d.Nama, d.Anonim, d.NoHPEnc, string(d.Jenis), nullStr(d.ProgramID), d.Jumlah, string(d.Sumber), nullStr(d.Metode),
		string(d.Status), nullStr(d.NomorKuitansi), nullStr(d.Keterangan), nullStr(d.BayarURL),
		d.DibayarPada, d.DilihatPada, nullStr(d.DicatatOleh))
	if err != nil {
		return fmt.Errorf("store.CreateDonasi: %w", mapErr(err))
	}
	dibuat, err := s.GetDonasi(ctx, id)
	if err != nil {
		return err
	}
	*d = dibuat
	return nil
}

// GetDonasi mengambil satu donasi; ErrNotFound bila tidak ada.
func (s *Store) GetDonasi(ctx context.Context, id string) (domain.Donasi, error) {
	return s.ambilDonasi(ctx, `id = ?`, id, false)
}

// GetDonasiByKode = GetDonasi lewat kode publik (tautan muzaki).
func (s *Store) GetDonasiByKode(ctx context.Context, kode string) (domain.Donasi, error) {
	return s.ambilDonasi(ctx, `kode_publik = ?`, kode, false)
}

// GetDonasiForUpdate mengunci barisnya sampai transaksi selesai, supaya dua
// notifikasi gateway yang datang bersamaan tidak sama-sama menerbitkan kuitansi.
// Hanya bermakna di dalam transaksi.
func (s *Store) GetDonasiForUpdate(ctx context.Context, id string) (domain.Donasi, error) {
	return s.ambilDonasi(ctx, `id = ?`, id, true)
}

func (s *Store) ambilDonasi(ctx context.Context, kondisi, arg string, kunci bool) (domain.Donasi, error) {
	q := `SELECT ` + kolomDonasi + ` FROM donasi WHERE ` + kondisi + ` AND deleted_at IS NULL`
	if kunci {
		q += ` FOR UPDATE OF donasi` // baris program (subquery judul) tidak ikut terkunci
	}
	var r donasiRow
	if err := sqlx.GetContext(ctx, s.q, &r, q, arg); err != nil {
		return domain.Donasi{}, fmt.Errorf("store.ambilDonasi: %w", mapErr(err))
	}
	return r.domain(), nil
}

// UpdateDonasi menulis kolom yang boleh berubah setelah dibuat: status, metode,
// nomor kuitansi, waktu bayar, tautan bayar, dan alasan batal. Data donatur
// dan jumlah tidak pernah diubah (kuitansi adalah bukti keuangan).
func (s *Store) UpdateDonasi(ctx context.Context, d domain.Donasi) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE donasi
		    SET status = ?, metode = ?, nomor_kuitansi = ?, dibayar_pada = ?, bayar_url = ?, alasan_batal = ?
		  WHERE id = ? AND deleted_at IS NULL`,
		string(d.Status), nullStr(d.Metode), nullStr(d.NomorKuitansi), d.DibayarPada,
		nullStr(d.BayarURL), nullStr(d.AlasanBatal), d.ID)
	if err != nil {
		return fmt.Errorf("store.UpdateDonasi: %w", mapErr(err))
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("store.UpdateDonasi: %w", err)
	} else if n == 0 {
		return fmt.Errorf("store.UpdateDonasi %s: %w", d.ID, domain.ErrNotFound)
	}
	return nil
}

// ListDonasi mengembalikan satu halaman (terbaru dulu) dan total yang cocok filter.
func (s *Store) ListDonasi(ctx context.Context, f DonasiFilter) (DonasiList, error) {
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
	if err := sqlx.GetContext(ctx, s.q, &total, `SELECT COUNT(*) FROM donasi WHERE `+kondisi, args...); err != nil {
		return DonasiList{}, fmt.Errorf("store.ListDonasi: hitung: %w", mapErr(err))
	}
	var rows []donasiRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT `+kolomDonasi+` FROM donasi WHERE `+kondisi+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`,
		append(args, f.PerPage, (f.Page-1)*f.PerPage)...)
	if err != nil {
		return DonasiList{}, fmt.Errorf("store.ListDonasi: %w", mapErr(err))
	}
	hasil := DonasiList{Items: make([]domain.Donasi, 0, len(rows)), Total: total}
	for _, r := range rows {
		hasil.Items = append(hasil.Items, r.domain())
	}
	return hasil, nil
}

// RingkasDonasi menghitung penanda notifikasi dan total penerimaan.
func (s *Store) RingkasDonasi(ctx context.Context) (DonasiRingkas, error) {
	var r struct {
		Baru   int   `db:"baru"`
		Jumlah int   `db:"jumlah"`
		Total  int64 `db:"total"`
	}
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT COALESCE(SUM(sumber = 'web' AND dilihat_pada IS NULL), 0) AS baru,
		        COUNT(*) AS jumlah,
		        COALESCE(SUM(CAST(jumlah AS SIGNED)), 0) AS total
		   FROM donasi WHERE status = 'diterima' AND deleted_at IS NULL`)
	if err != nil {
		return DonasiRingkas{}, fmt.Errorf("store.RingkasDonasi: %w", mapErr(err))
	}
	return DonasiRingkas{Baru: r.Baru, JumlahDiterima: r.Jumlah, TotalDiterima: r.Total}, nil
}

// TandaiDonasiDilihat menganggap semua donasi web yang sudah diterima telah
// dilihat pengurus; mengembalikan berapa baris yang berubah.
func (s *Store) TandaiDonasiDilihat(ctx context.Context) (int64, error) {
	res, err := s.q.ExecContext(ctx,
		`UPDATE donasi SET dilihat_pada = CURRENT_TIMESTAMP(3)
		  WHERE sumber = 'web' AND status = 'diterima' AND dilihat_pada IS NULL AND deleted_at IS NULL`)
	if err != nil {
		return 0, fmt.Errorf("store.TandaiDonasiDilihat: %w", mapErr(err))
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("store.TandaiDonasiDilihat: %w", err)
	}
	return n, nil
}

// NextNomorKuitansi mengambil nomor urut berikutnya untuk tahun tertentu.
// WAJIB di dalam transaksi yang juga menyimpan donasinya: baris penghitung
// terkunci sampai commit, jadi dua kuitansi tidak pernah mendapat nomor sama
// dan rollback tidak meninggalkan celah.
func (s *Store) NextNomorKuitansi(ctx context.Context, tahun int) (int, error) {
	if s.db != nil {
		return 0, errors.New("store.NextNomorKuitansi: harus dipanggil di dalam transaksi")
	}
	// LAST_INSERT_ID(expr) menyimpan nilai baru di koneksi ini, baik untuk baris
	// pertama (VALUES) maupun yang sudah ada (UPDATE), lalu dibaca kembali.
	if _, err := s.q.ExecContext(ctx,
		`INSERT INTO donasi_urut (tahun, terakhir) VALUES (?, LAST_INSERT_ID(1))
		 ON DUPLICATE KEY UPDATE terakhir = LAST_INSERT_ID(terakhir + 1)`, tahun); err != nil {
		return 0, fmt.Errorf("store.NextNomorKuitansi: %w", mapErr(err))
	}
	var n int
	if err := sqlx.GetContext(ctx, s.q, &n, `SELECT LAST_INSERT_ID()`); err != nil {
		return 0, fmt.Errorf("store.NextNomorKuitansi: baca nomor: %w", mapErr(err))
	}
	return n, nil
}
