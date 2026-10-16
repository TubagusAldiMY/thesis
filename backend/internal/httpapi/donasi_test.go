package httpapi

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"spkzakat/internal/crypto"
	"spkzakat/internal/domain"
	"spkzakat/internal/payment"
	"spkzakat/internal/service"
	"spkzakat/internal/store"
	"spkzakat/internal/testdb"
)

// bufAman = penampung log yang aman dipakai banyak goroutine (handler HTTP).
type bufAman struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *bufAman) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *bufAman) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// siapkanServerGerbang = siapkanServer dengan penyedia tiruan terpasang.
// Log server ditulis ke log (untuk memeriksa tidak ada rahasia yang tercatat).
func siapkanServerGerbang(t *testing.T, gerbang bool) (*httptest.Server, *payment.Tiruan, *bufAman) {
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
	log := &bufAman{}
	var opsi []Opsi
	ti := payment.NewTiruan()
	if gerbang {
		svc.AturGerbang(ti, "https://masjid.example")
		opsi = append(opsi, DenganSimulasiBayar(ti))
	}
	srv := httptest.NewServer(New(svc, slog.New(slog.NewJSONHandler(log, &slog.HandlerOptions{Level: slog.LevelDebug})), false, st.Ping, opsi...))
	t.Cleanup(srv.Close)
	for _, u := range []struct {
		username string
		role     domain.Role
	}{{"admin", domain.RoleAdmin}, {"ketua", domain.RoleKetua}} {
		if _, err := svc.BuatUser(ctx, u.username, "Nama "+u.username, u.role, pw); err != nil {
			t.Fatal(err)
		}
	}
	return srv, ti, log
}

var donasiWeb = map[string]any{"nama": "Hamba Allah", "no_hp": "081234567890", "jenis": "zakat_fitrah", "jumlah": 50000, "keterangan": "semoga berkah"}

