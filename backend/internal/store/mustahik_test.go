package store

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"spkzakat/internal/domain"
)

func TestMustahik_CreateGet(t *testing.T) {
	s := newStore(t)

	t.Run("tanpa No. HP", func(t *testing.T) {
		m := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)
		got, err := s.GetMustahik(bg, m.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.ID) != 36 || got.Nama != "Ahmad" || got.Asnaf != domain.AsnafFakir || !got.Aktif {
			t.Errorf("got = %+v", got)
		}
		if !bytes.Equal(got.NIKEnc, []byte{0x00, 0xff, 0x10}) {
			t.Errorf("data biner NIKEnc rusak: %v", got.NIKEnc)
		}
		if got.NoHPEnc != nil {
			t.Errorf("NoHPEnc = %v, mau nil (NULL)", got.NoHPEnc)
		}
		if got.CreatedAt.IsZero() || got.CreatedAt.Location() != time.UTC {
			t.Errorf("CreatedAt = %v (loc %v), mau waktu UTC terisi", got.CreatedAt, got.CreatedAt.Location())
		}
		if d := time.Since(got.CreatedAt); d < -time.Minute || d > time.Minute {
			t.Errorf("CreatedAt %v menyimpang %v dari sekarang (zona waktu salah?)", got.CreatedAt, d)
		}
	})

	t.Run("dengan No. HP", func(t *testing.T) {
		m := domain.Mustahik{Nama: "Budi", NIKEnc: []byte{1}, NIKHash: hashNIK("Budi"), Alamat: "a",
			NoHPEnc: []byte{0x00, 0x01, 0x02}, Asnaf: domain.AsnafMiskin, Aktif: true}
		if err := s.CreateMustahik(bg, &m); err != nil {
			t.Fatal(err)
		}
		got, _ := s.GetMustahik(bg, m.ID)
		if !bytes.Equal(got.NoHPEnc, []byte{0x00, 0x01, 0x02}) {
			t.Errorf("NoHPEnc = %v", got.NoHPEnc)
		}
	})

	t.Run("id tidak ada", func(t *testing.T) {
		if _, err := s.GetMustahik(bg, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})
}

func TestMustahik_NIKGandaDitolak(t *testing.T) {
	s := newStore(t)
	buatMustahik(t, s, "Ahmad", domain.AsnafFakir)

	dobel := domain.Mustahik{Nama: "Orang Lain", NIKEnc: []byte{9}, NIKHash: hashNIK("Ahmad"), Alamat: "a", Asnaf: domain.AsnafFakir, Aktif: true}
	err := s.CreateMustahik(bg, &dobel)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, mau ErrConflict", err)
	}
	var me *mysql.MySQLError
	if !errors.As(err, &me) || me.Number != 1062 {
		t.Errorf("error MySQL asli tidak ikut terbawa: %v", err)
	}
}

func TestMustahik_FindByNIKHash(t *testing.T) {
	s := newStore(t)
	m := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)

	got, err := s.FindMustahikByNIKHash(bg, hashNIK("Ahmad"))
	if err != nil || got.ID != m.ID {
		t.Errorf("Find = %+v, %v; mau ID %s", got, err, m.ID)
	}
	if _, err := s.FindMustahikByNIKHash(bg, hashNIK("tidak terdaftar")); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("belum terdaftar: err = %v, mau ErrNotFound", err)
	}
}

func TestMustahik_Update(t *testing.T) {
	s := newStore(t)
	a := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)
	b := buatMustahik(t, s, "Budi", domain.AsnafMiskin)

	t.Run("ubah semua kolom termasuk menonaktifkan", func(t *testing.T) {
		a.Nama, a.Alamat, a.Asnaf, a.Aktif = "Ahmad Baru", "Alamat Baru", domain.AsnafGharim, false
		a.NoHPEnc = []byte{7, 7}
		if err := s.UpdateMustahik(bg, a); err != nil {
			t.Fatal(err)
		}
		got, _ := s.GetMustahik(bg, a.ID)
		if got.Nama != "Ahmad Baru" || got.Alamat != "Alamat Baru" || got.Asnaf != domain.AsnafGharim || got.Aktif || !bytes.Equal(got.NoHPEnc, []byte{7, 7}) {
			t.Errorf("got = %+v", got)
		}
	})

	t.Run("simpan tanpa perubahan tetap sukses", func(t *testing.T) {
		if err := s.UpdateMustahik(bg, a); err != nil {
			t.Errorf("err = %v (clientFoundRows mati?)", err)
		}
	})

	t.Run("id tidak ada", func(t *testing.T) {
		hantu := a
		hantu.ID = "00000000-0000-4000-8000-000000000000"
		if err := s.UpdateMustahik(bg, hantu); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})

	t.Run("NIK diganti jadi milik orang lain", func(t *testing.T) {
		b.NIKHash = hashNIK("Ahmad") // hash milik a
		if err := s.UpdateMustahik(bg, b); !errors.Is(err, domain.ErrConflict) {
			t.Errorf("err = %v, mau ErrConflict", err)
		}
	})
}

