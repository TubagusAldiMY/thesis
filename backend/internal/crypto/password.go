// Package crypto membungkus tiga kebutuhan keamanan project ini dengan API
// sekecil mungkin, supaya pemakainya (service) tidak perlu mengurus detail:
//
//   - password: hash Argon2id (password.go)
//   - data pribadi (NIK, No. HP): enkripsi AES-256-GCM (cipher.go)
//   - pencarian/duplikat NIK tanpa membuka plaintext: HMAC-SHA256 (cipher.go)
//
// Paket ini murni (tanpa I/O) dan tidak pernah mencatat atau mengembalikan
// kunci/plaintext di pesan error.
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameter Argon2id mengikuti rekomendasi OWASP (m=64 MiB, t=3, p=4) —
// lihat rules/owasp-checklist.md A02. Parameter DISIMPAN di dalam string hash,
// jadi menaikkannya kelak tidak merusak hash lama.
const (
	argonMemoryKiB = 64 * 1024
	argonTime      = 3
	argonThreads   = 4
	argonSaltLen   = 16
	argonKeyLen    = 32
)

var (
	// ErrPasswordKosong: HashPassword menolak password kosong.
	ErrPasswordKosong = errors.New("crypto: password kosong")
	// ErrFormatHash: string hash bukan format PHC Argon2id yang valid/wajar.
	ErrFormatHash = errors.New("crypto: format hash password tidak valid")
)

var b64 = base64.RawStdEncoding

// HashPassword menghasilkan hash dalam format PHC:
//
//	$argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
//
// Salt 16 byte acak dibuat baru tiap panggilan, jadi dua hash dari password
// yang sama selalu berbeda. Kebijakan password (panjang minimal, dst.) ada di
// service, bukan di sini.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", ErrPasswordKosong
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("crypto: salt acak: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword memeriksa plain terhadap hash. Mengembalikan (false, nil)
// bila password salah, dan (false, ErrFormatHash) bila string hash rusak —
// pemanggil membedakan "password salah" dari "data hash di database rusak".
// Perbandingan memakai waktu konstan agar tidak bocor lewat selisih waktu.
func VerifyPassword(encoded, plain string) (bool, error) {
	p, salt, want, err := parseHash(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(plain), salt, p.time, p.memory, p.threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

// parseHash membaca parameter dari string hash dan MEMBATASI rentangnya.
// Hash datang dari database (tepercaya), tetapi tanpa batas, satu baris yang
// rusak/tertimpa dengan m=4 TiB akan membuat server kehabisan memori saat login.
func parseHash(encoded string) (argonParams, []byte, []byte, error) {
	var zero argonParams
	bagian := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, hash]
	if len(bagian) != 6 || bagian[0] != "" || bagian[1] != "argon2id" {
		return zero, nil, nil, ErrFormatHash
	}
	if bagian[2] != "v="+strconv.Itoa(argon2.Version) {
		return zero, nil, nil, fmt.Errorf("%w: versi tidak didukung", ErrFormatHash)
	}

	var p argonParams
	for _, kv := range strings.Split(bagian[3], ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return zero, nil, nil, ErrFormatHash
		}
		n, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return zero, nil, nil, ErrFormatHash
		}
		switch k {
		case "m":
			p.memory = uint32(n)
		case "t":
			p.time = uint32(n)
		case "p":
			if n > 255 {
				return zero, nil, nil, ErrFormatHash
			}
			p.threads = uint8(n)
		default:
			return zero, nil, nil, ErrFormatHash
		}
	}
	if p.threads < 1 || p.threads > 16 || p.time < 1 || p.time > 10 ||
		p.memory < 8*uint32(p.threads) || p.memory > 256*1024 {
		return zero, nil, nil, fmt.Errorf("%w: parameter di luar batas wajar", ErrFormatHash)
	}

	salt, err := b64.DecodeString(bagian[4])
	if err != nil || len(salt) < 8 {
		return zero, nil, nil, ErrFormatHash
	}
	hash, err := b64.DecodeString(bagian[5])
	if err != nil || len(hash) < 16 || len(hash) > 64 {
		return zero, nil, nil, ErrFormatHash
	}
	return p, salt, hash, nil
}
