package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// KeySize = panjang kunci yang diminta: 32 byte (AES-256 / HMAC-SHA256).
const KeySize = 32

// ErrDekripsi sengaja SATU pesan untuk semua kegagalan dekripsi (kunci salah,
// data diubah, AAD salah, data terpotong): rinciannya tidak berguna bagi
// pemanggil dan bisa membantu penyerang.
var ErrDekripsi = errors.New("crypto: data tidak bisa didekripsi")

// ParseKey membaca kunci base64 (mis. dari variabel lingkungan) dan memastikan
// panjangnya tepat 32 byte. Pesan error tidak memuat isi kunci.
func ParseKey(s string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, errors.New("crypto: kunci bukan base64 yang valid")
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("crypto: kunci harus %d byte, dapat %d (buat dengan `openssl rand -base64 32`)", KeySize, len(key))
	}
	return key, nil
}

// Cipher mengenkripsi data pribadi dengan AES-256-GCM (terautentikasi: data
// yang diubah satu bit pun akan ditolak saat dekripsi, bukan menghasilkan
// sampah diam-diam).
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher membuat Cipher dari kunci 32 byte. Kunci disalin; mengubah slice
// pemanggil sesudahnya tidak berpengaruh.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("crypto: kunci enkripsi harus %d byte, dapat %d", KeySize, len(key))
	}
	blok, err := aes.NewCipher(append([]byte(nil), key...))
	if err != nil {
		return nil, fmt.Errorf("crypto: AES: %w", err)
	}
	aead, err := cipher.NewGCM(blok)
	if err != nil {
		return nil, fmt.Errorf("crypto: GCM: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt mengembalikan nonce(12 byte) ‖ ciphertext ‖ tag(16 byte). Nonce acak
// BARU tiap panggilan (GCM celaka bila nonce berulang dengan kunci sama).
//
// aad = "konteks" yang ikut diautentikasi tetapi tidak dienkripsi, mis.
// []byte("mustahik.nik"). Dengan ini ciphertext kolom NIK tidak bisa
// ditukar-pasang ke kolom No. HP (atau baris lain) tanpa ketahuan.
func (c *Cipher) Encrypt(plain, aad []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: nonce acak: %w", err)
	}
	return c.aead.Seal(nonce, nonce, plain, aad), nil
}

// Decrypt kebalikan Encrypt. aad harus sama persis dengan saat Encrypt.
func (c *Cipher) Decrypt(blob, aad []byte) ([]byte, error) {
	n := c.aead.NonceSize()
	if len(blob) < n+c.aead.Overhead() {
		return nil, ErrDekripsi
	}
	plain, err := c.aead.Open(nil, blob[:n], blob[n:], aad)
	if err != nil {
		return nil, ErrDekripsi
	}
	return plain, nil
}

// Hasher menghitung HMAC-SHA256 sebagai "blind index": nilai tetap untuk
// input yang sama (jadi bisa dipakai UNIQUE/pencarian di database), tetapi
// tanpa kunci tidak bisa dibalik atau ditebak. SHA-256 polos tidak cukup —
// NIK hanya 16 digit sehingga bisa ditebak dengan tabel pelangi.
type Hasher struct {
	key []byte
}

// NewHasher membuat Hasher; kunci minimal 32 byte (disalin).
func NewHasher(key []byte) (*Hasher, error) {
	if len(key) < KeySize {
		return nil, fmt.Errorf("crypto: kunci HMAC minimal %d byte, dapat %d", KeySize, len(key))
	}
	return &Hasher{key: append([]byte(nil), key...)}, nil
}

// Sum mengembalikan HMAC-SHA256 heksadesimal huruf kecil (64 karakter,
// sesuai kolom CHAR(64) `nik_hash`).
func (h *Hasher) Sum(data string) string {
	m := hmac.New(sha256.New, h.key)
	m.Write([]byte(data))
	return hex.EncodeToString(m.Sum(nil))
}
