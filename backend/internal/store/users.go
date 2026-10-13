package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

const kolomUser = `u.id, u.username, u.password_hash, u.nama, u.role, u.is_active`

type userRow struct {
	ID           string `db:"id"`
	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
	Nama         string `db:"nama"`
	Role         string `db:"role"`
	Aktif        bool   `db:"is_active"`
}

func (r userRow) domain() domain.User {
	return domain.User{
		ID: r.ID, Username: r.Username, PasswordHash: r.PasswordHash,
		Nama: r.Nama, Role: domain.Role(r.Role), Aktif: r.Aktif,
	}
}

// CreateUser menyimpan akun baru dan mengisi u.ID. Username dobel →
// domain.ErrConflict (UNIQUE uq_users_username).
func (s *Store) CreateUser(ctx context.Context, u *domain.User) error {
	id := newID()
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, nama, role) VALUES (?, ?, ?, ?, ?)`,
		id, u.Username, u.PasswordHash, u.Nama, string(u.Role))
	if err != nil {
		return fmt.Errorf("store.CreateUser: %w", mapErr(err))
	}
	u.ID, u.Aktif = id, true
	return nil
}

// GetUserByUsername mencari akun AKTIF menurut username; ErrNotFound bila
// tidak ada, nonaktif, atau terhapus (pemanggil memperlakukan semuanya sama:
// "username atau password salah").
func (s *Store) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	var r userRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomUser+` FROM users u WHERE u.username = ? AND u.is_active = TRUE AND u.deleted_at IS NULL`, username)
	if err != nil {
		return domain.User{}, fmt.Errorf("store.GetUserByUsername: %w", mapErr(err))
	}
	return r.domain(), nil
}

// GetUser mengambil akun aktif menurut ID.
func (s *Store) GetUser(ctx context.Context, id string) (domain.User, error) {
	var r userRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomUser+` FROM users u WHERE u.id = ? AND u.is_active = TRUE AND u.deleted_at IS NULL`, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("store.GetUser: %w", mapErr(err))
	}
	return r.domain(), nil
}

// UpdatePasswordHash mengganti hash password akun.
func (s *Store) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ? AND deleted_at IS NULL`, hash, userID)
	if err != nil {
		return fmt.Errorf("store.UpdatePasswordHash: %w", mapErr(err))
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("store.UpdatePasswordHash: %w", domain.ErrNotFound)
	}
	return nil
}

// CreateSession menyimpan sesi baru. Yang disimpan hanya HASH token; token
// aslinya hanya ada di cookie pengguna, jadi bocornya isi tabel ini tidak
// bisa dipakai untuk masuk.
func (s *Store) CreateSession(ctx context.Context, userID, tokenHash string, kedaluwarsa time.Time) error {
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)`,
		newID(), userID, tokenHash, kedaluwarsa.UTC())
	if err != nil {
		return fmt.Errorf("store.CreateSession: %w", mapErr(err))
	}
	return nil
}

// GetUserBySession mengembalikan pemilik sesi yang BELUM kedaluwarsa dan
// akunnya masih aktif; selain itu ErrNotFound (→ 401).
func (s *Store) GetUserBySession(ctx context.Context, tokenHash string) (domain.User, error) {
	var r userRow
	err := sqlx.GetContext(ctx, s.q, &r,
		`SELECT `+kolomUser+`
		   FROM sessions se JOIN users u ON u.id = se.user_id
		  WHERE se.token_hash = ? AND se.expires_at > CURRENT_TIMESTAMP(3)
		    AND u.is_active = TRUE AND u.deleted_at IS NULL`, tokenHash)
	if err != nil {
		return domain.User{}, fmt.Errorf("store.GetUserBySession: %w", mapErr(err))
	}
	return r.domain(), nil
}

// DeleteSession mencabut satu sesi (logout). Tidak error bila sudah tidak ada.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	if _, err := s.q.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash); err != nil {
		return fmt.Errorf("store.DeleteSession: %w", mapErr(err))
	}
	return nil
}

// DeleteSessionsUser mencabut semua sesi milik user KECUALI sesi kecuali
// (hash token; "" = cabut semua). Dipakai saat ganti/reset password.
func (s *Store) DeleteSessionsUser(ctx context.Context, userID, kecuali string) error {
	if _, err := s.q.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ? AND token_hash <> ?`, userID, kecuali); err != nil {
		return fmt.Errorf("store.DeleteSessionsUser: %w", mapErr(err))
	}
	return nil
}

// DeleteExpiredSessions membersihkan sesi kedaluwarsa (dipanggil saat login,
// supaya tabel tidak tumbuh tanpa batas tanpa perlu job terjadwal).
func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	if _, err := s.q.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP(3)`); err != nil {
		return fmt.Errorf("store.DeleteExpiredSessions: %w", mapErr(err))
	}
	return nil
}
