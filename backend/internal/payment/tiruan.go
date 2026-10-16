package payment

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"spkzakat/internal/domain"
)

// kunciTiruan = "server key" penyedia tiruan. Sengaja bukan rahasia: penyedia
// ini hanya untuk development, demo sidang tanpa akun Midtrans, dan test.
// Server menolak menyalakannya di production (lihat cmd/server).
const kunciTiruan = "kunci-penyedia-tiruan-bukan-rahasia"

// Tiruan = penyedia pembayaran palsu: Buat mengalihkan ke halaman simulasi di
// frontend (/donasi/tiruan), dan notifikasinya memakai format serta tanda
// tangan yang sama dengan Midtrans, jadi seluruh jalur Verifikasi → ubah status
// → terbit kuitansi ikut teruji tanpa memanggil layanan luar.
type Tiruan struct{ *Midtrans }

// NewTiruan membuat penyedia tiruan.
func NewTiruan() *Tiruan {
	m, _ := NewMidtrans(kunciTiruan, "") // kunci tidak kosong dan URL bawaan https → tidak mungkin galat
	return &Tiruan{m}
}

// Buat tidak memanggil jaringan. Halaman simulasi berada di alamat yang sama
// dengan halaman selesai, hanya jalurnya berbeda.
func (t *Tiruan) Buat(_ context.Context, p domain.PermintaanBayar) (domain.HasilBayar, error) {
	return domain.HasilBayar{RedirectURL: strings.Replace(p.SelesaiURL, "/donasi/selesai", "/donasi/tiruan", 1)}, nil
}

// Muatan membuat notifikasi yang lolos Verifikasi untuk order tertentu.
// status memakai istilah Midtrans: settlement, pending, expire, deny, dst.
func (t *Tiruan) Muatan(orderID string, jumlah int64, status string) []byte {
	const kode = "200"
	gross := strconv.FormatInt(jumlah, 10) + ".00" // meniru format Midtrans: rupiah utuh, dua desimal
	h := sha512.Sum512([]byte(orderID + kode + gross + kunciTiruan))
	b, _ := json.Marshal(map[string]string{
		"order_id": orderID, "status_code": kode, "gross_amount": gross,
		"signature_key": hex.EncodeToString(h[:]), "transaction_status": status,
		"fraud_status": "accept", "payment_type": "qris",
	})
	return b
}
