package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"spkzakat/internal/crypto"
	"spkzakat/internal/domain"
	"spkzakat/internal/service"
	"spkzakat/internal/store"
	"spkzakat/internal/testdb"
)

// Test ini menjalankan server HTTP sungguhan (httptest) di atas MySQL uji:
// smoke test + alur demo sidang dari login sampai laporan.

const pw = "password-uji-panjang"

func siapkanServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(ctx, testdb.Migrated(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	enc, _ := crypto.NewCipher(bytes.Repeat([]byte{1}, crypto.KeySize))
	idx, _ := crypto.NewHasher(bytes.Repeat([]byte{2}, crypto.KeySize))
	svc := service.New(st, enc, idx)
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))
	srv := httptest.NewServer(New(svc, log, false, st.Ping))
	t.Cleanup(srv.Close)
	for _, u := range []struct {
		username string
		role     domain.Role
	}{{"admin", domain.RoleAdmin}, {"ketua", domain.RoleKetua}} {
		if _, err := svc.BuatUser(ctx, u.username, "Nama "+u.username, u.role, pw); err != nil {
			t.Fatal(err)
		}
	}
	return srv
}

// klien = pengguna dengan cookie jar sendiri.
type klien struct {
	t   *testing.T
	srv *httptest.Server
	c   *http.Client
}

func klienBaru(t *testing.T, srv *httptest.Server) *klien {
	jar, _ := cookiejar.New(nil)
	return &klien{t: t, srv: srv, c: &http.Client{Jar: jar}}
}

// minta mengirim JSON dan mendekode respons ke out (boleh nil); status dicek.
func (k *klien) minta(method, path string, body any, mauStatus int, out any) *http.Response {
	k.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, k.srv.URL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	res, err := k.c.Do(req)
	if err != nil {
		k.t.Fatal(err)
	}
	defer res.Body.Close()
	isi, _ := io.ReadAll(res.Body)
	if res.StatusCode != mauStatus {
		k.t.Fatalf("%s %s = %d, mau %d; body: %s", method, path, res.StatusCode, mauStatus, isi)
	}
	if out != nil {
		if err := json.Unmarshal(isi, out); err != nil {
			k.t.Fatalf("%s %s: dekode %q: %v", method, path, isi, err)
		}
	}
	return res
}

func (k *klien) login(username string) {
	k.t.Helper()
	k.minta("POST", "/api/auth/login", map[string]string{"username": username, "password": pw}, 200, nil)
}

func TestSmoke_HealthzDanRequestID(t *testing.T) {
	srv := siapkanServer(t)
	k := klienBaru(t, srv)
	var h map[string]bool
	res := k.minta("GET", "/healthz", nil, 200, &h)
	if !h["ok"] || res.Header.Get("X-Request-Id") == "" {
		t.Fatalf("healthz = %v, X-Request-Id = %q", h, res.Header.Get("X-Request-Id"))
	}

	req, _ := http.NewRequest("GET", srv.URL+"/api/auth/me", nil)
	req.Header.Set("X-Request-Id", "dari-klien-123")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var e map[string]any
	_ = json.NewDecoder(res.Body).Decode(&e)
	if res.StatusCode != 401 || res.Header.Get("X-Request-Id") != "dari-klien-123" || e["request_id"] != "dari-klien-123" {
		t.Fatalf("tanpa sesi: %d, header %q, body %v", res.StatusCode, res.Header.Get("X-Request-Id"), e)
	}
}

func TestAuth_CookieGagalTerkunciDanPeran(t *testing.T) {
	srv := siapkanServer(t)
	k := klienBaru(t, srv)

	var e map[string]any
	k.minta("POST", "/api/auth/login", map[string]string{"username": "admin", "password": "salah-salah-salah"}, 401, &e)
	if e["sisa_percobaan"] != float64(4) {
		t.Fatalf("login gagal = %v", e)
	}
	k.minta("POST", "/api/auth/login", map[string]string{"username": "admin", "pw": "x"}, 400, nil) // field tak dikenal

	res := k.minta("POST", "/api/auth/login", map[string]string{"username": "admin", "password": pw}, 200, nil)
	ck := res.Header.Get("Set-Cookie")
	if !strings.Contains(ck, "HttpOnly") || !strings.Contains(ck, "SameSite=Strict") {
		t.Fatalf("cookie tidak aman: %s", ck)
	}
	var me domain.User
	k.minta("GET", "/api/auth/me", nil, 200, &me)
	if me.Username != "admin" || me.Role != domain.RoleAdmin {
		t.Fatalf("me = %+v", me)
	}

	ketua := klienBaru(t, srv)
	ketua.login("ketua")
	ketua.minta("GET", "/api/mustahik", nil, 403, nil)                                              // Ketua tidak mengelola mustahik
	ketua.minta("PUT", "/api/kriteria", map[string]any{"items": []any{}}, 403, nil)                 // Kriteria hanya-baca
	ketua.minta("GET", "/api/kriteria", nil, 200, nil)                                              // ...tapi boleh lihat
	k.minta("POST", "/api/periode/x/keputusan", map[string]string{"keputusan": "setuju"}, 403, nil) // keputusan hanya Ketua

	k.minta("POST", "/api/auth/logout", nil, 204, nil)
	k.minta("GET", "/api/auth/me", nil, 401, nil)

	penyerang := klienBaru(t, srv)
	for i := 0; i < 4; i++ {
		penyerang.minta("POST", "/api/auth/login", map[string]string{"username": "ketua", "password": fmt.Sprintf("tebak-tebakan-%d", i)}, 401, nil)
	}
	res = penyerang.minta("POST", "/api/auth/login", map[string]string{"username": "ketua", "password": "tebak-lagi-lagi"}, 429, &e)
	if res.Header.Get("Retry-After") == "" || e["retry_after"] == nil {
		t.Fatalf("terkunci tanpa Retry-After: %v", e)
	}
}

