package store

import (
	"errors"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

// Sesi kedaluwarsa atau milik akun nonaktif tidak boleh dikenali (→ 401).
func TestSesi_KedaluwarsaDanAkunNonaktif(t *testing.T) {
	s := newStore(t)
	u := domain.User{Username: "admin1", PasswordHash: "x", Nama: "Admin", Role: domain.RoleAdmin}
	if err := s.CreateUser(bg, &u); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(bg, u.ID, "hidup", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(bg, u.ID, "basi", time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetUserBySession(bg, "hidup"); err != nil || got.ID != u.ID {
		t.Fatalf("sesi hidup = %+v, %v", got, err)
	}
	if _, err := s.GetUserBySession(bg, "basi"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("sesi kedaluwarsa: err = %v", err)
	}
	if err := s.DeleteExpiredSessions(bg); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(bg, `UPDATE users SET is_active = FALSE WHERE id = ?`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetUserBySession(bg, "hidup"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("akun nonaktif: err = %v", err)
	}
	if err := s.CreateUser(bg, &domain.User{Username: "admin1", PasswordHash: "x", Nama: "Lain", Role: domain.RoleKetua}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("username dobel: err = %v", err)
	}
}