// kirimNotifikasi mengirim muatan mentah ke webhook (tanpa sesi), seperti Midtrans.
func kirimNotifikasi(t *testing.T, srv *httptest.Server, isi []byte) (int, string) {
	t.Helper()
	res, err := http.Post(srv.URL+"/api/publik/midtrans/notifikasi", "application/json", bytes.NewReader(isi))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

// Alur utama (sanity): muzaki tanpa login berdonasi → gateway memberi tahu →
// pengurus melihat notifikasi dan kuitansi, dengan hak akses tiap peran benar.
func TestDonasi_AlurPublikSampaiPengurus(t *testing.T) {
	srv, ti, _ := siapkanServerGerbang(t, true)
	publik := klienBaru(t, srv) // tanpa login

	var dibuat struct {
		Kode     string `json:"kode"`
		Status   string `json:"status"`
		BayarURL string `json:"bayar_url"`
		ID       string `json:"id"`
	}
	publik.minta("POST", "/api/publik/donasi", donasiWeb, 201, &dibuat)
	if len(dibuat.Kode) != 32 || dibuat.Status != "menunggu" || !strings.Contains(dibuat.BayarURL, "/donasi/tiruan?kode="+dibuat.Kode) {
		t.Fatalf("donasi dibuat = %+v", dibuat)
	}
	if dibuat.ID != "" {
		t.Error("ID internal donasi bocor ke JSON publik")
	}

	// Gateway memberi tahu pembayaran berhasil. order_id = ID donasi (dari DB lewat admin).
	admin := klienBaru(t, srv)
	admin.login("admin")
	var daftar struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	admin.minta("GET", "/api/donasi", nil, 200, &daftar)
	if len(daftar.Items) != 1 || daftar.Items[0].Status != "menunggu" {
		t.Fatalf("daftar awal = %+v", daftar)
	}
	id := daftar.Items[0].ID
	if code, body := kirimNotifikasi(t, srv, ti.Muatan(id, 50000, "settlement")); code != 200 {
		t.Fatalf("webhook = %d %s", code, body)
	}

	var p struct {
		Status        string `json:"status"`
		NomorKuitansi string `json:"nomor_kuitansi"`
		BayarURL      string `json:"bayar_url"`
		Nama          string `json:"nama"`
	}
	publik.minta("GET", "/api/publik/donasi/"+dibuat.Kode, nil, 200, &p)
	if p.Status != "diterima" || !strings.HasPrefix(p.NomorKuitansi, "KWT/") || p.BayarURL != "" || p.Nama != "Hamba Allah" {
		t.Fatalf("publik setelah bayar = %+v", p)
	}

	// Notifikasi untuk pengurus: penanda baru = 1, lalu hilang setelah ditandai.
	var r struct {
		Baru          int   `json:"baru"`
		TotalDiterima int64 `json:"total_diterima"`
	}
	admin.minta("GET", "/api/donasi/ringkas", nil, 200, &r)
	if r.Baru != 1 || r.TotalDiterima != 50000 {
		t.Fatalf("ringkas = %+v", r)
	}

	// Hak akses: Admin melihat No. HP penuh, Ketua hanya tersamar.
	var det struct {
		NoHP         string `json:"no_hp"`
		NoHPTersamar string `json:"no_hp_tersamar"`
	}
	admin.minta("GET", "/api/donasi/"+id, nil, 200, &det)
	if det.NoHP != "081234567890" {
		t.Errorf("admin: no_hp = %q", det.NoHP)
	}
	ketua := klienBaru(t, srv)
	ketua.login("ketua")
	det.NoHP, det.NoHPTersamar = "", ""
	ketua.minta("GET", "/api/donasi/"+id, nil, 200, &det)
	if det.NoHP != "" || det.NoHPTersamar == "" {
		t.Errorf("ketua: %+v (hanya boleh tersamar)", det)
	}
	ketua.minta("GET", "/api/donasi/ringkas", nil, 200, &r)
	if r.Baru != 1 {
		t.Errorf("ketua juga melihat notifikasi: %+v", r)
	}
	ketua.minta("POST", "/api/donasi/dibaca", nil, 200, nil)
	admin.minta("GET", "/api/donasi/ringkas", nil, 200, &r)
	if r.Baru != 0 {
		t.Errorf("setelah ditandai dibaca = %+v", r)
	}

	// Audit mencatat penerimaan (tanpa pelaku, karena dari gateway) dan tanpa data pribadi.
	var au struct {
		Items []struct {
			Action string         `json:"action"`
			Detail map[string]any `json:"detail"`
		} `json:"items"`
	}
	admin.minta("GET", "/api/audit?kategori=donasi", nil, 200, &au)
	if len(au.Items) != 1 || au.Items[0].Action != "DONASI_TERIMA" {
		t.Fatalf("audit = %+v", au)
	}
	for k := range au.Items[0].Detail {
		if strings.Contains(k, "nama") || strings.Contains(k, "hp") {
			t.Errorf("audit memuat data pribadi: %v", au.Items[0].Detail)
		}
	}
}

// Keamanan (jalur kritis): webhook tanpa tanda tangan sah tidak boleh mengubah apa pun.
func TestDonasi_WebhookDitolak(t *testing.T) {
	srv, ti, _ := siapkanServerGerbang(t, true)
	publik := klienBaru(t, srv)
	var dibuat struct{ Kode string }
	publik.minta("POST", "/api/publik/donasi", donasiWeb, 201, &dibuat)
	admin := klienBaru(t, srv)
	admin.login("admin")
	var daftar struct {
		Items []struct{ ID string } `json:"items"`
	}
	admin.minta("GET", "/api/donasi", nil, 200, &daftar)
	id := daftar.Items[0].ID

	// Muatan bertanda tangan sah, lalu jumlah diubah penyerang.
	palsu := bytes.Replace(ti.Muatan(id, 50000, "settlement"), []byte("50000.00"), []byte("1.00"), 1)
	tests := []struct {
		nama string
		isi  []byte
		mau  int
	}{
		{"bukan JSON", []byte("halo"), 401},
		{"kosong", nil, 401},
		{"tanpa tanda tangan", []byte(`{"order_id":"` + id + `","status_code":"200","gross_amount":"50000.00","transaction_status":"settlement"}`), 401},
		{"jumlah diubah setelah ditandatangani", palsu, 401},
		{"terlalu besar", bytes.Repeat([]byte("a"), 70<<10), 400},
		{"jumlah beda tapi bertanda tangan sah", ti.Muatan(id, 1000, "settlement"), 409},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			if code, body := kirimNotifikasi(t, srv, tt.isi); code != tt.mau {
				t.Errorf("status = %d, mau %d; body %s", code, tt.mau, body)
			}
		})
	}
	var p struct{ Status string }
	publik.minta("GET", "/api/publik/donasi/"+dibuat.Kode, nil, 200, &p)
	if p.Status != "menunggu" {
		t.Errorf("donasi berubah menjadi %q oleh notifikasi yang ditolak", p.Status)
	}
}

