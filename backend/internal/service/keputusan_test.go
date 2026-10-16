package service

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"spkzakat/internal/domain"
)

// dihitung = contoh dokumen 02 (kuota 3: A2, A4, A1 usulan; A5, A3 cadangan)
// yang sudah dihitung, plus akun Ketua.
func dihitung(t *testing.T) (*contoh, domain.User) {
	t.Helper()
	c := siapkanContoh(t)
	c.isi(t)
	if _, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user); err != nil {
		t.Fatal(err)
	}
	return c, c.e.akun(t, "ketua1", domain.RoleKetua)
}

func cekKonflik(t *testing.T, err error) {
	t.Helper()
	var ce *domain.ConflictError
	if !errors.As(err, &ce) || !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, mau ConflictError", err)
	}
}

func TestPutuskan_Validasi(t *testing.T) {
	c, ketua := dihitung(t)
	tests := []struct {
		nama    string
		kep     domain.Keputusan
		catatan string
		field   string
	}{
		{"keputusan tak dikenal", "mungkin", "", "keputusan"},
		{"tolak tanpa catatan", domain.KeputusanTolak, "  ", "catatan"},
		{"catatan terlalu panjang", domain.KeputusanSetuju, strings.Repeat("a", 1001), "catatan"},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			_, err := c.e.svc.Putuskan(bg, c.periode.ID, ketua.ID, tt.kep, tt.catatan)
			cekValidasi(t, err, tt.field)
		})
	}
	if st := c.statusPeriode(t); st != domain.StatusDihitung {
		t.Fatalf("validasi gagal tidak boleh mengubah status, sekarang %s", st)
	}
}

func TestPutuskan_TolakLaluBukaKembaliLaluSetuju(t *testing.T) {
	c, ketua := dihitung(t)
	if _, err := c.e.svc.Putuskan(bg, c.periode.ID, ketua.ID, domain.KeputusanTolak, "Data A3 perlu dicek ulang"); err != nil {
		t.Fatal(err)
	}
	if st := c.statusPeriode(t); st != domain.StatusDitolak {
		t.Fatalf("status = %s, mau ditolak", st)
	}
	d, err := c.e.svc.AmbilDetailPeriode(bg, c.periode.ID)
	if err != nil || d.CatatanPenolakan != "Data A3 perlu dicek ulang" || d.Keputusan[0].OlehNama != ketua.Nama {
		t.Fatalf("detail = %+v, %v", d, err)
	}
	_, err = c.e.svc.Putuskan(bg, c.periode.ID, ketua.ID, domain.KeputusanSetuju, "")
	cekKonflik(t, err) // sudah ditolak, tidak bisa diputuskan lagi

	cekKonflik(t, c.e.svc.TutupPeriode(bg, c.periode.ID))
	if err := c.e.svc.BukaKembali(bg, c.periode.ID); err != nil {
		t.Fatal(err)
	}
	cekKonflik(t, c.e.svc.BukaKembali(bg, c.periode.ID)) // sudah draft
	if st := c.statusPeriode(t); st != domain.StatusDraft {
		t.Fatalf("status = %s, mau draft", st)
	}

	// Run lama sudah diputuskan; keputusan baru butuh hitung ulang (run v2).
	if _, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user); err != nil {
		t.Fatal(err)
	}
	p, err := c.e.svc.Putuskan(bg, c.periode.ID, ketua.ID, domain.KeputusanSetuju, "")
	if err != nil || p.Versi != 2 {
		t.Fatalf("setuju run v2 = %+v, %v", p, err)
	}
	if st := c.statusPeriode(t); st != domain.StatusDisetujui {
		t.Fatalf("status = %s, mau disetujui", st)
	}
	if d, _ := c.e.svc.AmbilDetailPeriode(bg, c.periode.ID); d.CatatanPenolakan != "" || len(d.Runs) != 2 {
		t.Fatalf("setelah disetujui: catatan=%q runs=%d", d.CatatanPenolakan, len(d.Runs))
	}
}

func TestPutuskan_HanyaSaatDihitung(t *testing.T) {
	c := siapkanContoh(t)
	ketua := c.e.akun(t, "ketua1", domain.RoleKetua)
	_, err := c.e.svc.Putuskan(bg, c.periode.ID, ketua.ID, domain.KeputusanSetuju, "")
	cekKonflik(t, err)
	if _, err := c.e.svc.Putuskan(bg, idHantu, ketua.ID, domain.KeputusanSetuju, ""); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("periode tidak ada: err = %v", err)
	}
}

