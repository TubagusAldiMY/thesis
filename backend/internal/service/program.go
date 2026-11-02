package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"spkzakat/internal/domain"
)

const (
	// Batas target program: bawah supaya tidak ada program iseng, atas = batas
	// wajar penggalangan satu masjid (Rp 10 miliar), jauh di bawah DECIMAL(15,2).
	targetMin  int64 = 100_000
	targetMaks int64 = 10_000_000_000

	// GambarMaks = ukuran maksimal gambar sampul (juga dipakai lapisan HTTP).
	GambarMaks = 2 << 20

	// donaturTampil = banyak donatur terbaru di halaman publik program.
	donaturTampil = 50

	namaAnonim = "Hamba Allah"
)

var reSlug = regexp.MustCompile(`^[a-z0-9-]{1,90}$`)

// ProgramInput = isian pengurus saat membuat/mengubah program.
type ProgramInput struct {
	Judul      string
	Ringkasan  string
	Cerita     string
	Target     int64
	BatasWaktu string // yyyy-mm-dd, kosong = tanpa batas
	Status     domain.StatusProgram
}

// ProgramView = bentuk program untuk halaman pengurus maupun publik. Angka
// turunan (persen, sisa hari, terbuka) dihitung di sini supaya halaman tidak
// perlu tahu aturan batas waktu.
type ProgramView struct {
	ID          string               `json:"id"`
	Slug        string               `json:"slug"`
	Judul       string               `json:"judul"`
	Ringkasan   string               `json:"ringkasan"`
	Cerita      string               `json:"cerita"`
	Target      int64                `json:"target"`
	BatasWaktu  string               `json:"batas_waktu"` // yyyy-mm-dd atau ""
	Status      domain.StatusProgram `json:"status"`
	AdaGambar   bool                 `json:"ada_gambar"`
	GambarVersi int64                `json:"gambar_versi"` // penanda cache: berubah tiap program diubah
	Terkumpul   int64                `json:"terkumpul"`
	Donatur     int                  `json:"donatur"`
	Persen      int                  `json:"persen"`    // 0–100 untuk lebar bar (terkumpul bisa melebihi target)
	SisaHari    *int                 `json:"sisa_hari"` // nil = tanpa batas; 0 = hari terakhir
	Terbuka     bool                 `json:"terbuka"`   // masih menerima donasi
	DibuatPada  time.Time            `json:"dibuat_pada"`
}

// KabarView = satu kabar terbaru.
type KabarView struct {
	ID         string    `json:"id"`
	Judul      string    `json:"judul"`
	Isi        string    `json:"isi"`
	DibuatPada time.Time `json:"dibuat_pada"`
}

// DonaturView = satu baris daftar donatur publik (tanpa No. HP/doa).
type DonaturView struct {
	Nama        string    `json:"nama"`
	Jumlah      int64     `json:"jumlah"`
	DibayarPada time.Time `json:"dibayar_pada"`
}

// ProgramDetail = program beserta kabar dan (untuk publik) donatur terbarunya.
type ProgramDetail struct {
	ProgramView
	Kabar   []KabarView   `json:"kabar"`
	Donatur []DonaturView `json:"donatur_terbaru"`
}

