package service

import (
	"testing"

	"spkzakat/internal/domain"
)

func TestDashboard_Keuangan(t *testing.T) {
	e := baruDonasi(t) // jam tetap 16/10/2026 10.00 WIB
	petugas := e.user(t)
	catat := func(jenis domain.JenisDonasi, jumlah int64, tanggal string) DonasiView {
		t.Helper()
		v, err := e.svc.CatatDonasiManual(bg, petugas, inputManual(func(i *DonasiManualInput) {
			i.Jenis, i.Jumlah, i.Tanggal = jenis, jumlah, tanggal
		}))
		if err != nil {
			t.Fatalf("CatatDonasiManual(%s, %d, %q): %v", jenis, jumlah, tanggal, err)
		}
		return v
	}

	// Kosong: tetap 12 bulan (Nov 2025 – Okt 2026) bernilai nol, bukan nil/galat.
	d, err := e.svc.AmbilDashboard(bg)
	if err != nil {
		t.Fatal(err)
	}
	k := d.Keuangan
	if len(k.Bulanan) != 12 || k.Bulanan[0].Bulan != "2025-11" || k.Bulanan[11].Bulan != "2026-10" {
		t.Fatalf("jendela bulan = %+v", k.Bulanan)
	}
	if k.TotalDiterima != 0 || k.Saldo != 0 || len(k.DonasiTerbaru) != 0 || k.DonasiTerbaru == nil {
		t.Fatalf("kosong: %+v", k)
	}

	catat(domain.DonasiZakatFitrah, 100_000, "")
	catat(domain.DonasiZakatMal, 200_000, "2026-10-01")
	catat(domain.DonasiInfak, 50_000, "2026-09-20")
	catat(domain.DonasiInfak, 30_000, "2025-10-31") // sebelum jendela: masuk total, tidak masuk grafik
	salah := catat(domain.DonasiZakatProfesi, 999_000, "")
	if _, err := e.svc.BatalkanDonasi(bg, salah.ID, "salah input"); err != nil {
		t.Fatal(err)
	}
	// Donasi web: yang belum dibayar tidak dihitung, yang dibayar (settlement) dihitung.
	if _, err := e.svc.BuatDonasiWeb(bg, inputWeb(func(i *DonasiWebInput) { i.Jumlah = 888_000 })); err != nil {
		t.Fatal(err)
	}
	lunas, err := e.svc.BuatDonasiWeb(bg, inputWeb(func(i *DonasiWebInput) { i.Jumlah = 70_000 }))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.bayar(t, e.idDariKode(t, lunas.Kode), 70_000, "settlement"); err != nil {
		t.Fatal(err)
	}

	if d, err = e.svc.AmbilDashboard(bg); err != nil {
		t.Fatal(err)
	}
	k = d.Keuangan
	if k.Zakat != (DanaJenis{Diterima: 370_000, Saldo: 370_000}) || k.Infak != (DanaJenis{Diterima: 80_000, Saldo: 80_000}) {
		t.Errorf("zakat=%+v infak=%+v, mau zakat 370000 & infak 80000 (tanpa yang dibatalkan/menunggu)", k.Zakat, k.Infak)
	}
	if k.TotalDiterima != 450_000 || k.Saldo != 450_000 || k.TotalTersalur != 0 || k.JumlahDonasi != 5 {
		t.Errorf("total=%d saldo=%d tersalur=%d jumlah=%d", k.TotalDiterima, k.Saldo, k.TotalTersalur, k.JumlahDonasi)
	}
	if k.DiterimaBulanIni != 370_000 {
		t.Errorf("bulan ini = %d, mau 370000", k.DiterimaBulanIni)
	}
	if k.Bulanan[10].Bulan != "2026-09" || k.Bulanan[10].Penerimaan != 50_000 || k.Bulanan[11].Penerimaan != 370_000 {
		t.Errorf("grafik = %+v", k.Bulanan[9:])
	}
	if len(k.DonasiTerbaru) != 5 || k.PerJenis[domain.DonasiInfak] != 80_000 {
		t.Errorf("terbaru=%d perJenis=%v", len(k.DonasiTerbaru), k.PerJenis)
	}
}
