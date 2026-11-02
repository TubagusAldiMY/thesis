package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// CMS konten kegiatan keagamaan: kegiatan (artikel/acara), jadwal rutin
// mingguan, dan pengumuman. Admin dan Ketua boleh mengelola (dicek di HTTP).
// Teks bebas hanya disimpan; halaman meng-escape-nya (teks biasa, bukan HTML).

// KategoriView = satu kategori kegiatan untuk pilihan di formulir dan chip filter.
type KategoriView struct {
	Kode  string `json:"kode"`
	Label string `json:"label"`
}

// kategoriKegiatan = daftar kategori yang sah, berurutan seperti di formulir.
// Menambah kategori cukup di sini (kolom DB-nya VARCHAR, tanpa migrasi).
var kategoriKegiatan = []KategoriView{
	{"maulid_nabi", "Maulid Nabi"},
	{"isra_miraj", "Isra Mi'raj"},
	{"tahun_baru_hijriah", "Tahun Baru Hijriah"},
	{"ramadan", "Ramadan"},
	{"idul_fitri", "Idulfitri"},
	{"idul_adha", "Iduladha"},
	{"kajian", "Kajian"},
	{"santunan", "Santunan & sosial"},
	{"pendidikan", "Pendidikan (TPQ)"},
	{"lainnya", "Lainnya"},
}

// DaftarKategoriKegiatan mengembalikan salinan daftar kategori.
func DaftarKategoriKegiatan() []KategoriView { return append([]KategoriView(nil), kategoriKegiatan...) }

func labelKategori(kode string) (string, bool) {
	for _, k := range kategoriKegiatan {
		if k.Kode == kode {
			return k.Label, true
		}
	}
	return "", false
}

