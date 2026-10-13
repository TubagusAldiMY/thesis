package service

import (
	"errors"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

// TestPembatas memeriksa aturan penguncian tanpa database, dengan jam palsu.
func TestPembatas(t *testing.T) {
	sekarang := time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)
	p := pembatasBaru()
	p.now = func() time.Time { return sekarang }

	tests := []struct {
		nama     string
		maju     time.Duration // jam dimajukan sebelum langkah
		aksi     string        // gagal | sukses | cek
		kunci    string
		mauSisa  int
		mauKunci bool // gagal ini memicu kunci / cek melihat terkunci
	}{
		{"gagal 1", 0, "gagal", "a|ip", 4, false},
		{"gagal 2", 0, "gagal", "a|ip", 3, false},
		{"kunci lain tidak terpengaruh", 0, "gagal", "b|ip", 4, false},
		{"gagal 3", 0, "gagal", "a|ip", 2, false},
		{"gagal 4", 0, "gagal", "a|ip", 1, false},
		{"gagal 5 mengunci", 0, "gagal", "a|ip", 0, true},
		{"masih terkunci 14 menit kemudian", 14 * time.Minute, "cek", "a|ip", 0, true},
		{"terbuka setelah 15 menit", 2 * time.Minute, "cek", "a|ip", 0, false},
		{"hitungan mulai lagi dari nol", 0, "gagal", "a|ip", 4, false},
		{"sukses menghapus hitungan", 0, "sukses", "a|ip", 0, false},
		{"setelah sukses kembali 4", 0, "gagal", "a|ip", 4, false},
		{"gagal lama (>15 mnt) tidak dihitung", 16 * time.Minute, "gagal", "a|ip", 4, false},
	}
	for _, tt := range tests {
		sekarang = sekarang.Add(tt.maju)
		switch tt.aksi {
		case "gagal":
			sisa, kunci := p.gagal(tt.kunci)
			if sisa != tt.mauSisa || (kunci > 0) != tt.mauKunci {
				t.Errorf("%s: sisa=%d kunci=%v, mau sisa=%d kunci=%v", tt.nama, sisa, kunci, tt.mauSisa, tt.mauKunci)
			}
		case "sukses":
			p.sukses(tt.kunci)
		case "cek":
			if got := p.terkunci(tt.kunci) > 0; got != tt.mauKunci {
				t.Errorf("%s: terkunci=%v, mau %v", tt.nama, got, tt.mauKunci)
			}
		}
	}
}

const pwUji = "password-uji-panjang"

func (e *lingkungan) akun(t *testing.T, username string, role domain.Role) domain.User {
	t.Helper()
	u, err := e.svc.BuatUser(bg, username, "Nama "+username, role, pwUji)
	if err != nil {
		t.Fatalf("BuatUser: %v", err)
	}
	return u
}

func TestLogin_SuksesSesiLogout(t *testing.T) {
	e := baru(t)
	u := e.akun(t, "admin1", domain.RoleAdmin)

	h, err := e.svc.Login(bg, "  ADMIN1 ", pwUji, "10.0.0.1") // username tidak peka huruf besar & spasi
	if err != nil {
		t.Fatal(err)
	}
	if h.Token == "" || h.User.ID != u.ID {
		t.Fatalf("hasil login = %+v", h)
	}
	var tersimpan string
	if err := e.db.QueryRow(`SELECT token_hash FROM sessions WHERE user_id = ?`, u.ID).Scan(&tersimpan); err != nil {
		t.Fatal(err)
	}
	if tersimpan == h.Token || tersimpan != hashToken(h.Token) {
		t.Fatal("tabel sessions harus menyimpan hash token, bukan token aslinya")
	}

	got, err := e.svc.Sesi(bg, h.Token)
	if err != nil || got.ID != u.ID || got.Role != domain.RoleAdmin {
		t.Fatalf("Sesi = %+v, %v", got, err)
	}
	if err := e.svc.Logout(bg, got, h.Token, "10.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Sesi(bg, h.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("setelah logout Sesi err = %v, mau ErrNotFound", err)
	}

	var aksi []string
	rows, err := e.db.Query(`SELECT action FROM audit_logs ORDER BY created_at, action`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		aksi = append(aksi, a)
	}
	if !samaNama(aksi, []string{"LOGIN", "LOGOUT"}) {
		t.Errorf("audit = %v, mau [LOGIN LOGOUT]", aksi)
	}
}

