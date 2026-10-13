package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

func zakat(ubah ...func(*PeriodeInput)) PeriodeInput {
	in := PeriodeInput{
		Nama: "Zakat Fitrah 1448 H", Jenis: domain.JenisZakat,
		DanaTersedia: 3_000_000, NominalPerPenerima: 1_000_000,
		AsnafDiizinkan: []domain.Asnaf{domain.AsnafFakir, domain.AsnafMiskin},
	}
	for _, f := range ubah {
		f(&in)
	}
	return in
}

func TestBuatPeriode_Sukses(t *testing.T) {
	e := baru(t)
	user := e.user(t)

	t.Run("zakat: nama dipangkas, asnaf dedupe, draft, kuota", func(t *testing.T) {
		p, err := e.svc.BuatPeriode(bg, user, zakat(func(in *PeriodeInput) {
			in.Nama = "  Zakat Fitrah  "
			in.AsnafDiizinkan = []domain.Asnaf{domain.AsnafMiskin, domain.AsnafFakir, domain.AsnafMiskin}
		}))
		if err != nil {
			t.Fatal(err)
		}
		if p.Nama != "Zakat Fitrah" || p.Status != domain.StatusDraft || p.Kuota() != 3 || len(p.AsnafDiizinkan) != 2 || p.CreatedBy != user {
			t.Errorf("periode = %+v", p)
		}
	})
	t.Run("infak: isian asnaf dibuang", func(t *testing.T) {
		p, err := e.svc.BuatPeriode(bg, user, zakat(func(in *PeriodeInput) {
			in.Jenis = domain.JenisInfak
			in.AsnafDiizinkan = []domain.Asnaf{domain.AsnafFakir}
		}))
		if err != nil || len(p.AsnafDiizinkan) != 0 {
			t.Errorf("periode = %+v, err = %v; infak tidak boleh membatasi golongan", p, err)
		}
	})
	t.Run("infak tanpa asnaf sah", func(t *testing.T) {
		if _, err := e.svc.BuatPeriode(bg, user, zakat(func(in *PeriodeInput) { in.Jenis, in.AsnafDiizinkan = domain.JenisInfak, nil })); err != nil {
			t.Error(err)
		}
	})
	t.Run("nominal sama dengan dana → kuota 1", func(t *testing.T) {
		p, err := e.svc.BuatPeriode(bg, user, zakat(func(in *PeriodeInput) { in.DanaTersedia, in.NominalPerPenerima = 1_000_000, 1_000_000 }))
		if err != nil || p.Kuota() != 1 {
			t.Errorf("kuota = %d, err = %v", p.Kuota(), err)
		}
	})
	t.Run("sisa pembagian dibuang (3,5 jt ÷ 1 jt = 3)", func(t *testing.T) {
		p, err := e.svc.BuatPeriode(bg, user, zakat(func(in *PeriodeInput) { in.DanaTersedia = 3_500_000 }))
		if err != nil || p.Kuota() != 3 {
			t.Errorf("kuota = %d, err = %v", p.Kuota(), err)
		}
	})
}

func TestBuatPeriode_Validasi(t *testing.T) {
	tests := []struct {
		nama   string
		ubah   func(*PeriodeInput)
		userID string
		field  string
	}{
		{"nama 2 karakter", func(in *PeriodeInput) { in.Nama = "ab" }, "u", "nama"},
		{"nama kosong", func(in *PeriodeInput) { in.Nama = "   " }, "u", "nama"},
		{"nama 101 karakter", func(in *PeriodeInput) { in.Nama = strings.Repeat("a", 101) }, "u", "nama"},
		{"jenis tidak dikenal", func(in *PeriodeInput) { in.Jenis = "wakaf" }, "u", "jenis"},
		{"dana nol", func(in *PeriodeInput) { in.DanaTersedia = 0 }, "u", "dana_tersedia"},
		{"dana negatif", func(in *PeriodeInput) { in.DanaTersedia = -1 }, "u", "dana_tersedia"},
		{"dana melebihi batas kolom", func(in *PeriodeInput) { in.DanaTersedia = maksUang + 1 }, "u", "dana_tersedia"},
		{"nominal nol", func(in *PeriodeInput) { in.NominalPerPenerima = 0 }, "u", "nominal_per_penerima"},
		{"nominal melebihi dana", func(in *PeriodeInput) { in.NominalPerPenerima = 3_000_001 }, "u", "nominal_per_penerima"},
		{"zakat tanpa asnaf", func(in *PeriodeInput) { in.AsnafDiizinkan = nil }, "u", "asnaf_diizinkan"},
		{"zakat dengan asnaf tidak dikenal", func(in *PeriodeInput) { in.AsnafDiizinkan = []domain.Asnaf{"xx"} }, "u", "asnaf_diizinkan"},
		{"pembuat tidak diketahui", func(in *PeriodeInput) {}, "  ", "pembuat"},
	}
	e := baru(t)
	e.user(t)
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			_, err := e.svc.BuatPeriode(bg, tt.userID, zakat(tt.ubah))
			cekValidasi(t, err, tt.field)
		})
	}

	t.Run("semua kesalahan dilaporkan sekaligus, tidak ada yang tersimpan", func(t *testing.T) {
		_, err := e.svc.BuatPeriode(bg, "", PeriodeInput{})
		ve := cekValidasi(t, err, "nama")
		for _, f := range []string{"nama", "jenis", "dana_tersedia", "nominal_per_penerima", "pembuat"} {
			if ve.Fields[f] == "" {
				t.Errorf("field %q tidak dilaporkan: %v", f, ve.Fields)
			}
		}
		if ps, _ := e.svc.DaftarPeriode(bg, ""); len(ps) != 0 {
			t.Errorf("tersimpan %d periode dari input tidak valid", len(ps))
		}
	})

	t.Run("user pembuat yang tidak ada → ErrNotFound (bukan panik)", func(t *testing.T) {
		_, err := e.svc.BuatPeriode(bg, idHantu, zakat())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})
}

