package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"spkzakat/internal/crypto"
	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// Aturan sesi & penguncian (docs/04 bagian 6, FR-01).
const (
	DurasiSesi      = 8 * time.Hour    // cookie & baris sessions kedaluwarsa setelah ini
	MaksGagal       = 5                // gagal berturut-turut sebelum dikunci
	DurasiKunci     = 15 * time.Minute // lama terkunci; juga jendela penghitung gagal
	minPassword     = 12
	maksPassword    = 128 // Argon2 mahal; password super panjang = celah DoS
	maksUsername    = 50
	maksPembatas    = 10_000
	pesanLoginGagal = "Username atau password salah"
)

var reUsername = regexp.MustCompile(`^[a-z0-9._-]{3,50}$`)

// LoginGagalError = username/password salah. Pesannya SAMA untuk username
// yang tidak terdaftar, supaya tidak bisa dipakai menebak akun.
type LoginGagalError struct{ SisaPercobaan int }

func (e *LoginGagalError) Error() string { return pesanLoginGagal }

// TerkunciError = terlalu banyak gagal; coba lagi setelah RetryAfter (→ 429).
type TerkunciError struct{ RetryAfter time.Duration }

func (e *TerkunciError) Error() string {
	return fmt.Sprintf("Terlalu banyak percobaan gagal. Coba lagi dalam %d menit", int(e.RetryAfter.Minutes())+1)
}

// pembatas menghitung kegagalan per kunci (username+IP) di memori proses.
//
// ponytail: hilang saat restart dan hanya berlaku untuk 1 instance server.
// Cukup untuk satu masjid; pindah ke tabel login_attempts bila perlu
// beberapa instance.
type pembatas struct {
	mu   sync.Mutex
	data map[string]*catatanGagal
	now  func() time.Time
}

type catatanGagal struct {
	gagal          int
	terakhir       time.Time
	terkunciSampai time.Time
}

func pembatasBaru() *pembatas {
	return &pembatas{data: map[string]*catatanGagal{}, now: time.Now}
}

// terkunci mengembalikan sisa waktu kunci (0 = tidak terkunci).
func (p *pembatas) terkunci(kunci string) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c := p.data[kunci]; c != nil {
		if sisa := c.terkunciSampai.Sub(p.now()); sisa > 0 {
			return sisa
		}
	}
	return 0
}

// gagal mencatat satu kegagalan dan mengembalikan sisa percobaan serta lama
// kunci (> 0 bila kegagalan ini membuatnya terkunci). Kegagalan yang lebih
// tua dari DurasiKunci tidak dihitung lagi.
func (p *pembatas) gagal(kunci string) (sisa int, kunciSelama time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	kedaluwarsa := func(c *catatanGagal) bool {
		return now.Sub(c.terakhir) > DurasiKunci && !c.terkunciSampai.After(now)
	}
	if len(p.data) >= maksPembatas { // cegah peta tumbuh tanpa batas
		for k, c := range p.data {
			if kedaluwarsa(c) {
				delete(p.data, k)
			}
		}
	}
	c := p.data[kunci]
	if c == nil || kedaluwarsa(c) {
		c = &catatanGagal{}
		p.data[kunci] = c
	}
	c.gagal++
	c.terakhir = now
	if c.gagal >= MaksGagal {
		c.gagal = 0
		c.terkunciSampai = now.Add(DurasiKunci)
		return 0, DurasiKunci
	}
	return MaksGagal - c.gagal, 0
}

// sukses menghapus catatan gagal (login berhasil = hitungan mulai dari nol).
func (p *pembatas) sukses(kunci string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.data, kunci)
}

// hashToken = SHA-256 heksadesimal; hanya ini yang disimpan di tabel sessions.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// hashPalsu dipakai saat username tidak ada: verifikasi tetap dijalankan
// supaya waktu respons tidak membedakan "username salah" dari "password salah".
var (
	hashPalsuOnce sync.Once
	hashPalsu     string
)

func hashPembanding() string {
	hashPalsuOnce.Do(func() {
		hashPalsu, _ = crypto.HashPassword("pembanding-waktu-bukan-password")
	})
	return hashPalsu
}

// LoginHasil = token sesi (untuk cookie, JANGAN dicatat) + pemiliknya.
type LoginHasil struct {
	Token string
	User  domain.User
}