// TestBuatPeriode_JumlahPenerimaLewatHTTP: pengurus boleh mengisi jumlah penerima (bukan nominal);
// nominal diturunkan dari dana. Mengisi keduanya sekaligus ditolak 422 dengan pesan di kolom jumlah_penerima.
func TestBuatPeriode_JumlahPenerimaLewatHTTP(t *testing.T) {
	srv := siapkanServer(t)
	admin := klienBaru(t, srv)
	admin.login("admin")
	dasar := func(tambah map[string]any) map[string]any {
		b := map[string]any{"nama": "Zakat Mal", "jenis": "zakat", "dana_tersedia": 3000000, "asnaf_diizinkan": []string{"fakir"}}
		for k, v := range tambah {
			b[k] = v
		}
		return b
	}

	var p struct {
		Nominal int64 `json:"nominal_per_penerima"`
	}
	admin.minta("POST", "/api/periode", dasar(map[string]any{"jumlah_penerima": 3}), 201, &p)
	if p.Nominal != 1000000 {
		t.Fatalf("nominal_per_penerima = %d, mau 1000000 (3 jt ÷ 3 orang)", p.Nominal)
	}

	var e struct {
		Fields map[string]string `json:"fields"`
	}
	admin.minta("POST", "/api/periode", dasar(map[string]any{"jumlah_penerima": 3, "nominal_per_penerima": 1000000}), 422, &e)
	if e.Fields["jumlah_penerima"] == "" {
		t.Errorf("kolom jumlah_penerima tidak dilaporkan saat nominal dan jumlah diisi bersamaan: %v", e)
	}
}

