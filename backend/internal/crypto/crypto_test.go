package crypto

import (
	"bytes"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func kunci(b byte) []byte { return bytes.Repeat([]byte{b}, KeySize) }

// ---------- password ----------

func TestHashPassword_FormatDanSalt(t *testing.T) {
	h1, err := HashPassword("kata-sandi-panjang-123")
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := HashPassword("kata-sandi-panjang-123")

	re := regexp.MustCompile(`^\$argon2id\$v=19\$m=65536,t=3,p=4\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`)
	if !re.MatchString(h1) {
		t.Errorf("format hash tidak sesuai PHC/parameter OWASP: %q", h1)
	}
	if h1 == h2 {
		t.Error("dua hash dari password sama identik: salt tidak acak")
	}
	if strings.Contains(h1, "kata-sandi") {
		t.Error("plaintext muncul di dalam hash")
	}
}

func TestHashPassword_Kosong(t *testing.T) {
	if _, err := HashPassword(""); !errors.Is(err, ErrPasswordKosong) {
		t.Errorf("err = %v, mau ErrPasswordKosong", err)
	}
}

func TestVerifyPassword(t *testing.T) {
	const asli = "P@ssw0rd-é日本-Panjang"
	hash, err := HashPassword(asli)
	if err != nil {
		t.Fatal(err)
	}
	panjang := strings.Repeat("x", 1000)
	hashPanjang, _ := HashPassword(panjang)

	tests := []struct {
		nama, hash, coba string
		mau              bool
	}{
		{"benar (termasuk unicode)", hash, asli, true},
		{"salah", hash, "salah", false},
		{"beda huruf besar/kecil", hash, strings.ToUpper(asli), false},
		{"spasi di ujung", hash, asli + " ", false},
		{"kosong", hash, "", false},
		{"password 1000 karakter benar", hashPanjang, panjang, true},
		{"password 1000 karakter, beda satu karakter", hashPanjang, panjang[:999] + "y", false},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			ok, err := VerifyPassword(tt.hash, tt.coba)
			if err != nil || ok != tt.mau {
				t.Errorf("VerifyPassword = %v, %v; mau %v, nil", ok, err, tt.mau)
			}
		})
	}
}

// Parameter harus dibaca dari string hash, bukan dari konstanta: kalau
// konstanta dinaikkan kelak, hash lama tetap bisa diverifikasi.
func TestVerifyPassword_ParameterDariHash(t *testing.T) {
	salt := []byte("garam-16-byte-ok")
	key := argon2.IDKey([]byte("rahasia"), salt, 1, 8*1024, 2, 32) // t=1, m=8MiB, p=2 (≠ bawaan)
	enc := "$argon2id$v=19$m=8192,t=1,p=2$" + b64.EncodeToString(salt) + "$" + b64.EncodeToString(key)

	if ok, err := VerifyPassword(enc, "rahasia"); err != nil || !ok {
		t.Errorf("hash berparameter lain gagal diverifikasi: %v, %v", ok, err)
	}
	if ok, _ := VerifyPassword(enc, "salah"); ok {
		t.Error("password salah diterima")
	}
}

func TestVerifyPassword_FormatRusak(t *testing.T) {
	s := b64.EncodeToString([]byte("garam-16-byte-ok"))
	h := b64.EncodeToString(bytes.Repeat([]byte{1}, 32))
	tests := []struct{ nama, enc string }{
		{"kosong", ""},
		{"teks biasa", "bukan-hash"},
		{"algoritma lain (argon2i)", "$argon2i$v=19$m=65536,t=3,p=4$" + s + "$" + h},
		{"versi lain", "$argon2id$v=16$m=65536,t=3,p=4$" + s + "$" + h},
		{"bagian kurang", "$argon2id$v=19$m=65536,t=3,p=4$" + s},
		{"bagian lebih", "$argon2id$v=19$m=65536,t=3,p=4$" + s + "$" + h + "$x"},
		{"tanpa awalan $", "argon2id$v=19$m=65536,t=3,p=4$" + s + "$" + h},
		{"parameter bukan angka", "$argon2id$v=19$m=abc,t=3,p=4$" + s + "$" + h},
		{"parameter tak dikenal", "$argon2id$v=19$m=65536,t=3,p=4,x=1$" + s + "$" + h},
		{"parameter tanpa nilai", "$argon2id$v=19$m,t=3,p=4$" + s + "$" + h},
		{"memori raksasa (DoS)", "$argon2id$v=19$m=4000000000,t=3,p=4$" + s + "$" + h},
		{"waktu raksasa (DoS)", "$argon2id$v=19$m=65536,t=4000000,p=4$" + s + "$" + h},
		{"thread nol", "$argon2id$v=19$m=65536,t=3,p=0$" + s + "$" + h},
		{"thread kebanyakan", "$argon2id$v=19$m=65536,t=3,p=200$" + s + "$" + h},
		{"memori terlalu kecil untuk thread", "$argon2id$v=19$m=8,t=3,p=4$" + s + "$" + h},
		{"salt bukan base64", "$argon2id$v=19$m=65536,t=3,p=4$!!!!$" + h},
		{"salt terlalu pendek", "$argon2id$v=19$m=65536,t=3,p=4$" + b64.EncodeToString([]byte("abc")) + "$" + h},
		{"hash bukan base64", "$argon2id$v=19$m=65536,t=3,p=4$" + s + "$!!!!"},
		{"hash terlalu pendek", "$argon2id$v=19$m=65536,t=3,p=4$" + s + "$" + b64.EncodeToString([]byte("abc"))},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			ok, err := VerifyPassword(tt.enc, "apa saja")
			if ok || !errors.Is(err, ErrFormatHash) {
				t.Errorf("VerifyPassword = %v, %v; mau false, ErrFormatHash", ok, err)
			}
		})
	}
}

