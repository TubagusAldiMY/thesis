package aras

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

// kriteriaDefault = 5 kriteria usulan di docs/02-perancangan-aras.md.
func kriteriaDefault() []Kriteria {
	return []Kriteria{
		{Kode: "C1", Tipe: Cost, Bobot: 0.30},
		{Kode: "C2", Tipe: Benefit, Bobot: 0.20},
		{Kode: "C3", Tipe: Benefit, Bobot: 0.20},
		{Kode: "C4", Tipe: Benefit, Bobot: 0.15},
		{Kode: "C5", Tipe: Benefit, Bobot: 0.15},
	}
}

// alternatifContoh = 5 alternatif contoh dokumen 02 bagian 3.
func alternatifContoh() []Alternatif {
	return []Alternatif{
		{ID: "A1", Nilai: []float64{800_000, 5, 3, 2, 3}},
		{ID: "A2", Nilai: []float64{500_000, 4, 4, 3, 4}},
		{ID: "A3", Nilai: []float64{1_500_000, 3, 2, 1, 2}},
		{ID: "A4", Nilai: []float64{600_000, 2, 3, 4, 3}},
		{ID: "A5", Nilai: []float64{1_000_000, 6, 2, 2, 1}},
	}
}

// kriteriaFinal = konfigurasi hasil konfirmasi Ketua Yayasan (Oktober 2026): C1 penghasilan kini TINGKAT 1–4
// bertipe benefit (4 = paling membutuhkan), bobot 0,30/0,30/0,10/0,15/0,15. Dipasang lewat
// scripts/terapkan-kriteria-final.sh; skripsi (BAB III 3.3.4) memakai angka ini.
func kriteriaFinal() []Kriteria {
	return []Kriteria{
		{Kode: "C1", Tipe: Benefit, Bobot: 0.30},
		{Kode: "C2", Tipe: Benefit, Bobot: 0.30},
		{Kode: "C3", Tipe: Benefit, Bobot: 0.10},
		{Kode: "C4", Tipe: Benefit, Bobot: 0.15},
		{Kode: "C5", Tipe: Benefit, Bobot: 0.15},
	}
}

// alternatifFinal = lima calon sintetis; C1 berupa tingkat penghasilan (bukan rupiah).
func alternatifFinal() []Alternatif {
	return []Alternatif{
		{ID: "A1", Nilai: []float64{3, 5, 3, 2, 3}},
		{ID: "A2", Nilai: []float64{4, 4, 4, 3, 4}},
		{ID: "A3", Nilai: []float64{1, 3, 2, 1, 2}},
		{ID: "A4", Nilai: []float64{3, 2, 3, 4, 3}},
		{ID: "A5", Nilai: []float64{2, 6, 2, 2, 1}},
	}
}

// Sanity test konfigurasi final: ranking di skripsi (Tabel 3.11) dan seed demo harus sama.
func TestHitung_KonfigurasiFinal(t *testing.T) {
	h, err := Hitung(kriteriaFinal(), alternatifFinal())
	if err != nil {
		t.Fatalf("Hitung error: %v", err)
	}
	if !dekat(h.S0, 0.2348, tolDokumen) {
		t.Errorf("S0 = %.6f, mau 0,2348", h.S0)
	}
	mau := []struct {
		id      string
		s, k    float64
		ranking int
	}{
		{"A2", 0.2024, 0.8618, 1},
		{"A1", 0.1725, 0.7346, 2},
		{"A4", 0.1567, 0.6671, 3},
		{"A5", 0.1432, 0.6098, 4},
		{"A3", 0.0904, 0.3849, 5},
	}
	if len(h.Alternatif) != len(mau) {
		t.Fatalf("jumlah hasil = %d, mau %d", len(h.Alternatif), len(mau))
	}
	for i, m := range mau {
		g := h.Alternatif[i]
		if g.ID != m.id || g.Ranking != m.ranking || !dekat(g.S, m.s, tolDokumen) || !dekat(g.K, m.k, tolDokumen) {
			t.Errorf("baris %d = %+v, mau %+v", i, g, m)
		}
	}
	// Tanpa kriteria cost, alternatif ideal hanya mengambil nilai terbesar tiap kriteria.
	if !reflect.DeepEqual(h.Langkah.A0, []float64{4, 6, 4, 4, 4}) {
		t.Errorf("A0 = %v, mau [4 6 4 4 4]", h.Langkah.A0)
	}
}

