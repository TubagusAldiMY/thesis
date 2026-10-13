package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationError(t *testing.T) {
	var err error = &ValidationError{Fields: map[string]string{
		"nik":  "NIK harus 16 digit angka",
		"nama": "Nama minimal 2 karakter",
	}}

	if !errors.Is(err, ErrValidation) {
		t.Error("errors.Is(err, ErrValidation) harus true")
	}
	if errors.Is(err, ErrConflict) || errors.Is(err, ErrNotFound) {
		t.Error("tidak boleh dikenali sebagai error domain lain")
	}

	// Tetap dikenali setelah dibungkus (pola fmt.Errorf("...: %w")) dan bisa diambil kembali.
	dibungkus := fmt.Errorf("service.BuatMustahik: %w", err)
	var ve *ValidationError
	if !errors.Is(dibungkus, ErrValidation) || !errors.As(dibungkus, &ve) || ve.Fields["nik"] == "" {
		t.Errorf("gagal mengambil ValidationError dari error terbungkus: %v", dibungkus)
	}

	// Urutan field tetap (abjad) agar pesan stabil.
	const mau = "validasi gagal: nama: Nama minimal 2 karakter; nik: NIK harus 16 digit angka"
	if got := err.Error(); got != mau {
		t.Errorf("Error() = %q, mau %q", got, mau)
	}
}