// ---------- cipher ----------

func TestCipher_Roundtrip(t *testing.T) {
	c, err := NewCipher(kunci(7))
	if err != nil {
		t.Fatal(err)
	}
	aad := []byte("mustahik.nik")
	tests := []struct {
		nama  string
		plain []byte
	}{
		{"NIK 16 digit", []byte("3271012345678901")},
		{"kosong", []byte{}},
		{"biner dengan byte 0x00 dan 0xff", []byte{0x00, 0xff, 0x00, 0x10}},
		{"unicode", []byte("日本語-é")},
		{"besar (1 MiB)", bytes.Repeat([]byte("a"), 1<<20)},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			blob, err := c.Encrypt(tt.plain, aad)
			if err != nil {
				t.Fatal(err)
			}
			if want := 12 + len(tt.plain) + 16; len(blob) != want {
				t.Errorf("panjang blob = %d, mau nonce(12)+data+tag(16) = %d", len(blob), want)
			}
			if len(tt.plain) >= 8 && bytes.Contains(blob, tt.plain) {
				t.Error("plaintext terlihat di dalam ciphertext")
			}
			got, err := c.Decrypt(blob, aad)
			if err != nil || !bytes.Equal(got, tt.plain) {
				t.Errorf("Decrypt = %q, %v; mau %q", got, err, tt.plain)
			}
		})
	}
}

func TestCipher_NonceTidakPernahBerulang(t *testing.T) {
	c, _ := NewCipher(kunci(1))
	lihat := map[string]bool{}
	var pertama []byte
	for i := 0; i < 5000; i++ {
		blob, err := c.Encrypt([]byte("data sama"), nil)
		if err != nil {
			t.Fatal(err)
		}
		nonce := string(blob[:12])
		if lihat[nonce] {
			t.Fatalf("nonce berulang pada iterasi %d (GCM jadi tidak aman)", i)
		}
		lihat[nonce] = true
		if pertama == nil {
			pertama = blob
		} else if bytes.Equal(pertama, blob) {
			t.Fatal("ciphertext sama untuk plaintext sama: enkripsi deterministik")
		}
	}
}

// Mengubah SATU bit di posisi mana pun (nonce, isi, tag) harus ditolak.
func TestCipher_DataDiubahDitolak(t *testing.T) {
	c, _ := NewCipher(kunci(2))
	blob, _ := c.Encrypt([]byte("3271012345678901"), []byte("ctx"))
	for i := range blob {
		rusak := append([]byte(nil), blob...)
		rusak[i] ^= 0x01
		if _, err := c.Decrypt(rusak, []byte("ctx")); !errors.Is(err, ErrDekripsi) {
			t.Fatalf("byte ke-%d diubah tetapi dekripsi tidak ditolak (err=%v)", i, err)
		}
	}
}