// hariIni = tanggal hari ini menurut WIB, sebagai 00.00 UTC (setara kolom DATE).
func (s *Service) hariIni() time.Time {
	y, m, d := s.sekarang().In(wib).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// programTerbuka = program aktif yang belum lewat batas waktunya (batas
// waktu termasuk: donasi masih diterima sepanjang hari itu, WIB).
func (s *Service) programTerbuka(p domain.Program) bool {
	return p.Status == domain.ProgramAktif && (p.BatasWaktu == nil || !s.hariIni().After(*p.BatasWaktu))
}

func (s *Service) viewProgram(p domain.Program) ProgramView {
	v := ProgramView{
		ID: p.ID, Slug: p.Slug, Judul: p.Judul, Ringkasan: p.Ringkasan, Cerita: p.Cerita, Target: p.Target,
		Status: p.Status, AdaGambar: p.AdaGambar, GambarVersi: p.UpdatedAt.UnixMilli(),
		Terkumpul: p.Terkumpul, Donatur: p.Donatur, Terbuka: s.programTerbuka(p), DibuatPada: p.CreatedAt,
	}
	if p.Target > 0 {
		v.Persen = int(min(100, p.Terkumpul*100/p.Target))
	}
	if p.BatasWaktu != nil {
		v.BatasWaktu = p.BatasWaktu.Format("2006-01-02")
		sisa := max(0, int(p.BatasWaktu.Sub(s.hariIni()).Hours()/24))
		v.SisaHari = &sisa
	}
	return v
}

func viewKabar(ks []domain.KabarProgram) []KabarView {
	out := make([]KabarView, 0, len(ks))
	for _, k := range ks {
		out = append(out, KabarView{ID: k.ID, Judul: k.Judul, Isi: k.Isi, DibuatPada: k.CreatedAt})
	}
	return out
}

// buatSlug mengubah judul menjadi alamat yang aman di URL: huruf kecil, angka,
// dan tanda hubung ("Renovasi Atap Masjid!" → "renovasi-atap-masjid").
func buatSlug(judul string) string {
	var b strings.Builder
	hubung := true // true = karakter terakhir tanda hubung (atau awal): cegah "--" dan "-" di depan
	for _, r := range strings.ToLower(judul) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			hubung = false
		} else if !hubung {
			b.WriteByte('-')
			hubung = true
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if len(slug) > 80 {
		slug = strings.TrimRight(slug[:80], "-")
	}
	if slug == "" {
		slug = "program"
	}
	return slug
}

// validasiProgram memeriksa isian dan mengubah batas waktu menjadi tanggal.
// baru=true (membuat) menolak batas waktu yang sudah lewat.
func (s *Service) validasiProgram(in ProgramInput, baru bool) (*time.Time, error) {
	g := galat{}
	if n := panjangRune(strings.TrimSpace(in.Judul)); n < 5 || n > 120 {
		g.tambah("judul", "Judul harus 5–120 karakter")
	}
	if n := panjangRune(strings.TrimSpace(in.Ringkasan)); n < 10 || n > 200 {
		g.tambah("ringkasan", "Ringkasan harus 10–200 karakter")
	}
	if n := panjangRune(strings.TrimSpace(in.Cerita)); n < 20 || n > 10_000 {
		g.tambah("cerita", "Cerita program harus 20–10.000 karakter")
	}
	if in.Target < targetMin || in.Target > targetMaks {
		g.tambah("target", fmt.Sprintf("Target harus antara Rp %s dan Rp %s", titik(targetMin), titik(targetMaks)))
	}
	switch in.Status {
	case domain.ProgramDraf, domain.ProgramAktif, domain.ProgramSelesai:
	default:
		g.tambah("status", "Pilih draf, aktif, atau selesai")
	}
	var batas *time.Time
	if t := strings.TrimSpace(in.BatasWaktu); t != "" {
		w, err := time.Parse("2006-01-02", t)
		switch {
		case err != nil:
			g.tambah("batas_waktu", "Tanggal tidak valid")
		case baru && w.Before(s.hariIni()):
			g.tambah("batas_waktu", "Batas waktu tidak boleh sebelum hari ini")
		default:
			batas = &w
		}
	}
	return batas, g.err()
}

// BuatProgram membuat program baru. Slug diambil dari judul; bila sudah
// dipakai program lain, diberi akhiran acak pendek (maksimal 3 percobaan).
func (s *Service) BuatProgram(ctx context.Context, userID string, in ProgramInput) (ProgramView, error) {
	batas, err := s.validasiProgram(in, true)
	if err != nil {
		return ProgramView{}, err
	}
	p := domain.Program{
		Judul: strings.TrimSpace(in.Judul), Ringkasan: strings.TrimSpace(in.Ringkasan),
		Cerita: strings.TrimSpace(in.Cerita), Target: in.Target, BatasWaktu: batas, Status: in.Status, DibuatOleh: userID,
	}
	dasar := buatSlug(p.Judul)
	for coba := 0; ; coba++ {
		p.Slug = dasar
		if coba > 0 {
			kode, err := kodeAcak()
			if err != nil {
				return ProgramView{}, err
			}
			p.Slug = dasar + "-" + kode[:4]
		}
		err = s.st.CreateProgram(ctx, &p)
		if !errors.Is(err, domain.ErrConflict) || coba == 3 {
			break
		}
	}
	if err != nil {
		return ProgramView{}, fmt.Errorf("service.BuatProgram: %w", err)
	}
	return s.viewProgram(p), nil
}

// UbahProgram mengubah isian dan status program. Program yang sudah menerima
// donasi tidak boleh kembali ke draf (halaman publiknya akan hilang padahal
// donaturnya memegang tautan).
func (s *Service) UbahProgram(ctx context.Context, id string, in ProgramInput) (ProgramView, error) {
	batas, err := s.validasiProgram(in, false)
	if err != nil {
		return ProgramView{}, err
	}
	p, err := s.st.GetProgram(ctx, id)
	if err != nil {
		return ProgramView{}, fmt.Errorf("service.UbahProgram: %w", err)
	}
	if in.Status == domain.ProgramDraf && p.Donatur > 0 {
		return ProgramView{}, domain.Konflik("Program yang sudah menerima donasi tidak bisa dikembalikan ke draf")
	}
	p.Judul, p.Ringkasan, p.Cerita = strings.TrimSpace(in.Judul), strings.TrimSpace(in.Ringkasan), strings.TrimSpace(in.Cerita)
	p.Target, p.BatasWaktu, p.Status = in.Target, batas, in.Status
	if err := s.st.UpdateProgram(ctx, p); err != nil {
		return ProgramView{}, fmt.Errorf("service.UbahProgram: %w", err)
	}
	if p, err = s.st.GetProgram(ctx, id); err != nil {
		return ProgramView{}, fmt.Errorf("service.UbahProgram: %w", err)
	}
	return s.viewProgram(p), nil
}

// tipeGambar mengenali JPEG, PNG, dan WebP dari isi berkas (magic bytes),
// bukan dari nama/Content-Type kiriman yang bisa dipalsukan. Selain tiga itu
// (termasuk SVG yang bisa memuat skrip) ditolak.
func tipeGambar(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return "image/jpeg"
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return "image/webp"
	}
	return ""
}

// SimpanGambarProgram mengganti gambar sampul program.
func (s *Service) SimpanGambarProgram(ctx context.Context, id string, isi []byte) error {
	if len(isi) == 0 || len(isi) > GambarMaks {
		return (galat{"gambar": "Pilih gambar berukuran maksimal 2 MB"}).err()
	}
	tipe := tipeGambar(isi)
	if tipe == "" {
		return (galat{"gambar": "Gambar harus berformat JPG, PNG, atau WebP"}).err()
	}
	if err := s.st.SetGambarProgram(ctx, id, isi, tipe); err != nil {
		return fmt.Errorf("service.SimpanGambarProgram: %w", err)
	}
	return nil
}

// AmbilGambarProgram untuk pengurus (semua status, lewat ID).
func (s *Service) AmbilGambarProgram(ctx context.Context, id string) ([]byte, string, error) {
	isi, tipe, err := s.st.GetGambarProgram(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("service.AmbilGambarProgram: %w", err)
	}
	return isi, tipe, nil
}

// AmbilGambarProgramPublik = gambar sampul lewat slug; draf dianggap tidak ada.
func (s *Service) AmbilGambarProgramPublik(ctx context.Context, slug string) ([]byte, string, error) {
	p, err := s.programPublik(ctx, slug)
	if err != nil {
		return nil, "", err
	}
	return s.AmbilGambarProgram(ctx, p.ID)
}

// DaftarProgram = semua program untuk pengurus (aktif, draf, selesai).
func (s *Service) DaftarProgram(ctx context.Context) ([]ProgramView, error) {
	return s.daftarProgram(ctx)
}

// DaftarProgramPublik = program yang tampil di halaman publik (tanpa draf).
func (s *Service) DaftarProgramPublik(ctx context.Context) ([]ProgramView, error) {
	return s.daftarProgram(ctx, domain.ProgramAktif, domain.ProgramSelesai)
}

func (s *Service) daftarProgram(ctx context.Context, status ...domain.StatusProgram) ([]ProgramView, error) {
	ps, err := s.st.ListProgram(ctx, status...)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarProgram: %w", err)
	}
	out := make([]ProgramView, 0, len(ps))
	for _, p := range ps {
		out = append(out, s.viewProgram(p))
	}
	return out, nil
}

// AmbilProgram = detail untuk pengurus: program + semua kabarnya.
func (s *Service) AmbilProgram(ctx context.Context, id string) (ProgramDetail, error) {
	p, err := s.st.GetProgram(ctx, id)
	if err != nil {
		return ProgramDetail{}, fmt.Errorf("service.AmbilProgram: %w", err)
	}
	ks, err := s.st.ListKabar(ctx, p.ID)
	if err != nil {
		return ProgramDetail{}, fmt.Errorf("service.AmbilProgram: %w", err)
	}
	return ProgramDetail{ProgramView: s.viewProgram(p), Kabar: viewKabar(ks), Donatur: []DonaturView{}}, nil
}

// programLewatSlug mengambil program lewat slug; slug salah format = tidak ada,
// supaya tidak ada petunjuk bagi penebak dan tidak ada query sia-sia.
func (s *Service) programLewatSlug(ctx context.Context, slug string) (domain.Program, error) {
	if !reSlug.MatchString(slug) {
		return domain.Program{}, domain.ErrNotFound
	}
	return s.st.GetProgramBySlug(ctx, slug)
}

