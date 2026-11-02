package httpapi

import (
	"strings"
	"testing"
)

// Sanity: hanya Ketua yang mengelola pengguna; akun baru bisa masuk; nonaktif
// langsung memutus sesi yang sedang berjalan; password tidak masuk log.
func TestPengguna_HanyaKetuaDanNonaktifMemutusSesi(t *testing.T) {
	srv, _, log := siapkanServerGerbang(t, false)
	ketua, admin, anon := klienBaru(t, srv), klienBaru(t, srv), klienBaru(t, srv)
	ketua.login("ketua")
	admin.login("admin")

	baru := map[string]any{"username": "petugas2", "nama": "Petugas Dua", "role": "admin", "password": "rahasia-awal-12345"}
	anon.minta("GET", "/api/pengguna", nil, 401, nil)
	admin.minta("GET", "/api/pengguna", nil, 403, nil)
	admin.minta("POST", "/api/pengguna", baru, 403, nil)

	var u struct {
		ID    string `json:"id"`
		Aktif bool   `json:"aktif"`
	}
	ketua.minta("POST", "/api/pengguna", baru, 201, &u)
	if u.ID == "" || !u.Aktif {
		t.Fatalf("pengguna baru = %+v", u)
	}
	ketua.minta("POST", "/api/pengguna", baru, 422, nil) // username dobel

	// Akun baru bisa masuk. Ketua nonaktifkan: sesi yang sedang berjalan langsung mati.
	petugas := klienBaru(t, srv)
	petugas.minta("POST", "/api/auth/login", map[string]string{"username": "petugas2", "password": "rahasia-awal-12345"}, 200, nil)
	petugas.minta("GET", "/api/auth/me", nil, 200, nil)
	ketua.minta("POST", "/api/pengguna/"+u.ID+"/nonaktif", nil, 200, nil)
	petugas.minta("GET", "/api/auth/me", nil, 401, nil)
	anon.minta("POST", "/api/auth/login", map[string]string{"username": "petugas2", "password": "rahasia-awal-12345"}, 401, nil)

	// Diaktifkan lagi + reset password: password lama tidak berlaku, yang baru berlaku.
	ketua.minta("POST", "/api/pengguna/"+u.ID+"/aktif", nil, 200, nil)
	ketua.minta("POST", "/api/pengguna/"+u.ID+"/reset-password", map[string]string{"password": "pendek"}, 422, nil)
	ketua.minta("POST", "/api/pengguna/"+u.ID+"/reset-password", map[string]string{"password": "password-baru-67890"}, 204, nil)
	anon.minta("POST", "/api/auth/login", map[string]string{"username": "petugas2", "password": "rahasia-awal-12345"}, 401, nil)
	anon.minta("POST", "/api/auth/login", map[string]string{"username": "petugas2", "password": "password-baru-67890"}, 200, nil)

	// Ketua tidak bisa memutus aksesnya sendiri.
	var daftar []struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	ketua.minta("GET", "/api/pengguna", nil, 200, &daftar)
	idKetua := ""
	for _, d := range daftar {
		if d.Username == "ketua" {
			idKetua = d.ID
		}
	}
	if idKetua == "" {
		t.Fatal("akun ketua tidak ada di daftar")
	}
	ketua.minta("POST", "/api/pengguna/"+idKetua+"/nonaktif", nil, 409, nil)
	ketua.minta("PUT", "/api/pengguna/"+idKetua, map[string]any{"nama": "Ketua", "role": "admin"}, 409, nil)

	for _, rahasia := range []string{"rahasia-awal-12345", "password-baru-67890"} {
		if strings.Contains(log.String(), rahasia) {
			t.Errorf("password %q tercatat di log server", rahasia)
		}
	}

	// Audit mencatat aksi pengguna tanpa password.
	var audit struct {
		Items []struct {
			Action string         `json:"action"`
			Detail map[string]any `json:"detail"`
		} `json:"items"`
	}
	ketua.minta("GET", "/api/audit?kategori=pengguna", nil, 200, &audit)
	if len(audit.Items) < 4 {
		t.Fatalf("audit pengguna = %+v", audit.Items)
	}
	for _, it := range audit.Items {
		if _, ada := it.Detail["password"]; ada {
			t.Errorf("audit %s memuat password", it.Action)
		}
	}
}
