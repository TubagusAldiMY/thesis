// Package domain berisi tipe inti dan error bersama. Tidak mengimpor paket
// lain di project ini (aturan arah dependensi: domain di paling dalam).
package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrNotFound: data yang dicari/dirujuk tidak ada.
	ErrNotFound = errors.New("data tidak ditemukan")
	// ErrConflict: bertabrakan dengan data lain (duplikat) atau status tidak
	// sesuai (mis. periode sudah berpindah status).
	ErrConflict = errors.New("data bertentangan dengan data yang ada")
)

// Asnaf = golongan penerima zakat.
type Asnaf string

const (
	AsnafFakir        Asnaf = "fakir"
	AsnafMiskin       Asnaf = "miskin"
	AsnafAmil         Asnaf = "amil"
	AsnafMuallaf      Asnaf = "muallaf"
	AsnafRiqab        Asnaf = "riqab"
	AsnafGharim       Asnaf = "gharim"
	AsnafFisabilillah Asnaf = "fisabilillah"
	AsnafIbnuSabil    Asnaf = "ibnu_sabil"
)

// JenisPeriode: zakat (dibatasi asnaf) atau infak (semua mustahik aktif).
type JenisPeriode string

const (
	JenisZakat JenisPeriode = "zakat"
	JenisInfak JenisPeriode = "infak"
)

// StatusPeriode mengikuti diagram status di docs/03 bagian 2.
type StatusPeriode string

const (
	StatusDraft     StatusPeriode = "draft"
	StatusDihitung  StatusPeriode = "dihitung"
	StatusDisetujui StatusPeriode = "disetujui"
	StatusDitolak   StatusPeriode = "ditolak"
	StatusSelesai   StatusPeriode = "selesai"
)

// TipeKriteria: arah "lebih baik" sebuah kriteria (lihat paket aras).
type TipeKriteria string

const (
	TipeBenefit TipeKriteria = "benefit"
	TipeCost    TipeKriteria = "cost"
)

// Kriteria = satu kolom penilaian beserta bobotnya (pecahan 0–1).
type Kriteria struct {
	ID     string            `json:"id"`
	Kode   string            `json:"kode"`
	Nama   string            `json:"nama"`
	Tipe   TipeKriteria      `json:"tipe"`
	Bobot  float64           `json:"bobot"`
	Skala  map[string]string `json:"skala"` // keterangan skala nilai ("1" → "sehat"); nil bila tidak ada
	Urutan int               `json:"urutan"`
	Aktif  bool              `json:"aktif"`
}

// Mustahik = calon penerima. NIK dan No. HP sudah TERENKRIPSI oleh pemanggil
// (lapisan service); store tidak pernah melihat plaintext.
type Mustahik struct {
	ID        string
	Nama      string
	NIKEnc    []byte
	NIKHash   string // HMAC-SHA256 heksadesimal (64 karakter), untuk cek duplikat
	Alamat    string
	NoHPEnc   []byte // nil bila tidak diisi
	Asnaf     Asnaf
	Aktif     bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Periode = satu putaran penyaluran. Uang dalam rupiah utuh (int64): zakat
// tidak memakai sen, dan menghindari galat pecahan float.
type Periode struct {
	ID                 string        `json:"id"`
	Nama               string        `json:"nama"`
	Jenis              JenisPeriode  `json:"jenis"`
	DanaTersedia       int64         `json:"dana_tersedia"`
	NominalPerPenerima int64         `json:"nominal_per_penerima"`
	AsnafDiizinkan     []Asnaf       `json:"asnaf_diizinkan"` // kosong untuk infak
	Status             StatusPeriode `json:"status"`
	CreatedBy          string        `json:"created_by"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// Kuota = jumlah penerima = ⌊dana ÷ nominal per penerima⌋ (FR-11).
// Nominal ≤ 0 menghasilkan 0 (hindari pembagian nol; data seperti itu
// sudah ditolak CHECK di database).
func (p Periode) Kuota() int {
	if p.NominalPerPenerima <= 0 || p.DanaTersedia <= 0 {
		return 0
	}
	return int(p.DanaTersedia / p.NominalPerPenerima)
}

// ConflictError = bentrok dengan keadaan data (status periode sudah berubah,
// mustahik sedang dipakai, dst.) yang pesannya AMAN ditampilkan ke pengguna.
// errors.Is(err, ErrConflict) tetap benar, jadi pemeriksa lama tidak berubah.
// Bentrok lain (mis. duplikat dari database) tetap ErrConflict polos dan
// lapisan HTTP menampilkan pesan umum, supaya teks error MySQL tidak bocor.
type ConflictError struct{ Pesan string }

func (e *ConflictError) Error() string        { return e.Pesan + ": " + ErrConflict.Error() }
func (e *ConflictError) Is(target error) bool { return target == ErrConflict }

// Konflik membuat ConflictError dengan pesan berbahasa Indonesia untuk pengguna.
func Konflik(format string, args ...any) error {
	return &ConflictError{Pesan: fmt.Sprintf(format, args...)}
}
