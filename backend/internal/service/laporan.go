package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// DashboardPeriode = periode aktif (terbaru yang belum selesai) beserta posisi
// stepper 5 langkah (FR-18, docs/04 bagian 4).
type DashboardPeriode struct {
	Periode          domain.Periode `json:"periode"`
	Kuota            int            `json:"kuota"`
	LangkahAktif     int            `json:"langkah_aktif"` // 1 kandidat · 2 penilaian · 3 hitung · 4 persetujuan · 5 penyaluran
	JumlahKandidat   int            `json:"jumlah_kandidat"`
	PenilaianLengkap int            `json:"penilaian_lengkap"`
	TotalTersalur    int64          `json:"total_tersalur"`
}

// Dashboard = isi halaman Beranda.
type Dashboard struct {
	MustahikAktif   int                  `json:"mustahik_aktif"`
	AsnafCounts     map[domain.Asnaf]int `json:"asnaf_counts"`
	PeriodeBerjalan int                  `json:"periode_berjalan"`
	Aktif           *DashboardPeriode    `json:"aktif"` // nil bila semua periode selesai / belum ada
	Terbaru         []domain.Periode     `json:"terbaru"`
	// TotalTersalurSemua = realisasi semua periode; ditampilkan saat Aktif nil
	// supaya kartu "Total tersalur" tidak jatuh ke Rp 0 setelah periode ditutup.
	TotalTersalurSemua int64 `json:"total_tersalur_semua"`
	// Keuangan = ringkasan dana masjid secara umum (penerimaan vs penyaluran).
	Keuangan Keuangan `json:"keuangan"`
}

// DanaJenis = pembukuan satu jenis dana. Saldo = Diterima − Tersalur; bisa
// negatif bila ada penyaluran dari sebelum donasi dicatat di aplikasi ini.
type DanaJenis struct {
	Diterima int64 `json:"diterima"`
	Tersalur int64 `json:"tersalur"`
	Saldo    int64 `json:"saldo"`
}

// ArusBulan = penerimaan donasi dan penyaluran dalam satu bulan kalender WIB.
type ArusBulan struct {
	Bulan      string `json:"bulan"` // "2026-10"
	Penerimaan int64  `json:"penerimaan"`
	Penyaluran int64  `json:"penyaluran"`
}

// DonasiTerbaru = satu baris daftar donasi terbaru di Beranda. Sengaja tanpa
// No. HP: Beranda hanya butuh siapa, berapa, dan kapan.
type DonasiTerbaru struct {
	ID          string              `json:"id"`
	Nama        string              `json:"nama"`
	Jenis       domain.JenisDonasi  `json:"jenis"`
	Jumlah      int64               `json:"jumlah"`
	Sumber      domain.SumberDonasi `json:"sumber"`
	DibayarPada *time.Time          `json:"dibayar_pada"`
}

// Keuangan = isi bagian "dana masjid" di Beranda. Zakat dan infak dibukukan
// terpisah (dana zakat hanya boleh disalurkan ke delapan asnaf), sehingga
// saldonya tidak digabung per jenis.
type Keuangan struct {
	Zakat            DanaJenis                    `json:"zakat"`
	Infak            DanaJenis                    `json:"infak"`
	TotalDiterima    int64                        `json:"total_diterima"`
	TotalTersalur    int64                        `json:"total_tersalur"`
	Saldo            int64                        `json:"saldo"`
	DiterimaBulanIni int64                        `json:"diterima_bulan_ini"`
	JumlahDonasi     int                          `json:"jumlah_donasi"`
	PerJenis         map[domain.JenisDonasi]int64 `json:"per_jenis"`
	Bulanan          []ArusBulan                  `json:"bulanan"` // 12 bulan, terlama dulu
	DonasiTerbaru    []DonasiTerbaru              `json:"donasi_terbaru"`
}

