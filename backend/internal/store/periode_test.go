package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

func periodeBaru(userID string) domain.Periode {
	return domain.Periode{
		Nama: "Zakat Fitrah 1448 H", Jenis: domain.JenisZakat,
		DanaTersedia: 3_000_000, NominalPerPenerima: 1_000_000,
		AsnafDiizinkan: []domain.Asnaf{domain.AsnafFakir, domain.AsnafMiskin},
		CreatedBy:      userID,
	}
}

func TestPeriode_CreateGet(t *testing.T) {
	s := newStore(t)
	user := seedUser(t, s)

	t.Run("zakat dengan asnaf", func(t *testing.T) {
		p := periodeBaru(user)
		if err := s.CreatePeriode(bg, &p); err != nil {
			t.Fatal(err)
		}
		got, err := s.GetPeriode(bg, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != domain.StatusDraft || got.Jenis != domain.JenisZakat || got.CreatedBy != user {
			t.Errorf("got = %+v", got)
		}
		if got.DanaTersedia != 3_000_000 || got.NominalPerPenerima != 1_000_000 || got.Kuota() != 3 {
			t.Errorf("uang tidak utuh: dana=%d nominal=%d kuota=%d", got.DanaTersedia, got.NominalPerPenerima, got.Kuota())
		}
		if len(got.AsnafDiizinkan) != 2 || got.AsnafDiizinkan[0] != domain.AsnafFakir || got.AsnafDiizinkan[1] != domain.AsnafMiskin {
			t.Errorf("AsnafDiizinkan = %v, mau [fakir miskin]", got.AsnafDiizinkan)
		}
		if got.CreatedAt.IsZero() {
			t.Error("CreatedAt kosong")
		}
	})

	t.Run("urutan asnaf input dinormalkan MySQL (urutan definisi SET)", func(t *testing.T) {
		p := periodeBaru(user)
		p.AsnafDiizinkan = []domain.Asnaf{domain.AsnafMiskin, domain.AsnafFakir}
		if err := s.CreatePeriode(bg, &p); err != nil {
			t.Fatal(err)
		}
		if p.AsnafDiizinkan[0] != domain.AsnafFakir || p.AsnafDiizinkan[1] != domain.AsnafMiskin {
			t.Errorf("AsnafDiizinkan = %v", p.AsnafDiizinkan)
		}
	})

	t.Run("infak tanpa asnaf disimpan NULL", func(t *testing.T) {
		p := periodeBaru(user)
		p.Jenis, p.AsnafDiizinkan = domain.JenisInfak, nil
		if err := s.CreatePeriode(bg, &p); err != nil {
			t.Fatal(err)
		}
		if len(p.AsnafDiizinkan) != 0 {
			t.Errorf("AsnafDiizinkan = %v, mau kosong", p.AsnafDiizinkan)
		}
	})

	t.Run("created_by tidak ada → ErrNotFound (FK)", func(t *testing.T) {
		p := periodeBaru("00000000-0000-4000-8000-000000000000")
		if err := s.CreatePeriode(bg, &p); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})

	t.Run("data yang dilarang CHECK/ENUM database ditolak", func(t *testing.T) {
		nol := periodeBaru(user)
		nol.DanaTersedia = 0
		salah := periodeBaru(user)
		salah.AsnafDiizinkan = []domain.Asnaf{"bukan-asnaf"}
		for nama, p := range map[string]domain.Periode{"dana nol": nol, "asnaf tidak dikenal": salah} {
			if err := s.CreatePeriode(bg, &p); err == nil || errors.Is(err, domain.ErrNotFound) {
				t.Errorf("%s: err = %v, mau error constraint", nama, err)
			}
		}
	})

	t.Run("id tidak ada", func(t *testing.T) {
		if _, err := s.GetPeriode(bg, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})
}

func TestPeriode_List(t *testing.T) {
	s := newStore(t)
	user := seedUser(t, s)

	var ids []string
	for i, nama := range []string{"Lama", "Tengah", "Baru"} {
		p := periodeBaru(user)
		p.Nama = nama
		if err := s.CreatePeriode(bg, &p); err != nil {
			t.Fatal(err)
		}
		// created_at diatur eksplisit: dua INSERT bisa jatuh di milidetik yang sama.
		waktu := time.Now().UTC().Add(time.Duration(i-3) * time.Hour)
		if _, err := s.db.ExecContext(bg, `UPDATE periode SET created_at = ? WHERE id = ?`, waktu, p.ID); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	if err := s.SetStatusPeriode(bg, ids[0], domain.StatusDraft, domain.StatusSelesai); err != nil {
		t.Fatal(err)
	}

	semua, err := s.ListPeriode(bg, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(semua) != 3 || semua[0].Nama != "Baru" || semua[1].Nama != "Tengah" || semua[2].Nama != "Lama" {
		t.Fatalf("urutan = %+v, mau terbaru dulu (Baru Tengah Lama)", semua)
	}
	if draft, _ := s.ListPeriode(bg, domain.StatusDraft); len(draft) != 2 {
		t.Errorf("filter draft = %d, mau 2", len(draft))
	}
	if selesai, _ := s.ListPeriode(bg, domain.StatusSelesai); len(selesai) != 1 || selesai[0].Nama != "Lama" {
		t.Errorf("filter selesai = %+v", selesai)
	}
	if kosong, err := s.ListPeriode(bg, domain.StatusDitolak); err != nil || len(kosong) != 0 {
		t.Errorf("filter ditolak = %v, %v; mau slice kosong", kosong, err)
	}
}

func TestPeriode_SetStatus(t *testing.T) {
	s := newStore(t)
	user := seedUser(t, s)
	p := periodeBaru(user)
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}

	t.Run("draft → dihitung berhasil", func(t *testing.T) {
		if err := s.SetStatusPeriode(bg, p.ID, domain.StatusDraft, domain.StatusDihitung); err != nil {
			t.Fatal(err)
		}
		if got, _ := s.GetPeriode(bg, p.ID); got.Status != domain.StatusDihitung {
			t.Errorf("status = %s", got.Status)
		}
	})
	t.Run("status saat ini bukan 'dari' → ErrConflict, data tak berubah", func(t *testing.T) {
		err := s.SetStatusPeriode(bg, p.ID, domain.StatusDraft, domain.StatusDitolak)
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("err = %v, mau ErrConflict", err)
		}
		if got, _ := s.GetPeriode(bg, p.ID); got.Status != domain.StatusDihitung {
			t.Errorf("status berubah menjadi %s", got.Status)
		}
	})
	t.Run("periode tidak ada → ErrNotFound", func(t *testing.T) {
		err := s.SetStatusPeriode(bg, "00000000-0000-4000-8000-000000000000", domain.StatusDraft, domain.StatusDihitung)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})
}

// Dua pengguna menekan tombol bersamaan: tepat satu yang boleh menang.
func TestPeriode_SetStatus_Balapan(t *testing.T) {
	s := newStore(t)
	p := periodeBaru(seedUser(t, s))
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}

	const n = 20
	var wg sync.WaitGroup
	hasil := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ke := domain.StatusDihitung
			if i%2 == 1 {
				ke = domain.StatusDitolak
			}
			hasil[i] = s.SetStatusPeriode(bg, p.ID, domain.StatusDraft, ke)
		}(i)
	}
	wg.Wait()

	menang := 0
	for _, err := range hasil {
		switch {
		case err == nil:
			menang++
		case !errors.Is(err, domain.ErrConflict):
			t.Errorf("error tak terduga: %v", err)
		}
	}
	if menang != 1 {
		t.Fatalf("pemenang = %d, mau tepat 1", menang)
	}
}