func dekat(a, b, toleransi float64) bool { return math.Abs(a-b) <= toleransi }

// Angka di dokumen dibulatkan ke 4 desimal, jadi selisih maksimal 5e-5.
const tolDokumen = 5e-5

// Sanity test: hasil harus sama dengan hitung manual di dokumen 02.
func TestHitung_ContohDokumen02(t *testing.T) {
	h, err := Hitung(kriteriaDefault(), alternatifContoh())
	if err != nil {
		t.Fatalf("Hitung error: %v", err)
	}

	if !dekat(h.S0, 0.2333, tolDokumen) {
		t.Errorf("S0 = %.6f, mau 0,2333", h.S0)
	}

	// Urutan sudah menurut ranking.
	mau := []struct {
		id      string
		s, k    float64
		ranking int
	}{
		{"A2", 0.2085, 0.8939, 1},
		{"A4", 0.1709, 0.7327, 2},
		{"A1", 0.1607, 0.6888, 3},
		{"A5", 0.1309, 0.5611, 4},
		{"A3", 0.0956, 0.4099, 5},
	}
	if len(h.Alternatif) != len(mau) {
		t.Fatalf("jumlah hasil = %d, mau %d", len(h.Alternatif), len(mau))
	}
	for i, m := range mau {
		g := h.Alternatif[i]
		if g.ID != m.id || g.Ranking != m.ranking ||
			!dekat(g.S, m.s, tolDokumen) || !dekat(g.K, m.k, tolDokumen) {
			t.Errorf("baris %d = %+v, mau %+v", i, g, m)
		}
	}

	// Langkah antara untuk UI: A0, dan beberapa sel yang dikutip di dokumen.
	if !reflect.DeepEqual(h.Langkah.A0, []float64{500_000, 6, 4, 4, 4}) {
		t.Errorf("A0 = %v", h.Langkah.A0)
	}
	if got := h.Langkah.Normalisasi[1][0]; !dekat(got, 0.1456, tolDokumen) { // r(A1,C1)
		t.Errorf("r(A1,C1) = %.6f, mau 0,1456", got)
	}
	if got := h.Langkah.Terbobot[1][0]; !dekat(got, 0.0437, tolDokumen) { // d(A1,C1)
		t.Errorf("d(A1,C1) = %.6f, mau 0,0437", got)
	}
	for j := range kriteriaDefault() { // tiap kolom normalisasi (termasuk A0) berjumlah 1
		jumlah := 0.0
		for i := range h.Langkah.Normalisasi {
			jumlah += h.Langkah.Normalisasi[i][j]
		}
		if !dekat(jumlah, 1, 1e-12) {
			t.Errorf("Σ kolom %d = %.15f, mau 1", j, jumlah)
		}
	}
}