var namaHari = [7]string{"Ahad", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

var reJam = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

const (
	// terdahuluTampil = banyak kegiatan lampau di halaman publik (tanpa paging).
	terdahuluTampil = 60
)

// ---------- bentuk keluaran ----------

// KegiatanView = kegiatan untuk pengurus maupun publik.
type KegiatanView struct {
	ID            string                `json:"id"`
	Slug          string                `json:"slug"`
	Judul         string                `json:"judul"`
	Kategori      string                `json:"kategori"`
	KategoriLabel string                `json:"kategori_label"`
	Ringkasan     string                `json:"ringkasan"`
	Isi           string                `json:"isi"`
	TanggalAcara  string                `json:"tanggal_acara"` // yyyy-mm-dd atau ""
	JamAcara      string                `json:"jam_acara"`
	Lokasi        string                `json:"lokasi"`
	Status        domain.StatusKegiatan `json:"status"`
	AdaGambar     bool                  `json:"ada_gambar"`
	GambarVersi   int64                 `json:"gambar_versi"` // penanda cache: berubah tiap kegiatan diubah
	Mendatang     bool                  `json:"mendatang"`    // tanggal acara hari ini atau nanti
	DibuatPada    time.Time             `json:"dibuat_pada"`
}

// KegiatanDaftarPengurus = semua kegiatan (draf dan terbit) + kategori untuk formulir.
type KegiatanDaftarPengurus struct {
	Kategori []KategoriView `json:"kategori"`
	Items    []KegiatanView `json:"items"`
}

// KegiatanDaftarPublik = kegiatan terbit: yang akan datang (terdekat dulu) dan
// yang sudah lewat atau tanpa tanggal (terbaru dulu).
type KegiatanDaftarPublik struct {
	Kategori  []KategoriView `json:"kategori"`
	Mendatang []KegiatanView `json:"mendatang"`
	Terdahulu []KegiatanView `json:"terdahulu"`
}

func (s *Service) viewKegiatan(k domain.Kegiatan) KegiatanView {
	label, _ := labelKategori(k.Kategori)
	v := KegiatanView{
		ID: k.ID, Slug: k.Slug, Judul: k.Judul, Kategori: k.Kategori, KategoriLabel: label,
		Ringkasan: k.Ringkasan, Isi: k.Isi, JamAcara: k.JamAcara, Lokasi: k.Lokasi, Status: k.Status,
		AdaGambar: k.AdaGambar, GambarVersi: k.UpdatedAt.UnixMilli(), DibuatPada: k.CreatedAt,
	}
	if k.TanggalAcara != nil {
		v.TanggalAcara = k.TanggalAcara.Format("2006-01-02")
		v.Mendatang = !k.TanggalAcara.Before(s.hariIni())
	}
	return v
}

// JadwalView = satu jadwal rutin mingguan.
type JadwalView struct {
	ID         string `json:"id"`
	Judul      string `json:"judul"`
	Pemateri   string `json:"pemateri"`
	Hari       int    `json:"hari"` // 0 = Ahad ... 6 = Sabtu
	HariLabel  string `json:"hari_label"`
	Jam        string `json:"jam"`
	Lokasi     string `json:"lokasi"`
	Keterangan string `json:"keterangan"`
	Aktif      bool   `json:"aktif"`
}

func viewJadwal(j domain.JadwalRutin) JadwalView {
	return JadwalView{
		ID: j.ID, Judul: j.Judul, Pemateri: j.Pemateri, Hari: j.Hari, HariLabel: namaHari[j.Hari%7], Jam: j.Jam,
		Lokasi: j.Lokasi, Keterangan: j.Keterangan, Aktif: j.Aktif,
	}
}

// PengumumanView = satu pengumuman; Tayang = terbit dan masih dalam masa tayang.
type PengumumanView struct {
	ID           string    `json:"id"`
	Judul        string    `json:"judul"`
	Isi          string    `json:"isi"`
	TampilSampai string    `json:"tampil_sampai"` // yyyy-mm-dd atau ""
	Terbit       bool      `json:"terbit"`
	Tayang       bool      `json:"tayang"`
	DibuatPada   time.Time `json:"dibuat_pada"`
}

func (s *Service) viewPengumuman(p domain.Pengumuman) PengumumanView {
	v := PengumumanView{ID: p.ID, Judul: p.Judul, Isi: p.Isi, Terbit: p.Terbit, DibuatPada: p.CreatedAt}
	v.Tayang = p.Terbit
	if p.TampilSampai != nil {
		v.TampilSampai = p.TampilSampai.Format("2006-01-02")
		v.Tayang = v.Tayang && !s.hariIni().After(*p.TampilSampai)
	}
	return v
}

// parseTanggal mengubah "yyyy-mm-dd" (boleh kosong) menjadi tanggal. Kosong → nil.
func parseTanggal(g galat, field, nilai string) *time.Time {
	nilai = strings.TrimSpace(nilai)
	if nilai == "" {
		return nil
	}
	w, err := time.Parse("2006-01-02", nilai)
	if err != nil {
		g.tambah(field, "Tanggal tidak valid")
		return nil
	}
	return &w
}

// simpanDenganSlug memanggil simpan dengan slug dasar; bila bentrok dengan
// konten lain dicoba lagi dengan akhiran acak pendek (maksimal 3 kali).
func simpanDenganSlug(dasar string, simpan func(slug string) error) error {
	var err error
	for coba := 0; coba <= 3; coba++ {
		slug := dasar
		if coba > 0 {
			kode, kerr := kodeAcak()
			if kerr != nil {
				return kerr
			}
			slug = dasar + "-" + kode[:4]
		}
		if err = simpan(slug); !errors.Is(err, domain.ErrConflict) {
			return err
		}
	}
	return err
}

// ---------- kegiatan ----------

// KegiatanInput = isian pengurus saat membuat/mengubah kegiatan.
type KegiatanInput struct {
	Judul        string
	Kategori     string
	Ringkasan    string
	Isi          string
	TanggalAcara string // yyyy-mm-dd, kosong = tanpa tanggal acara
	JamAcara     string // HH:MM, hanya bila ada tanggal
	Lokasi       string
	Status       domain.StatusKegiatan
}

func validasiKegiatan(in KegiatanInput) (domain.Kegiatan, error) {
	g := galat{}
	k := domain.Kegiatan{
		Judul: strings.TrimSpace(in.Judul), Kategori: in.Kategori, Ringkasan: strings.TrimSpace(in.Ringkasan),
		Isi: strings.TrimSpace(in.Isi), JamAcara: strings.TrimSpace(in.JamAcara), Lokasi: strings.TrimSpace(in.Lokasi),
		Status: in.Status,
	}
	if n := panjangRune(k.Judul); n < 5 || n > 120 {
		g.tambah("judul", "Judul harus 5–120 karakter")
	}
	if _, ok := labelKategori(k.Kategori); !ok {
		g.tambah("kategori", "Pilih kategori kegiatan")
	}
	if n := panjangRune(k.Ringkasan); n < 10 || n > 200 {
		g.tambah("ringkasan", "Ringkasan harus 10–200 karakter")
	}
	if n := panjangRune(k.Isi); n < 20 || n > 10_000 {
		g.tambah("isi", "Isi harus 20–10.000 karakter")
	}
	k.TanggalAcara = parseTanggal(g, "tanggal_acara", in.TanggalAcara)
	if k.JamAcara != "" {
		switch {
		case !reJam.MatchString(k.JamAcara):
			g.tambah("jam_acara", "Jam harus berformat HH:MM, mis. 19:30")
		case k.TanggalAcara == nil:
			g.tambah("jam_acara", "Isi tanggal acara bila mengisi jam")
		}
	}
	if panjangRune(k.Lokasi) > 120 {
		g.tambah("lokasi", "Lokasi maksimal 120 karakter")
	}
	if k.Status != domain.KegiatanDraf && k.Status != domain.KegiatanTerbit {
		g.tambah("status", "Pilih draf atau terbit")
	}
	return k, g.err()
}

// BuatKegiatan membuat kegiatan baru. Slug dari judul; bentrok → akhiran acak.
func (s *Service) BuatKegiatan(ctx context.Context, userID string, in KegiatanInput) (KegiatanView, error) {
	k, err := validasiKegiatan(in)
	if err != nil {
		return KegiatanView{}, err
	}
	k.DibuatOleh = userID
	err = simpanDenganSlug(buatSlug(k.Judul), func(slug string) error {
		k.Slug = slug
		return s.st.CreateKegiatan(ctx, &k)
	})
	if err != nil {
		return KegiatanView{}, fmt.Errorf("service.BuatKegiatan: %w", err)
	}
	return s.viewKegiatan(k), nil
}

// UbahKegiatan mengubah isian dan status kegiatan (slug tetap).
func (s *Service) UbahKegiatan(ctx context.Context, id string, in KegiatanInput) (KegiatanView, error) {
	k, err := validasiKegiatan(in)
	if err != nil {
		return KegiatanView{}, err
	}
	k.ID = id
	if err := s.st.UpdateKegiatan(ctx, k); err != nil {
		return KegiatanView{}, fmt.Errorf("service.UbahKegiatan: %w", err)
	}
	baru, err := s.st.GetKegiatan(ctx, id)
	if err != nil {
		return KegiatanView{}, fmt.Errorf("service.UbahKegiatan: %w", err)
	}
	return s.viewKegiatan(baru), nil
}

// HapusKegiatan menghapus (soft delete) satu kegiatan.
func (s *Service) HapusKegiatan(ctx context.Context, id string) error {
	if err := s.st.DeleteKegiatan(ctx, id); err != nil {
		return fmt.Errorf("service.HapusKegiatan: %w", err)
	}
	return nil
}

// AmbilKegiatan = satu kegiatan untuk pengurus (draf juga).
func (s *Service) AmbilKegiatan(ctx context.Context, id string) (KegiatanView, error) {
	k, err := s.st.GetKegiatan(ctx, id)
	if err != nil {
		return KegiatanView{}, fmt.Errorf("service.AmbilKegiatan: %w", err)
	}
	return s.viewKegiatan(k), nil
}

// DaftarKegiatan = semua kegiatan (draf dan terbit) untuk pengurus.
func (s *Service) DaftarKegiatan(ctx context.Context) (KegiatanDaftarPengurus, error) {
	ks, err := s.st.ListKegiatan(ctx, store.KegiatanFilter{})
	if err != nil {
		return KegiatanDaftarPengurus{}, fmt.Errorf("service.DaftarKegiatan: %w", err)
	}
	d := KegiatanDaftarPengurus{Kategori: DaftarKategoriKegiatan(), Items: make([]KegiatanView, 0, len(ks))}
	for _, k := range ks {
		d.Items = append(d.Items, s.viewKegiatan(k))
	}
	return d, nil
}

// DaftarKegiatanPublik = kegiatan terbit, dipisah mendatang dan terdahulu.
// kategori kosong = semua; kategori tidak dikenal → galat validasi.
func (s *Service) DaftarKegiatanPublik(ctx context.Context, kategori string) (KegiatanDaftarPublik, error) {
	if kategori != "" {
		if _, ok := labelKategori(kategori); !ok {
			return KegiatanDaftarPublik{}, (galat{"kategori": "Kategori tidak dikenal"}).err()
		}
	}
	ks, err := s.st.ListKegiatan(ctx, store.KegiatanFilter{Status: domain.KegiatanTerbit, Kategori: kategori})
	if err != nil {
		return KegiatanDaftarPublik{}, fmt.Errorf("service.DaftarKegiatanPublik: %w", err)
	}
	d := KegiatanDaftarPublik{Kategori: DaftarKategoriKegiatan(), Mendatang: []KegiatanView{}, Terdahulu: []KegiatanView{}}
	// Store mengurutkan terbaru dulu; yang akan datang dibalik supaya terdekat di atas.
	var depan []KegiatanView
	for _, k := range ks {
		v := s.viewKegiatan(k)
		if v.Mendatang {
			depan = append(depan, v)
		} else if len(d.Terdahulu) < terdahuluTampil {
			d.Terdahulu = append(d.Terdahulu, v)
		}
	}
	for i := len(depan) - 1; i >= 0; i-- {
		d.Mendatang = append(d.Mendatang, depan[i])
	}
	return d, nil
}

// kegiatanPublik mengambil kegiatan lewat slug untuk pengunjung; slug salah
// format dan draf diperlakukan sama dengan tidak ada (404).
func (s *Service) kegiatanPublik(ctx context.Context, slug string) (domain.Kegiatan, error) {
	if !reSlug.MatchString(slug) {
		return domain.Kegiatan{}, domain.ErrNotFound
	}
	k, err := s.st.GetKegiatanBySlug(ctx, slug)
	if err != nil {
		return domain.Kegiatan{}, fmt.Errorf("service.kegiatanPublik: %w", err)
	}
	if k.Status != domain.KegiatanTerbit {
		return domain.Kegiatan{}, domain.ErrNotFound
	}
	return k, nil
}

// AmbilKegiatanPublik = halaman satu kegiatan terbit.
func (s *Service) AmbilKegiatanPublik(ctx context.Context, slug string) (KegiatanView, error) {
	k, err := s.kegiatanPublik(ctx, slug)
	if err != nil {
		return KegiatanView{}, err
	}
	return s.viewKegiatan(k), nil
}

// SimpanGambarKegiatan mengganti gambar sampul (aturan sama dengan program donasi).
func (s *Service) SimpanGambarKegiatan(ctx context.Context, id string, isi []byte) error {
	if len(isi) == 0 || len(isi) > GambarMaks {
		return (galat{"gambar": "Pilih gambar berukuran maksimal 2 MB"}).err()
	}
	tipe := tipeGambar(isi)
	if tipe == "" {
		return (galat{"gambar": "Gambar harus berformat JPG, PNG, atau WebP"}).err()
	}
	if err := s.st.SetGambarKegiatan(ctx, id, isi, tipe); err != nil {
		return fmt.Errorf("service.SimpanGambarKegiatan: %w", err)
	}
	return nil
}

// AmbilGambarKegiatan untuk pengurus (semua status, lewat ID).
func (s *Service) AmbilGambarKegiatan(ctx context.Context, id string) ([]byte, string, error) {
	isi, tipe, err := s.st.GetGambarKegiatan(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("service.AmbilGambarKegiatan: %w", err)
	}
	return isi, tipe, nil
}

// AmbilGambarKegiatanPublik = gambar sampul lewat slug; draf dianggap tidak ada.
func (s *Service) AmbilGambarKegiatanPublik(ctx context.Context, slug string) ([]byte, string, error) {
	k, err := s.kegiatanPublik(ctx, slug)
	if err != nil {
		return nil, "", err
	}
	return s.AmbilGambarKegiatan(ctx, k.ID)
}

// ---------- jadwal rutin ----------

// JadwalInput = isian pengurus untuk jadwal rutin mingguan.
type JadwalInput struct {
	Judul      string
	Pemateri   string
	Hari       int
	Jam        string
	Lokasi     string
	Keterangan string
	Aktif      bool
}

func validasiJadwal(in JadwalInput) (domain.JadwalRutin, error) {
	g := galat{}
	j := domain.JadwalRutin{
		Judul: strings.TrimSpace(in.Judul), Pemateri: strings.TrimSpace(in.Pemateri), Hari: in.Hari,
		Jam: strings.TrimSpace(in.Jam), Lokasi: strings.TrimSpace(in.Lokasi), Keterangan: strings.TrimSpace(in.Keterangan),
		Aktif: in.Aktif,
	}
	if n := panjangRune(j.Judul); n < 3 || n > 120 {
		g.tambah("judul", "Judul harus 3–120 karakter")
	}
	if panjangRune(j.Pemateri) > 100 {
		g.tambah("pemateri", "Nama pemateri maksimal 100 karakter")
	}
	if j.Hari < 0 || j.Hari > 6 {
		g.tambah("hari", "Pilih hari")
	}
	if !reJam.MatchString(j.Jam) {
		g.tambah("jam", "Jam harus berformat HH:MM, mis. 05:30")
	}
	if panjangRune(j.Lokasi) > 120 {
		g.tambah("lokasi", "Lokasi maksimal 120 karakter")
	}
	if panjangRune(j.Keterangan) > 200 {
		g.tambah("keterangan", "Keterangan maksimal 200 karakter")
	}
	return j, g.err()
}

// DaftarJadwal = semua jadwal (aktif dan nonaktif) untuk pengurus.
func (s *Service) DaftarJadwal(ctx context.Context) ([]JadwalView, error) {
	return s.daftarJadwal(ctx, false)
}

// DaftarJadwalPublik = jadwal aktif, menurut hari lalu jam.
func (s *Service) DaftarJadwalPublik(ctx context.Context) ([]JadwalView, error) {
	return s.daftarJadwal(ctx, true)
}

func (s *Service) daftarJadwal(ctx context.Context, hanyaAktif bool) ([]JadwalView, error) {
	js, err := s.st.ListJadwal(ctx, hanyaAktif)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarJadwal: %w", err)
	}
	out := make([]JadwalView, 0, len(js))
	for _, j := range js {
		out = append(out, viewJadwal(j))
	}
	return out, nil
}

// BuatJadwal membuat jadwal rutin baru.
func (s *Service) BuatJadwal(ctx context.Context, in JadwalInput) (JadwalView, error) {
	j, err := validasiJadwal(in)
	if err != nil {
		return JadwalView{}, err
	}
	if err := s.st.CreateJadwal(ctx, &j); err != nil {
		return JadwalView{}, fmt.Errorf("service.BuatJadwal: %w", err)
	}
	return viewJadwal(j), nil
}

// UbahJadwal mengubah seluruh isian jadwal.
func (s *Service) UbahJadwal(ctx context.Context, id string, in JadwalInput) (JadwalView, error) {
	j, err := validasiJadwal(in)
	if err != nil {
		return JadwalView{}, err
	}
	j.ID = id
	if err := s.st.UpdateJadwal(ctx, j); err != nil {
		return JadwalView{}, fmt.Errorf("service.UbahJadwal: %w", err)
	}
	return viewJadwal(j), nil
}

// HapusJadwal menghapus (soft delete) satu jadwal.
func (s *Service) HapusJadwal(ctx context.Context, id string) error {
	if err := s.st.DeleteJadwal(ctx, id); err != nil {
		return fmt.Errorf("service.HapusJadwal: %w", err)
	}
	return nil
}

// ---------- pengumuman ----------

