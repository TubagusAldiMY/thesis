package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"

	"spkzakat/internal/domain"
	"spkzakat/internal/testdb"
)

var bg = context.Background()

// newStore membuka Store ke database uji yang sudah dimigrasi (dihapus otomatis).
func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(bg, testdb.Migrated(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// seedUser menyisipkan user langsung lewat SQL (store users belum ada) dan
// mengembalikan ID-nya, untuk kolom created_by pada periode.
func seedUser(t *testing.T, s *Store) string {
	t.Helper()
	id := newID()
	_, err := s.db.ExecContext(bg,
		`INSERT INTO users (id, username, password_hash, nama, role) VALUES (?, ?, 'x', 'Admin Uji', 'admin')`,
		id, "u-"+id[:8])
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	return id
}

func hashNIK(nama string) string {
	h := sha256.Sum256([]byte(nama))
	return hex.EncodeToString(h[:])
}

// buatMustahik membuat mustahik aktif; NIKEnc sengaja berisi byte 0x00/0xff
// untuk memastikan data biner utuh pulang-pergi.
func buatMustahik(t *testing.T, s *Store, nama string, asnaf domain.Asnaf) domain.Mustahik {
	t.Helper()
	m := domain.Mustahik{
		Nama: nama, NIKEnc: []byte{0x00, 0xff, 0x10}, NIKHash: hashNIK(nama),
		Alamat: "Jl. Uji No. 1", Asnaf: asnaf, Aktif: true,
	}
	if err := s.CreateMustahik(bg, &m); err != nil {
		t.Fatalf("buatMustahik %q: %v", nama, err)
	}
	return m
}

func TestNewID(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	lihat := map[string]bool{}
	for i := 0; i < 2000; i++ {
		id := newID()
		if !re.MatchString(id) {
			t.Fatalf("ID %q bukan UUID v4 (36 karakter, CHAR(36))", id)
		}
		if lihat[id] {
			t.Fatalf("ID kembar: %s", id)
		}
		lihat[id] = true
	}
}

func TestMapErr(t *testing.T) {
	tests := []struct {
		nama   string
		in     error
		mau    error // error domain yang diharapkan; nil = diteruskan apa adanya
		nilNil bool  // in nil → hasil nil
	}{
		{nama: "nil tetap nil", in: nil, nilNil: true},
		{nama: "ErrNoRows → NotFound", in: sql.ErrNoRows, mau: domain.ErrNotFound},
		{nama: "ErrNoRows terbungkus → NotFound", in: fmt.Errorf("x: %w", sql.ErrNoRows), mau: domain.ErrNotFound},
		{nama: "1062 duplikat → Conflict", in: &mysql.MySQLError{Number: 1062}, mau: domain.ErrConflict},
		{nama: "1451 masih dirujuk → Conflict", in: &mysql.MySQLError{Number: 1451}, mau: domain.ErrConflict},
		{nama: "1452 referensi tak ada → NotFound", in: &mysql.MySQLError{Number: 1452}, mau: domain.ErrNotFound},
		{nama: "error MySQL lain diteruskan", in: &mysql.MySQLError{Number: 1146}},
		{nama: "error biasa diteruskan", in: errors.New("boom")},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			got := mapErr(tt.in)
			switch {
			case tt.nilNil:
				if got != nil {
					t.Fatalf("mapErr(nil) = %v", got)
				}
			case tt.mau != nil:
				if !errors.Is(got, tt.mau) {
					t.Fatalf("mapErr = %v, mau Is %v", got, tt.mau)
				}
				// Error asli harus tetap bisa dicek (errors.As / Is).
				if !errors.Is(got, tt.in) {
					t.Errorf("error asli hilang dari rantai: %v", got)
				}
			default:
				if got != tt.in {
					t.Fatalf("mapErr = %v, mau diteruskan apa adanya (%v)", got, tt.in)
				}
			}
		})
	}
}

