package store

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

type siapRun struct {
	s       *Store
	user    string
	p       domain.Periode
	a, b, c domain.Mustahik
	bobot   []domain.BobotKriteria
}

func siapkanRun(t *testing.T) siapRun {
	t.Helper()
	s := newStore(t)
	user := seedUser(t, s)
	p := periodeBaru(user)
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}
	return siapRun{
		s: s, user: user, p: p,
		a: buatMustahik(t, s, "Ahmad", domain.AsnafFakir),
		b: buatMustahik(t, s, "Budi", domain.AsnafMiskin),
		c: buatMustahik(t, s, "Citra", domain.AsnafMiskin),
		bobot: []domain.BobotKriteria{
			{Kode: "C1", Tipe: domain.TipeCost, Bobot: 0.30}, {Kode: "C2", Tipe: domain.TipeBenefit, Bobot: 0.20},
			{Kode: "C3", Tipe: domain.TipeBenefit, Bobot: 0.20}, {Kode: "C4", Tipe: domain.TipeBenefit, Bobot: 0.15},
			{Kode: "C5", Tipe: domain.TipeBenefit, Bobot: 0.15},
		},
	}
}

func (x siapRun) hasil() []domain.ArasHasil {
	return []domain.ArasHasil{
		{MustahikID: x.b.ID, S: 0.2085357, K: 0.89387007, Ranking: 1, Status: domain.HasilUsulan},
		{MustahikID: x.a.ID, S: 0.1709409, K: 0.73272350, Ranking: 2, Status: domain.HasilUsulan},
		{MustahikID: x.c.ID, S: 0.0956221, K: 0.40987609, Ranking: 3, Status: domain.HasilCadangan},
	}
}

func (x siapRun) run(langkah []byte) *domain.ArasRun {
	return &domain.ArasRun{PeriodeID: x.p.ID, Bobot: x.bobot, S0: 0.23329549, Langkah: langkah, DijalankanOleh: x.user}
}