// PengumumanInput = isian pengurus untuk pengumuman singkat.
type PengumumanInput struct {
	Judul        string
	Isi          string
	TampilSampai string // yyyy-mm-dd, kosong = tayang terus
	Terbit       bool
}

func (s *Service) validasiPengumuman(in PengumumanInput, baru bool) (domain.Pengumuman, error) {
	g := galat{}
	p := domain.Pengumuman{Judul: strings.TrimSpace(in.Judul), Isi: strings.TrimSpace(in.Isi), Terbit: in.Terbit}
	if n := panjangRune(p.Judul); n < 3 || n > 120 {
		g.tambah("judul", "Judul harus 3–120 karakter")
	}
	if n := panjangRune(p.Isi); n < 10 || n > 500 {
		g.tambah("isi", "Isi harus 10–500 karakter")
	}
	p.TampilSampai = parseTanggal(g, "tampil_sampai", in.TampilSampai)
	if baru && p.TampilSampai != nil && p.TampilSampai.Before(s.hariIni()) {
		g.tambah("tampil_sampai", "Masa tayang tidak boleh berakhir sebelum hari ini")
	}
	return p, g.err()
}

// DaftarPengumuman = semua pengumuman (termasuk yang sudah berakhir) untuk pengurus.
func (s *Service) DaftarPengumuman(ctx context.Context) ([]PengumumanView, error) {
	return s.daftarPengumuman(ctx, nil)
}