// Login memverifikasi username & password (FR-01). Error:
//   - *TerkunciError bila kunci username+IP sedang aktif atau baru saja terpicu;
//   - *LoginGagalError (dengan sisa percobaan) bila salah;
//   - *domain.ValidationError bila isian kosong/terlalu panjang.
//
// Setiap percobaan dicatat di audit log (LOGIN / LOGIN_GAGAL) tanpa password.
func (s *Service) Login(ctx context.Context, username, password, ip string) (LoginHasil, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	g := galat{}
	if username == "" || len(username) > maksUsername {
		g.tambah("username", "Username wajib diisi")
	}
	if password == "" || len(password) > maksPassword {
		g.tambah("password", "Password wajib diisi")
	}
	if err := g.err(); err != nil {
		return LoginHasil{}, err
	}

	kunci := username + "|" + ip
	if sisa := s.batas.terkunci(kunci); sisa > 0 {
		return LoginHasil{}, &TerkunciError{RetryAfter: sisa}
	}

	u, err := s.st.GetUserByUsername(ctx, username)
	hash := u.PasswordHash
	if errors.Is(err, domain.ErrNotFound) {
		hash = hashPembanding()
	} else if err != nil {
		return LoginHasil{}, fmt.Errorf("service.Login: %w", err)
	}
	cocok, verr := crypto.VerifyPassword(hash, password)
	if verr != nil && u.ID != "" {
		return LoginHasil{}, fmt.Errorf("service.Login: hash password akun rusak: %w", verr)
	}
	if !cocok || u.ID == "" {
		s.catatDiam(ctx, domain.Audit{ActorID: u.ID, Action: domain.AksiLoginGagal, Resource: "auth", IP: ip})
		sisa, kunciSelama := s.batas.gagal(kunci)
		if kunciSelama > 0 {
			return LoginHasil{}, &TerkunciError{RetryAfter: kunciSelama}
		}
		return LoginHasil{}, &LoginGagalError{SisaPercobaan: sisa}
	}

	s.batas.sukses(kunci)
	if err := s.st.DeleteExpiredSessions(ctx); err != nil {
		return LoginHasil{}, fmt.Errorf("service.Login: %w", err)
	}
	token, err := tokenAcak()
	if err != nil {
		return LoginHasil{}, fmt.Errorf("service.Login: %w", err)
	}
	if err := s.st.CreateSession(ctx, u.ID, hashToken(token), time.Now().Add(DurasiSesi)); err != nil {
		return LoginHasil{}, fmt.Errorf("service.Login: %w", err)
	}
	s.catatDiam(ctx, domain.Audit{ActorID: u.ID, Action: domain.AksiLogin, Resource: "auth", ResourceID: u.ID, IP: ip})
	return LoginHasil{Token: token, User: u}, nil
}

// tokenAcak = 32 byte acak (256 bit) dalam base64 URL-safe.
func tokenAcak() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token acak: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Sesi mengembalikan pemilik token; ErrNotFound bila token tidak dikenal,
// kedaluwarsa, atau akunnya nonaktif.
func (s *Service) Sesi(ctx context.Context, token string) (domain.User, error) {
	if token == "" || len(token) > 100 {
		return domain.User{}, domain.ErrNotFound
	}
	u, err := s.st.GetUserBySession(ctx, hashToken(token))
	if err != nil {
		return domain.User{}, fmt.Errorf("service.Sesi: %w", err)
	}
	return u, nil
}

// Logout mencabut sesi di database (bukan hanya menghapus cookie, OWASP A07).
func (s *Service) Logout(ctx context.Context, u domain.User, token, ip string) error {
	if err := s.st.DeleteSession(ctx, hashToken(token)); err != nil {
		return fmt.Errorf("service.Logout: %w", err)
	}
	s.catatDiam(ctx, domain.Audit{ActorID: u.ID, Action: domain.AksiLogout, Resource: "auth", ResourceID: u.ID, IP: ip})
	return nil
}

// validasiPasswordBaru: ≥ 12 karakter, ≤ 128 byte.
func validasiPasswordBaru(baru string) string {
	if panjangRune(baru) < minPassword {
		return fmt.Sprintf("Password minimal %d karakter", minPassword)
	}
	if len(baru) > maksPassword {
		return fmt.Sprintf("Password maksimal %d karakter", maksPassword)
	}
	return ""
}