func TestLogin_GagalSamaUntukUsernameTakDikenalLaluTerkunci(t *testing.T) {
	e := baru(t)
	e.akun(t, "admin1", domain.RoleAdmin)

	var lg *LoginGagalError
	_, err := e.svc.Login(bg, "tidakada", pwUji, "10.0.0.1")
	if !errors.As(err, &lg) || lg.SisaPercobaan != 4 {
		t.Fatalf("username tak dikenal: err = %v", err)
	}
	pesanTakDikenal := err.Error()

	for i := 4; i >= 1; i-- {
		_, err = e.svc.Login(bg, "admin1", "salah-salah-salah", "10.0.0.2")
		if !errors.As(err, &lg) || lg.SisaPercobaan != i {
			t.Fatalf("percobaan salah: err = %v, mau sisa %d", err, i)
		}
		if err.Error() != pesanTakDikenal {
			t.Fatal("pesan password salah harus sama dengan username tak dikenal")
		}
	}
	_, err = e.svc.Login(bg, "admin1", "salah-salah-salah", "10.0.0.2")
	var tk *TerkunciError
	if !errors.As(err, &tk) {
		t.Fatalf("percobaan ke-5: err = %v, mau TerkunciError", err)
	}
	if _, err := e.svc.Login(bg, "admin1", pwUji, "10.0.0.2"); !errors.As(err, &tk) {
		t.Fatalf("password benar saat terkunci harus tetap ditolak, err = %v", err)
	}
	if _, err := e.svc.Login(bg, "admin1", pwUji, "10.0.0.9"); err != nil {
		t.Fatalf("IP lain tidak ikut terkunci: %v", err)
	}
	if _, err := e.svc.Login(bg, "", "", "x"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("isian kosong: err = %v", err)
	}
}

func TestGantiPassword_MencabutSesiLainSaja(t *testing.T) {
	e := baru(t)
	e.akun(t, "ketua1", domain.RoleKetua)
	a, err := e.svc.Login(bg, "ketua1", pwUji, "ip")
	if err != nil {
		t.Fatal(err)
	}
	b, err := e.svc.Login(bg, "ketua1", pwUji, "ip")
	if err != nil {
		t.Fatal(err)
	}

	cekValidasi(t, e.svc.GantiPassword(bg, a.User, a.Token, pwUji, "pendek", "ip"), "password_baru")
	cekValidasi(t, e.svc.GantiPassword(bg, a.User, a.Token, pwUji, pwUji, "ip"), "password_baru")
	cekValidasi(t, e.svc.GantiPassword(bg, a.User, a.Token, "lama-yang-salah", "password-baru-panjang", "ip"), "password_lama")

	if err := e.svc.GantiPassword(bg, a.User, a.Token, pwUji, "password-baru-panjang", "ip"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Sesi(bg, a.Token); err != nil {
		t.Fatalf("sesi yang dipakai harus tetap hidup: %v", err)
	}
	if _, err := e.svc.Sesi(bg, b.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sesi lain harus dicabut, err = %v", err)
	}
	if _, err := e.svc.Login(bg, "ketua1", "password-baru-panjang", "ip"); err != nil {
		t.Fatalf("login dengan password baru: %v", err)
	}
}

func TestBuatUserDanResetPassword(t *testing.T) {
	e := baru(t)
	tests := []struct {
		nama, username, namaLengkap string
		role                        domain.Role
		pw, field                   string
	}{
		{"username tidak sah", "A B", "Nama", domain.RoleAdmin, pwUji, "username"},
		{"nama pendek", "abc", "A", domain.RoleAdmin, pwUji, "nama"},
		{"peran salah", "abc", "Nama", "root", pwUji, "role"},
		{"password pendek", "abc", "Nama", domain.RoleAdmin, "pendek", "password"},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			_, err := e.svc.BuatUser(bg, tt.username, tt.namaLengkap, tt.role, tt.pw)
			cekValidasi(t, err, tt.field)
		})
	}

	e.akun(t, "admin1", domain.RoleAdmin)
	_, err := e.svc.BuatUser(bg, "admin1", "Lain", domain.RoleKetua, pwUji)
	cekValidasi(t, err, "username")

	s, err := e.svc.Login(bg, "admin1", pwUji, "ip")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.ResetPassword(bg, "admin1", "password-reset-panjang"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Sesi(bg, s.Token); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("reset password harus mencabut semua sesi")
	}
	if err := e.svc.ResetPassword(bg, "tidakada", "password-reset-panjang"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("reset username tak dikenal: err = %v", err)
	}
}

func TestDaftarAudit_FilterKategori(t *testing.T) {
	e := baru(t)
	u := e.akun(t, "admin1", domain.RoleAdmin)
	for _, a := range []string{domain.AksiLogin, domain.AksiHitung, domain.AksiSalur} {
		if err := e.svc.Catat(bg, domain.Audit{ActorID: u.ID, Action: a, Resource: "periode"}); err != nil {
			t.Fatal(err)
		}
	}
	h, err := e.svc.DaftarAudit(bg, "hitung", 1)
	if err != nil {
		t.Fatal(err)
	}
	if h.Total != 1 || h.Items[0].Action != domain.AksiHitung || h.Items[0].ActorNama != u.Nama {
		t.Fatalf("kategori hitung = %+v", h)
	}
	if h, _ := e.svc.DaftarAudit(bg, "", 1); h.Total != 3 {
		t.Fatalf("semua = %d, mau 3", h.Total)
	}
	_, err = e.svc.DaftarAudit(bg, "ngawur", 1)
	cekValidasi(t, err, "kategori")
}
