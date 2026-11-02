package service

import (
	"errors"
	"regexp"
	"sync"
	"testing"

	"spkzakat/internal/domain"
)

var reKodeSetup = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestSetup_HanyaAktifSelamaTidakAdaAkun(t *testing.T) {
	e := baru(t) // basis data baru: tanpa akun

	if aktif, _ := e.svc.SetupAktif(bg); aktif {
		t.Error("setup aktif sebelum SiapkanSetup dipanggil (belum ada kode)")
	}
	kode, err := e.svc.SiapkanSetup(bg)
	if err != nil || !reKodeSetup.MatchString(kode) {
		t.Fatalf("kode = %q, err %v; mau 128 bit heksadesimal", kode, err)
	}
	if aktif, err := e.svc.SetupAktif(bg); err != nil || !aktif {
		t.Fatalf("SetupAktif = %v, %v; mau true", aktif, err)
	}

	// Akun yang dibuat lewat jalur lain (CLI useradmin) saat server berjalan menutup setup.
	e.akun(t, "dibuatcli", domain.RoleKetua)
	if aktif, _ := e.svc.SetupAktif(bg); aktif {
		t.Error("setup masih aktif padahal sudah ada akun")
	}
	if _, err := e.svc.BuatAkunPertama(bg, "1.1.1.1", kode, "ketua2", "Ketua Dua", pwUji); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("setup setelah ada akun: err = %v, mau not found", err)
	}

	// Server start ulang pada basis data yang sudah berisi akun: tidak ada kode baru.
	if k, err := e.svc.SiapkanSetup(bg); err != nil || k != "" {
		t.Errorf("SiapkanSetup dengan akun ada = %q, %v; mau kosong", k, err)
	}
}

func TestBuatAkunPertama_Alur(t *testing.T) {
	e := baru(t)
	kode, _ := e.svc.SiapkanSetup(bg)

	// Kode salah: ditolak dengan galat field "token", tidak ada akun terbentuk.
	var ve *domain.ValidationError
	if _, err := e.svc.BuatAkunPertama(bg, "2.2.2.2", "salah", "ketua", "Ketua", pwUji); !errors.As(err, &ve) || ve.Fields["token"] == "" {
		t.Fatalf("kode salah: err = %v", err)
	}
	if n, _ := e.st.CountUsers(bg); n != 0 {
		t.Fatalf("akun terbentuk dengan kode salah: %d", n)
	}

	// Kode benar tetapi isian tidak sah: ditolak, kode TIDAK hangus.
	for _, tt := range []struct{ nama, username, nama2, pw, field string }{
		{"username pendek", "ab", "Ketua", pwUji, "username"},
		{"nama kosong", "ketua", " ", pwUji, "nama"},
		{"password pendek", "ketua", "Ketua", "pendek", "password"},
	} {
		if _, err := e.svc.BuatAkunPertama(bg, "2.2.2.2", kode, tt.username, tt.nama2, tt.pw); fieldGalat(err) != tt.field {
			t.Errorf("%s: err = %v, mau galat field %q", tt.nama, err, tt.field)
		}
	}
	if aktif, _ := e.svc.SetupAktif(bg); !aktif {
		t.Fatal("setup tertutup oleh isian yang tidak sah")
	}

	// Berhasil: akun pertama selalu Ketua, aktif, bisa langsung masuk.
	v, err := e.svc.BuatAkunPertama(bg, "2.2.2.2", " "+kode+" ", "Ketua.Baru", "H. Ketua Baru", pwUji)
	if err != nil || v.Role != domain.RoleKetua || !v.Aktif || v.Username != "ketua.baru" {
		t.Fatalf("buat akun pertama: %+v, %v", v, err)
	}
	if _, err := e.svc.Login(bg, "ketua.baru", pwUji, "2.2.2.2"); err != nil {
		t.Errorf("login akun pertama: %v", err)
	}

	// Setup langsung mati; kode lama tidak berlaku lagi, bahkan untuk username lain.
	if aktif, _ := e.svc.SetupAktif(bg); aktif {
		t.Error("setup masih aktif setelah akun pertama dibuat")
	}
	if _, err := e.svc.BuatAkunPertama(bg, "3.3.3.3", kode, "penyusup", "Penyusup", pwUji); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("pakai ulang kode: err = %v, mau not found", err)
	}

	// Audit mencatat SETUP_AWAL tanpa kode maupun password.
	a, _ := e.svc.DaftarAudit(bg, "pengguna", 1)
	ada := false
	for _, it := range a.Items {
		if it.Action == domain.AksiSetupAwal {
			ada = true
			if _, bocor := it.Detail["token"]; bocor {
				t.Error("audit SETUP_AWAL memuat kode")
			}
		}
	}
	if !ada {
		t.Error("audit SETUP_AWAL tidak tercatat")
	}
}

// Menebak kode dibatasi seperti login: 5 salah dari satu IP mengunci 15 menit,
// termasuk untuk kode yang benar sesudahnya.
func TestBuatAkunPertama_PenebakKodeDikunci(t *testing.T) {
	e := baru(t)
	kode, _ := e.svc.SiapkanSetup(bg)

	var kunci *TerkunciError
	for i := 1; i <= MaksGagal; i++ {
		_, err := e.svc.BuatAkunPertama(bg, "9.9.9.9", "tebakan", "ketua", "Ketua", pwUji)
		if i < MaksGagal && fieldGalat(err) != "token" {
			t.Fatalf("percobaan %d: err = %v, mau galat token", i, err)
		}
		if i == MaksGagal && !errors.As(err, &kunci) {
			t.Fatalf("percobaan %d: err = %v, mau TerkunciError", i, err)
		}
	}
	if _, err := e.svc.BuatAkunPertama(bg, "9.9.9.9", kode, "ketua", "Ketua", pwUji); !errors.As(err, &kunci) {
		t.Errorf("kode benar saat terkunci: err = %v, mau TerkunciError", err)
	}
	// IP lain tidak ikut terkunci.
	if _, err := e.svc.BuatAkunPertama(bg, "8.8.8.8", kode, "ketua", "Ketua", pwUji); err != nil {
		t.Errorf("IP lain: %v", err)
	}
}

// Dua permintaan bersamaan dengan kode yang benar: hanya satu akun yang boleh
// terbentuk. Tanpa penyerialan keduanya lolos dan ada dua Ketua tak terduga.
func TestBuatAkunPertama_BersamaanHanyaSatu(t *testing.T) {
	e := baru(t)
	kode, _ := e.svc.SiapkanSetup(bg)

	var wg sync.WaitGroup
	hasil := make([]error, 4)
	for i := range hasil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, hasil[i] = e.svc.BuatAkunPertama(bg, "7.7.7.7", kode, "ketua"+string(rune('a'+i)), "Ketua", pwUji)
		}()
	}
	wg.Wait()

	berhasil := 0
	for _, err := range hasil {
		if err == nil {
			berhasil++
		} else if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("galat tak terduga: %v", err)
		}
	}
	if n, _ := e.st.CountUsers(bg); berhasil != 1 || n != 1 {
		t.Fatalf("berhasil = %d, akun = %d; mau tepat 1 dan 1", berhasil, n)
	}
}
