package store

import (
	"errors"
	"math"
	"testing"

	"spkzakat/internal/domain"
)

// Variabel (bukan konstanta) supaya 0,1+0,2 dihitung saat runtime sebagai
// float64 = 0.30000000000000004. Konstanta Go dihitung eksak = 0,3 dan tidak
// akan menguji apa pun.
var poin1, poin2 = 0.1, 0.2

// Sanity: seed V002 terbaca benar (tipe, bobot, skala JSON, urutan).
func TestListKriteria_Seed(t *testing.T) {
	s := newStore(t)
	ks, err := s.ListKriteria(bg, false)
	if err != nil {
		t.Fatal(err)
	}
	mau := []struct {
		kode  string
		tipe  domain.TipeKriteria
		bobot float64
	}{
		{"C1", domain.TipeCost, 0.30}, {"C2", domain.TipeBenefit, 0.20}, {"C3", domain.TipeBenefit, 0.20},
		{"C4", domain.TipeBenefit, 0.15}, {"C5", domain.TipeBenefit, 0.15},
	}
	if len(ks) != len(mau) {
		t.Fatalf("jumlah kriteria = %d, mau %d", len(ks), len(mau))
	}
	total := 0.0
	for i, m := range mau {
		k := ks[i]
		if k.Kode != m.kode || k.Tipe != m.tipe || k.Bobot != m.bobot || !k.Aktif || k.Urutan != i+1 || len(k.ID) != 36 {
			t.Errorf("kriteria %d = %+v, mau %+v", i, k, m)
		}
		total += k.Bobot
	}
	if math.Abs(total-1) > 1e-9 {
		t.Errorf("Σ bobot = %v, mau 1", total)
	}
	if ks[0].Skala != nil {
		t.Errorf("C1 tidak punya skala, dapat %v", ks[0].Skala)
	}
	if got := ks[2].Skala["4"]; got != "tidak layak / tanpa tempat tinggal" {
		t.Errorf("skala C3[4] = %q", got)
	}
}

