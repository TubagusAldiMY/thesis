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
	d := Dashboard{AsnafCounts: m.AsnafCounts, Terbaru: ps, TotalTersalurSemua: total}
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
