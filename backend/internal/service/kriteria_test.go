package service

import (
	"errors"
	"math"
	"strings"
	"testing"

	"spkzakat/internal/domain"
)

func TestBobotSah(t *testing.T) {
	tests := []struct {
		b   float64
		mau bool
	}{
		{0.05, true}, {0.10, true}, {0.15, true}, {0.30, true}, {0.55, true}, {0.95, true}, {1.00, true},
		{0, false}, {-0.05, false}, {0.04, false}, {0.12, false}, {0.325, false}, {1.05, false}, {2, false},
		{math.NaN(), false}, {math.Inf(1), false}, {math.Inf(-1), false},
	}
	for _, tt := range tests {
		if got := bobotSah(tt.b); got != tt.mau {
			t.Errorf("bobotSah(%v) = %v, mau %v", tt.b, got, tt.mau)
		}
	}
}

// idKriteria memetakan kode (C1..C5) → ID.
func idKriteria(t *testing.T, e *lingkungan) map[string]string {
	t.Helper()
	d, err := e.svc.DaftarKriteria(bg)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, k := range d.Items {
		m[k.Kode] = k.ID
	}
	return m
}

func bobotDiDB(t *testing.T, e *lingkungan) map[string]float64 {
	t.Helper()
	d, err := e.svc.DaftarKriteria(bg)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]float64{}
	for _, k := range d.Items {
		m[k.Kode] = k.Bobot
	}
	return m
}

func TestDaftarKriteria_Seed(t *testing.T) {
	e := baru(t)
	d, err := e.svc.DaftarKriteria(bg)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 5 || d.TotalPersen != 100 {
		t.Errorf("items=%d total=%d%%, mau 5 dan 100%%", len(d.Items), d.TotalPersen)
	}
}

func TestUbahKriteria(t *testing.T) {
	type u = KriteriaUbah
	cost, benefit := domain.TipeCost, domain.TipeBenefit

	tests := []struct {
		nama   string
		ubah   func(id map[string]string) []u
		mauErr string  // "" sukses; "validasi:<field>"; "notfound"
		pesan  string  // potongan pesan yang harus ada pada validasi
		c1     float64 // bobot C1 yang diharapkan di database setelahnya
	}{
		{"semua kriteria, total 100%", func(id map[string]string) []u {
			return []u{{id["C1"], cost, 0.35}, {id["C2"], benefit, 0.15}, {id["C3"], benefit, 0.20}, {id["C4"], benefit, 0.15}, {id["C5"], benefit, 0.15}}
		}, "", "", 0.35},
		{"sebagian kriteria asal total tetap 100%", func(id map[string]string) []u {
			return []u{{id["C1"], cost, 0.35}, {id["C2"], benefit, 0.15}}
		}, "", "", 0.35},
		{"ganti tipe saja", func(id map[string]string) []u {
			return []u{{id["C1"], benefit, 0.30}}
		}, "", "", 0.30},
		{"total kurang dari 100% → dibatalkan", func(id map[string]string) []u {
			return []u{{id["C1"], cost, 0.25}}
		}, "validasi:bobot", "95%", 0.30},
		{"total lebih dari 100% → dibatalkan", func(id map[string]string) []u {
			return []u{{id["C1"], cost, 0.35}}
		}, "validasi:bobot", "105%", 0.30},
		{"bukan kelipatan 0,05", func(id map[string]string) []u {
			return []u{{id["C1"], cost, 0.325}, {id["C2"], benefit, 0.175}}
		}, "validasi:bobot", "kelipatan", 0.30},
		{"bobot nol", func(id map[string]string) []u { return []u{{id["C1"], cost, 0}} }, "validasi:bobot", "", 0.30},
		{"bobot negatif", func(id map[string]string) []u { return []u{{id["C1"], cost, -0.05}} }, "validasi:bobot", "", 0.30},
		{"bobot di atas 1", func(id map[string]string) []u { return []u{{id["C1"], cost, 1.05}} }, "validasi:bobot", "", 0.30},
		{"bobot NaN", func(id map[string]string) []u { return []u{{id["C1"], cost, math.NaN()}} }, "validasi:bobot", "", 0.30},
		{"tipe tidak dikenal", func(id map[string]string) []u { return []u{{id["C1"], "lainnya", 0.30}} }, "validasi:tipe", "", 0.30},
		{"kriteria dikirim dua kali", func(id map[string]string) []u {
			return []u{{id["C1"], cost, 0.30}, {id["C1"], cost, 0.30}}
		}, "validasi:kriteria", "", 0.30},
		{"tidak ada perubahan", func(id map[string]string) []u { return nil }, "validasi:bobot", "", 0.30},
		{"ID tidak ada: perubahan lain dalam paket ikut dibatalkan", func(id map[string]string) []u {
			return []u{{id["C1"], cost, 0.35}, {idHantu, benefit, 0.15}}
		}, "notfound", "", 0.30},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			e := baru(t)
			err := e.svc.UbahKriteria(bg, tt.ubah(idKriteria(t, e)))

			switch {
			case tt.mauErr == "":
				if err != nil {
					t.Fatalf("err = %v, mau sukses", err)
				}
			case tt.mauErr == "notfound":
				if !errors.Is(err, domain.ErrNotFound) {
					t.Fatalf("err = %v, mau ErrNotFound", err)
				}
			default:
				ve := cekValidasi(t, err, strings.TrimPrefix(tt.mauErr, "validasi:"))
				if tt.pesan != "" && !strings.Contains(ve.Error(), tt.pesan) {
					t.Errorf("pesan = %q, harus memuat %q", ve.Error(), tt.pesan)
				}
			}
			if got := bobotDiDB(t, e)["C1"]; got != tt.c1 {
				t.Errorf("bobot C1 di database = %v, mau %v (rollback gagal / perubahan tidak tersimpan)", got, tt.c1)
			}
		})
	}
}

func TestUbahKriteria_TipeTersimpan(t *testing.T) {
	e := baru(t)
	id := idKriteria(t, e)
	if err := e.svc.UbahKriteria(bg, []KriteriaUbah{{id["C1"], domain.TipeBenefit, 0.30}}); err != nil {
		t.Fatal(err)
	}
	d, _ := e.svc.DaftarKriteria(bg)
	if d.Items[0].Kode != "C1" || d.Items[0].Tipe != domain.TipeBenefit {
		t.Errorf("C1 = %+v, mau tipe benefit", d.Items[0])
	}
}

// Kriteria nonaktif tidak ikut dihitung dalam total 100%.
func TestUbahKriteria_NonaktifTidakDihitung(t *testing.T) {
	e := baru(t)
	id := idKriteria(t, e)
	e.exec(t, `UPDATE kriteria SET is_active = FALSE WHERE kode = 'C5'`)

	d, _ := e.svc.DaftarKriteria(bg)
	if d.TotalPersen != 85 {
		t.Fatalf("total aktif = %d%%, mau 85%% (C5 nonaktif)", d.TotalPersen)
	}
	// Aktif tinggal C1..C4: harus 100% tanpa C5.
	err := e.svc.UbahKriteria(bg, []KriteriaUbah{
		{id["C1"], domain.TipeCost, 0.35}, {id["C2"], domain.TipeBenefit, 0.25}, {id["C3"], domain.TipeBenefit, 0.25},
	})
	if err != nil {
		t.Fatalf("err = %v; total aktif C1..C4 = 100%% mestinya lolos", err)
	}
	if d, _ := e.svc.DaftarKriteria(bg); d.TotalPersen != 100 {
		t.Errorf("total = %d%%", d.TotalPersen)
	}
}