// ambilKeuangan menyusun Keuangan dari beberapa agregat kecil. Hanya donasi
// berstatus diterima yang dihitung; bulan mengikuti kalender WIB.
func (s *Service) ambilKeuangan(ctx context.Context) (Keuangan, error) {
	gagal := func(err error) (Keuangan, error) {
		return Keuangan{}, fmt.Errorf("service.ambilKeuangan: %w", err)
	}
	terima, err := s.st.PenerimaanPerJenis(ctx)
	if err != nil {
		return gagal(err)
	}
	salur, err := s.st.PenyaluranPerJenis(ctx)
	if err != nil {
		return gagal(err)
	}
	// 12 bulan berakhir bulan ini; time.Date menormalkan bulan <= 0 ke tahun lalu.
	now := s.sekarang().In(wib)
	bulan := make([]string, 12)
	for i := range bulan {
		bulan[i] = time.Date(now.Year(), now.Month()-time.Month(11-i), 1, 0, 0, 0, 0, wib).Format("2006-01")
	}
	masuk, err := s.st.PenerimaanPerBulan(ctx, bulan[0])
	if err != nil {
		return gagal(err)
	}
	keluar, err := s.st.PenyaluranPerBulan(ctx, bulan[0])
	if err != nil {
		return gagal(err)
	}
	ringkas, err := s.st.RingkasDonasi(ctx)
	if err != nil {
		return gagal(err)
	}
	terbaru, err := s.st.ListDonasi(ctx, store.DonasiFilter{Status: domain.DonasiDiterima, Page: 1, PerPage: 5})
	if err != nil {
		return gagal(err)
	}

	jenis := func(diterima, tersalur int64) DanaJenis {
		return DanaJenis{Diterima: diterima, Tersalur: tersalur, Saldo: diterima - tersalur}
	}
	k := Keuangan{
		Zakat: jenis(terima[domain.DonasiZakatFitrah]+terima[domain.DonasiZakatMal]+terima[domain.DonasiZakatProfesi], salur[domain.JenisZakat]),
		Infak: jenis(terima[domain.DonasiInfak], salur[domain.JenisInfak]),
		// JumlahDonasi = jumlah kuitansi aktif (status diterima).
		JumlahDonasi:     ringkas.JumlahDiterima,
		DiterimaBulanIni: masuk[bulan[11]],
		PerJenis:         terima,
		Bulanan:          make([]ArusBulan, len(bulan)),
		DonasiTerbaru:    make([]DonasiTerbaru, 0, len(terbaru.Items)),
	}
	k.TotalDiterima = k.Zakat.Diterima + k.Infak.Diterima
	k.TotalTersalur = k.Zakat.Tersalur + k.Infak.Tersalur
	k.Saldo = k.TotalDiterima - k.TotalTersalur
	for i, b := range bulan {
		k.Bulanan[i] = ArusBulan{Bulan: b, Penerimaan: masuk[b], Penyaluran: keluar[b]}
	}
	for _, d := range terbaru.Items {
		k.DonasiTerbaru = append(k.DonasiTerbaru, DonasiTerbaru{
			ID: d.ID, Nama: d.Nama, Jenis: d.Jenis, Jumlah: d.Jumlah, Sumber: d.Sumber, DibayarPada: d.DibayarPada,
		})
	}
	return k, nil
}

// langkahAktif menentukan posisi stepper dari data periode.
func langkahAktif(p domain.Periode, kandidat, lengkap int) int {
	switch {
	case p.Status == domain.StatusDisetujui:
		return 5
	case p.Status == domain.StatusDihitung || p.Status == domain.StatusDitolak:
		return 4
	case kandidat == 0:
		return 1
	case lengkap < kandidat:
		return 2
	default:
		return 3
	}
}

