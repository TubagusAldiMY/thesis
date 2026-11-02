package service

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"spkzakat/internal/domain"
)

// Setup awal: membuat akun Ketua pertama lewat halaman /setup, tanpa terminal.
//
// Keamanannya: halaman ini hanya hidup selama tabel users KOSONG dan hanya bisa
// dipakai oleh orang yang memegang kode setup. Kode (128 bit acak) dibuat saat
// server start dan hanya dicetak ke log server, jadi yang bisa membacanya adalah
// orang yang punya akses ke server itu sendiri. Tanpa kode ini, siapa pun yang
// membuka /setup lebih dulu setelah deploy bisa menjadi Ketua. Kode hanya ada di
// memori: server restart (selama masih kosong) = kode baru; setelah akun pertama
// terbentuk kode dibuang dan halaman mati selamanya.

// SiapkanSetup dipanggil sekali saat server start. Bila belum ada akun, membuat
// kode setup baru dan mengembalikannya (untuk dicetak ke log). Bila sudah ada
// akun, mengembalikan "" dan setup tidak aktif.
func (s *Service) SiapkanSetup(ctx context.Context) (string, error) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	n, err := s.st.CountUsers(ctx)
	if err != nil {
		return "", fmt.Errorf("service.SiapkanSetup: %w", err)
	}
	if n > 0 {
		s.setupToken = ""
		return "", nil
	}
	kode, err := kodeAcak()
	if err != nil {
		return "", err
	}
	s.setupToken = kode
	return kode, nil
}

// SetupAktif = apakah /setup boleh dipakai sekarang: kode ada DAN belum ada akun.
// Jumlah akun dicek ke database setiap kali, jadi akun yang dibuat lewat CLI
// `useradmin` saat server berjalan juga langsung menutup setup.
func (s *Service) SetupAktif(ctx context.Context) (bool, error) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	return s.setupAktifLocked(ctx)
}

func (s *Service) setupAktifLocked(ctx context.Context) (bool, error) {
	if s.setupToken == "" {
		return false, nil
	}
	n, err := s.st.CountUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("service.SetupAktif: %w", err)
	}
	if n > 0 {
		s.setupToken = ""
		return false, nil
	}
	return true, nil
}

// BuatAkunPertama membuat akun Ketua pertama. Error:
//   - domain.ErrNotFound bila setup tidak aktif (sengaja sama dengan "tidak ada",
//     supaya halaman tidak membocorkan keadaan instalasi);
//   - *TerkunciError setelah 5 kode salah dari IP yang sama (15 menit);
//   - *domain.ValidationError untuk kode salah (field "token") atau isian tidak sah.
//
// Seluruhnya berjalan di bawah setupMu: dua permintaan bersamaan dengan kode yang
// benar tidak bisa sama-sama membuat akun (yang kedua melihat tabel sudah terisi).
func (s *Service) BuatAkunPertama(ctx context.Context, ip, token, username, nama, password string) (PenggunaView, error) {
	kunci := "setup|" + ip
	if sisa := s.batas.terkunci(kunci); sisa > 0 {
		return PenggunaView{}, &TerkunciError{RetryAfter: sisa}
	}

	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	aktif, err := s.setupAktifLocked(ctx)
	if err != nil {
		return PenggunaView{}, err
	}
	if !aktif {
		return PenggunaView{}, domain.ErrNotFound
	}

	// Perbandingan waktu-konstan; salah dihitung seperti login gagal (bisa mengunci).
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(token)), []byte(s.setupToken)) != 1 {
		sisa, kunciSelama := s.batas.gagal(kunci)
		if kunciSelama > 0 {
			return PenggunaView{}, &TerkunciError{RetryAfter: kunciSelama}
		}
		return PenggunaView{}, (galat{"token": fmt.Sprintf("Kode setup salah. Sisa percobaan: %d dari %d", sisa, MaksGagal)}).err()
	}
	s.batas.sukses(kunci)

	// Akun pertama selalu Ketua: hanya Ketua yang bisa membuat akun lain lewat aplikasi.
	u, err := s.BuatUser(ctx, username, nama, domain.RoleKetua, password)
	if err != nil {
		return PenggunaView{}, err
	}
	s.setupToken = "" // setup selesai; kode dibuang dan halaman mati
	u.Aktif = true
	s.catatDiam(ctx, domain.Audit{ActorID: u.ID, Action: domain.AksiSetupAwal, Resource: "users", ResourceID: u.ID, IP: ip})
	return viewPengguna(u), nil
}
