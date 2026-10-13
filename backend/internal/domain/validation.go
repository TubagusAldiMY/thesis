package domain

import (
	"errors"
	"sort"
	"strings"
)

// ErrValidation dikenali lewat errors.Is(err, domain.ErrValidation); lapisan
// HTTP memetakannya ke 422 dan memakai ValidationError.Fields sebagai pesan
// per kolom formulir.
var ErrValidation = errors.New("validasi gagal")

// ValidationError membawa pesan per field (nama field = nama kolom di UI,
// mis. "nik", "bobot", "kandidat"). Pesannya untuk MANUSIA (pengurus masjid),
// jadi berbahasa Indonesia dan tidak memuat data pribadi.
type ValidationError struct {
	Fields map[string]string
}

// Error menyusun pesan dengan urutan field tetap supaya log/test stabil.
func (e *ValidationError) Error() string {
	nama := make([]string, 0, len(e.Fields))
	for f := range e.Fields {
		nama = append(nama, f)
	}
	sort.Strings(nama)
	bagian := make([]string, len(nama))
	for i, f := range nama {
		bagian[i] = f + ": " + e.Fields[f]
	}
	return "validasi gagal: " + strings.Join(bagian, "; ")
}

// Is membuat errors.Is(err, ErrValidation) benar untuk semua ValidationError.
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }
