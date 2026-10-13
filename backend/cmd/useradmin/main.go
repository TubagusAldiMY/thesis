// Command useradmin membuat akun dan me-reset password (FR-17). Tidak ada
// halaman kelola akun di aplikasi; hanya pengembang/pengelola server yang
// memegang akses ini.
//
//	go run ./cmd/useradmin buat --username admin --nama "Ahmad" --role admin
//	go run ./cmd/useradmin reset-password --username admin
//
// Password diminta interaktif (tidak tampil di layar, tidak masuk riwayat
// shell). Bila stdin bukan terminal (skrip), password dibaca dari baris
// pertama stdin.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"spkzakat/internal/domain"
	"spkzakat/internal/service"
	"spkzakat/internal/store"
)

func main() {
	if err := jalankan(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

const pakai = `pemakaian:
  useradmin buat --username U --nama "Nama Lengkap" --role admin|ketua
  useradmin reset-password --username U`

func jalankan(args []string) error {
	if len(args) == 0 {
		return errors.New(pakai)
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	username := fs.String("username", "", "username login")
	nama := fs.String("nama", "", "nama lengkap (tampil di aplikasi & laporan)")
	role := fs.String("role", "", "admin atau ketua")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL wajib diset (lihat .env.example)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()
	svc := service.New(st, nil, nil) // perintah akun tidak menyentuh NIK/No. HP

	switch args[0] {
	case "buat":
		pw, err := mintaPassword()
		if err != nil {
			return err
		}
		u, err := svc.BuatUser(ctx, *username, *nama, domain.Role(*role), pw)
		if err != nil {
			return err
		}
		fmt.Printf("akun dibuat: %s (%s)\n", u.Username, u.Role)
	case "reset-password":
		pw, err := mintaPassword()
		if err != nil {
			return err
		}
		if err := svc.ResetPassword(ctx, *username, pw); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return errors.New("username tidak ditemukan atau nonaktif")
			}
			return err
		}
		fmt.Println("password diganti; semua sesi akun ini dicabut")
	default:
		return errors.New(pakai)
	}
	return nil
}

// mintaPassword membaca password dua kali tanpa echo bila di terminal.
func mintaPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		baris, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && baris == "" {
			return "", errors.New("password tidak terbaca dari stdin")
		}
		return strings.TrimRight(baris, "\r\n"), nil
	}
	fmt.Fprint(os.Stderr, "Password (min. 12 karakter): ")
	a, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, "Ulangi password: ")
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if string(a) != string(b) {
		return "", errors.New("kedua password tidak sama")
	}
	return string(a), nil
}