func jumlahPenilaian(t *testing.T, s *Store, periodeID string) int {
	t.Helper()
	var n int
	if err := s.db.GetContext(bg, &n, `SELECT COUNT(*) FROM penilaian WHERE periode_id = ?`, periodeID); err != nil {
		t.Fatal(err)
	}
	return n
}

func namaKandidat(t *testing.T, s *Store, periodeID string) []string {
	t.Helper()
	ms, err := s.ListKandidat(bg, periodeID)
	if err != nil {
		t.Fatal(err)
	}
	var nama []string
	for _, m := range ms {
		nama = append(nama, m.Nama)
	}
	return nama
}

func samaNama(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPeriode_ReplaceKandidat(t *testing.T) {
	s := newStore(t)
	p := periodeBaru(seedUser(t, s))
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}
	m1 := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)
	m2 := buatMustahik(t, s, "Budi", domain.AsnafMiskin)
	m3 := buatMustahik(t, s, "Citra", domain.AsnafMiskin)
	ks, _ := s.ListKriteria(bg, false)
	nilai := func(mustahikID string) {
		t.Helper()
		_, err := s.db.ExecContext(bg,
			`INSERT INTO penilaian (id, periode_id, mustahik_id, kriteria_id, nilai) VALUES (?, ?, ?, ?, 5)`,
			newID(), p.ID, mustahikID, ks[0].ID)
		if err != nil {
			t.Fatalf("seed penilaian: %v", err)
		}
	}

	if err := s.ReplaceKandidat(bg, p.ID, []string{m2.ID, m1.ID}); err != nil {
		t.Fatal(err)
	}
	if got := namaKandidat(t, s, p.ID); !samaNama(got, []string{"Ahmad", "Budi"}) {
		t.Fatalf("kandidat = %v, mau [Ahmad Budi] (urut nama)", got)
	}
	nilai(m1.ID)
	nilai(m2.ID)

	// Regression guard: mempertahankan m1 TIDAK boleh menghapus penilaiannya.
	// (Implementasi "hapus semua lalu isi ulang" akan menghapus penilaian m1
	// lewat ON DELETE CASCADE.)
	if err := s.ReplaceKandidat(bg, p.ID, []string{m1.ID, m3.ID}); err != nil {
		t.Fatal(err)
	}
	if got := namaKandidat(t, s, p.ID); !samaNama(got, []string{"Ahmad", "Citra"}) {
		t.Errorf("kandidat = %v, mau [Ahmad Citra]", got)
	}
	if n := jumlahPenilaian(t, s, p.ID); n != 1 {
		t.Errorf("penilaian tersisa = %d, mau 1 (m1 dipertahankan, m2 dihapus bersama kandidatnya)", n)
	}

	t.Run("mustahik tidak ada → ErrNotFound dan semua perubahan dibatalkan", func(t *testing.T) {
		err := s.ReplaceKandidat(bg, p.ID, []string{m1.ID, "00000000-0000-4000-8000-000000000000", m2.ID})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, mau ErrNotFound", err)
		}
		if got := namaKandidat(t, s, p.ID); !samaNama(got, []string{"Ahmad", "Citra"}) {
			t.Errorf("daftar berubah sebagian: %v (harus atomik)", got)
		}
	})

	t.Run("ID dobel di input aman", func(t *testing.T) {
		if err := s.ReplaceKandidat(bg, p.ID, []string{m1.ID, m1.ID, m2.ID}); err != nil {
			t.Fatal(err)
		}
		if got := namaKandidat(t, s, p.ID); !samaNama(got, []string{"Ahmad", "Budi"}) {
			t.Errorf("kandidat = %v", got)
		}
	})

	t.Run("daftar kosong mengosongkan kandidat", func(t *testing.T) {
		if err := s.ReplaceKandidat(bg, p.ID, nil); err != nil {
			t.Fatal(err)
		}
		if got := namaKandidat(t, s, p.ID); len(got) != 0 {
			t.Errorf("kandidat = %v, mau kosong", got)
		}
	})

	t.Run("periode tidak ada → ErrNotFound", func(t *testing.T) {
		err := s.ReplaceKandidat(bg, "00000000-0000-4000-8000-000000000000", []string{m1.ID})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})

	t.Run("ikut transaksi pemanggil dan ikut dibatalkan bersamanya", func(t *testing.T) {
		galat := errors.New("batal")
		err := s.WithTx(bg, func(tx *Store) error {
			if err := tx.ReplaceKandidat(bg, p.ID, []string{m3.ID}); err != nil {
				return err
			}
			return galat
		})
		if !errors.Is(err, galat) {
			t.Fatalf("err = %v", err)
		}
		if got := namaKandidat(t, s, p.ID); len(got) != 0 {
			t.Errorf("kandidat = %v; rollback transaksi luar harus membatalkannya", got)
		}
	})
}

func TestPeriode_ListKandidat_TanpaMustahikTerhapus(t *testing.T) {
	s := newStore(t)
	p := periodeBaru(seedUser(t, s))
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}
	a := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)
	b := buatMustahik(t, s, "Budi", domain.AsnafMiskin)
	if err := s.ReplaceKandidat(bg, p.ID, []string{a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(bg, `UPDATE mustahik SET deleted_at = NOW(3) WHERE id = ?`, b.ID); err != nil {
		t.Fatal(err)
	}
	if got := namaKandidat(t, s, p.ID); !samaNama(got, []string{"Ahmad"}) {
		t.Errorf("kandidat = %v, mau [Ahmad]", got)
	}
}
