package service

import (
	"errors"
	"sync"
	"testing"

	"spkzakat/internal/domain"
)

func TestBuatPengguna_Validasi(t *testing.T) {
	e := baru(t)
	e.akun(t, "sudahada", domain.RoleAdmin)
	tests := []struct {
		nama, username, namaLengkap, password string
		role                                  domain.Role
		field                                 string
	}{
		{"username pendek", "ab", "Budi", pwUji, domain.RoleAdmin, "username"},
		{"username berspasi", "bu di", "Budi", pwUji, domain.RoleAdmin, "username"},
		{"username dobel", "sudahada", "Budi", pwUji, domain.RoleAdmin, "username"},
		{"nama kosong", "budi", " ", pwUji, domain.RoleAdmin, "nama"},
		{"peran asing", "budi", "Budi", pwUji, "superadmin", "role"},
		{"password pendek", "budi", "Budi", "pendek", domain.RoleAdmin, "password"},
	}
	for _, tt := range tests {
		_, err := e.svc.BuatPengguna(bg, tt.username, tt.namaLengkap, tt.role, tt.password)
		var ve *domain.ValidationError
		if !errors.As(err, &ve) || ve.Fields[tt.field] == "" {
			t.Errorf("%s: err = %v, mau galat field %q", tt.nama, err, tt.field)
		}
	}
	v, err := e.svc.BuatPengguna(bg, "budi", "Budi", domain.RoleKetua, pwUji)
	if err != nil || !v.Aktif || v.Role != domain.RoleKetua {
		t.Fatalf("buat sah: %+v, %v", v, err)
	}
}

// Nonaktif harus benar-benar memutus akses: login gagal, sesi yang sedang
// dipakai langsung tidak berlaku, dan akun bisa dihidupkan lagi.
func TestNonaktifkanPengguna_MemutusAksesDanBisaDiaktifkanLagi(t *testing.T) {
	e := baru(t)
	ketua := e.akun(t, "ketua1", domain.RoleKetua)
	e.akun(t, "admin1", domain.RoleAdmin)
	target := e.akun(t, "admin2", domain.RoleAdmin)

	masuk, err := e.svc.Login(bg, "admin2", pwUji, "1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Sesi(bg, masuk.Token); err != nil {
		t.Fatalf("sesi awal: %v", err)
	}

	v, err := e.svc.NonaktifkanPengguna(bg, ketua, target.ID)
	if err != nil || v.Aktif {
		t.Fatalf("nonaktifkan: %+v, %v", v, err)
	}
	if _, err := e.svc.Sesi(bg, masuk.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("sesi setelah nonaktif: err = %v, mau not found", err)
	}
	var lg *LoginGagalError
	if _, err := e.svc.Login(bg, "admin2", pwUji, "1.1.1.1"); !errors.As(err, &lg) {
		t.Errorf("login nonaktif: err = %v, mau LoginGagalError", err)
	}
	if _, err := e.svc.NonaktifkanPengguna(bg, ketua, target.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("nonaktif dua kali: err = %v, mau konflik", err)
	}

	// Akun nonaktif tetap tampil di daftar (di bawah yang aktif) dan username-nya tetap terpakai.
	daftar, _ := e.svc.DaftarPengguna(bg)
	if len(daftar) != 3 || daftar[2].Username != "admin2" || daftar[2].Aktif {
		t.Errorf("daftar = %+v", daftar)
	}
	if _, err := e.svc.BuatPengguna(bg, "admin2", "Lain", domain.RoleAdmin, pwUji); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("username akun nonaktif dipakai ulang: err = %v", err)
	}

	if v, err := e.svc.AktifkanPengguna(bg, target.ID); err != nil || !v.Aktif {
		t.Fatalf("aktifkan: %+v, %v", v, err)
	}
	if _, err := e.svc.Login(bg, "admin2", pwUji, "1.1.1.1"); err != nil {
		t.Errorf("login setelah diaktifkan: %v", err)
	}
	if _, err := e.svc.AktifkanPengguna(bg, target.ID); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("aktifkan dua kali: err = %v", err)
	}
}