// Tanpa gateway: halaman donasi web mati (503), pencatatan petugas tetap jalan.
func TestDonasi_TanpaGerbang(t *testing.T) {
	srv, ti, _ := siapkanServerGerbang(t, false)
	publik := klienBaru(t, srv)
	publik.minta("POST", "/api/publik/donasi", donasiWeb, 503, nil)
	if code, _ := kirimNotifikasi(t, srv, ti.Muatan("x", 10000, "settlement")); code != 503 {
		t.Errorf("webhook tanpa gateway = %d, mau 503", code)
	}
	// Endpoint simulasi tidak boleh ada tanpa penyedia tiruan.
	publik.minta("POST", "/api/publik/tiruan/"+strings.Repeat("a", 32)+"/bayar", nil, 404, nil)

	admin := klienBaru(t, srv)
	admin.login("admin")
	var v struct {
		NomorKuitansi string `json:"nomor_kuitansi"`
		Status        string `json:"status"`
	}
	admin.minta("POST", "/api/donasi", map[string]any{"nama": "Pak Haji", "jenis": "infak", "jumlah": 200000, "metode": "tunai"}, 201, &v)
	if v.Status != "diterima" || !strings.HasPrefix(v.NomorKuitansi, "KWT/") {
		t.Errorf("catat petugas = %+v", v)
	}
}

// Deny-by-default: semua rute pengurus butuh sesi; Ketua hanya boleh melihat.
func TestDonasi_HakAkses(t *testing.T) {
	srv, _, _ := siapkanServerGerbang(t, true)
	anon := klienBaru(t, srv)
	for _, r := range [][2]string{
		{"GET", "/api/donasi"}, {"POST", "/api/donasi"}, {"GET", "/api/donasi/ringkas"},
		{"POST", "/api/donasi/dibaca"}, {"GET", "/api/donasi/x"}, {"POST", "/api/donasi/x/batal"},
	} {
		anon.minta(r[0], r[1], nil, 401, nil)
	}

	admin := klienBaru(t, srv)
	admin.login("admin")
	var v struct {
		ID string `json:"id"`
	}
	admin.minta("POST", "/api/donasi", map[string]any{"nama": "Pak Haji", "jenis": "infak", "jumlah": 200000, "metode": "tunai"}, 201, &v)

	ketua := klienBaru(t, srv)
	ketua.login("ketua")
	ketua.minta("POST", "/api/donasi", map[string]any{"nama": "X Y", "jenis": "infak", "jumlah": 200000, "metode": "tunai"}, 403, nil)
	ketua.minta("POST", "/api/donasi/"+v.ID+"/batal", map[string]string{"alasan": "coba coba"}, 403, nil)
	ketua.minta("GET", "/api/donasi", nil, 200, nil)

	var b struct {
		Status string `json:"status"`
	}
	admin.minta("POST", "/api/donasi/"+v.ID+"/batal", map[string]string{"alasan": ""}, 422, nil)
	admin.minta("POST", "/api/donasi/"+v.ID+"/batal", map[string]string{"alasan": "salah ketik"}, 200, &b)
	if b.Status != "dibatalkan" {
		t.Errorf("status = %q", b.Status)
	}
	admin.minta("POST", "/api/donasi/"+v.ID+"/batal", map[string]string{"alasan": "dua kali"}, 409, nil)
}

