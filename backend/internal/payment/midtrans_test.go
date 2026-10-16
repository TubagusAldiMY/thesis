package payment

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"spkzakat/internal/domain"
)

const kunciUji = "SB-Mid-server-kunci-uji"

// notif membuat muatan notifikasi bertanda tangan sah untuk kunciUji.
func notif(orderID, kode, gross, status, fraud string) []byte {
	h := sha512.Sum512([]byte(orderID + kode + gross + kunciUji))
	b, _ := json.Marshal(map[string]string{
		"order_id": orderID, "status_code": kode, "gross_amount": gross,
		"signature_key": hex.EncodeToString(h[:]), "transaction_status": status,
		"fraud_status": fraud, "payment_type": "qris",
	})
	return b
}

func klienUji(t *testing.T) *Midtrans {
	t.Helper()
	m, err := NewMidtrans(kunciUji, "")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// Smoke/sanity: notifikasi settlement yang sah → diterima dengan jumlah benar.
func TestVerifikasi_SettlementSah(t *testing.T) {
	h, err := klienUji(t).Verifikasi(notif("order-1", "200", "50000.00", "settlement", "accept"))
	if err != nil {
		t.Fatalf("Verifikasi: %v", err)
	}
	if h.OrderID != "order-1" || h.Jumlah != 50000 || h.Status != domain.DonasiDiterima || h.Metode != "qris" {
		t.Errorf("hasil = %+v", h)
	}
}

// Tabel pemetaan status Midtrans → status donasi.
func TestVerifikasi_PemetaanStatus(t *testing.T) {
	tests := []struct {
		nama, status, fraud string
		mau                 domain.StatusDonasi
	}{
		{"settlement", "settlement", "accept", domain.DonasiDiterima},
		{"capture kartu aman", "capture", "accept", domain.DonasiDiterima},
		{"capture challenge belum final", "capture", "challenge", domain.DonasiMenunggu},
		{"pending", "pending", "", domain.DonasiMenunggu},
		{"deny", "deny", "", domain.DonasiGagal},
		{"cancel", "cancel", "", domain.DonasiGagal},
		{"failure", "failure", "", domain.DonasiGagal},
		{"expire", "expire", "", domain.DonasiKedaluwarsa},
		{"status tak dikenal tidak mengubah apa pun", "refund", "", domain.DonasiMenunggu},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			h, err := klienUji(t).Verifikasi(notif("o", "200", "10000.00", tt.status, tt.fraud))
			if err != nil {
				t.Fatalf("Verifikasi: %v", err)
			}
			if h.Status != tt.mau {
				t.Errorf("status = %q, mau %q", h.Status, tt.mau)
			}
		})
	}
}

// Keamanan (jalur kritis): semua muatan yang tidak sah harus ditolak sebagai
// ErrTandaTanganTidakSah, termasuk yang diubah setelah ditandatangani.
func TestVerifikasi_Ditolak(t *testing.T) {
	sah := notif("order-1", "200", "50000.00", "settlement", "accept")
	var m map[string]string
	_ = json.Unmarshal(sah, &m)
	ubah := func(k, v string) []byte {
		c := map[string]string{}
		for a, b := range m {
			c[a] = b
		}
		c[k] = v
		b, _ := json.Marshal(c)
		return b
	}
	hapus := func(k string) []byte {
		c := map[string]string{}
		for a, b := range m {
			if a != k {
				c[a] = b
			}
		}
		b, _ := json.Marshal(c)
		return b
	}
	// Jumlah bersen dengan tanda tangan yang benar untuknya: lolos tanda tangan tapi bukan rupiah utuh.
	bersen := notif("order-1", "200", "50000.50", "settlement", "accept")

	tests := []struct {
		nama string
		isi  []byte
	}{
		{"bukan JSON", []byte("bukan json")},
		{"kosong", nil},
		{"tanda tangan diganti", ubah("signature_key", strings.Repeat("0", 128))},
		{"jumlah diubah setelah ditandatangani", ubah("gross_amount", "1.00")},
		{"order_id diubah", ubah("order_id", "order-lain")},
		{"status_code diubah", ubah("status_code", "201")},
		{"tanpa tanda tangan", hapus("signature_key")},
		{"tanpa order_id", hapus("order_id")},
		{"jumlah bersen", bersen},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			if _, err := klienUji(t).Verifikasi(tt.isi); !errors.Is(err, domain.ErrTandaTanganTidakSah) {
				t.Errorf("err = %v, mau ErrTandaTanganTidakSah", err)
			}
		})
	}
}

// Kunci server yang berbeda tidak boleh bisa memalsukan notifikasi.
func TestVerifikasi_KunciLain(t *testing.T) {
	lain, _ := NewMidtrans("kunci-lain", "")
	if _, err := lain.Verifikasi(notif("o", "200", "10000.00", "settlement", "accept")); !errors.Is(err, domain.ErrTandaTanganTidakSah) {
		t.Errorf("err = %v", err)
	}
}

func TestNewMidtrans_Validasi(t *testing.T) {
	if _, err := NewMidtrans("", ""); err == nil {
		t.Error("server key kosong harus ditolak")
	}
	if _, err := NewMidtrans("k", "http://insecure.example"); err == nil {
		t.Error("URL bukan https harus ditolak")
	}
}

