package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"spkzakat/internal/crypto"
	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// Kelola pengguna oleh Ketua (Ketua Yayasan / Ketua DKM = peran "ketua").
// Pengecekan peran ada di lapisan HTTP; di sini hanya aturan bisnis:
//   - akun tidak pernah dihapus, hanya dinonaktifkan (audit log merujuk ke ID);
//   - Ketua tidak boleh menonaktifkan, menurunkan peran, atau mereset password
//     akunnya sendiri lewat halaman ini (mencegah terkunci sendiri; ganti
//     password sendiri tetap lewat Profil dengan password lama);
//   - selalu tersisa minimal satu Admin aktif, kalau tidak tidak ada yang bisa
//     mengisi data. Jumlah Ketua tidak perlu dijaga: pelakunya sendiri Ketua
//     aktif yang tidak boleh menyentuh akunnya.

// PenggunaView = akun untuk halaman pengurus. Tidak memuat hash password.
type PenggunaView struct {
	ID         string      `json:"id"`
	Username   string      `json:"username"`
	Nama       string      `json:"nama"`
	Role       domain.Role `json:"role"`
	Aktif      bool        `json:"aktif"`
	DibuatPada time.Time   `json:"dibuat_pada"`
}

func viewPengguna(u domain.User) PenggunaView {
	return PenggunaView{ID: u.ID, Username: u.Username, Nama: u.Nama, Role: u.Role, Aktif: u.Aktif, DibuatPada: u.CreatedAt}
}

// DaftarPengguna = semua akun, aktif lebih dulu.
func (s *Service) DaftarPengguna(ctx context.Context) ([]PenggunaView, error) {
	us, err := s.st.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarPengguna: %w", err)
	}
	out := make([]PenggunaView, 0, len(us))
	for _, u := range us {
		out = append(out, viewPengguna(u))
	}
	return out, nil
}

// AmbilPengguna = satu akun (aktif atau tidak).
func (s *Service) AmbilPengguna(ctx context.Context, id string) (PenggunaView, error) {
	u, err := s.st.GetUserKelola(ctx, id, false)
	if err != nil {
		return PenggunaView{}, fmt.Errorf("service.AmbilPengguna: %w", err)
	}
	return viewPengguna(u), nil
}

// BuatPengguna membuat akun baru dengan password awal yang ditetapkan Ketua.
// Pengguna sebaiknya menggantinya sendiri lewat Profil setelah masuk pertama.
func (s *Service) BuatPengguna(ctx context.Context, username, nama string, role domain.Role, password string) (PenggunaView, error) {
	u, err := s.BuatUser(ctx, username, nama, role, password)
	if err != nil {
		return PenggunaView{}, err
	}
	u.Aktif = true
	return viewPengguna(u), nil
}

func validasiProfil(nama string, role domain.Role) (string, error) {
	nama = strings.TrimSpace(nama)
	g := galat{}
	if n := panjangRune(nama); n < 2 || n > 100 {
		g.tambah("nama", "Nama harus 2–100 karakter")
	}
	if role != domain.RoleAdmin && role != domain.RoleKetua {
		g.tambah("role", "Peran harus admin atau ketua")
	}
	return nama, g.err()
}

// pastikanAdaAdminLain menolak bila `u` adalah satu-satunya Admin aktif.
// WAJIB dipanggil di dalam transaksi: baris Admin aktif dikunci.
func pastikanAdaAdminLain(ctx context.Context, tx *store.Store, u domain.User) error {
	if u.Role != domain.RoleAdmin || !u.Aktif {
		return nil
	}
	n, err := tx.KunciAdminAktif(ctx)
	if err != nil {
		return err
	}
	if n <= 1 {
		return domain.Konflik("Harus tersisa minimal satu Admin aktif. Buat atau aktifkan Admin lain lebih dulu")
	}
	return nil
}