// Landing publik menampilkan angka transparansi tanpa login. Yang dijaga: hanya
// enam angka agregat, dan nama/No. HP donatur tidak pernah ikut keluar.
func TestRingkasanPublik_TanpaLoginTanpaDataPribadi(t *testing.T) {
	srv, _, _ := siapkanServerGerbang(t, false)
	anon := klienBaru(t, srv)
	kunci := []string{"infak_diterima", "jumlah_donasi", "penerima_manfaat", "total_diterima", "total_tersalur", "zakat_diterima"}

	// periksa memastikan bentuknya tepat enam kunci dan SEMUA nilainya angka:
	// dengan begitu nama atau No. HP (string) mustahil ikut keluar.
	periksa := func(r map[string]any) {
		t.Helper()
		if len(r) != len(kunci) {
			t.Fatalf("kunci = %v, mau tepat %v", r, kunci)
		}
		for _, k := range kunci {
			if _, angka := r[k].(float64); !angka {
				t.Errorf("%s = %#v, mau angka", k, r[k])
			}
		}
	}

	var r map[string]any
	anon.minta("GET", "/api/publik/ringkasan", nil, 200, &r)
	periksa(r)
	for _, k := range kunci {
		if r[k] != float64(0) {
			t.Errorf("%s = %v, mau 0 pada database kosong", k, r[k])
		}
	}

	admin := klienBaru(t, srv)
	admin.login("admin")
	admin.minta("POST", "/api/donasi", map[string]any{
		"nama": "Pak Haji Rahasia", "no_hp": "081234567890", "jenis": "zakat_mal", "jumlah": 200000, "metode": "tunai",
	}, 201, nil)

	r = nil
	anon.minta("GET", "/api/publik/ringkasan", nil, 200, &r)
	periksa(r)
	if r["zakat_diterima"] != float64(200000) || r["total_diterima"] != float64(200000) || r["jumlah_donasi"] != float64(1) {
		t.Errorf("setelah satu donasi = %v", r)
	}
}

func TestDonasi_Aktif(t *testing.T) {
	for _, gerbang := range []bool{true, false} {
		srv, _, _ := siapkanServerGerbang(t, gerbang)
		var a map[string]bool
		klienBaru(t, srv).minta("GET", "/api/publik/donasi-aktif", nil, 200, &a)
		if a["aktif"] != gerbang {
			t.Errorf("gerbang=%v: aktif = %v", gerbang, a["aktif"])
		}
	}
}

func TestDonasi_Validasi(t *testing.T) {
	srv, _, _ := siapkanServerGerbang(t, true)
	publik := klienBaru(t, srv)
	var e struct {
		Fields map[string]string `json:"fields"`
	}
	publik.minta("POST", "/api/publik/donasi", map[string]any{"nama": "A", "jenis": "x", "jumlah": 5}, 422, &e)
	for _, f := range []string{"nama", "jenis", "jumlah"} {
		if e.Fields[f] == "" {
			t.Errorf("tidak ada pesan untuk field %q: %v", f, e.Fields)
		}
	}
	publik.minta("POST", "/api/publik/donasi", map[string]any{"nama": "Budi", "jenis": "infak", "jumlah": 50000, "lain": 1}, 400, nil) // field tak dikenal
	publik.minta("GET", "/api/publik/donasi/tidak-valid", nil, 404, nil)
	publik.minta("GET", "/api/publik/donasi/"+strings.Repeat("0", 32), nil, 404, nil)
}

