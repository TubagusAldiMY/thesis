// Package payment berisi penyedia pembayaran untuk donasi web (domain.GerbangBayar):
// Midtrans Snap untuk sandbox/production, dan penyedia tiruan untuk dev/test.
//
// Alur Midtrans Snap: server membuat transaksi (Buat) → muzaki dialihkan ke
// halaman bayar Midtrans (kartu/QRIS/VA diurus Midtrans, aplikasi tidak pernah
// menyentuh data pembayaran) → Midtrans memanggil URL notifikasi kita dengan
// tanda tangan SHA512 (Verifikasi) → status donasi berubah.
// Rujukan: dokumentasi Midtrans "Snap API" dan "HTTP(S) Notification / Webhooks".
package payment

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"spkzakat/internal/domain"
)

// URLSnapSandbox = endpoint pembuatan transaksi Snap di lingkungan sandbox.
// Production memakai alamat lain; set lewat MIDTRANS_SNAP_URL.
const URLSnapSandbox = "https://app.sandbox.midtrans.com/snap/v1/transactions"

// batasBalasan mencegah balasan tak terduga memenuhi memori.
const batasBalasan = 1 << 20

// Midtrans = klien Snap. ServerKey bersifat RAHASIA: tidak boleh dilog atau
// dikirim ke browser.
type Midtrans struct {
	serverKey string
	snapURL   string
	http      *http.Client
}

