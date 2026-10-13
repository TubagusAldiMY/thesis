package service

import (
	"errors"
	"math"
	"strings"
	"testing"

	"spkzakat/internal/domain"
)

// contoh = periode zakat (kuota 3) dengan 5 kandidat A1..A5 persis seperti
// contoh hitung di docs/02-perancangan-aras.md bagian 3.
type contoh struct {
	e       *lingkungan
	user    string
	periode domain.Periode
	m       map[string]MustahikView // "A1".."A5"
}

type dataAlt struct {
	asnaf domain.Asnaf
	nilai map[string]float64 // kode kriteria → nilai
}

var dataContoh = map[string]dataAlt{
	"A1": {domain.AsnafMiskin, map[string]float64{"C1": 800_000, "C2": 5, "C3": 3, "C4": 2, "C5": 3}},
	"A2": {domain.AsnafFakir, map[string]float64{"C1": 500_000, "C2": 4, "C3": 4, "C4": 3, "C5": 4}},
	"A3": {domain.AsnafMiskin, map[string]float64{"C1": 1_500_000, "C2": 3, "C3": 2, "C4": 1, "C5": 2}},
	"A4": {domain.AsnafFakir, map[string]float64{"C1": 600_000, "C2": 2, "C3": 3, "C4": 4, "C5": 3}},
	"A5": {domain.AsnafMiskin, map[string]float64{"C1": 1_000_000, "C2": 6, "C3": 2, "C4": 2, "C5": 1}},
}

var urutKunci = []string{"A1", "A2", "A3", "A4", "A5"}

// siapkanContoh membuat mustahik, periode, dan kandidat — BELUM ada penilaian.
func siapkanContoh(t *testing.T) *contoh {
	t.Helper()
	e := baru(t)
	c := &contoh{e: e, user: e.user(t), m: map[string]MustahikView{}}
	var ids []string
	for i, kunci := range urutKunci {
		nama := "Mustahik " + kunci[1:]
		nik := "327101234567890" + string(rune('1'+i))
		c.m[kunci] = e.buatMustahik(t, nama, nik, dataContoh[kunci].asnaf, true)
		ids = append(ids, c.m[kunci].ID)
	}
	p, err := e.svc.BuatPeriode(bg, c.user, zakat())
	if err != nil {
		t.Fatal(err)
	}
	c.periode = p
	if err := e.svc.AturKandidat(bg, p.ID, ids); err != nil {
		t.Fatal(err)
	}
	return c
}

// masukan membentuk PenilaianInput untuk kunci-kunci yang diminta.
func (c *contoh) masukan(kunci ...string) []PenilaianInput {
	var in []PenilaianInput
	for _, k := range kunci {
		in = append(in, PenilaianInput{MustahikID: c.m[k].ID, Nilai: dataContoh[k].nilai})
	}
	return in
}

// isi menyimpan penilaian untuk kunci yang diminta (bawaan: semua).
func (c *contoh) isi(t *testing.T, kunci ...string) {
	t.Helper()
	if len(kunci) == 0 {
		kunci = urutKunci
	}
	if err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, c.masukan(kunci...)); err != nil {
		t.Fatalf("isi penilaian: %v", err)
	}
}