func TestDaftarDanAmbilPeriode(t *testing.T) {
	e := baru(t)
	user := e.user(t)
	p, _ := e.svc.BuatPeriode(bg, user, zakat())

	if _, err := e.svc.DaftarPeriode(bg, "bukan-status"); err == nil {
		t.Error("status tak dikenal harus ditolak")
	} else {
		cekValidasi(t, err, "status")
	}
	if ps, err := e.svc.DaftarPeriode(bg, domain.StatusDraft); err != nil || len(ps) != 1 {
		t.Errorf("draft = %v, %v", ps, err)
	}
	if ps, err := e.svc.DaftarPeriode(bg, domain.StatusSelesai); err != nil || len(ps) != 0 {
		t.Errorf("selesai = %v, %v", ps, err)
	}
	if got, err := e.svc.AmbilPeriode(bg, p.ID); err != nil || got.ID != p.ID {
		t.Errorf("Ambil = %+v, %v", got, err)
	}
	if _, err := e.svc.AmbilPeriode(bg, idHantu); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, mau ErrNotFound", err)
	}
}

// siap = satu periode zakat (fakir+miskin) dan 4 mustahik.
type siap struct {
	e                        *lingkungan
	periode                  domain.Periode
	ahmad, budi, citra, dewi MustahikView // fakir, miskin, gharim, miskin-nonaktif
}

func siapkan(t *testing.T) siap {
	t.Helper()
	e := baru(t)
	p, err := e.svc.BuatPeriode(bg, e.user(t), zakat())
	if err != nil {
		t.Fatal(err)
	}
	return siap{
		e: e, periode: p,
		ahmad: e.buatMustahik(t, "Ahmad", "3271012345678901", domain.AsnafFakir, true),
		budi:  e.buatMustahik(t, "Budi", "3271012345678902", domain.AsnafMiskin, true),
		citra: e.buatMustahik(t, "Citra", "3271012345678903", domain.AsnafGharim, true),
		dewi:  e.buatMustahik(t, "Dewi", "3271012345678904", domain.AsnafMiskin, false),
	}
}

func (s siap) nama(t *testing.T) []string {
	t.Helper()
	ks, err := s.e.svc.DaftarKandidat(bg, s.periode.ID)
	if err != nil {
		t.Fatal(err)
	}
	var n []string
	for _, k := range ks {
		n = append(n, k.Nama)
	}
	return n
}

