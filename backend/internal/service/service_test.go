package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"spkzakat/internal/crypto"
	"spkzakat/internal/domain"
	"spkzakat/internal/store"
	"spkzakat/internal/testdb"
)

var bg = context.Background()

const idHantu = "00000000-0000-4000-8000-000000000000"

// lingkungan = Service + akses langsung ke database uji (untuk menyiapkan
// keadaan dan memeriksa isi tabel mentah, mis. memastikan NIK terenkripsi).
type lingkungan struct {
	svc *Service
	st  *store.Store
	db  *sql.DB
	enc *crypto.Cipher
	idx *crypto.Hasher
}

func baru(t *testing.T) *lingkungan {
	t.Helper()
	dsn := testdb.Migrated(t)
	st, err := store.Open(bg, dsn)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	enc, err := crypto.NewCipher(bytes.Repeat([]byte{1}, crypto.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	idx, err := crypto.NewHasher(bytes.Repeat([]byte{2}, crypto.KeySize))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(); _ = st.Close() })
	return &lingkungan{svc: New(st, enc, idx), st: st, db: db, enc: enc, idx: idx}
}

func (e *lingkungan) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := e.db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// user menyisipkan satu user (idempoten) dan mengembalikan ID-nya, untuk
// kolom created_by periode.
func (e *lingkungan) user(t *testing.T) string {
	t.Helper()
	id := "11111111-1111-4111-8111-111111111111"
	e.exec(t, `INSERT IGNORE INTO users (id, username, password_hash, nama, role) VALUES (?, 'admin', 'x', 'Admin', 'admin')`, id)
	return id
}

// masukanSah = isian mustahik yang valid; ubah lewat fungsi opsional.
func masukanSah(ubah ...func(*MustahikInput)) MustahikInput {
	in := MustahikInput{
		Nama: "Ahmad Fauzi", NIK: "3271012345678901", NoHP: "081234567890",
		Alamat: "Jl. Mawar No. 1", Asnaf: domain.AsnafFakir, Aktif: true,
	}
	for _, f := range ubah {
		f(&in)
	}
	return in
}

// buatMustahik membuat mustahik valid; aktif=false langsung menonaktifkannya.
func (e *lingkungan) buatMustahik(t *testing.T, nama, nik string, asnaf domain.Asnaf, aktif bool) MustahikView {
	t.Helper()
	v, err := e.svc.BuatMustahik(bg, masukanSah(func(in *MustahikInput) {
		in.Nama, in.NIK, in.Asnaf, in.Aktif = nama, nik, asnaf, true
	}))
	if err != nil {
		t.Fatalf("buatMustahik %s: %v", nama, err)
	}
	if !aktif {
		in := masukanSah(func(in *MustahikInput) { in.Nama, in.NIK, in.NoHP, in.Asnaf, in.Aktif = nama, "", "", asnaf, false })
		if v, err = e.svc.UbahMustahik(bg, v.ID, in); err != nil {
			t.Fatalf("nonaktifkan %s: %v", nama, err)
		}
	}
	return v
}

// cekValidasi memastikan err adalah ValidationError yang memuat pesan untuk field.
func cekValidasi(t *testing.T, err error, field string) *domain.ValidationError {
	t.Helper()
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, mau ErrValidation", err)
	}
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("bukan *ValidationError: %v", err)
	}
	if field != "" && ve.Fields[field] == "" {
		t.Fatalf("tidak ada pesan untuk field %q; isi = %v", field, ve.Fields)
	}
	return ve
}

func samaNama(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