func TestHitung_Validasi(t *testing.T) {
	ok := func() ([]Kriteria, []Alternatif) { return kriteriaDefault(), alternatifContoh() }

	tests := []struct {
		nama string
		ubah func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif)
	}{
		{"kriteria kosong", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) { return nil, a }},
		{"alternatif kosong", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) { return k, nil }},
		{"tipe tidak dikenal", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			k[0].Tipe = "lainnya"
			return k, a
		}},
		{"total bobot kurang dari 100%", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			k[0].Bobot = 0.25
			return k, a
		}},
		{"total bobot lebih dari 100%", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			k[0].Bobot = 0.35
			return k, a
		}},
		{"bobot bukan persen bulat", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			k[0].Bobot, k[1].Bobot = 0.325, 0.175 // total tetap 1, tetapi 32,5% bukan bulat
			return k, a
		}},
		{"bobot nol", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			k[0].Bobot, k[1].Bobot = 0, 0.5
			return k, a
		}},
		{"bobot negatif", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			k[0].Bobot, k[1].Bobot = -0.1, 0.6
			return k, a
		}},
		{"bobot NaN", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			k[0].Bobot = math.NaN()
			return k, a
		}},
		{"jumlah nilai tidak sama dengan kriteria", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			a[0].Nilai = a[0].Nilai[:4]
			return k, a
		}},
		{"nilai nol (cost dibagi 0)", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			a[1].Nilai[0] = 0
			return k, a
		}},
		{"nilai negatif", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			a[1].Nilai[1] = -3
			return k, a
		}},
		{"nilai NaN", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			a[2].Nilai[2] = math.NaN()
			return k, a
		}},
		{"nilai tak hingga", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			a[2].Nilai[3] = math.Inf(1)
			return k, a
		}},
		{"ID alternatif duplikat", func(k []Kriteria, a []Alternatif) ([]Kriteria, []Alternatif) {
			a[1].ID = a[0].ID
			return k, a
		}},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			k, a := tt.ubah(ok())
			_, err := Hitung(k, a)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("err = %v, mau ErrInvalid", err)
			}
		})
	}
}

// Edge case: satu alternatif = A0 itu sendiri, jadi K tepat 1.
func TestHitung_SatuAlternatif(t *testing.T) {
	h, err := Hitung(kriteriaDefault(), alternatifContoh()[:1])
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Alternatif) != 1 || h.Alternatif[0].Ranking != 1 || !dekat(h.Alternatif[0].K, 1, 1e-12) {
		t.Errorf("hasil = %+v, mau 1 alternatif, ranking 1, K = 1", h.Alternatif)
	}
}

// Seri K (4 desimal) harus dipecahkan deterministik (docs/02 bagian 4).
func TestHitung_Seri(t *testing.T) {
	benefit2 := []Kriteria{{"C1", Benefit, 0.5}, {"C2", Benefit, 0.5}}
	cost2 := []Kriteria{{"C1", Cost, 0.5}, {"C2", Cost, 0.5}}

	tests := []struct {
		nama     string
		kriteria []Kriteria
		alt      []Alternatif
		urutan   []string // ID menurut ranking yang diharapkan
	}{
		{
			// Simetris → K sama persis; benefit: C1 lebih besar menang.
			"benefit: kriteria pertama yang beda menentukan",
			benefit2,
			[]Alternatif{{ID: "Y", Nilai: []float64{2, 4}}, {ID: "X", Nilai: []float64{4, 2}}},
			[]string{"X", "Y"},
		},
		{
			// Simetris → K sama persis; cost: C1 lebih kecil menang.
			"cost: nilai lebih kecil menang",
			cost2,
			[]Alternatif{{ID: "Y", Nilai: []float64{4, 2}}, {ID: "X", Nilai: []float64{2, 4}}},
			[]string{"X", "Y"},
		},
		{
			"nilai identik: urut ID naik",
			benefit2,
			[]Alternatif{{ID: "B", Nilai: []float64{3, 3}}, {ID: "A", Nilai: []float64{3, 3}}},
			[]string{"A", "B"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			h, err := Hitung(tt.kriteria, tt.alt)
			if err != nil {
				t.Fatal(err)
			}
			if !dekat(h.Alternatif[0].K, h.Alternatif[1].K, 1e-9) {
				t.Fatalf("K tidak seri: %v vs %v (kasus uji salah dirancang)", h.Alternatif[0].K, h.Alternatif[1].K)
			}
			for i, id := range tt.urutan {
				if h.Alternatif[i].ID != id || h.Alternatif[i].Ranking != i+1 {
					t.Errorf("posisi %d = %+v, mau ID %s", i, h.Alternatif[i], id)
				}
			}
		})
	}
}