func TestCipher_DekripsiGagal(t *testing.T) {
	c, _ := NewCipher(kunci(3))
	lain, _ := NewCipher(kunci(4))
	blob, _ := c.Encrypt([]byte("rahasia"), []byte("mustahik.nik"))

	tests := []struct {
		nama string
		c    *Cipher
		blob []byte
		aad  []byte
	}{
		{"kunci salah", lain, blob, []byte("mustahik.nik")},
		{"AAD salah (ciphertext NIK dipindah ke kolom lain)", c, blob, []byte("mustahik.no_hp")},
		{"AAD hilang", c, blob, nil},
		{"blob nil", c, nil, []byte("mustahik.nik")},
		{"blob kosong", c, []byte{}, []byte("mustahik.nik")},
		{"blob lebih pendek dari nonce+tag", c, make([]byte, 27), []byte("mustahik.nik")},
		{"blob terpotong", c, blob[:len(blob)-1], []byte("mustahik.nik")},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			got, err := tt.c.Decrypt(tt.blob, tt.aad)
			if !errors.Is(err, ErrDekripsi) || got != nil {
				t.Errorf("Decrypt = %q, %v; mau nil, ErrDekripsi", got, err)
			}
		})
	}
}

func TestNewCipher_PanjangKunci(t *testing.T) {
	for _, n := range []int{0, 16, 24, 31, 33, 64} {
		if _, err := NewCipher(make([]byte, n)); err == nil {
			t.Errorf("kunci %d byte diterima; harus tepat 32", n)
		}
	}
	if _, err := NewCipher(kunci(1)); err != nil {
		t.Errorf("kunci 32 byte ditolak: %v", err)
	}
}

func TestNewCipher_KunciDisalin(t *testing.T) {
	k := kunci(5)
	c, _ := NewCipher(k)
	blob, _ := c.Encrypt([]byte("x"), nil)
	for i := range k {
		k[i] = 0 // pemanggil menimpa kuncinya
	}
	if got, err := c.Decrypt(blob, nil); err != nil || string(got) != "x" {
		t.Errorf("Cipher ikut berubah saat slice kunci pemanggil diubah: %q, %v", got, err)
	}
}

func TestParseKey(t *testing.T) {
	asli := kunci(9)
	valid := base64.StdEncoding.EncodeToString(asli)
	pendek := base64.StdEncoding.EncodeToString(make([]byte, 16))

	if got, err := ParseKey(" " + valid + "\n"); err != nil || !bytes.Equal(got, asli) {
		t.Errorf("kunci valid (dengan spasi/newline) gagal: %v", err)
	}
	for nama, s := range map[string]string{"kosong": "", "bukan base64": "###", "16 byte": pendek} {
		_, err := ParseKey(s)
		if err == nil {
			t.Errorf("%s: harus error", nama)
		} else if s != "" && strings.Contains(err.Error(), s) {
			t.Errorf("%s: pesan error memuat isi kunci: %v", nama, err)
		}
	}
}

// ---------- hasher ----------

// RFC 4231 test case 6 (HMAC-SHA-256): kunci 131 byte 0xaa. Memastikan Hasher
// benar-benar HMAC-SHA256 standar, bukan sekadar "sesuatu yang konsisten".
func TestHasher_VektorRFC4231(t *testing.T) {
	h, err := NewHasher(bytes.Repeat([]byte{0xaa}, 131))
	if err != nil {
		t.Fatal(err)
	}
	const mau = "60e431591ee0b67f0d8a26aacbf5b77f8e0bc6213728c5140546040f0ee37f54"
	if got := h.Sum("Test Using Larger Than Block-Size Key - Hash Key First"); got != mau {
		t.Errorf("HMAC = %s, mau %s", got, mau)
	}
}

func TestHasher_Sifat(t *testing.T) {
	h1, _ := NewHasher(kunci(1))
	h2, _ := NewHasher(kunci(2))
	re := regexp.MustCompile(`^[0-9a-f]{64}$`)

	a := h1.Sum("3271012345678901")
	if !re.MatchString(a) {
		t.Errorf("keluaran bukan 64 hex huruf kecil: %q", a)
	}
	if a != h1.Sum("3271012345678901") {
		t.Error("tidak deterministik (UNIQUE/pencarian jadi mustahil)")
	}
	if a == h1.Sum("3271012345678902") {
		t.Error("NIK berbeda menghasilkan nilai sama")
	}
	if a == h2.Sum("3271012345678901") {
		t.Error("kunci berbeda menghasilkan nilai sama (kunci tidak berpengaruh)")
	}
	if strings.Contains(a, "3271012345678901") {
		t.Error("plaintext muncul di keluaran")
	}
}

func TestNewHasher_PanjangDanSalinanKunci(t *testing.T) {
	if _, err := NewHasher(make([]byte, 31)); err == nil {
		t.Error("kunci 31 byte diterima; minimal 32")
	}
	k := kunci(6)
	h, _ := NewHasher(k)
	sebelum := h.Sum("x")
	for i := range k {
		k[i] = 0
	}
	if h.Sum("x") != sebelum {
		t.Error("Hasher ikut berubah saat slice kunci pemanggil diubah")
	}
}
