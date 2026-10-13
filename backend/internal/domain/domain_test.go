package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestPeriode_Kuota(t *testing.T) {
	tests := []struct {
		nama          string
		dana, nominal int64
		mau           int
	}{
		{"pas dibagi (contoh dokumen)", 3_000_000, 1_000_000, 3},
		{"sisa dibuang (pembulatan ke bawah)", 3_500_000, 1_000_000, 3},
		{"dana kurang dari satu nominal", 999_999, 1_000_000, 0},
		{"nominal nol: tidak panik", 3_000_000, 0, 0},
		{"nominal negatif", 3_000_000, -1, 0},
		{"dana nol", 0, 1_000_000, 0},
		{"angka besar tidak overflow int", 9_000_000_000_000, 1_000, 9_000_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			p := Periode{DanaTersedia: tt.dana, NominalPerPenerima: tt.nominal}
			if got := p.Kuota(); got != tt.mau {
				t.Errorf("Kuota() = %d, mau %d", got, tt.mau)
			}
		})
	}
}

// ConflictError harus tetap dikenali sebagai ErrConflict (pemeriksa lama) dan
// membawa pesan untuk pengguna.
func TestKonflik(t *testing.T) {
	err := fmt.Errorf("lapisan: %w", Konflik("Periode berstatus %s", "selesai"))
	var ce *ConflictError
	if !errors.Is(err, ErrConflict) || !errors.As(err, &ce) || ce.Pesan != "Periode berstatus selesai" {
		t.Fatalf("Konflik tidak terbungkus benar: %v", err)
	}
}