// Pemecah seri harus memulai dari kriteria berbobot TERBESAR, bukan urutan kolom.
func TestMenang_BobotTerbesarDulu(t *testing.T) {
	k := []Kriteria{{"C1", Benefit, 0.4}, {"C2", Benefit, 0.6}}
	urutan := indeksBobot(k)
	if !reflect.DeepEqual(urutan, []int{1, 0}) {
		t.Fatalf("indeksBobot = %v, mau [1 0]", urutan)
	}
	// p unggul di C1 (bobot kecil), q unggul di C2 (bobot besar) → q menang.
	p, q := []float64{9, 1}, []float64{1, 9}
	if menang(k, urutan, p, q, "p", "q") || !menang(k, urutan, q, p, "q", "p") {
		t.Error("kriteria berbobot terbesar (C2) harus menentukan")
	}
}

// Hasil tidak boleh bergantung pada urutan input. ID dan ranking harus
// identik; S/K hanya boleh beda di galat floating point (penjumlahan kolom
// tidak asosiatif, jadi digit ke-16 bisa bergeser) — maka pakai toleransi.
func TestHitung_UrutanInputTidakMempengaruhi(t *testing.T) {
	asli, err := Hitung(kriteriaDefault(), alternatifContoh())
	if err != nil {
		t.Fatal(err)
	}
	terbalik := alternatifContoh()
	for i, j := 0, len(terbalik)-1; i < j; i, j = i+1, j-1 {
		terbalik[i], terbalik[j] = terbalik[j], terbalik[i]
	}
	h, err := Hitung(kriteriaDefault(), terbalik)
	if err != nil {
		t.Fatal(err)
	}
	if !dekat(asli.S0, h.S0, 1e-12) {
		t.Errorf("S0 berbeda: %v vs %v", asli.S0, h.S0)
	}
	for i := range asli.Alternatif {
		a, b := asli.Alternatif[i], h.Alternatif[i]
		if a.ID != b.ID || a.Ranking != b.Ranking || !dekat(a.S, b.S, 1e-12) || !dekat(a.K, b.K, 1e-12) {
			t.Errorf("baris %d berbeda:\n asli     = %+v\n terbalik = %+v", i, a, b)
		}
	}
}

// Normalisasi rasio tak terpengaruh satuan: mengalikan satu kolom dengan
// konstanta (mis. rupiah → ribu rupiah) tidak boleh mengubah K.
func TestHitung_InvarianSkalaKolom(t *testing.T) {
	asli, _ := Hitung(kriteriaDefault(), alternatifContoh())

	skala := alternatifContoh()
	for i := range skala {
		skala[i].Nilai[0] *= 1000 // penghasilan (cost)
		skala[i].Nilai[1] *= 7    // tanggungan (benefit)
	}
	h, err := Hitung(kriteriaDefault(), skala)
	if err != nil {
		t.Fatal(err)
	}
	for i := range asli.Alternatif {
		if asli.Alternatif[i].ID != h.Alternatif[i].ID || !dekat(asli.Alternatif[i].K, h.Alternatif[i].K, 1e-12) {
			t.Errorf("baris %d: %+v vs %+v", i, asli.Alternatif[i], h.Alternatif[i])
		}
	}
}

// Hitung tidak boleh mengubah slice milik pemanggil.
func TestHitung_InputTidakDiubah(t *testing.T) {
	k, a := kriteriaDefault(), alternatifContoh()
	if _, err := Hitung(k, a); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(k, kriteriaDefault()) || !reflect.DeepEqual(a, alternatifContoh()) {
		t.Error("input berubah setelah Hitung")
	}
}