func TestRupiahUtuh(t *testing.T) {
	ok := map[string]int64{"10000.00": 10000, "10000": 10000, "1.0": 1}
	for in, mau := range ok {
		if n, err := rupiahUtuh(in); err != nil || n != mau {
			t.Errorf("rupiahUtuh(%q) = %d, %v; mau %d", in, n, err, mau)
		}
	}
	for _, in := range []string{"", "abc", "0.00", "-5.00", "100.01", "1e3"} {
		if _, err := rupiahUtuh(in); err == nil {
			t.Errorf("rupiahUtuh(%q) seharusnya galat", in)
		}
	}
}

// serverSnap menjalankan server Snap palsu dan merekam permintaan terakhir.
func serverSnap(t *testing.T, status int, balasan string, dapat *http.Request, isi *[]byte) *Midtrans {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*dapat = *r
		*isi, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(balasan))
	}))
	t.Cleanup(srv.Close)
	m, err := NewMidtrans(kunciUji, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	m.http = srv.Client()
	return m
}

func TestBuat_Sukses(t *testing.T) {
	var r http.Request
	var isi []byte
	m := serverSnap(t, 201, `{"token":"tok","redirect_url":"https://app.sandbox.midtrans.com/snap/v4/redirection/tok"}`, &r, &isi)

	h, err := m.Buat(context.Background(), domain.PermintaanBayar{
		OrderID: "order-1", Jumlah: 75000, NamaDonatur: "Hamba Allah", NamaItem: "Zakat Fitrah",
		SelesaiURL: "https://masjid.example/donasi/selesai?kode=abc", KedaluwarsaMenit: 1440,
	})
	if err != nil {
		t.Fatalf("Buat: %v", err)
	}
	if h.RedirectURL != "https://app.sandbox.midtrans.com/snap/v4/redirection/tok" {
		t.Errorf("RedirectURL = %q", h.RedirectURL)
	}
	if r.Method != http.MethodPost {
		t.Errorf("method = %s", r.Method)
	}
	// Basic auth = base64(serverKey + ":")
	if mau := "Basic " + base64.StdEncoding.EncodeToString([]byte(kunciUji+":")); r.Header.Get("Authorization") != mau {
		t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
	}
	var req map[string]any
	if err := json.Unmarshal(isi, &req); err != nil {
		t.Fatal(err)
	}
	td := req["transaction_details"].(map[string]any)
	if td["order_id"] != "order-1" || td["gross_amount"].(float64) != 75000 {
		t.Errorf("transaction_details = %v", td)
	}
	item := req["item_details"].([]any)[0].(map[string]any)
	if item["price"].(float64) != 75000 || item["name"] != "Zakat Fitrah" {
		t.Errorf("item_details = %v", item) // Σ item harus sama dengan gross_amount
	}
	if req["callbacks"].(map[string]any)["finish"] != "https://masjid.example/donasi/selesai?kode=abc" {
		t.Errorf("callbacks = %v", req["callbacks"])
	}
	// Privasi: hanya nama yang dikirim sebagai data donatur (tanpa No. HP/email/alamat).
	cd := req["customer_details"].(map[string]any)
	if len(cd) != 1 || cd["first_name"] != "Hamba Allah" {
		t.Errorf("customer_details = %v", cd)
	}
}

func TestBuat_Galat(t *testing.T) {
	tests := []struct {
		nama   string
		status int
		isi    string
	}{
		{"ditolak Midtrans", 401, `{"error_messages":["Access denied"]}`},
		{"galat server", 500, `oops`},
		{"tanpa redirect_url", 201, `{"token":"x"}`},
		{"redirect_url bukan https", 201, `{"redirect_url":"http://evil.example"}`},
		{"bukan JSON", 201, `<html>`},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			var r http.Request
			var isi []byte
			m := serverSnap(t, tt.status, tt.isi, &r, &isi)
			if _, err := m.Buat(context.Background(), domain.PermintaanBayar{OrderID: "o", Jumlah: 10000, NamaDonatur: "A", NamaItem: "Infak"}); err == nil {
				t.Error("seharusnya galat")
			}
		})
	}
}

// Penyedia tiruan: muatan buatannya harus lolos Verifikasi, dan jumlah/status sesuai.
func TestTiruan_MuatanLolosVerifikasi(t *testing.T) {
	ti := NewTiruan()
	h, err := ti.Verifikasi(ti.Muatan("order-9", 25000, "settlement"))
	if err != nil {
		t.Fatalf("Verifikasi: %v", err)
	}
	if h.OrderID != "order-9" || h.Jumlah != 25000 || h.Status != domain.DonasiDiterima {
		t.Errorf("hasil = %+v", h)
	}
	// Tidak boleh lolos di penyedia Midtrans sungguhan (kunci beda).
	if _, err := klienUji(t).Verifikasi(ti.Muatan("order-9", 25000, "settlement")); !errors.Is(err, domain.ErrTandaTanganTidakSah) {
		t.Errorf("muatan tiruan lolos di Midtrans asli: %v", err)
	}
	r, _ := ti.Buat(context.Background(), domain.PermintaanBayar{SelesaiURL: "http://localhost:5173/donasi/selesai?kode=abc"})
	if r.RedirectURL != "http://localhost:5173/donasi/tiruan?kode=abc" {
		t.Errorf("RedirectURL tiruan = %q", r.RedirectURL)
	}
}