// programPublik = programLewatSlug untuk pengunjung: draf diperlakukan tidak ada (404).
func (s *Service) programPublik(ctx context.Context, slug string) (domain.Program, error) {
	p, err := s.programLewatSlug(ctx, slug)
	if err != nil {
		return domain.Program{}, fmt.Errorf("service.programPublik: %w", err)
	}
	if p.Status == domain.ProgramDraf {
		return domain.Program{}, domain.ErrNotFound
	}
	return p, nil
}

// AmbilProgramPublik = halaman program untuk pengunjung: program, kabar, dan
// donatur terbaru. Nama donatur anonim diganti "Hamba Allah" DI SINI, jadi
// nama aslinya tidak pernah meninggalkan server.
func (s *Service) AmbilProgramPublik(ctx context.Context, slug string) (ProgramDetail, error) {
	p, err := s.programPublik(ctx, slug)
	if err != nil {
		return ProgramDetail{}, err
	}
	ks, err := s.st.ListKabar(ctx, p.ID)
	if err != nil {
		return ProgramDetail{}, fmt.Errorf("service.AmbilProgramPublik: %w", err)
	}
	ds, err := s.st.ListDonaturProgram(ctx, p.ID, donaturTampil)
	if err != nil {
		return ProgramDetail{}, fmt.Errorf("service.AmbilProgramPublik: %w", err)
	}
	donatur := make([]DonaturView, 0, len(ds))
	for _, d := range ds {
		nama := d.Nama
		if d.Anonim {
			nama = namaAnonim
		}
		donatur = append(donatur, DonaturView{Nama: nama, Jumlah: d.Jumlah, DibayarPada: d.DibayarPada})
	}
	return ProgramDetail{ProgramView: s.viewProgram(p), Kabar: viewKabar(ks), Donatur: donatur}, nil
}

// TambahKabar menambah kabar terbaru pada sebuah program.
func (s *Service) TambahKabar(ctx context.Context, userID, programID, judul, isi string) (KabarView, error) {
	judul, isi = strings.TrimSpace(judul), strings.TrimSpace(isi)
	g := galat{}
	if n := panjangRune(judul); n < 3 || n > 120 {
		g.tambah("judul", "Judul kabar harus 3–120 karakter")
	}
	if n := panjangRune(isi); n < 10 || n > 5_000 {
		g.tambah("isi", "Isi kabar harus 10–5.000 karakter")
	}
	if err := g.err(); err != nil {
		return KabarView{}, err
	}
	k := domain.KabarProgram{ProgramID: programID, Judul: judul, Isi: isi, DibuatOleh: userID}
	if err := s.st.CreateKabar(ctx, &k); err != nil {
		return KabarView{}, fmt.Errorf("service.TambahKabar: %w", err) // program tidak ada → FK 1452 → ErrNotFound
	}
	return viewKabar([]domain.KabarProgram{k})[0], nil
}

// HapusKabar menghapus satu kabar (soft delete).
func (s *Service) HapusKabar(ctx context.Context, programID, kabarID string) error {
	if err := s.st.DeleteKabar(ctx, programID, kabarID); err != nil {
		return fmt.Errorf("service.HapusKabar: %w", err)
	}
	return nil
}

// programDonasi memeriksa program tujuan sebuah donasi. kunci kosong = donasi
// umum (zero Program, nil). Program tidak ada atau tidak sah dicatat sebagai
// galat field "program"/"jenis" supaya formulir bisa menampilkannya; error
// database dikembalikan apa adanya. web=true (muzaki) mewajibkan program
// masih terbuka; petugas boleh mencatat untuk program aktif/selesai (mis. uang
// tunai yang baru disetor setelah batas waktu), tetapi tidak untuk draf.
func (s *Service) programDonasi(ctx context.Context, g galat, kunci string, jenis domain.JenisDonasi, web bool) (domain.Program, error) {
	if kunci == "" {
		return domain.Program{}, nil
	}
	var p domain.Program
	var err error
	if web {
		p, err = s.programLewatSlug(ctx, kunci)
	} else {
		p, err = s.st.GetProgram(ctx, kunci)
	}
	if errors.Is(err, domain.ErrNotFound) {
		g.tambah("program", "Program tidak ditemukan")
		return domain.Program{}, nil
	}
	if err != nil {
		return domain.Program{}, fmt.Errorf("service.programDonasi: %w", err)
	}
	if jenis != domain.DonasiInfak {
		g.tambah("jenis", "Program donasi hanya menerima infak; zakat disalurkan lewat donasi umum")
	}
	switch {
	case p.Status == domain.ProgramDraf:
		g.tambah("program", "Program belum diterbitkan")
	case web && !s.programTerbuka(p):
		g.tambah("program", "Program ini sudah tidak menerima donasi")
	}
	return p, nil
}