// GantiPassword mengganti password milik sendiri (FR-21): password lama wajib
// benar (salah dihitung seperti login gagal dan bisa mengunci), password baru
// ≥ 12 karakter dan berbeda dari yang lama. Semua sesi lain milik user
// dicabut; sesi yang sedang dipakai (token) tetap hidup.
func (s *Service) GantiPassword(ctx context.Context, u domain.User, token, lama, baru, ip string) error {
	kunci := "pw|" + u.ID + "|" + ip
	if sisa := s.batas.terkunci(kunci); sisa > 0 {
		return &TerkunciError{RetryAfter: sisa}
	}
	g := galat{}
	if lama == "" || len(lama) > maksPassword {
		g.tambah("password_lama", "Password lama wajib diisi")
	}
	if pesan := validasiPasswordBaru(baru); pesan != "" {
		g.tambah("password_baru", pesan)
	} else if baru == lama {
		g.tambah("password_baru", "Password baru harus berbeda dari password lama")
	}
	if err := g.err(); err != nil {
		return err
	}

	segar, err := s.st.GetUser(ctx, u.ID) // hash terbaru, bukan salinan saat sesi dibaca
	if err != nil {
		return fmt.Errorf("service.GantiPassword: %w", err)
	}
	cocok, err := crypto.VerifyPassword(segar.PasswordHash, lama)
	if err != nil {
		return fmt.Errorf("service.GantiPassword: %w", err)
	}
	if !cocok {
		sisa, kunciSelama := s.batas.gagal(kunci)
		if kunciSelama > 0 {
			return &TerkunciError{RetryAfter: kunciSelama}
		}
		g.tambah("password_lama", fmt.Sprintf("Password lama salah. Sisa percobaan: %d dari %d", sisa, MaksGagal))
		return g.err()
	}
	s.batas.sukses(kunci)

	hash, err := crypto.HashPassword(baru)
	if err != nil {
		return fmt.Errorf("service.GantiPassword: %w", err)
	}
	err = s.st.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.UpdatePasswordHash(ctx, u.ID, hash); err != nil {
			return err
		}
		return tx.DeleteSessionsUser(ctx, u.ID, hashToken(token))
	})
	if err != nil {
		return fmt.Errorf("service.GantiPassword: %w", err)
	}
	s.catatDiam(ctx, domain.Audit{ActorID: u.ID, Action: domain.AksiPasswordUbah, Resource: "users", ResourceID: u.ID, IP: ip})
	return nil
}

// BuatUser membuat akun Admin/Ketua (FR-17, dipanggil CLI useradmin).
func (s *Service) BuatUser(ctx context.Context, username, nama string, role domain.Role, password string) (domain.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	nama = strings.TrimSpace(nama)
	g := galat{}
	if !reUsername.MatchString(username) {
		g.tambah("username", "Username 3–50 karakter: huruf kecil, angka, titik, garis bawah, atau strip")
	}
	if n := panjangRune(nama); n < 2 || n > 100 {
		g.tambah("nama", "Nama harus 2–100 karakter")
	}
	if role != domain.RoleAdmin && role != domain.RoleKetua {
		g.tambah("role", "Peran harus admin atau ketua")
	}
	if pesan := validasiPasswordBaru(password); pesan != "" {
		g.tambah("password", pesan)
	}
	if err := g.err(); err != nil {
		return domain.User{}, err
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("service.BuatUser: %w", err)
	}
	u := domain.User{Username: username, Nama: nama, Role: role, PasswordHash: hash}
	if err := s.st.CreateUser(ctx, &u); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return domain.User{}, (galat{"username": "Username sudah dipakai"}).err()
		}
		return domain.User{}, fmt.Errorf("service.BuatUser: %w", err)
	}
	return u, nil
}

// ResetPassword mengganti password akun lewat CLI dan mencabut SEMUA sesinya.
func (s *Service) ResetPassword(ctx context.Context, username, password string) error {
	if pesan := validasiPasswordBaru(password); pesan != "" {
		return (galat{"password": pesan}).err()
	}
	u, err := s.st.GetUserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return fmt.Errorf("service.ResetPassword: %w", err)
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return fmt.Errorf("service.ResetPassword: %w", err)
	}
	return s.st.WithTx(ctx, func(tx *store.Store) error {
		if err := tx.UpdatePasswordHash(ctx, u.ID, hash); err != nil {
			return fmt.Errorf("service.ResetPassword: %w", err)
		}
		return tx.DeleteSessionsUser(ctx, u.ID, "")
	})
}