// Kegagalan konek tidak boleh membocorkan password dari DSN ke pesan error/log.
func TestOpen_TidakMembocorkanKredensial(t *testing.T) {
	const rahasia = "p4ssw0rd-rahasia"
	dsns := map[string]string{
		"DSN rusak":      "root:" + rahasia + "@tcp(host-tidak-lengkap",
		"server tak ada": "root:" + rahasia + "@tcp(127.0.0.1:1)/x?timeout=1s",
	}
	for nama, dsn := range dsns {
		t.Run(nama, func(t *testing.T) {
			_, err := Open(bg, dsn)
			if err == nil {
				t.Fatal("mau error")
			}
			if strings.Contains(err.Error(), rahasia) {
				t.Errorf("pesan error membocorkan kredensial: %q", err)
			}
		})
	}
}

// Pengaturan sesi yang diandalkan seluruh lapisan atas.
func TestOpen_PengaturanSesi(t *testing.T) {
	s := newStore(t)

	var zona string
	if err := s.db.GetContext(bg, &zona, "SELECT @@session.time_zone"); err != nil {
		t.Fatal(err)
	}
	if zona != "+00:00" {
		t.Errorf("time_zone sesi = %q, mau +00:00", zona)
	}

	// multiStatements harus MATI di koneksi aplikasi (batas dampak SQL injection).
	if _, err := s.db.ExecContext(bg, "SELECT 1; SELECT 2"); err == nil {
		t.Error("multi-statement diterima; seharusnya ditolak di koneksi aplikasi")
	}

	if err := s.Ping(bg); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestWithTx(t *testing.T) {
	s := newStore(t)
	galat := errors.New("gagal sengaja")

	t.Run("commit bila fn sukses", func(t *testing.T) {
		m := domain.Mustahik{Nama: "Commit", NIKEnc: []byte{1}, NIKHash: hashNIK("Commit"), Alamat: "a", Asnaf: domain.AsnafFakir, Aktif: true}
		if err := s.WithTx(bg, func(tx *Store) error { return tx.CreateMustahik(bg, &m) }); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetMustahik(bg, m.ID); err != nil {
			t.Errorf("data hilang setelah commit: %v", err)
		}
	})

	t.Run("rollback bila fn error", func(t *testing.T) {
		var id string
		err := s.WithTx(bg, func(tx *Store) error {
			m := domain.Mustahik{Nama: "Rollback", NIKEnc: []byte{1}, NIKHash: hashNIK("Rollback"), Alamat: "a", Asnaf: domain.AsnafFakir, Aktif: true}
			if err := tx.CreateMustahik(bg, &m); err != nil {
				return err
			}
			id = m.ID
			return galat
		})
		if !errors.Is(err, galat) {
			t.Fatalf("err = %v, mau galat asli diteruskan", err)
		}
		if _, err := s.GetMustahik(bg, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("data masih ada setelah rollback (err=%v)", err)
		}
	})

	t.Run("rollback bila panic", func(t *testing.T) {
		hash := hashNIK("Panic")
		func() {
			defer func() { _ = recover() }()
			_ = s.WithTx(bg, func(tx *Store) error {
				m := domain.Mustahik{Nama: "Panic", NIKEnc: []byte{1}, NIKHash: hash, Alamat: "a", Asnaf: domain.AsnafFakir, Aktif: true}
				if err := tx.CreateMustahik(bg, &m); err != nil {
					return err
				}
				panic("boom")
			})
		}()
		if _, err := s.FindMustahikByNIKHash(bg, hash); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("data masih ada setelah panic (err=%v)", err)
		}
	})

	t.Run("transaksi bersarang ditolak", func(t *testing.T) {
		err := s.WithTx(bg, func(tx *Store) error {
			return tx.WithTx(bg, func(*Store) error { return nil })
		})
		if err == nil {
			t.Error("mau error untuk transaksi bersarang")
		}
	})
}