// AmbilDashboard menyusun ringkasan Beranda. ponytail: beberapa query kecil
// per muat halaman; cukup untuk ≤ 500 mustahik (NFR-06).
func (s *Service) AmbilDashboard(ctx context.Context) (Dashboard, error) {
	m, err := s.st.ListMustahik(ctx, store.MustahikFilter{Page: 1, PerPage: 1})
	if err != nil {
		return Dashboard{}, fmt.Errorf("service.AmbilDashboard: %w", err)
	}
	ps, err := s.st.ListPeriode(ctx, "")
	if err != nil {
		return Dashboard{}, fmt.Errorf("service.AmbilDashboard: %w", err)
	}
	total, err := s.st.TotalPenyaluran(ctx)
	if err != nil {
		return Dashboard{}, fmt.Errorf("service.AmbilDashboard: %w", err)
	}
	keu, err := s.ambilKeuangan(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	d := Dashboard{AsnafCounts: m.AsnafCounts, Terbaru: ps, TotalTersalurSemua: total, Keuangan: keu}
	for _, n := range m.AsnafCounts {
		d.MustahikAktif += n
	}
	if len(d.Terbaru) > 5 {
		d.Terbaru = d.Terbaru[:5]
	}
	for _, p := range ps { // ps terurut terbaru dulu
		if p.Status == domain.StatusSelesai {
			continue
		}
		d.PeriodeBerjalan++
		if d.Aktif != nil {
			continue
		}
		pen, err := s.AmbilPenilaian(ctx, p.ID)
		if err != nil {
			return Dashboard{}, err
		}
		rek, err := s.DaftarPenyaluran(ctx, p.ID)
		if err != nil {
			return Dashboard{}, err
		}
		d.Aktif = &DashboardPeriode{
			Periode: p, Kuota: p.Kuota(), LangkahAktif: langkahAktif(p, pen.Total, pen.Lengkap),
			JumlahKandidat: pen.Total, PenilaianLengkap: pen.Lengkap, TotalTersalur: rek.TotalTersalur,
		}
	}
	return d, nil
}

// KriteriaLaporan = baris bagian "A. Kriteria & bobot" (bobot dari potret run,
// jadi laporan tetap benar walau bobot diubah sesudahnya).
type KriteriaLaporan struct {
	Kode  string              `json:"kode"`
	Nama  string              `json:"nama"`
	Tipe  domain.TipeKriteria `json:"tipe"`
	Bobot float64             `json:"bobot"`
}

// Laporan = isi template laporan periode (FR-14).
type Laporan struct {
	Periode       domain.Periode    `json:"periode"`
	Kuota         int               `json:"kuota"`
	Run           domain.ArasRun    `json:"run"`
	Kriteria      []KriteriaLaporan `json:"kriteria"`
	Ranking       []HasilArasItem   `json:"ranking"`
	Penyaluran    []BarisPenyaluran `json:"penyaluran"` // hanya yang sudah tercatat
	TotalTersalur int64             `json:"total_tersalur"`
	AdminNama     string            `json:"admin_nama"`
	KetuaNama     string            `json:"ketua_nama"` // kosong bila belum disetujui
	DisetujuiPada *time.Time        `json:"disetujui_pada"`
	DibuatPada    time.Time         `json:"dibuat_pada"`
}

// AmbilLaporan menyusun laporan dari run terkini. ErrNotFound bila periode
// belum pernah dihitung (belum ada yang bisa dilaporkan).
func (s *Service) AmbilLaporan(ctx context.Context, periodeID string) (Laporan, error) {
	hasil, err := s.AmbilHasilAras(ctx, periodeID, 0)
	if err != nil {
		return Laporan{}, err
	}
	rek, err := s.DaftarPenyaluran(ctx, periodeID)
	if err != nil {
		return Laporan{}, err
	}
	ks, err := s.st.ListKriteria(ctx, false)
	if err != nil {
		return Laporan{}, fmt.Errorf("service.AmbilLaporan: %w", err)
	}
	nama := make(map[string]string, len(ks))
	for _, k := range ks {
		nama[k.Kode] = k.Nama
	}
	l := Laporan{
		Periode: hasil.Periode, Kuota: hasil.Kuota, Run: hasil.Run, Ranking: hasil.Items,
		Penyaluran: []BarisPenyaluran{}, TotalTersalur: rek.TotalTersalur, DibuatPada: time.Now().UTC(),
	}
	for _, b := range hasil.Run.Bobot {
		l.Kriteria = append(l.Kriteria, KriteriaLaporan{Kode: b.Kode, Nama: nama[b.Kode], Tipe: b.Tipe, Bobot: b.Bobot})
	}
	for _, b := range rek.Baris {
		if b.Penyaluran != nil {
			l.Penyaluran = append(l.Penyaluran, b)
		}
	}
	if u, err := s.st.GetUser(ctx, hasil.Periode.CreatedBy); err == nil {
		l.AdminNama = u.Nama
	}
	kep, err := s.st.ListPersetujuan(ctx, periodeID)
	if err != nil {
		return Laporan{}, fmt.Errorf("service.AmbilLaporan: %w", err)
	}
	for _, k := range kep {
		if k.RunID == hasil.Run.ID && k.Keputusan == domain.KeputusanSetuju {
			l.KetuaNama, l.DisetujuiPada = k.OlehNama, &k.CreatedAt
		}
	}
	return l, nil
}

// selCSV mencegah CSV/formula injection: sel yang diawali = + - @ (atau tab/CR)
// akan dijalankan sebagai rumus oleh Excel, jadi diberi awalan petik.
func selCSV(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// TulisCSV menulis laporan sebagai CSV (FR-14) dengan bagian yang sama seperti
// template cetak. Angka ditulis polos (titik desimal) supaya bisa diolah.
func (l Laporan) TulisCSV(w io.Writer) error {
	c := csv.NewWriter(w)
	f4 := func(x float64) string { return strconv.FormatFloat(x, 'f', 4, 64) }
	baris := [][]string{
		{"Laporan Penyaluran", selCSV(l.Periode.Nama)},
		{"Jenis", string(l.Periode.Jenis)},
		{"Dana tersedia", strconv.FormatInt(l.Periode.DanaTersedia, 10)},
		{"Nominal per penerima", strconv.FormatInt(l.Periode.NominalPerPenerima, 10)},
		{"Kuota", strconv.Itoa(l.Kuota)},
		{"Status", string(l.Periode.Status)},
		{},
		{"A. Kriteria & bobot"},
		{"Kode", "Nama", "Tipe", "Bobot"},
	}
	for _, k := range l.Kriteria {
		baris = append(baris, []string{k.Kode, selCSV(k.Nama), string(k.Tipe), f4(k.Bobot)})
	}
	baris = append(baris, []string{}, []string{"B. Ranking ARAS (run v" + strconv.Itoa(l.Run.Versi) + ")"},
		[]string{"Ranking", "Nama", "Asnaf", "S", "K", "Keterangan"})
	for _, r := range l.Ranking {
		ket := "Cadangan"
		if r.Status == domain.HasilUsulan {
			ket = "Penerima"
		}
		baris = append(baris, []string{strconv.Itoa(r.Ranking), selCSV(r.Mustahik.Nama), string(r.Mustahik.Asnaf), f4(r.S), f4(r.K), ket})
	}
	baris = append(baris, []string{}, []string{"C. Realisasi penyaluran"}, []string{"Nama", "Tanggal", "Nominal"})
	for _, b := range l.Penyaluran {
		baris = append(baris, []string{selCSV(b.Mustahik.Nama), b.Penyaluran.Tanggal.Format(time.DateOnly), strconv.FormatInt(b.Penyaluran.Nominal, 10)})
	}
	baris = append(baris, []string{"Total", "", strconv.FormatInt(l.TotalTersalur, 10)})
	if err := c.WriteAll(baris); err != nil {
		return fmt.Errorf("laporan CSV: %w", err)
	}
	return nil
}