func TestPenyaluran_AturanLaluTutupDanLaporan(t *testing.T) {
	c, ketua := dihitung(t)
	salur := func(kunci string, nominal int64) error {
		_, err := c.e.svc.CatatPenyaluran(bg, c.periode.ID, c.user, PenyaluranInput{
			MustahikID: c.m[kunci].ID, Nominal: nominal, Tanggal: "2026-10-12",
		})
		return err
	}

	cekKonflik(t, salur("A2", 1_000_000)) // belum disetujui
	if _, err := c.e.svc.Putuskan(bg, c.periode.ID, ketua.ID, domain.KeputusanSetuju, "Lanjut"); err != nil {
		t.Fatal(err)
	}

	cekValidasi(t, salur("A3", 1_000_000), "mustahik_id") // cadangan, bukan usulan
	cekValidasi(t, salur("A2", 0), "nominal")
	_, err := c.e.svc.CatatPenyaluran(bg, c.periode.ID, c.user, PenyaluranInput{MustahikID: c.m["A2"].ID, Nominal: 1, Tanggal: "12/10/2026"})
	cekValidasi(t, err, "tanggal")

	if err := salur("A2", 1_000_000); err != nil {
		t.Fatal(err)
	}
	cekKonflik(t, salur("A2", 1_000_000))                                // sekali per penerima
	cekValidasi(t, salur("A4", 2_000_001), "nominal")                    // 1jt + 2jt+1 > dana 3jt
	cekValidasi(t, c.e.svc.TutupPeriode(bg, c.periode.ID), "penyaluran") // A4, A1 belum
	if err := salur("A4", 1_000_000); err != nil {
		t.Fatal(err)
	}

	rek, err := c.e.svc.DaftarPenyaluran(bg, c.periode.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rek.JumlahPenerima != 3 || rek.Tercatat != 2 || rek.TotalTersalur != 2_000_000 || rek.SisaDana != 1_000_000 || len(rek.Baris) != 5 {
		t.Fatalf("rekap = penerima %d tercatat %d total %d sisa %d baris %d",
			rek.JumlahPenerima, rek.Tercatat, rek.TotalTersalur, rek.SisaDana, len(rek.Baris))
	}

	if err := salur("A1", 1_000_000); err != nil {
		t.Fatal(err)
	}
	if err := c.e.svc.TutupPeriode(bg, c.periode.ID); err != nil {
		t.Fatal(err)
	}
	if st := c.statusPeriode(t); st != domain.StatusSelesai {
		t.Fatalf("status = %s, mau selesai", st)
	}
	// Regresi: setelah ditutup tidak ada periode aktif, tetapi Beranda harus
	// tetap tahu total yang sudah tersalur (dulu kartu jatuh ke Rp 0).
	if d, err := c.e.svc.AmbilDashboard(bg); err != nil || d.Aktif != nil || d.TotalTersalurSemua != 3_000_000 {
		t.Fatalf("dashboard setelah tutup: err=%v aktif=%v total=%d, mau nil & 3000000", err, d.Aktif, d.TotalTersalurSemua)
	} else if k := d.Keuangan; k.Zakat.Tersalur != 3_000_000 || k.Infak.Tersalur != 0 || k.TotalTersalur != 3_000_000 || k.Saldo != -3_000_000 {
		// Periode contoh berjenis zakat; belum ada donasi diterima, jadi saldo negatif.
		t.Fatalf("keuangan setelah tutup: %+v", k)
	}

	l, err := c.e.svc.AmbilLaporan(bg, c.periode.ID)
	if err != nil {
		t.Fatal(err)
	}
	if l.KetuaNama != ketua.Nama || l.AdminNama != "Admin" || l.TotalTersalur != 3_000_000 ||
		len(l.Ranking) != 5 || len(l.Penyaluran) != 3 || len(l.Kriteria) != 5 || l.Kriteria[0].Nama == "" {
		t.Fatalf("laporan = %+v", l)
	}
	var buf bytes.Buffer
	if err := l.TulisCSV(&buf); err != nil {
		t.Fatal(err)
	}
	csv := buf.String()
	for _, mau := range []string{"A. Kriteria & bobot", "C1,Penghasilan per bulan,cost,0.3000", "0.8939,Penerima", "Total,,3000000"} {
		if !strings.Contains(csv, mau) {
			t.Errorf("CSV tidak memuat %q:\n%s", mau, csv)
		}
	}
	if strings.Contains(csv, "3271012345678901") {
		t.Error("CSV tidak boleh memuat NIK")
	}
}

func TestSelCSV_CegahFormulaInjection(t *testing.T) {
	tests := map[string]string{"=1+1": "'=1+1", "+62": "'+62", "-x": "'-x", "@SUM": "'@SUM", "Ahmad": "Ahmad", "": ""}
	for masuk, mau := range tests {
		if got := selCSV(masuk); got != mau {
			t.Errorf("selCSV(%q) = %q, mau %q", masuk, got, mau)
		}
	}
}

func TestDashboard_LangkahAktif(t *testing.T) {
	c := siapkanContoh(t)
	d, err := c.e.svc.AmbilDashboard(bg)
	if err != nil {
		t.Fatal(err)
	}
	if d.MustahikAktif != 5 || d.PeriodeBerjalan != 1 || d.Aktif == nil || d.Aktif.LangkahAktif != 2 || d.Aktif.Kuota != 3 {
		t.Fatalf("sebelum penilaian: %+v aktif=%+v", d, d.Aktif)
	}
	c.isi(t)
	if d, _ = c.e.svc.AmbilDashboard(bg); d.Aktif.LangkahAktif != 3 {
		t.Fatalf("penilaian lengkap: langkah %d, mau 3", d.Aktif.LangkahAktif)
	}
	if _, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user); err != nil {
		t.Fatal(err)
	}
	if d, _ = c.e.svc.AmbilDashboard(bg); d.Aktif.LangkahAktif != 4 {
		t.Fatalf("dihitung: langkah %d, mau 4", d.Aktif.LangkahAktif)
	}

	tests := []struct {
		status            domain.StatusPeriode
		kandidat, lengkap int
		mau               int
	}{
		{domain.StatusDraft, 0, 0, 1},
		{domain.StatusDraft, 5, 4, 2},
		{domain.StatusDraft, 5, 5, 3},
		{domain.StatusDitolak, 5, 5, 4},
		{domain.StatusDisetujui, 5, 5, 5},
	}
	for _, tt := range tests {
		if got := langkahAktif(domain.Periode{Status: tt.status}, tt.kandidat, tt.lengkap); got != tt.mau {
			t.Errorf("langkahAktif(%s,%d,%d) = %d, mau %d", tt.status, tt.kandidat, tt.lengkap, got, tt.mau)
		}
	}
}
