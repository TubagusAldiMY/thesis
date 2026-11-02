package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

const kolomUser = `u.id, u.username, u.password_hash, u.nama, u.role, u.is_active, u.created_at`

type userRow struct {
	ID           string    `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	Nama         string    `db:"nama"`
	Role         string    `db:"role"`
	Aktif        bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
}

func (r userRow) domain() domain.User {
	return domain.User{
		ID: r.ID, Username: r.Username, PasswordHash: r.PasswordHash,
		Nama: r.Nama, Role: domain.Role(r.Role), Aktif: r.Aktif, CreatedAt: r.CreatedAt,
	}
}

// CountUsers menghitung SEMUA akun (aktif maupun nonaktif). Nol = instalasi baru
// yang belum punya akun sama sekali, satu-satunya keadaan saat /setup boleh dipakai.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	if err := sqlx.GetContext(ctx, s.q, &n, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		return 0, fmt.Errorf("store.CountUsers: %w", mapErr(err))
	}
	return n, nil
}

// ListUsers mengembalikan SEMUA akun (aktif lebih dulu, lalu menurut nama) untuk
// halaman kelola pengguna. Hash password ikut terbaca di struct tetapi tidak
// pernah diteruskan ke klien (service memakai PenggunaView).
func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	var rows []userRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT `+kolomUser+` FROM users u WHERE u.deleted_at IS NULL ORDER BY u.is_active DESC, u.nama, u.username`)
	if err != nil {
		return nil, fmt.Errorf("store.ListUsers: %w", mapErr(err))
	}
	hasil := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, r.domain())
	}
	return hasil, nil
}

// GetUserKelola mengambil akun menurut ID TANPA melihat status aktif, supaya akun
// nonaktif tetap bisa dibuka, diubah, atau diaktifkan lagi. forUpdate mengunci
// barisnya sampai transaksi selesai.
func (s *Store) GetUserKelola(ctx context.Context, id string, forUpdate bool) (domain.User, error) {
	q := `SELECT ` + kolomUser + ` FROM users u WHERE u.id = ? AND u.deleted_at IS NULL`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	var r userRow
	if err := sqlx.GetContext(ctx, s.q, &r, q, id); err != nil {
		return domain.User{}, fmt.Errorf("store.GetUserKelola: %w", mapErr(err))
	}
	return r.domain(), nil
}

// UpdateUserProfil mengubah nama dan peran akun.
func (s *Store) UpdateUserProfil(ctx context.Context, id, nama string, role domain.Role) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE users SET nama = ?, role = ? WHERE id = ? AND deleted_at IS NULL`, nama, string(role), id)
	return cekSatu(res, err, "store.UpdateUserProfil", id)
}

// SetUserAktif menonaktifkan atau mengaktifkan kembali akun. Akun tidak pernah
// dihapus: audit log dan data lain merujuk ke ID-nya.
func (s *Store) SetUserAktif(ctx context.Context, id string, aktif bool) error {
	res, err := s.q.ExecContext(ctx,
		`UPDATE users SET is_active = ? WHERE id = ? AND deleted_at IS NULL`, aktif, id)
	return cekSatu(res, err, "store.SetUserAktif", id)
}

// KunciAdminAktif mengunci semua baris Admin aktif dan mengembalikan jumlahnya.
// Dipanggil di dalam transaksi sebelum menonaktifkan/menurunkan seorang Admin,
// supaya dua permintaan bersamaan tidak sama-sama menyisakan nol Admin aktif.
func (s *Store) KunciAdminAktif(ctx context.Context) (int, error) {
	var ids []string
	err := sqlx.SelectContext(ctx, s.q, &ids,
		`SELECT id FROM users WHERE role = 'admin' AND is_active = TRUE AND deleted_at IS NULL FOR UPDATE`)
	if err != nil {
		return 0, fmt.Errorf("store.KunciAdminAktif: %w", mapErr(err))
	}
	return len(ids), nil
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