// TestAlurDemo = skenario sidang: input data → ranking ARAS → persetujuan →
// penyaluran → laporan, semua lewat HTTP. Angka K dicocokkan dengan docs/02.
func TestAlurDemo_DariMustahikSampaiLaporan(t *testing.T) {
	srv := siapkanServer(t)
	admin, ketua := klienBaru(t, srv), klienBaru(t, srv)
	admin.login("admin")
	ketua.login("ketua")

	data := []struct {
		nama  string
		asnaf string
		nilai map[string]float64
	}{
		{"Mustahik 1", "miskin", map[string]float64{"C1": 800000, "C2": 5, "C3": 3, "C4": 2, "C5": 3}},
		{"Mustahik 2", "fakir", map[string]float64{"C1": 500000, "C2": 4, "C3": 4, "C4": 3, "C5": 4}},
		{"Mustahik 3", "miskin", map[string]float64{"C1": 1500000, "C2": 3, "C3": 2, "C4": 1, "C5": 2}},
		{"Mustahik 4", "fakir", map[string]float64{"C1": 600000, "C2": 2, "C3": 3, "C4": 4, "C5": 3}},
		{"Mustahik 5", "miskin", map[string]float64{"C1": 1000000, "C2": 6, "C3": 2, "C4": 2, "C5": 1}},
	}
	var ids []string
	for i, d := range data {
		var m struct {
			ID  string `json:"id"`
			NIK string `json:"nik_tersamar"`
		}
		admin.minta("POST", "/api/mustahik", map[string]any{
			"nama": d.nama, "nik": fmt.Sprintf("327101234567890%d", i+1), "alamat": "Jl. Melati No. 1", "asnaf": d.asnaf,
		}, 201, &m)
		if !strings.HasPrefix(m.NIK, "•") {
			t.Fatalf("NIK tidak tersamar di respons: %q", m.NIK)
		}
		ids = append(ids, m.ID)
	}

	var p struct {
		ID string `json:"id"`
	}
	admin.minta("POST", "/api/periode", map[string]any{
		"nama": "Zakat Fitrah 1448 H", "jenis": "zakat", "dana_tersedia": 3000000,
		"nominal_per_penerima": 1000000, "asnaf_diizinkan": []string{"fakir", "miskin"},
	}, 201, &p)
	base := "/api/periode/" + p.ID
	admin.minta("PUT", base+"/kandidat", map[string]any{"mustahik_ids": ids}, 200, nil)

	var items []map[string]any
	for i, d := range data {
		items = append(items, map[string]any{"mustahik_id": ids[i], "nilai": d.nilai})
	}
	var grid struct {
		Lengkap int `json:"lengkap"`
		Total   int `json:"total"`
	}
	admin.minta("PUT", base+"/penilaian", map[string]any{"items": items}, 200, &grid)
	if grid.Lengkap != 5 || grid.Total != 5 {
		t.Fatalf("grid = %+v", grid)
	}

	var hasil struct {
		Kuota int `json:"kuota"`
		Items []struct {
			Mustahik struct {
				Nama string `json:"nama"`
			} `json:"mustahik"`
			K      float64 `json:"k"`
			Status string  `json:"status"`
		} `json:"items"`
		Langkah *struct {
			Normalisasi [][]float64 `json:"normalisasi"`
		} `json:"langkah"`
	}
	admin.minta("POST", base+"/hitung", nil, 200, &hasil)
	mauNama := []string{"Mustahik 2", "Mustahik 4", "Mustahik 1", "Mustahik 5", "Mustahik 3"}
	mauK := []float64{0.8939, 0.7327, 0.6888, 0.5611, 0.4099}
	for i, it := range hasil.Items {
		if it.Mustahik.Nama != mauNama[i] || it.K < mauK[i]-1e-4 || it.K > mauK[i]+1e-4 {
			t.Errorf("ranking %d = %s K %.4f, mau %s K %.4f", i+1, it.Mustahik.Nama, it.K, mauNama[i], mauK[i])
		}
	}
	if hasil.Kuota != 3 || hasil.Items[2].Status != "usulan" || hasil.Items[3].Status != "cadangan" || hasil.Langkah == nil {
		t.Fatalf("kuota/status/langkah salah: %+v", hasil)
	}

	ketua.minta("GET", base+"/hasil", nil, 200, nil)
	var e map[string]any
	ketua.minta("POST", base+"/keputusan", map[string]string{"keputusan": "tolak"}, 422, &e) // catatan wajib
	ketua.minta("POST", base+"/keputusan", map[string]string{"keputusan": "setuju", "catatan": "Bismillah"}, 200, nil)
	ketua.minta("POST", base+"/keputusan", map[string]string{"keputusan": "setuju"}, 409, &e)
	if !strings.Contains(e["error"].(string), "disetujui") {
		t.Fatalf("pesan konflik harus menjelaskan status: %v", e)
	}

	salur := func(id string, nominal int64, mau int) {
		t.Helper()
		admin.minta("POST", base+"/penyaluran", map[string]any{"mustahik_id": id, "nominal": nominal, "tanggal": "2026-10-12"}, mau, nil)
	}
	salur(ids[2], 1000000, 422) // Mustahik 3 = cadangan
	admin.minta("POST", base+"/tutup", nil, 422, nil)
	salur(ids[1], 1000000, 201)
	salur(ids[1], 1000000, 409)
	salur(ids[3], 1000000, 201)
	salur(ids[0], 1000000, 201)
	admin.minta("POST", base+"/tutup", nil, 200, nil)

	var d struct {
		Periode struct {
			Status string `json:"status"`
		} `json:"periode"`
	}
	ketua.minta("GET", base, nil, 200, &d)
	if d.Periode.Status != "selesai" {
		t.Fatalf("status akhir = %s", d.Periode.Status)
	}

	req, _ := http.NewRequest("GET", srv.URL+base+"/laporan?format=csv", nil)
	res, err := ketua.c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	csv, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/csv") ||
		!bytes.Contains(csv, []byte("Total,,3000000")) || !bytes.HasPrefix(csv, []byte("\xEF\xBB\xBF")) {
		t.Fatalf("laporan CSV = %d %s\n%s", res.StatusCode, res.Header.Get("Content-Type"), csv)
	}

	var aud struct {
		Total int `json:"total"`
		Items []struct {
			Action    string `json:"action"`
			ActorNama string `json:"actor_nama"`
		} `json:"items"`
	}
	ketua.minta("GET", "/api/audit?kategori=keputusan", nil, 200, &aud)
	if aud.Total != 1 || aud.Items[0].Action != "SETUJU" || aud.Items[0].ActorNama != "Nama ketua" {
		t.Fatalf("audit keputusan = %+v", aud)
	}
	ketua.minta("GET", "/api/audit?kategori=penyaluran", nil, 200, &aud)
	if aud.Total != 4 { // 3 SALUR + 1 TUTUP
		t.Fatalf("audit penyaluran = %d, mau 4", aud.Total)
	}

	var dash struct {
		MustahikAktif int `json:"mustahik_aktif"`
		Aktif         any `json:"aktif"`
	}
	admin.minta("GET", "/api/dashboard", nil, 200, &dash)
	if dash.MustahikAktif != 5 || dash.Aktif != nil {
		t.Fatalf("dashboard = %+v", dash)
	}
}