// DaftarPengumumanPublik = pengumuman terbit yang masih dalam masa tayang hari ini (WIB).
func (s *Service) DaftarPengumumanPublik(ctx context.Context) ([]PengumumanView, error) {
	hari := s.hariIni()
	return s.daftarPengumuman(ctx, &hari)
}

func (s *Service) daftarPengumuman(ctx context.Context, tayangPada *time.Time) ([]PengumumanView, error) {
	ps, err := s.st.ListPengumuman(ctx, tayangPada)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarPengumuman: %w", err)
	}
	out := make([]PengumumanView, 0, len(ps))
	for _, p := range ps {
		out = append(out, s.viewPengumuman(p))
	}
	return out, nil
}

// BuatPengumuman membuat pengumuman baru.
func (s *Service) BuatPengumuman(ctx context.Context, in PengumumanInput) (PengumumanView, error) {
	p, err := s.validasiPengumuman(in, true)
	if err != nil {
		return PengumumanView{}, err
	}
	if err := s.st.CreatePengumuman(ctx, &p); err != nil {
		return PengumumanView{}, fmt.Errorf("service.BuatPengumuman: %w", err)
	}
	return s.viewPengumuman(p), nil
}

// UbahPengumuman mengubah seluruh isian pengumuman (masa tayang boleh lampau).
func (s *Service) UbahPengumuman(ctx context.Context, id string, in PengumumanInput) (PengumumanView, error) {
	p, err := s.validasiPengumuman(in, false)
	if err != nil {
		return PengumumanView{}, err
	}
	p.ID = id
	if err := s.st.UpdatePengumuman(ctx, p); err != nil {
		return PengumumanView{}, fmt.Errorf("service.UbahPengumuman: %w", err)
	}
	baru, err := s.st.GetPengumuman(ctx, id)
	if err != nil {
		return PengumumanView{}, fmt.Errorf("service.UbahPengumuman: %w", err)
	}
	return s.viewPengumuman(baru), nil
}

// HapusPengumuman menghapus (soft delete) satu pengumuman.
func (s *Service) HapusPengumuman(ctx context.Context, id string) error {
	if err := s.st.DeletePengumuman(ctx, id); err != nil {
		return fmt.Errorf("service.HapusPengumuman: %w", err)
	}
	return nil
}