func TestAturKandidat(t *testing.T) {
	s := siapkan(t)

	t.Run("sukses; ID dobel dan kosong dibuang", func(t *testing.T) {
		if err := s.e.svc.AturKandidat(bg, s.periode.ID, []string{s.budi.ID, "", s.ahmad.ID, s.budi.ID}); err != nil {
			t.Fatal(err)
		}
		if got := s.nama(t); !samaNama(got, []string{"Ahmad", "Budi"}) {
			t.Errorf("kandidat = %v", got)
		}
	})

	// Kasus gagal: daftar kandidat sebelumnya HARUS tetap utuh (atomik).
	gagal := []struct {
		nama string
		ids  func() []string
		mau  string // potongan pesan pada field "kandidat"
	}{
		{"golongan tidak diizinkan (gharim di periode fakir+miskin)", func() []string { return []string{s.ahmad.ID, s.citra.ID} }, "Citra (gharim)"},
		{"mustahik nonaktif", func() []string { return []string{s.dewi.ID} }, "Dewi"},
		{"mustahik tidak ada", func() []string { return []string{s.ahmad.ID, idHantu} }, "1 mustahik tidak ditemukan"},
	}
	for _, tt := range gagal {
		t.Run(tt.nama, func(t *testing.T) {
			err := s.e.svc.AturKandidat(bg, s.periode.ID, tt.ids())
			ve := cekValidasi(t, err, "kandidat")
			if !strings.Contains(ve.Fields["kandidat"], tt.mau) {
				t.Errorf("pesan = %q, harus memuat %q", ve.Fields["kandidat"], tt.mau)
			}
			if got := s.nama(t); !samaNama(got, []string{"Ahmad", "Budi"}) {
				t.Errorf("kandidat berubah menjadi %v padahal ditolak", got)
			}
		})
	}

	t.Run("kombinasi: semua jenis masalah muncul di satu pesan", func(t *testing.T) {
		err := s.e.svc.AturKandidat(bg, s.periode.ID, []string{s.citra.ID, s.dewi.ID, idHantu})
		pesan := cekValidasi(t, err, "kandidat").Fields["kandidat"]
		for _, mau := range []string{"1 mustahik tidak ditemukan", "tidak aktif: Dewi", "Citra (gharim)"} {
			if !strings.Contains(pesan, mau) {
				t.Errorf("pesan %q tidak memuat %q", pesan, mau)
			}
		}
	})

	t.Run("lebih dari batas per periode", func(t *testing.T) {
		ids := make([]string, maksKandidat+1)
		for i := range ids {
			ids[i] = fmt.Sprintf("id-%d", i)
		}
		cekValidasi(t, s.e.svc.AturKandidat(bg, s.periode.ID, ids), "kandidat")
	})

	t.Run("daftar kosong mengosongkan kandidat", func(t *testing.T) {
		if err := s.e.svc.AturKandidat(bg, s.periode.ID, nil); err != nil {
			t.Fatal(err)
		}
		if got := s.nama(t); len(got) != 0 {
			t.Errorf("kandidat = %v", got)
		}
	})

	t.Run("periode tidak ada", func(t *testing.T) {
		if err := s.e.svc.AturKandidat(bg, idHantu, []string{s.ahmad.ID}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})
}

func TestAturKandidat_PeriodeInfakMenerimaSemuaGolongan(t *testing.T) {
	s := siapkan(t)
	infak, err := s.e.svc.BuatPeriode(bg, s.e.user(t), zakat(func(in *PeriodeInput) { in.Jenis, in.AsnafDiizinkan = domain.JenisInfak, nil }))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.e.svc.AturKandidat(bg, infak.ID, []string{s.ahmad.ID, s.citra.ID}); err != nil {
		t.Errorf("infak harus menerima golongan apa pun: %v", err)
	}
}

func TestAturKandidat_HanyaSaatDraft(t *testing.T) {
	s := siapkan(t)
	if err := s.e.svc.AturKandidat(bg, s.periode.ID, []string{s.ahmad.ID}); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"dihitung", "disetujui", "ditolak", "selesai"} {
		t.Run(status, func(t *testing.T) {
			s.e.exec(t, `UPDATE periode SET status = ? WHERE id = ?`, status, s.periode.ID)
			err := s.e.svc.AturKandidat(bg, s.periode.ID, []string{s.ahmad.ID, s.budi.ID})
			if !errors.Is(err, domain.ErrConflict) {
				t.Errorf("err = %v, mau ErrConflict", err)
			}
			if got := s.nama(t); !samaNama(got, []string{"Ahmad"}) {
				t.Errorf("kandidat berubah: %v", got)
			}
		})
	}
}

// Inti penguncian: bila status periode berubah ke "dihitung" SAAT AturKandidat
// sedang menunggu, ia harus melihat status baru (ErrConflict) dan TIDAK menulis
// kandidat ke periode yang sudah dihitung. Tanpa FOR UPDATE, AturKandidat
// membaca "draft" dari snapshot lama dan menulis juga.
func TestAturKandidat_StatusBerubahSaatMenunggu(t *testing.T) {
	s := siapkan(t)
	hasil := make(chan error, 1)

	err := s.e.st.WithTx(bg, func(tx *store.Store) error {
		if _, err := tx.GetPeriodeForUpdate(bg, s.periode.ID); err != nil { // "pengguna lain" memegang kunci
			return err
		}
		go func() { hasil <- s.e.svc.AturKandidat(bg, s.periode.ID, []string{s.ahmad.ID}) }()
		time.Sleep(400 * time.Millisecond) // beri waktu AturKandidat sampai ke titik menunggu
		return tx.SetStatusPeriode(bg, s.periode.ID, domain.StatusDraft, domain.StatusDihitung)
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-hasil:
		if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("err = %v, mau ErrConflict (status sudah dihitung)", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("AturKandidat tidak selesai")
	}
	if got := s.nama(t); len(got) != 0 {
		t.Errorf("kandidat tertulis ke periode yang sudah dihitung: %v", got)
	}
}

func TestDaftarKandidat_PeriodeTidakAda(t *testing.T) {
	e := baru(t)
	if _, err := e.svc.DaftarKandidat(bg, idHantu); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, mau ErrNotFound", err)
	}
}