func TestPengguna_AturanKeselamatan(t *testing.T) {
	e := baru(t)
	ketua := e.akun(t, "ketua1", domain.RoleKetua)
	admin := e.akun(t, "admin1", domain.RoleAdmin) // satu-satunya Admin

	tests := []struct {
		nama string
		err  error
	}{
		{"nonaktifkan diri sendiri", func() error { _, err := e.svc.NonaktifkanPengguna(bg, ketua, ketua.ID); return err }()},
		{"turunkan peran diri sendiri", func() error {
			_, err := e.svc.UbahPengguna(bg, ketua, ketua.ID, "Ketua", domain.RoleAdmin)
			return err
		}()},
		{"reset password diri sendiri", e.svc.ResetPasswordPengguna(bg, ketua, ketua.ID, "password-baru-12345")},
		{"nonaktifkan Admin terakhir", func() error { _, err := e.svc.NonaktifkanPengguna(bg, ketua, admin.ID); return err }()},
		{"jadikan Admin terakhir sebagai ketua", func() error {
			_, err := e.svc.UbahPengguna(bg, ketua, admin.ID, "Admin", domain.RoleKetua)
			return err
		}()},
	}
	for _, tt := range tests {
		if !errors.Is(tt.err, domain.ErrConflict) {
			t.Errorf("%s: err = %v, mau konflik", tt.nama, tt.err)
		}
	}

	// Ada Admin lain → Admin pertama boleh diturunkan; mengubah nama saja selalu boleh.
	e.akun(t, "admin2", domain.RoleAdmin)
	if v, err := e.svc.UbahPengguna(bg, ketua, admin.ID, "Nama Baru", domain.RoleAdmin); err != nil || v.Nama != "Nama Baru" {
		t.Errorf("ubah nama: %+v, %v", v, err)
	}
	if v, err := e.svc.UbahPengguna(bg, ketua, admin.ID, "Nama Baru", domain.RoleKetua); err != nil || v.Role != domain.RoleKetua {
		t.Errorf("turunkan Admin ketika ada Admin lain: %+v, %v", v, err)
	}
	var ve *domain.ValidationError
	if _, err := e.svc.UbahPengguna(bg, ketua, admin.ID, "x", domain.RoleKetua); !errors.As(err, &ve) {
		t.Errorf("nama 1 huruf: err = %v", err)
	}
	if _, err := e.svc.UbahPengguna(bg, ketua, "00000000-0000-4000-8000-000000000000", "Nama", domain.RoleAdmin); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("akun tidak ada: err = %v", err)
	}
}

// Dua Ketua menonaktifkan dua Admin terakhir bersamaan: hanya satu yang boleh
// berhasil. Tanpa kunci baris, keduanya lolos dan tersisa nol Admin aktif.
func TestNonaktifkanPengguna_BersamaanTidakMenyisakanNolAdmin(t *testing.T) {
	e := baru(t)
	k1 := e.akun(t, "ketua1", domain.RoleKetua)
	k2 := e.akun(t, "ketua2", domain.RoleKetua)
	a1 := e.akun(t, "admin1", domain.RoleAdmin)
	a2 := e.akun(t, "admin2", domain.RoleAdmin)

	var wg sync.WaitGroup
	hasil := make([]error, 2)
	for i, c := range []struct{ aktor, sasaran domain.User }{{k1, a1}, {k2, a2}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, hasil[i] = e.svc.NonaktifkanPengguna(bg, c.aktor, c.sasaran.ID)
		}()
	}
	wg.Wait()

	berhasil := 0
	for _, err := range hasil {
		if err == nil {
			berhasil++
		} else if !errors.Is(err, domain.ErrConflict) {
			t.Errorf("galat tak terduga: %v", err)
		}
	}
	if berhasil != 1 {
		t.Fatalf("berhasil = %d, mau tepat 1 (hasil: %v)", berhasil, hasil)
	}
	aktif := 0
	daftar, _ := e.svc.DaftarPengguna(bg)
	for _, u := range daftar {
		if u.Role == domain.RoleAdmin && u.Aktif {
			aktif++
		}
	}
	if aktif != 1 {
		t.Errorf("Admin aktif = %d, mau 1", aktif)
	}
}

func TestResetPasswordPengguna(t *testing.T) {
	e := baru(t)
	ketua := e.akun(t, "ketua1", domain.RoleKetua)
	target := e.akun(t, "admin1", domain.RoleAdmin)
	masuk, err := e.svc.Login(bg, "admin1", pwUji, "2.2.2.2")
	if err != nil {
		t.Fatal(err)
	}

	var ve *domain.ValidationError
	if err := e.svc.ResetPasswordPengguna(bg, ketua, target.ID, "pendek"); !errors.As(err, &ve) || ve.Fields["password"] == "" {
		t.Errorf("password pendek: err = %v", err)
	}
	if err := e.svc.ResetPasswordPengguna(bg, ketua, "00000000-0000-4000-8000-000000000000", "password-baru-12345"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("akun tidak ada: err = %v", err)
	}
	if err := e.svc.ResetPasswordPengguna(bg, ketua, target.ID, "password-baru-12345"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Sesi(bg, masuk.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("sesi lama setelah reset: err = %v, mau dicabut", err)
	}
	var lg *LoginGagalError
	if _, err := e.svc.Login(bg, "admin1", pwUji, "2.2.2.2"); !errors.As(err, &lg) {
		t.Errorf("password lama masih berlaku: err = %v", err)
	}
	if _, err := e.svc.Login(bg, "admin1", "password-baru-12345", "2.2.2.2"); err != nil {
		t.Errorf("password baru: %v", err)
	}
}