func TestMustahik_List(t *testing.T) {
	s := newStore(t)
	buatMustahik(t, s, "Ahmad", domain.AsnafFakir)
	buatMustahik(t, s, "Budi", domain.AsnafMiskin)
	citra := buatMustahik(t, s, "Citra", domain.AsnafMiskin)
	buatMustahik(t, s, "dewi_x", domain.AsnafFakir)
	buatMustahik(t, s, "100% Jujur", domain.AsnafMiskin)
	citra.Aktif = false
	if err := s.UpdateMustahik(bg, citra); err != nil {
		t.Fatal(err)
	}
	ya, tidak := true, false

	tests := []struct {
		nama  string
		f     MustahikFilter
		mau   []string // nama yang diharapkan, berurutan
		total int
	}{
		{"semua, urut nama", MustahikFilter{}, []string{"100% Jujur", "Ahmad", "Budi", "Citra", "dewi_x"}, 5},
		{"cari potongan, tak peka huruf besar", MustahikFilter{Q: "AH"}, []string{"Ahmad"}, 1},
		{"% dibaca harfiah (bukan wildcard)", MustahikFilter{Q: "%"}, []string{"100% Jujur"}, 1},
		{"_ dibaca harfiah (bukan 1 karakter bebas)", MustahikFilter{Q: "_"}, []string{"dewi_x"}, 1},
		{"backslash dibaca harfiah", MustahikFilter{Q: `\`}, nil, 0},
		{"filter asnaf", MustahikFilter{Asnaf: domain.AsnafMiskin}, []string{"100% Jujur", "Budi", "Citra"}, 3},
		{"hanya aktif", MustahikFilter{Aktif: &ya}, []string{"100% Jujur", "Ahmad", "Budi", "dewi_x"}, 4},
		{"hanya nonaktif", MustahikFilter{Aktif: &tidak}, []string{"Citra"}, 1},
		{"gabungan q + asnaf + aktif", MustahikFilter{Q: "u", Asnaf: domain.AsnafMiskin, Aktif: &ya}, []string{"100% Jujur", "Budi"}, 2},
		{"halaman 2 (2 per halaman)", MustahikFilter{Page: 2, PerPage: 2}, []string{"Budi", "Citra"}, 5},
		{"halaman terakhir sebagian", MustahikFilter{Page: 3, PerPage: 2}, []string{"dewi_x"}, 5},
		{"halaman di luar jangkauan", MustahikFilter{Page: 99, PerPage: 2}, nil, 5},
		{"PerPage kebesaran dibatasi, bukan error", MustahikFilter{PerPage: 100000}, []string{"100% Jujur", "Ahmad", "Budi", "Citra", "dewi_x"}, 5},
		{"percobaan SQL injection di q → 0 baris, bukan error", MustahikFilter{Q: `' OR '1'='1`}, nil, 0},
		{"percobaan DROP TABLE di q → 0 baris", MustahikFilter{Q: `'; DROP TABLE mustahik; --`}, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			got, err := s.ListMustahik(bg, tt.f)
			if err != nil {
				t.Fatal(err)
			}
			var nama []string
			for _, m := range got.Items {
				nama = append(nama, m.Nama)
			}
			if !samaNama(nama, tt.mau) {
				t.Fatalf("nama = %v, mau %v", nama, tt.mau)
			}
			if got.Total != tt.total {
				t.Errorf("Total = %d, mau %d", got.Total, tt.total)
			}
		})
	}

	// Setelah semua percobaan injeksi, tabel harus masih utuh.
	if all, err := s.ListMustahik(bg, MustahikFilter{}); err != nil || all.Total != 5 {
		t.Fatalf("tabel rusak setelah uji injeksi: total=%d err=%v", all.Total, err)
	}

	t.Run("AsnafCounts hanya menghitung mustahik AKTIF, tak terpengaruh filter", func(t *testing.T) {
		got, err := s.ListMustahik(bg, MustahikFilter{Q: "ahmad"}) // filter sempit
		if err != nil {
			t.Fatal(err)
		}
		// aktif: Ahmad(fakir), dewi_x(fakir), Budi(miskin), 100% Jujur(miskin); Citra nonaktif.
		if got.AsnafCounts[domain.AsnafFakir] != 2 || got.AsnafCounts[domain.AsnafMiskin] != 2 || len(got.AsnafCounts) != 2 {
			t.Errorf("AsnafCounts = %v, mau fakir:2 miskin:2", got.AsnafCounts)
		}
	})

	t.Run("soft delete tidak ikut terbaca", func(t *testing.T) {
		all, _ := s.ListMustahik(bg, MustahikFilter{Q: "budi"})
		id := all.Items[0].ID
		if _, err := s.db.ExecContext(bg, `UPDATE mustahik SET deleted_at = NOW(3) WHERE id = ?`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetMustahik(bg, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Get: err = %v, mau ErrNotFound", err)
		}
		if got, _ := s.ListMustahik(bg, MustahikFilter{}); got.Total != 4 {
			t.Errorf("Total setelah soft delete = %d, mau 4", got.Total)
		}
	})
}