func TestSimpanRun_VersiBertambahDanHanyaSatuYangTerkini(t *testing.T) {
	x := siapkanRun(t)

	r1 := x.run(nil)
	if err := x.s.SimpanRun(bg, r1, x.hasil()); err != nil {
		t.Fatal(err)
	}
	if r1.Versi != 1 || !r1.IsCurrent || len(r1.ID) != 36 || r1.CreatedAt.IsZero() || r1.CreatedAt.Location() != time.UTC {
		t.Errorf("run 1 = %+v", r1)
	}

	r2 := x.run(nil)
	if err := x.s.SimpanRun(bg, r2, x.hasil()); err != nil {
		t.Fatal(err)
	}
	if r2.Versi != 2 || !r2.IsCurrent {
		t.Errorf("run 2 = %+v", r2)
	}

	riwayat, err := x.s.ListRun(bg, x.p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(riwayat) != 2 || riwayat[0].Versi != 2 || !riwayat[0].IsCurrent || riwayat[1].Versi != 1 || riwayat[1].IsCurrent {
		t.Errorf("riwayat = %+v, mau [v2 terkini, v1 bukan]", riwayat)
	}

	if cur, _ := x.s.GetRun(bg, x.p.ID, 0); cur.Versi != 2 {
		t.Errorf("GetRun(0) = v%d, mau v2 (terkini)", cur.Versi)
	}
	if lama, err := x.s.GetRun(bg, x.p.ID, 1); err != nil || lama.Versi != 1 || lama.IsCurrent {
		t.Errorf("GetRun(1) = %+v, %v", lama, err)
	}
}

func TestSimpanRun_VersiPerPeriodeTerpisah(t *testing.T) {
	x := siapkanRun(t)
	p2 := periodeBaru(x.user)
	if err := x.s.CreatePeriode(bg, &p2); err != nil {
		t.Fatal(err)
	}
	if err := x.s.SimpanRun(bg, x.run(nil), x.hasil()); err != nil {
		t.Fatal(err)
	}
	r := &domain.ArasRun{PeriodeID: p2.ID, Bobot: x.bobot, S0: 0.2, DijalankanOleh: x.user}
	if err := x.s.SimpanRun(bg, r, x.hasil()); err != nil {
		t.Fatal(err)
	}
	if r.Versi != 1 || !r.IsCurrent {
		t.Errorf("periode 2: versi=%d current=%v, mau 1/true", r.Versi, r.IsCurrent)
	}
	if cur, _ := x.s.GetRun(bg, x.p.ID, 0); !cur.IsCurrent {
		t.Error("run periode 1 kehilangan status terkini karena run periode lain")
	}
}

func TestSimpanRun_DataUtuhPulangPergi(t *testing.T) {
	x := siapkanRun(t)
	langkah := []byte(`{"a0":[500000,6,4,4,4],"matriks":[[2e-06,6],[1.25e-06,5]]}`)
	r := x.run(langkah)
	if err := x.s.SimpanRun(bg, r, x.hasil()); err != nil {
		t.Fatal(err)
	}
	got, err := x.s.GetRun(bg, x.p.ID, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got.Bobot, x.bobot) {
		t.Errorf("bobot snapshot = %+v, mau %+v", got.Bobot, x.bobot)
	}
	if math.Abs(got.S0-0.23329549) > 1e-8 || got.DijalankanOleh != x.user || got.PeriodeID != x.p.ID {
		t.Errorf("run = %+v", got)
	}
	// JSON dinormalkan MySQL (spasi/urutan kunci), jadi bandingkan isinya, bukan teksnya.
	var a, b any
	if json.Unmarshal(langkah, &a) != nil || json.Unmarshal(got.Langkah, &b) != nil || !reflect.DeepEqual(a, b) {
		t.Errorf("langkah = %s, mau setara dengan %s", got.Langkah, langkah)
	}

	hs, err := x.s.ListHasil(bg, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	mau := x.hasil()
	if len(hs) != 3 {
		t.Fatalf("hasil = %d baris", len(hs))
	}
	for i := range mau {
		if hs[i].MustahikID != mau[i].MustahikID || hs[i].Ranking != i+1 || hs[i].Status != mau[i].Status ||
			math.Abs(hs[i].S-mau[i].S) > 1e-8 || math.Abs(hs[i].K-mau[i].K) > 1e-8 {
			t.Errorf("hasil[%d] = %+v, mau %+v", i, hs[i], mau[i])
		}
	}
}

func TestSimpanRun_TanpaLangkahTersimpanNULL(t *testing.T) {
	x := siapkanRun(t)
	if err := x.s.SimpanRun(bg, x.run(nil), x.hasil()); err != nil {
		t.Fatal(err)
	}
	got, _ := x.s.GetRun(bg, x.p.ID, 1)
	if got.Langkah != nil {
		t.Errorf("Langkah = %s, mau nil (NULL)", got.Langkah)
	}
	// ListRun sengaja tidak memuat kolom langkah.
	riwayat, _ := x.s.ListRun(bg, x.p.ID)
	if len(riwayat) != 1 || riwayat[0].Langkah != nil {
		t.Errorf("ListRun memuat langkah: %+v", riwayat)
	}
}

func TestSimpanRun_GagalMembatalkanSemua(t *testing.T) {
	x := siapkanRun(t)
	if err := x.s.SimpanRun(bg, x.run(nil), x.hasil()); err != nil {
		t.Fatal(err)
	}

	hitungRun := func() int {
		var n int
		if err := x.s.db.GetContext(bg, &n, `SELECT COUNT(*) FROM aras_run WHERE periode_id = ?`, x.p.ID); err != nil {
			t.Fatal(err)
		}
		return n
	}
	hitungHasil := func() int {
		var n int
		if err := x.s.db.GetContext(bg, &n, `SELECT COUNT(*) FROM aras_hasil`); err != nil {
			t.Fatal(err)
		}
		return n
	}

	tests := []struct {
		nama  string
		run   *domain.ArasRun
		hasil []domain.ArasHasil
		mau   error
	}{
		{"mustahik dobel di hasil (UNIQUE)", x.run(nil), append(x.hasil(), domain.ArasHasil{MustahikID: x.a.ID, S: 0.1, K: 0.1, Ranking: 4, Status: domain.HasilCadangan}), domain.ErrConflict},
		{"mustahik di hasil tidak ada (FK)", x.run(nil), []domain.ArasHasil{{MustahikID: "00000000-0000-4000-8000-000000000000", S: 0.1, K: 0.1, Ranking: 1, Status: domain.HasilUsulan}}, domain.ErrNotFound},
		{"periode tidak ada (FK)", &domain.ArasRun{PeriodeID: "00000000-0000-4000-8000-000000000000", Bobot: x.bobot, DijalankanOleh: x.user}, x.hasil(), domain.ErrNotFound},
		{"user penjalan tidak ada (FK)", &domain.ArasRun{PeriodeID: x.p.ID, Bobot: x.bobot, DijalankanOleh: "00000000-0000-4000-8000-000000000000"}, x.hasil(), domain.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			runSebelum, hasilSebelum := hitungRun(), hitungHasil()
			err := x.s.SimpanRun(bg, tt.run, tt.hasil)
			if !errors.Is(err, tt.mau) {
				t.Fatalf("err = %v, mau %v", err, tt.mau)
			}
			if hitungRun() != runSebelum || hitungHasil() != hasilSebelum {
				t.Errorf("ada data tertinggal setelah gagal: run %d→%d, hasil %d→%d", runSebelum, hitungRun(), hasilSebelum, hitungHasil())
			}
			// Penanda terkini run sebelumnya TIDAK boleh ikut tercabut oleh percobaan yang gagal.
			if cur, err := x.s.GetRun(bg, x.p.ID, 0); err != nil || cur.Versi != 1 {
				t.Errorf("run terkini = %+v, %v; mau v1 tetap terkini", cur, err)
			}
		})
	}
}

func TestGetRun_TidakAda(t *testing.T) {
	x := siapkanRun(t)
	if _, err := x.s.GetRun(bg, x.p.ID, 0); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("belum pernah dihitung: err = %v, mau ErrNotFound", err)
	}
	if err := x.s.SimpanRun(bg, x.run(nil), x.hasil()); err != nil {
		t.Fatal(err)
	}
	if _, err := x.s.GetRun(bg, x.p.ID, 99); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("versi 99: err = %v, mau ErrNotFound", err)
	}
	if runs, err := x.s.ListRun(bg, "00000000-0000-4000-8000-000000000000"); err != nil || len(runs) != 0 {
		t.Errorf("ListRun periode lain = %v, %v; mau kosong", runs, err)
	}
}

func TestSimpanRun_PresisiDelapanDesimal(t *testing.T) {
	x := siapkanRun(t)
	h := []domain.ArasHasil{{MustahikID: x.a.ID, S: 0.123456789012, K: 0.987654321098, Ranking: 1, Status: domain.HasilUsulan}}
	r := x.run(nil)
	if err := x.s.SimpanRun(bg, r, h); err != nil {
		t.Fatal(err)
	}
	got, _ := x.s.ListHasil(bg, r.ID)
	if len(got) != 1 || got[0].S != 0.12345679 || got[0].K != 0.98765432 {
		t.Errorf("hasil = %+v, mau dibulatkan ke 8 desimal (0.12345679 / 0.98765432)", got)
	}
}
