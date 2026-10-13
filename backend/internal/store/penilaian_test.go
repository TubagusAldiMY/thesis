package store

import (
	"errors"
	"testing"

	"spkzakat/internal/domain"
)

// siapPenilaian: satu periode zakat dengan dua kandidat dan daftar kriteria seed.
type siapPenilaian struct {
	s      *Store
	p      domain.Periode
	m1, m2 domain.Mustahik
	ks     []domain.Kriteria
}

func siapkanPenilaian(t *testing.T) siapPenilaian {
	t.Helper()
	s := newStore(t)
	p := periodeBaru(seedUser(t, s))
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}
	m1 := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)
	m2 := buatMustahik(t, s, "Budi", domain.AsnafMiskin)
	if err := s.ReplaceKandidat(bg, p.ID, []string{m1.ID, m2.ID}); err != nil {
		t.Fatal(err)
	}
	ks, err := s.ListKriteria(bg, true)
	if err != nil {
		t.Fatal(err)
	}
	return siapPenilaian{s: s, p: p, m1: m1, m2: m2, ks: ks}
}

func nilaiMap(t *testing.T, s *Store, periodeID string) map[[2]string]float64 {
	t.Helper()
	ps, err := s.ListPenilaian(bg, periodeID)
	if err != nil {
		t.Fatal(err)
	}
	m := map[[2]string]float64{}
	for _, p := range ps {
		m[[2]string{p.MustahikID, p.KriteriaID}] = p.Nilai
	}
	return m
}

func TestUpsertPenilaian_TambahLaluTimpa(t *testing.T) {
	x := siapkanPenilaian(t)
	c1, c2 := x.ks[0].ID, x.ks[1].ID

	if err := x.s.UpsertPenilaian(bg, x.p.ID, []domain.Penilaian{
		{MustahikID: x.m1.ID, KriteriaID: c1, Nilai: 800000},
		{MustahikID: x.m1.ID, KriteriaID: c2, Nilai: 5},
		{MustahikID: x.m2.ID, KriteriaID: c1, Nilai: 500000},
	}); err != nil {
		t.Fatal(err)
	}
	got := nilaiMap(t, x.s, x.p.ID)
	if len(got) != 3 || got[[2]string{x.m1.ID, c1}] != 800000 || got[[2]string{x.m1.ID, c2}] != 5 {
		t.Fatalf("tersimpan = %v", got)
	}

	// Menimpa satu sel: jumlah baris tetap, nilai berubah, sel lain tak tersentuh.
	if err := x.s.UpsertPenilaian(bg, x.p.ID, []domain.Penilaian{{MustahikID: x.m1.ID, KriteriaID: c1, Nilai: 900000}}); err != nil {
		t.Fatal(err)
	}
	got = nilaiMap(t, x.s, x.p.ID)
	if len(got) != 3 || got[[2]string{x.m1.ID, c1}] != 900000 || got[[2]string{x.m2.ID, c1}] != 500000 {
		t.Errorf("setelah timpa = %v, mau 3 baris dengan m1/c1=900000 dan m2/c1 utuh", got)
	}

	// Mengirim nilai yang sama lagi tidak error (idempotent).
	if err := x.s.UpsertPenilaian(bg, x.p.ID, []domain.Penilaian{{MustahikID: x.m1.ID, KriteriaID: c1, Nilai: 900000}}); err != nil {
		t.Errorf("upsert ulang: %v", err)
	}
}

func TestUpsertPenilaian_Presisi(t *testing.T) {
	x := siapkanPenilaian(t)
	c := x.ks[0].ID
	tests := []struct{ masuk, mau float64 }{
		{1234567.8912, 1234567.8912},
		{1.23456, 1.2346}, // dibulatkan ke 4 desimal secara eksplisit
		{0.5, 0.5},
		{9_999_999_999, 9_999_999_999},
	}
	for _, tt := range tests {
		if err := x.s.UpsertPenilaian(bg, x.p.ID, []domain.Penilaian{{MustahikID: x.m1.ID, KriteriaID: c, Nilai: tt.masuk}}); err != nil {
			t.Fatalf("simpan %v: %v", tt.masuk, err)
		}
		if got := nilaiMap(t, x.s, x.p.ID)[[2]string{x.m1.ID, c}]; got != tt.mau {
			t.Errorf("simpan %v → baca %v, mau %v", tt.masuk, got, tt.mau)
		}
	}
}

func TestUpsertPenilaian_GagalMembatalkanSemuaDalamPaket(t *testing.T) {
	x := siapkanPenilaian(t)
	c := x.ks[0].ID
	orangLuar := buatMustahik(t, x.s, "Bukan Kandidat", domain.AsnafFakir)

	tests := []struct {
		nama     string
		sel      domain.Penilaian
		notFound bool
	}{
		{"mustahik bukan kandidat periode", domain.Penilaian{MustahikID: orangLuar.ID, KriteriaID: c, Nilai: 5}, true},
		{"kriteria tidak ada", domain.Penilaian{MustahikID: x.m2.ID, KriteriaID: "00000000-0000-4000-8000-000000000000", Nilai: 5}, true},
		{"nilai nol ditolak CHECK database", domain.Penilaian{MustahikID: x.m2.ID, KriteriaID: c, Nilai: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			err := x.s.UpsertPenilaian(bg, x.p.ID, []domain.Penilaian{
				{MustahikID: x.m1.ID, KriteriaID: c, Nilai: 7}, // sah, tetapi harus ikut dibatalkan
				tt.sel,
			})
			if err == nil {
				t.Fatal("mau error")
			}
			if tt.notFound != errors.Is(err, domain.ErrNotFound) {
				t.Errorf("err = %v; NotFound diharapkan = %v", err, tt.notFound)
			}
			if got := nilaiMap(t, x.s, x.p.ID); len(got) != 0 {
				t.Errorf("tersimpan %v padahal paket gagal (harus atomik)", got)
			}
		})
	}

	t.Run("daftar kosong tidak melakukan apa pun", func(t *testing.T) {
		if err := x.s.UpsertPenilaian(bg, x.p.ID, nil); err != nil {
			t.Error(err)
		}
	})
}

func TestListPenilaian_PeriodeLainTerpisah(t *testing.T) {
	x := siapkanPenilaian(t)
	p2 := periodeBaru(x.p.CreatedBy)
	if err := x.s.CreatePeriode(bg, &p2); err != nil {
		t.Fatal(err)
	}
	if err := x.s.ReplaceKandidat(bg, p2.ID, []string{x.m1.ID}); err != nil {
		t.Fatal(err)
	}
	c := x.ks[0].ID
	if err := x.s.UpsertPenilaian(bg, x.p.ID, []domain.Penilaian{{MustahikID: x.m1.ID, KriteriaID: c, Nilai: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := x.s.UpsertPenilaian(bg, p2.ID, []domain.Penilaian{{MustahikID: x.m1.ID, KriteriaID: c, Nilai: 2}}); err != nil {
		t.Fatal(err)
	}
	if got := nilaiMap(t, x.s, x.p.ID)[[2]string{x.m1.ID, c}]; got != 1 {
		t.Errorf("periode 1 = %v, mau 1 (nilai periode lain bocor)", got)
	}
	if got := nilaiMap(t, x.s, p2.ID)[[2]string{x.m1.ID, c}]; got != 2 {
		t.Errorf("periode 2 = %v, mau 2", got)
	}
}