// Pembatas laju: pembuatan donasi publik dibatasi per IP.
func TestDonasi_PembatasLaju(t *testing.T) {
	srv, _, _ := siapkanServerGerbang(t, true)
	publik := klienBaru(t, srv)
	for i := 0; i < 5; i++ {
		publik.minta("POST", "/api/publik/donasi", donasiWeb, 201, nil)
	}
	res := publik.minta("POST", "/api/publik/donasi", donasiWeb, 429, nil)
	if res.Header.Get("Retry-After") == "" {
		t.Error("429 tanpa Retry-After")
	}
	// IP lain tidak ikut terkunci (kunci = IP dari X-Real-IP).
	req, _ := http.NewRequest("POST", srv.URL+"/api/publik/donasi", bytes.NewReader([]byte(`{"nama":"Siti","jenis":"infak","jumlah":20000}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "203.0.113.9")
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != 201 {
		t.Errorf("IP lain = %d, mau 201", r2.StatusCode)
	}
}

// Privasi (regresi): kode rahasia tautan kuitansi tidak boleh masuk log akses.
func TestDonasi_KodeRahasiaTidakDiLog(t *testing.T) {
	srv, _, log := siapkanServerGerbang(t, true)
	publik := klienBaru(t, srv)
	var d struct{ Kode string }
	publik.minta("POST", "/api/publik/donasi", donasiWeb, 201, &d)
	publik.minta("GET", "/api/publik/donasi/"+d.Kode, nil, 200, nil)
	publik.minta("POST", "/api/publik/tiruan/"+d.Kode+"/bayar", nil, 200, nil)
	publik.minta("GET", "/api/publik/donasi/"+strings.Repeat("f", 32), nil, 404, nil)

	keluaran := log.String()
	if strings.Contains(keluaran, d.Kode) {
		t.Fatalf("kode rahasia donasi muncul di log:\n%s", keluaran)
	}
	if !strings.Contains(keluaran, "/api/publik/donasi/{kode}") {
		t.Errorf("log akses tidak memakai path tersamar:\n%s", keluaran)
	}
}

func TestPathLog(t *testing.T) {
	tests := map[string]string{
		"/api/publik/donasi/abc123":       "/api/publik/donasi/{kode}",
		"/api/publik/tiruan/abc123/bayar": "/api/publik/tiruan/{kode}",
		"/api/publik/donasi":              "/api/publik/donasi",
		"/api/mustahik/11111111":          "/api/mustahik/11111111",
		"/healthz":                        "/healthz",
	}
	for in, mau := range tests {
		if got := pathLog(in); got != mau {
			t.Errorf("pathLog(%q) = %q, mau %q", in, got, mau)
		}
	}
}

// Simulasi pembayaran (hanya penyedia tiruan): menandai lunas lewat jalur webhook yang sama.
func TestDonasi_SimulasiBayar(t *testing.T) {
	srv, _, _ := siapkanServerGerbang(t, true)
	publik := klienBaru(t, srv)
	var d struct{ Kode string }
	publik.minta("POST", "/api/publik/donasi", donasiWeb, 201, &d)
	var p struct {
		Status        string `json:"status"`
		NomorKuitansi string `json:"nomor_kuitansi"`
	}
	publik.minta("POST", "/api/publik/tiruan/"+d.Kode+"/bayar", nil, 200, &p)
	if p.Status != "diterima" || p.NomorKuitansi == "" {
		t.Fatalf("setelah simulasi = %+v", p)
	}
	// Panggilan kedua idempoten: nomor tidak berubah.
	var p2 struct {
		NomorKuitansi string `json:"nomor_kuitansi"`
	}
	publik.minta("POST", "/api/publik/tiruan/"+d.Kode+"/bayar", nil, 200, &p2)
	if p2.NomorKuitansi != p.NomorKuitansi {
		t.Errorf("nomor berubah: %q → %q", p.NomorKuitansi, p2.NomorKuitansi)
	}
}