func TestListKriteria_HanyaAktif(t *testing.T) {
	s := newStore(t)
	if _, err := s.db.ExecContext(bg, `UPDATE kriteria SET is_active = FALSE WHERE kode = 'C5'`); err != nil {
		t.Fatal(err)
	}
	semua, _ := s.ListKriteria(bg, false)
	aktif, err := s.ListKriteria(bg, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(semua) != 5 || len(aktif) != 4 {
		t.Errorf("semua=%d aktif=%d, mau 5 dan 4", len(semua), len(aktif))
	}
	for _, k := range aktif {
		if k.Kode == "C5" {
			t.Error("kriteria nonaktif ikut terbaca")
		}
	}
}

func TestUpdateKriteria(t *testing.T) {
	s := newStore(t)
	ks, _ := s.ListKriteria(bg, false)
	c1 := ks[0]

	tests := []struct {
		nama     string
		id       string
		tipe     domain.TipeKriteria
		bobot    float64
		mauBobot float64
		mauErr   error
		errBiasa bool // error apa pun selain NotFound (mis. CHECK/ENUM database)
	}{
		{nama: "ubah tipe dan bobot", id: c1.ID, tipe: domain.TipeBenefit, bobot: 0.35, mauBobot: 0.35},
		{nama: "bobot hasil penjumlahan float disimpan persis 0,3", id: c1.ID, tipe: domain.TipeCost, bobot: poin1 + poin2, mauBobot: 0.3},
		{nama: "nilai sama dengan yang tersimpan tetap sukses (bukan 'tidak ditemukan')", id: c1.ID, tipe: domain.TipeCost, bobot: 0.3, mauBobot: 0.3},
		{nama: "id tidak ada", id: "00000000-0000-4000-8000-000000000000", tipe: domain.TipeCost, bobot: 0.3, mauErr: domain.ErrNotFound},
		{nama: "bobot di atas 1 ditolak CHECK database", id: c1.ID, tipe: domain.TipeCost, bobot: 1.5, errBiasa: true},
		{nama: "tipe tidak dikenal ditolak ENUM database", id: c1.ID, tipe: "lainnya", bobot: 0.3, errBiasa: true},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			err := s.UpdateKriteria(bg, tt.id, tt.tipe, tt.bobot)
			switch {
			case tt.mauErr != nil:
				if !errors.Is(err, tt.mauErr) {
					t.Fatalf("err = %v, mau %v", err, tt.mauErr)
				}
			case tt.errBiasa:
				if err == nil || errors.Is(err, domain.ErrNotFound) {
					t.Fatalf("err = %v, mau error constraint database", err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				got, err := s.GetKriteria(bg, tt.id)
				if err != nil {
					t.Fatal(err)
				}
				if got.Bobot != tt.mauBobot || got.Tipe != tt.tipe {
					t.Errorf("tersimpan {%v %v}, mau {%v %v}", got.Tipe, got.Bobot, tt.tipe, tt.mauBobot)
				}
			}
		})
	}
}

func TestGetKriteria_TidakAdaDanTerhapus(t *testing.T) {
	s := newStore(t)
	ks, _ := s.ListKriteria(bg, false)

	if _, err := s.GetKriteria(bg, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("id tak ada: err = %v", err)
	}
	if _, err := s.db.ExecContext(bg, `UPDATE kriteria SET deleted_at = NOW(3) WHERE id = ?`, ks[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetKriteria(bg, ks[0].ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("soft-deleted: err = %v, mau ErrNotFound", err)
	}
	if err := s.UpdateKriteria(bg, ks[0].ID, domain.TipeCost, 0.3); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("update soft-deleted: err = %v, mau ErrNotFound", err)
	}
	if lst, _ := s.ListKriteria(bg, false); len(lst) != 4 {
		t.Errorf("List memuat kriteria terhapus: %d", len(lst))
	}
}

func TestCreateKriteria_DanUpdateKriteriaPenuh(t *testing.T) {
	s := newStore(t)

	k6, err := s.CreateKriteria(bg, "Jumlah anak sekolah", domain.TipeBenefit, 0.10, map[string]string{"1": "sedikit", "2": "banyak"})
	if err != nil {
		t.Fatal(err)
	}
	if k6.Kode != "C6" || k6.Urutan != 6 || !k6.Aktif || k6.Nama != "Jumlah anak sekolah" || math.Abs(k6.Bobot-0.10) > 1e-9 ||
		k6.Skala["1"] != "sedikit" || k6.Skala["2"] != "banyak" {
		t.Errorf("kriteria baru = %+v", k6)
	}
	k7, err := s.CreateKriteria(bg, "Beban utang", domain.TipeCost, 0.05, nil) // nil = bernilai asli
	if err != nil {
		t.Fatal(err)
	}
	if k7.Kode != "C7" || k7.Urutan != 7 || k7.Skala != nil {
		t.Errorf("kriteria kedua = %+v, mau C7 urutan 7 tanpa skala", k7)
	}

	// Penuh-ganti: nama, tipe, bobot, skala dihapus, nonaktif. Kode dan urutan tetap.
	if err := s.UpdateKriteriaPenuh(bg, k6.ID, "Anak usia sekolah", domain.TipeCost, 0.15, nil, false); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetKriteria(bg, k6.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kode != "C6" || got.Urutan != 6 || got.Nama != "Anak usia sekolah" || got.Tipe != domain.TipeCost ||
		math.Abs(got.Bobot-0.15) > 1e-9 || got.Skala != nil || got.Aktif {
		t.Errorf("sesudah ubah = %+v", got)
	}
	if aktif, _ := s.ListKriteria(bg, true); len(aktif) != 6 { // C1..C5 + C7; C6 nonaktif
		t.Errorf("kriteria aktif = %d, mau 6", len(aktif))
	}

	// Isian yang sama persis tetap dianggap berhasil (ClientFoundRows), bukan "tidak ketemu".
	if err := s.UpdateKriteriaPenuh(bg, k6.ID, "Anak usia sekolah", domain.TipeCost, 0.15, nil, false); err != nil {
		t.Errorf("ubah tanpa perubahan nilai: %v", err)
	}
	if err := s.UpdateKriteriaPenuh(bg, "00000000-0000-4000-8000-000000000000", "Hantu", domain.TipeCost, 0.15, nil, true); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("ID tak ada: err = %v, mau ErrNotFound", err)
	}
}

// UNIQUE(kode) berlaku atas semua baris, jadi kode kriteria yang sudah dihapus tidak dipakai ulang.
func TestCreateKriteria_KodeTidakDipakaiUlangSetelahTerhapus(t *testing.T) {
	s := newStore(t)
	k6, err := s.CreateKriteria(bg, "Sementara", domain.TipeBenefit, 0.05, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(bg, `UPDATE kriteria SET deleted_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, k6.ID); err != nil {
		t.Fatal(err)
	}
	k, err := s.CreateKriteria(bg, "Pengganti", domain.TipeBenefit, 0.05, nil)
	if err != nil {
		t.Fatalf("membuat setelah ada yang terhapus: %v", err)
	}
	if k.Kode != "C7" {
		t.Errorf("kode = %s, mau C7 (C6 pernah dipakai)", k.Kode)
	}
}
