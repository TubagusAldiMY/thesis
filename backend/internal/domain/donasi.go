package domain

import (
	"context"
	"errors"
	"time"
)

// JenisDonasi = ragam penerimaan yang dikelola masjid (hasil wawancara
// 15/10/2026: zakat fitrah, zakat mal, zakat profesi, dan infak/sedekah).
type JenisDonasi string

const (
	DonasiZakatFitrah  JenisDonasi = "zakat_fitrah"
	DonasiZakatMal     JenisDonasi = "zakat_mal"
	DonasiZakatProfesi JenisDonasi = "zakat_profesi"
	DonasiInfak        JenisDonasi = "infak"
)

// StatusDonasi:
//
//	menunggu ──bayar berhasil──► diterima ──petugas batalkan──► dibatalkan
//	    ├──────gagal/ditolak───► gagal
//	    └──────waktu habis─────► kedaluwarsa
//
// Hanya "diterima" yang punya nomor kuitansi dan dihitung sebagai penerimaan.
type StatusDonasi string

const (
	DonasiMenunggu    StatusDonasi = "menunggu"
	DonasiDiterima    StatusDonasi = "diterima"
	DonasiGagal       StatusDonasi = "gagal"
	DonasiKedaluwarsa StatusDonasi = "kedaluwarsa"
	DonasiDibatalkan  StatusDonasi = "dibatalkan"
)

// SumberDonasi: lewat halaman publik (gateway) atau dicatat petugas di masjid.
type SumberDonasi string

const (
	SumberWeb     SumberDonasi = "web"
	SumberPetugas SumberDonasi = "petugas"
)

// Donasi = satu penerimaan zakat/infak. No. HP sudah TERENKRIPSI oleh
// pemanggil (service); store tidak pernah melihat plaintext. Uang dalam
// rupiah utuh (int64), sama seperti Periode.
type Donasi struct {
	ID            string
	KodePublik    string // rahasia tautan muzaki; jangan dilog
	Nama          string
	NoHPEnc       []byte // nil bila tidak diisi
	Jenis         JenisDonasi
	Jumlah        int64
	Sumber        SumberDonasi
	Metode        string
	Status        StatusDonasi
	NomorKuitansi string // kosong sampai diterima
	Keterangan    string
	BayarURL      string
	DibayarPada   *time.Time
	DilihatPada   *time.Time
	DicatatOleh   string
	AlasanBatal   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Kode aksi audit untuk donasi (dikelompokkan di service.KategoriAudit["donasi"]).
const (
	AksiDonasiTerima = "DONASI_TERIMA" // pembayaran web dikonfirmasi gateway
	AksiDonasiCatat  = "DONASI_CATAT"  // petugas mencatat penerimaan langsung
	AksiDonasiBatal  = "DONASI_BATAL"  // petugas membatalkan kuitansi
)

var (
	// ErrNonaktif: fitur belum dikonfigurasi (mis. gateway pembayaran belum diatur).
	ErrNonaktif = errors.New("fitur tidak aktif")
	// ErrGerbang: payment gateway gagal/tidak menjawab. Detailnya hanya untuk log.
	ErrGerbang = errors.New("payment gateway bermasalah")
	// ErrTandaTanganTidakSah: notifikasi gateway tidak lolos verifikasi tanda tangan.
	ErrTandaTanganTidakSah = errors.New("tanda tangan notifikasi tidak sah")
)

// PermintaanBayar = isi transaksi yang dibuat di gateway. Hanya nama (bukan
// No. HP/alamat) yang dikirim ke gateway: data minimum yang dibutuhkan.
type PermintaanBayar struct {
	OrderID          string // = Donasi.ID
	Jumlah           int64
	NamaDonatur      string
	NamaItem         string // tampil di halaman bayar, mis. "Zakat Fitrah"
	SelesaiURL       string // tujuan pengalihan setelah muzaki selesai di halaman gateway
	KedaluwarsaMenit int
}

// HasilBayar = tautan halaman bayar yang harus dibuka muzaki.
type HasilBayar struct {
	RedirectURL string
}

// HasilNotifikasi = isi notifikasi gateway yang SUDAH lolos verifikasi tanda
// tangan dan dipetakan ke status donasi.
type HasilNotifikasi struct {
	OrderID string
	Jumlah  int64
	Status  StatusDonasi // menunggu | diterima | gagal | kedaluwarsa
	Metode  string       // payment_type dari gateway
}

// Berkas = satu lampiran pesan. CID terisi = gambar inline yang dirujuk HTML
// lewat src="cid:<CID>"; kosong = lampiran biasa yang diunduh penerima.
type Berkas struct {
	Nama string // nama berkas, tanpa jalur
	Tipe string // media type, mis. "application/pdf"
	CID  string
	Isi  []byte
}

// Pesan = isi satu pemberitahuan. Teks wajib (cadangan bagi klien yang tidak
// menampilkan HTML); HTML boleh kosong. Pemanggil yang memastikan isinya aman:
// HTML sudah di-escape, dan tidak memuat rahasia (kode kuitansi publik, No. HP penuh).
type Pesan struct {
	Subjek   string
	Teks     string
	HTML     string
	Inline   []Berkas // gambar yang dipakai HTML
	Lampiran []Berkas
}

// Pemberitahu mengirim pesan kepada Ketua (saat ini lewat email, lihat
// internal/notif). Penerimanya ditetapkan oleh implementasi, bukan pemanggil,
// jadi service tidak perlu tahu alamat email.
type Pemberitahu interface {
	Kirim(ctx context.Context, p Pesan) error
}

// GerbangBayar memisahkan aturan donasi dari penyedia pembayaran: Midtrans
// untuk production/sandbox, penyedia tiruan untuk dev dan test.
type GerbangBayar interface {
	// Buat membuat transaksi dan mengembalikan tautan halaman bayar.
	Buat(ctx context.Context, p PermintaanBayar) (HasilBayar, error)
	// Verifikasi memeriksa tanda tangan notifikasi (body mentah dari gateway)
	// lalu memetakan statusnya. Tanda tangan salah → ErrTandaTanganTidakSah.
	Verifikasi(body []byte) (HasilNotifikasi, error)
}