func TestSimpanPenilaian_LaluAmbilGrid(t *testing.T) {
	c := siapkanContoh(t)

	t.Run("sebelum diisi: 0 dari 5 lengkap, 5 kolom kriteria", func(t *testing.T) {
		g, err := c.e.svc.AmbilPenilaian(bg, c.periode.ID)
		if err != nil {
			t.Fatal(err)
		}
		if g.Lengkap != 0 || g.Total != 5 || len(g.Baris) != 5 || len(g.Kriteria) != 5 {
			t.Errorf("grid = lengkap %d/%d, baris %d, kriteria %d", g.Lengkap, g.Total, len(g.Baris), len(g.Kriteria))
		}
	})

	t.Run("simpan sebagian: hanya sel yang dikirim yang terisi", func(t *testing.T) {
		err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, []PenilaianInput{
			{MustahikID: c.m["A1"].ID, Nilai: map[string]float64{"C1": 800_000, "C2": 5}},
		})
		if err != nil {
			t.Fatal(err)
		}
		g, _ := c.e.svc.AmbilPenilaian(bg, c.periode.ID)
		for _, b := range g.Baris {
			if b.Mustahik.ID == c.m["A1"].ID && (len(b.Nilai) != 2 || b.Lengkap) {
				t.Errorf("A1 = %+v, mau 2 sel terisi dan belum lengkap", b)
			}
		}
		if g.Lengkap != 0 {
			t.Errorf("lengkap = %d, mau 0", g.Lengkap)
		}
	})

	t.Run("isi semua: 5 dari 5 lengkap dan nilai persis", func(t *testing.T) {
		c.isi(t)
		g, _ := c.e.svc.AmbilPenilaian(bg, c.periode.ID)
		if g.Lengkap != 5 {
			t.Fatalf("lengkap = %d, mau 5", g.Lengkap)
		}
		for _, b := range g.Baris {
			for kunci, mv := range c.m {
				if mv.ID != b.Mustahik.ID {
					continue
				}
				for kode, mau := range dataContoh[kunci].nilai {
					if b.Nilai[kode] != mau {
						t.Errorf("%s/%s = %v, mau %v", kunci, kode, b.Nilai[kode], mau)
					}
				}
			}
		}
	})

	t.Run("menimpa satu sel", func(t *testing.T) {
		err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, []PenilaianInput{
			{MustahikID: c.m["A1"].ID, Nilai: map[string]float64{"C1": 750_000}},
		})
		if err != nil {
			t.Fatal(err)
		}
		g, _ := c.e.svc.AmbilPenilaian(bg, c.periode.ID)
		for _, b := range g.Baris {
			if b.Mustahik.ID == c.m["A1"].ID && (b.Nilai["C1"] != 750_000 || b.Nilai["C2"] != 5 || !b.Lengkap) {
				t.Errorf("A1 = %+v, mau C1=750000 dan sel lain utuh", b.Nilai)
			}
		}
	})
}

func TestSimpanPenilaian_Validasi(t *testing.T) {
	c := siapkanContoh(t)
	a1 := c.m["A1"].ID
	luar := c.e.buatMustahik(t, "Bukan Kandidat", "3271012345679999", domain.AsnafFakir, true).ID

	tests := []struct {
		nama  string
		masuk []PenilaianInput
		field string
		pesan string
	}{
		{"kriteria berskala: di luar 1–4", []PenilaianInput{{a1, map[string]float64{"C3": 5}}}, "nilai." + a1 + ".C3", "salah satu dari: 1, 2, 3, 4"},
		{"kriteria berskala: pecahan", []PenilaianInput{{a1, map[string]float64{"C3": 2.5}}}, "nilai." + a1 + ".C3", "salah satu dari"},
		{"kriteria berskala: nol", []PenilaianInput{{a1, map[string]float64{"C4": 0}}}, "nilai." + a1 + ".C4", "salah satu dari"},
		{"penghasilan nol (UI: isi 1)", []PenilaianInput{{a1, map[string]float64{"C1": 0}}}, "nilai." + a1 + ".C1", "isi 1 bila tidak ada"},
		{"penghasilan negatif", []PenilaianInput{{a1, map[string]float64{"C1": -5}}}, "nilai." + a1 + ".C1", "antara 1 dan"},
		{"tanggungan di bawah 1", []PenilaianInput{{a1, map[string]float64{"C2": 0.5}}}, "nilai." + a1 + ".C2", "antara 1 dan"},
		{"nilai melebihi batas", []PenilaianInput{{a1, map[string]float64{"C1": nilaiMaks + 1}}}, "nilai." + a1 + ".C1", "antara 1 dan"},
		{"NaN", []PenilaianInput{{a1, map[string]float64{"C1": math.NaN()}}}, "nilai." + a1 + ".C1", "tidak valid"},
		{"tak hingga", []PenilaianInput{{a1, map[string]float64{"C2": math.Inf(1)}}}, "nilai." + a1 + ".C2", "tidak valid"},
		{"kode kriteria tidak dikenal", []PenilaianInput{{a1, map[string]float64{"C9": 3}}}, "nilai." + a1 + ".C9", "tidak dikenal"},
		{"mustahik bukan kandidat", []PenilaianInput{{luar, map[string]float64{"C1": 5}}}, "mustahik." + luar, "Bukan kandidat"},
		{"mustahik dikirim dua kali", []PenilaianInput{{a1, map[string]float64{"C1": 5}}, {a1, map[string]float64{"C2": 5}}}, "penilaian", "lebih dari sekali"},
		{"permintaan kosong", nil, "penilaian", "Tidak ada nilai"},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, tt.masuk)
			ve := cekValidasi(t, err, tt.field)
			if !strings.Contains(ve.Fields[tt.field], tt.pesan) {
				t.Errorf("pesan = %q, harus memuat %q", ve.Fields[tt.field], tt.pesan)
			}
		})
	}

	t.Run("satu sel salah membatalkan seluruh paket (atomik)", func(t *testing.T) {
		a2 := c.m["A2"].ID
		err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, []PenilaianInput{
			{a1, map[string]float64{"C1": 111_111, "C2": 3}}, // sah
			{a2, map[string]float64{"C3": 9}},                // salah
		})
		cekValidasi(t, err, "nilai."+a2+".C3")
		g, _ := c.e.svc.AmbilPenilaian(bg, c.periode.ID)
		for _, b := range g.Baris {
			if len(b.Nilai) != 0 {
				t.Errorf("%s punya nilai %v padahal paket ditolak", b.Mustahik.Nama, b.Nilai)
			}
		}
	})

	t.Run("semua kesalahan dilaporkan sekaligus", func(t *testing.T) {
		err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, []PenilaianInput{
			{a1, map[string]float64{"C1": 0, "C3": 9}},
			{luar, map[string]float64{"C1": 5}},
		})
		ve := cekValidasi(t, err, "")
		for _, f := range []string{"nilai." + a1 + ".C1", "nilai." + a1 + ".C3", "mustahik." + luar} {
			if ve.Fields[f] == "" {
				t.Errorf("field %q tidak dilaporkan: %v", f, ve.Fields)
			}
		}
	})
}