// UbahPengguna mengubah nama dan peran. aktor = Ketua yang melakukannya.
func (s *Service) UbahPengguna(ctx context.Context, aktor domain.User, id, nama string, role domain.Role) (PenggunaView, error) {
	nama, err := validasiProfil(nama, role)
	if err != nil {
		return PenggunaView{}, err
	}
	var hasil domain.User
	err = s.st.WithTx(ctx, func(tx *store.Store) error {
		u, err := tx.GetUserKelola(ctx, id, true)
		if err != nil {
			return err
		}
		if u.ID == aktor.ID && role != u.Role {
			return domain.Konflik("Anda tidak bisa mengubah peran akun sendiri")
		}
		if role != u.Role {
			if err := pastikanAdaAdminLain(ctx, tx, u); err != nil {
				return err
			}
		}
		if err := tx.UpdateUserProfil(ctx, id, nama, role); err != nil {
			return err
		}
		u.Nama, u.Role = nama, role
		hasil = u
		return nil
	})
	if err != nil {
		return PenggunaView{}, fmt.Errorf("service.UbahPengguna: %w", err)
	}
	return viewPengguna(hasil), nil
}

// NonaktifkanPengguna mematikan akun dan mencabut SEMUA sesinya seketika:
// orang yang sedang masuk langsung terlempar, bukan menunggu sesinya habis.
func (s *Service) NonaktifkanPengguna(ctx context.Context, aktor domain.User, id string) (PenggunaView, error) {
	var hasil domain.User
	err := s.st.WithTx(ctx, func(tx *store.Store) error {
		u, err := tx.GetUserKelola(ctx, id, true)
		if err != nil {
			return err
		}
		if u.ID == aktor.ID {
			return domain.Konflik("Anda tidak bisa menonaktifkan akun sendiri")
		}
		if !u.Aktif {
			return domain.Konflik("Akun sudah nonaktif")
		}
		if err := pastikanAdaAdminLain(ctx, tx, u); err != nil {
			return err
		}
		if err := tx.SetUserAktif(ctx, id, false); err != nil {
			return err
		}
		if err := tx.DeleteSessionsUser(ctx, id, ""); err != nil {
			return err
		}
		u.Aktif = false
		hasil = u
		return nil
	})
	if err != nil {
		return PenggunaView{}, fmt.Errorf("service.NonaktifkanPengguna: %w", err)
	}
	return viewPengguna(hasil), nil
}

// AktifkanPengguna menghidupkan kembali akun nonaktif.
func (s *Service) AktifkanPengguna(ctx context.Context, id string) (PenggunaView, error) {
	var hasil domain.User
	err := s.st.WithTx(ctx, func(tx *store.Store) error {
		u, err := tx.GetUserKelola(ctx, id, true)
		if err != nil {
			return err
		}
		if u.Aktif {
			return domain.Konflik("Akun sudah aktif")
		}
		if err := tx.SetUserAktif(ctx, id, true); err != nil {
			return err
		}
		u.Aktif = true
		hasil = u
		return nil
	})
	if err != nil {
		return PenggunaView{}, fmt.Errorf("service.AktifkanPengguna: %w", err)
	}
	return viewPengguna(hasil), nil
}

// ResetPasswordPengguna mengganti password akun orang lain dan mencabut semua
// sesinya. Akun sendiri harus lewat Profil (butuh password lama).
func (s *Service) ResetPasswordPengguna(ctx context.Context, aktor domain.User, id, password string) error {
	if pesan := validasiPasswordBaru(password); pesan != "" {
		return (galat{"password": pesan}).err()
	}
	if id == aktor.ID {
		return domain.Konflik("Untuk mengganti password sendiri gunakan menu Profil & password")
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return fmt.Errorf("service.ResetPasswordPengguna: %w", err)
	}
	err = s.st.WithTx(ctx, func(tx *store.Store) error {
		if _, err := tx.GetUserKelola(ctx, id, true); err != nil {
			return err
		}
		if err := tx.UpdatePasswordHash(ctx, id, hash); err != nil {
			return err
		}
		return tx.DeleteSessionsUser(ctx, id, "")
	})
	if err != nil {
		return fmt.Errorf("service.ResetPasswordPengguna: %w", err)
	}
	return nil
}