// NewMidtrans membuat klien. snapURL kosong = sandbox.
func NewMidtrans(serverKey, snapURL string) (*Midtrans, error) {
	if strings.TrimSpace(serverKey) == "" {
		return nil, errors.New("payment: MIDTRANS_SERVER_KEY kosong")
	}
	if snapURL == "" {
		snapURL = URLSnapSandbox
	}
	if !strings.HasPrefix(snapURL, "https://") {
		return nil, errors.New("payment: MIDTRANS_SNAP_URL harus https")
	}
	return &Midtrans{serverKey: serverKey, snapURL: snapURL, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

type snapItem struct {
	ID       string `json:"id"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type snapRequest struct {
	TransactionDetails struct {
		OrderID     string `json:"order_id"`
		GrossAmount int64  `json:"gross_amount"`
	} `json:"transaction_details"`
	ItemDetails     []snapItem `json:"item_details"`
	CustomerDetails struct {
		FirstName string `json:"first_name"`
	} `json:"customer_details"`
	Callbacks struct {
		Finish string `json:"finish"`
	} `json:"callbacks"`
	Expiry struct {
		Unit     string `json:"unit"`
		Duration int    `json:"duration"`
	} `json:"expiry"`
}

// potong membatasi teks agar muat di kolom Midtrans (nama item maks. 50 karakter).
func potong(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// Buat membuat transaksi Snap dan mengembalikan tautan halaman bayar. Yang
// dikirim ke Midtrans hanya: order_id, jumlah, nama item, dan nama donatur.
// Detail balasan error Midtrans hanya masuk pesan galat (untuk log), tidak ke muzaki.
func (m *Midtrans) Buat(ctx context.Context, p domain.PermintaanBayar) (domain.HasilBayar, error) {
	var req snapRequest
	req.TransactionDetails.OrderID = p.OrderID
	req.TransactionDetails.GrossAmount = p.Jumlah
	// Midtrans mensyaratkan Σ item_details = gross_amount; satu item cukup.
	req.ItemDetails = []snapItem{{ID: "donasi", Price: p.Jumlah, Quantity: 1, Name: potong(p.NamaItem, 50)}}
	req.CustomerDetails.FirstName = potong(p.NamaDonatur, 100)
	req.Callbacks.Finish = p.SelesaiURL
	req.Expiry.Unit, req.Expiry.Duration = "minutes", p.KedaluwarsaMenit

	body, err := json.Marshal(req)
	if err != nil {
		return domain.HasilBayar{}, fmt.Errorf("payment.Midtrans.Buat: %w", err)
	}
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, m.snapURL, bytes.NewReader(body))
	if err != nil {
		return domain.HasilBayar{}, fmt.Errorf("payment.Midtrans.Buat: %w", err)
	}
	hr.Header.Set("Accept", "application/json")
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(m.serverKey+":")))

	res, err := m.http.Do(hr)
	if err != nil {
		return domain.HasilBayar{}, fmt.Errorf("payment.Midtrans.Buat: kirim: %w", err)
	}
	defer res.Body.Close()
	isi, err := io.ReadAll(io.LimitReader(res.Body, batasBalasan))
	if err != nil {
		return domain.HasilBayar{}, fmt.Errorf("payment.Midtrans.Buat: baca balasan: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return domain.HasilBayar{}, fmt.Errorf("payment.Midtrans.Buat: status %d: %s", res.StatusCode, potong(string(isi), 200))
	}
	var out struct {
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.Unmarshal(isi, &out); err != nil || out.RedirectURL == "" {
		return domain.HasilBayar{}, errors.New("payment.Midtrans.Buat: balasan tanpa redirect_url")
	}
	// Muzaki akan diarahkan ke tautan ini; tolak yang bukan https.
	if !strings.HasPrefix(out.RedirectURL, "https://") {
		return domain.HasilBayar{}, errors.New("payment.Midtrans.Buat: redirect_url bukan https")
	}
	return domain.HasilBayar{RedirectURL: out.RedirectURL}, nil
}

// notifikasi = bidang notifikasi yang dipakai. Semua bertipe string menurut
// dokumentasi Midtrans; gross_amount harus dipakai PERSIS seperti diterima
// ("10000.00") karena ikut dalam perhitungan tanda tangan.
type notifikasi struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	PaymentType       string `json:"payment_type"`
}

// Verifikasi memeriksa tanda tangan notifikasi lalu memetakan statusnya.
// signature_key = SHA512(order_id + status_code + gross_amount + ServerKey).
// Muatan yang rusak, kurang bidang, atau salah tanda tangan semuanya
// dikembalikan sebagai ErrTandaTanganTidakSah (tidak dipercaya sama sekali).
func (m *Midtrans) Verifikasi(body []byte) (domain.HasilNotifikasi, error) {
	var n notifikasi
	if err := json.Unmarshal(body, &n); err != nil {
		return domain.HasilNotifikasi{}, fmt.Errorf("%w: muatan bukan JSON valid", domain.ErrTandaTanganTidakSah)
	}
	if n.OrderID == "" || n.StatusCode == "" || n.GrossAmount == "" || n.SignatureKey == "" {
		return domain.HasilNotifikasi{}, fmt.Errorf("%w: bidang wajib kosong", domain.ErrTandaTanganTidakSah)
	}
	h := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + m.serverKey))
	harapan := hex.EncodeToString(h[:])
	if subtle.ConstantTimeCompare([]byte(harapan), []byte(strings.ToLower(n.SignatureKey))) != 1 {
		return domain.HasilNotifikasi{}, domain.ErrTandaTanganTidakSah
	}

	jumlah, err := rupiahUtuh(n.GrossAmount)
	if err != nil {
		// Tanda tangan benar tapi jumlah bukan rupiah utuh: tidak mungkin dari
		// transaksi kita. Perlakukan sebagai muatan tak tepercaya, bukan galat server
		// (galat 5xx membuat Midtrans mengirim ulang terus).
		return domain.HasilNotifikasi{}, fmt.Errorf("%w: %v", domain.ErrTandaTanganTidakSah, err)
	}
	return domain.HasilNotifikasi{
		OrderID: n.OrderID, Jumlah: jumlah, Status: petakanStatus(n.TransactionStatus, n.FraudStatus), Metode: n.PaymentType,
	}, nil
}

// petakanStatus menerjemahkan status Midtrans ke status donasi:
//   - settlement → diterima; capture (kartu) → diterima hanya bila fraud_status accept
//   - deny / cancel / failure → gagal; expire → kedaluwarsa
//   - pending, capture berstatus challenge, dan status lain (refund, dst.) → menunggu
//     (tidak mengubah apa pun; refund ditangani petugas lewat pembatalan kuitansi).
func petakanStatus(transaksi, fraud string) domain.StatusDonasi {
	switch transaksi {
	case "settlement":
		return domain.DonasiDiterima
	case "capture":
		if fraud == "accept" {
			return domain.DonasiDiterima
		}
		return domain.DonasiMenunggu
	case "deny", "cancel", "failure":
		return domain.DonasiGagal
	case "expire":
		return domain.DonasiKedaluwarsa
	default:
		return domain.DonasiMenunggu
	}
}

// rupiahUtuh membaca "10000.00" → 10000. Sen selain .00 ditolak: donasi selalu
// dalam rupiah utuh, jadi nilai lain berarti notifikasi tidak sesuai transaksi kita.
func rupiahUtuh(s string) (int64, error) {
	utuh, pecahan, ada := strings.Cut(s, ".")
	if ada && strings.Trim(pecahan, "0") != "" {
		return 0, fmt.Errorf("gross_amount %q bukan rupiah utuh", s)
	}
	n, err := strconv.ParseInt(utuh, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("gross_amount %q tidak valid", s)
	}
	return n, nil
}
