package store

import (
	"reflect"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

// diterimaPada membuat donasi berstatus diterima dengan waktu bayar (UTC) tertentu.
func diterimaPada(t *testing.T, s *Store, jenis domain.JenisDonasi, jumlah int64, waktu time.Time) {
	t.Helper()
	buatDonasi(t, s, func(d *domain.Donasi) {
		d.Status = domain.DonasiDiterima
		d.NomorKuitansi = "K-" + kodeUji()[:8]
		d.Jenis, d.Jumlah, d.DibayarPada = jenis, jumlah, &waktu
	})
}

// salurkan mencatat satu penyaluran Rp nominal untuk mustahik baru pada periode p.
func salurkan(t *testing.T, s *Store, p domain.Periode, nama string, nominal int64, tanggal time.Time) domain.Mustahik {
	t.Helper()
	m := buatMustahik(t, s, nama, domain.AsnafFakir)
	salurkanKe(t, s, p, m, nominal, tanggal)
	return m
}

// salurkanKe mencatat penyaluran untuk mustahik yang sudah ada.
func salurkanKe(t *testing.T, s *Store, p domain.Periode, m domain.Mustahik, nominal int64, tanggal time.Time) {
	t.Helper()
	err := s.CreatePenyaluran(bg, &domain.Penyaluran{
		PeriodeID: p.ID, MustahikID: m.ID, Nominal: nominal, Tanggal: tanggal, DicatatOleh: p.CreatedBy,
	})
	if err != nil {
		t.Fatalf("CreatePenyaluran: %v", err)
	}
}

func TestKeuangan_Agregat(t *testing.T) {
	s := newStore(t)
	utc := func(y int, m time.Month, d, jam, mnt int) time.Time {
		return time.Date(y, m, d, jam, mnt, 0, 0, time.UTC)
	}

	// Kosong: semua peta kosong, bukan galat.
	if m, err := s.PenerimaanPerJenis(bg); err != nil || len(m) != 0 {
		t.Fatalf("kosong: %v, %v", m, err)
	}

	diterimaPada(t, s, domain.DonasiZakatFitrah, 100_000, utc(2026, 10, 16, 3, 0))
	diterimaPada(t, s, domain.DonasiZakatMal, 50_000, utc(2026, 10, 2, 3, 0))
	// 30 Sep 17.30 UTC = 1 Okt 00.30 WIB → masuk Oktober, bukan September.
	diterimaPada(t, s, domain.DonasiInfak, 25_000, utc(2026, 9, 30, 17, 30))
	diterimaPada(t, s, domain.DonasiInfak, 40_000, utc(2026, 9, 15, 3, 0))
	diterimaPada(t, s, domain.DonasiInfak, 10_000, utc(2026, 8, 10, 3, 0)) // sebelum jendela "2026-09"
	// Tidak boleh dihitung: belum bayar dan dibatalkan.
	buatDonasi(t, s, func(d *domain.Donasi) { d.Jumlah = 999_999 })
	buatDonasi(t, s, func(d *domain.Donasi) {
		d.Status, d.NomorKuitansi, d.Jumlah = domain.DonasiDibatalkan, "K-BATAL", 777_777
		w := utc(2026, 10, 3, 3, 0)
		d.DibayarPada = &w
	})

	user := seedUser(t, s)
	zakat := periodeBaru(user)
	if err := s.CreatePeriode(bg, &zakat); err != nil {
		t.Fatal(err)
	}
	infak := domain.Periode{Nama: "Infak Dhuafa", Jenis: domain.JenisInfak, DanaTersedia: 1_000_000, NominalPerPenerima: 500_000, CreatedBy: user}
	if err := s.CreatePeriode(bg, &infak); err != nil {
		t.Fatal(err)
	}
	ahmad := salurkan(t, s, zakat, "Ahmad", 1_000_000, utc(2026, 10, 15, 0, 0))
	salurkan(t, s, zakat, "Budi", 1_000_000, utc(2026, 10, 15, 0, 0))
	salurkan(t, s, infak, "Cici", 500_000, utc(2026, 9, 10, 0, 0))
	salurkanKe(t, s, infak, ahmad, 500_000, utc(2026, 9, 12, 0, 0)) // Ahmad menerima lagi dari periode infak

	t.Run("penerimaan per jenis", func(t *testing.T) {
		got, err := s.PenerimaanPerJenis(bg)
		want := map[domain.JenisDonasi]int64{domain.DonasiZakatFitrah: 100_000, domain.DonasiZakatMal: 50_000, domain.DonasiInfak: 75_000}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("= %v, %v; mau %v", got, err, want)
		}
	})
	t.Run("penyaluran per jenis", func(t *testing.T) {
		got, err := s.PenyaluranPerJenis(bg)
		want := map[domain.JenisPeriode]int64{domain.JenisZakat: 2_000_000, domain.JenisInfak: 1_000_000}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("= %v, %v; mau %v", got, err, want)
		}
	})
	t.Run("penerimaan per bulan WIB dengan batas jendela", func(t *testing.T) {
		got, err := s.PenerimaanPerBulan(bg, "2026-09")
		want := map[string]int64{"2026-09": 40_000, "2026-10": 175_000}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("= %v, %v; mau %v", got, err, want)
		}
	})
	t.Run("penyaluran per bulan", func(t *testing.T) {
		got, err := s.PenyaluranPerBulan(bg, "2026-10")
		want := map[string]int64{"2026-10": 2_000_000}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("dari 2026-10 = %v, %v; mau %v", got, err, want)
		}
		got, _ = s.PenyaluranPerBulan(bg, "2026-09")
		if got["2026-09"] != 1_000_000 || got["2026-10"] != 2_000_000 {
			t.Errorf("dari 2026-09 = %v", got)
		}
	})
	t.Run("penerima manfaat dihitung per orang, bukan per penyaluran", func(t *testing.T) {
		// 4 baris penyaluran, tetapi Ahmad menerima dua kali → 3 orang berbeda.
		if n, err := s.JumlahPenerimaManfaat(bg); err != nil || n != 3 {
			t.Errorf("= %d, %v; mau 3", n, err)
		}
	})
}