func TestSimpanPenilaian_HanyaSaatDraft(t *testing.T) {
	c := siapkanContoh(t)
	for _, status := range []string{"dihitung", "disetujui", "ditolak", "selesai"} {
		t.Run(status, func(t *testing.T) {
			c.e.exec(t, `UPDATE periode SET status = ? WHERE id = ?`, status, c.periode.ID)
			err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, c.masukan("A1"))
			if !errors.Is(err, domain.ErrConflict) {
				t.Errorf("err = %v, mau ErrConflict", err)
			}
		})
	}
	t.Run("periode tidak ada", func(t *testing.T) {
		if err := c.e.svc.SimpanPenilaian(bg, idHantu, c.masukan("A1")); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
		if _, err := c.e.svc.AmbilPenilaian(bg, idHantu); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("AmbilPenilaian: err = %v, mau ErrNotFound", err)
		}
	})
}

func TestSimpanPenilaian_KriteriaNonaktif(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t) // semua 5 kriteria terisi dulu
	c.e.exec(t, `UPDATE kriteria SET is_active = FALSE WHERE kode = 'C5'`)

	err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, []PenilaianInput{{c.m["A1"].ID, map[string]float64{"C5": 2}}})
	cekValidasi(t, err, "nilai."+c.m["A1"].ID+".C5")

	g, _ := c.e.svc.AmbilPenilaian(bg, c.periode.ID)
	if len(g.Kriteria) != 4 || g.Lengkap != 5 {
		t.Errorf("kriteria=%d lengkap=%d, mau 4 kolom (C5 tersembunyi) dan 5 lengkap", len(g.Kriteria), g.Lengkap)
	}
	for _, b := range g.Baris {
		if _, ada := b.Nilai["C5"]; ada {
			t.Errorf("nilai C5 (nonaktif) ikut ditampilkan: %v", b.Nilai)
		}
	}
}

func TestPesanNilai(t *testing.T) {
	skala := domain.Kriteria{Kode: "C3", Skala: map[string]string{"1": "a", "2": "b", "3": "c", "4": "d"}}
	bebas := domain.Kriteria{Kode: "C1"}
	tests := []struct {
		nama string
		k    domain.Kriteria
		v    float64
		sah  bool
	}{
		{"skala 1", skala, 1, true}, {"skala 4", skala, 4, true},
		{"skala 0", skala, 0, false}, {"skala 5", skala, 5, false}, {"skala 2,5", skala, 2.5, false}, {"skala -1", skala, -1, false},
		{"bebas tepat 1", bebas, 1, true}, {"bebas 0,99", bebas, 0.99, false}, {"bebas 800000", bebas, 800_000, true},
		{"bebas batas atas", bebas, nilaiMaks, true}, {"bebas lewat batas atas", bebas, nilaiMaks + 1, false},
		{"bebas NaN", bebas, math.NaN(), false}, {"bebas +Inf", bebas, math.Inf(1), false}, {"skala -Inf", skala, math.Inf(-1), false},
	}
	for _, tt := range tests {
		if got := pesanNilai(tt.k, tt.v) == ""; got != tt.sah {
			t.Errorf("%s: sah = %v, mau %v", tt.nama, got, tt.sah)
		}
	}
}
