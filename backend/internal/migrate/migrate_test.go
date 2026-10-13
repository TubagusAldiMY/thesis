package migrate_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-sql-driver/mysql"

	"spkzakat/internal/migrate"
	"spkzakat/internal/testdb"
)

func buka(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func tabelAda(t *testing.T, db *sql.DB, nama string) bool {
	t.Helper()
	var n int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", nama).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func nomorError(err error) uint16 {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number
	}
	return 0
}

func TestUp_TerapkanLaluIdempoten(t *testing.T) {
	dsn := testdb.New(t)
	fsys := fstest.MapFS{
		"V001__a.sql": {Data: []byte("CREATE TABLE a (id INT);")},
		"V002__b.sql": {Data: []byte("CREATE TABLE b (id INT); CREATE TABLE c (id INT);")}, // multi-statement
	}
	ctx := context.Background()

	baru, err := migrate.Up(ctx, dsn, fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(baru) != 2 || baru[0] != "V001" || baru[1] != "V002" {
		t.Errorf("run pertama menerapkan %v, mau [V001 V002]", baru)
	}
	db := buka(t, dsn)
	for _, tb := range []string{"a", "b", "c", "schema_migrations"} {
		if !tabelAda(t, db, tb) {
			t.Errorf("tabel %q tidak ada", tb)
		}
	}

	baru, err = migrate.Up(ctx, dsn, fsys)
	if err != nil || len(baru) != 0 {
		t.Errorf("run kedua = %v, %v; mau tidak menerapkan apa pun", baru, err)
	}
}

// Menjaga aturan "migrasi tidak boleh diedit setelah deploy".
func TestUp_ChecksumBerubahDitolak(t *testing.T) {
	dsn := testdb.New(t)
	fsys := fstest.MapFS{"V001__a.sql": {Data: []byte("CREATE TABLE a (id INT);")}}
	if _, err := migrate.Up(context.Background(), dsn, fsys); err != nil {
		t.Fatal(err)
	}

	fsys["V001__a.sql"] = &fstest.MapFile{Data: []byte("CREATE TABLE a (id INT, nama TEXT);")}
	_, err := migrate.Up(context.Background(), dsn, fsys)
	if !errors.Is(err, migrate.ErrChecksum) {
		t.Fatalf("err = %v, mau ErrChecksum", err)
	}
}

func TestUp_NamaFileTidakValid(t *testing.T) {
	tests := []struct {
		nama string
		fsys fstest.MapFS
	}{
		{"nomor bukan angka", fstest.MapFS{"Vx__a.sql": {Data: []byte("SELECT 1;")}}},
		{"tanpa dua underscore", fstest.MapFS{"V001_a.sql": {Data: []byte("SELECT 1;")}}},
		{"huruf besar di deskripsi", fstest.MapFS{"V001__Awal.sql": {Data: []byte("SELECT 1;")}}},
		{"nomor kembar", fstest.MapFS{
			"V001__a.sql": {Data: []byte("SELECT 1;")},
			"V1__b.sql":   {Data: []byte("SELECT 1;")},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			_, err := migrate.Up(context.Background(), testdb.New(t), tt.fsys)
			if !errors.Is(err, migrate.ErrNamaFile) {
				t.Fatalf("err = %v, mau ErrNamaFile", err)
			}
		})
	}
}

// V10 harus jalan SETELAH V2 (urut angka), bukan urut abjad (V10 < V2).
func TestUp_UrutanMenurutNomor(t *testing.T) {
	dsn := testdb.New(t)
	fsys := fstest.MapFS{
		"V10__isi.sql":  {Data: []byte("INSERT INTO t VALUES (1);")}, // butuh tabel dari V2
		"V2__tabel.sql": {Data: []byte("CREATE TABLE t (id INT);")},
	}
	baru, err := migrate.Up(context.Background(), dsn, fsys)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(baru) != 2 || baru[0] != "V2" || baru[1] != "V10" {
		t.Errorf("urutan = %v, mau [V2 V10]", baru)
	}
}

func TestUp_SQLGagalTidakDicatat(t *testing.T) {
	dsn := testdb.New(t)
	fsys := fstest.MapFS{
		"V001__ok.sql":    {Data: []byte("CREATE TABLE a (id INT);")},
		"V002__rusak.sql": {Data: []byte("CREATE TABLEE salah;")},
	}
	baru, err := migrate.Up(context.Background(), dsn, fsys)
	if err == nil {
		t.Fatal("mau error untuk SQL rusak")
	}
	if len(baru) != 1 || baru[0] != "V001" {
		t.Errorf("diterapkan sebelum gagal = %v, mau [V001]", baru)
	}
	var n int
	if err := buka(t, dsn).QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 'V002'").Scan(&n); err != nil || n != 0 {
		t.Errorf("V002 tercatat padahal gagal (n=%d, err=%v)", n, err)
	}
}

// DSN rusak harus ditolak tanpa membocorkan isi DSN (bisa memuat password).
func TestUp_DSNRusakTidakBocor(t *testing.T) {
	const rahasia = "p4ssw0rd-rahasia"
	_, err := migrate.Up(context.Background(), "root:"+rahasia+"@tcp(host-tidak-lengkap", fstest.MapFS{})
	if err == nil {
		t.Fatal("mau error untuk DSN rusak")
	}
	if strings.Contains(err.Error(), rahasia) {
		t.Errorf("pesan error membocorkan kredensial: %q", err.Error())
	}
}

// Smoke/sanity migrasi ASLI: semua tabel rancangan ada dan seed kriteria benar.
func TestMigrasiAsli_TabelDanSeed(t *testing.T) {
	db := buka(t, testdb.Migrated(t))

	tabel := []string{
		"users", "sessions", "mustahik", "kriteria", "periode", "periode_kandidat",
		"penilaian", "aras_run", "aras_hasil", "persetujuan", "penyaluran", "audit_logs",
		"schema_migrations",
	}
	for _, tb := range tabel {
		if !tabelAda(t, db, tb) {
			t.Errorf("tabel %q tidak ada setelah migrasi", tb)
		}
	}

	rows, err := db.Query("SELECT kode, tipe, CAST(bobot AS CHAR) FROM kriteria ORDER BY urutan")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	mau := [][3]string{
		{"C1", "cost", "0.3000"}, {"C2", "benefit", "0.2000"}, {"C3", "benefit", "0.2000"},
		{"C4", "benefit", "0.1500"}, {"C5", "benefit", "0.1500"},
	}
	var got [][3]string
	for rows.Next() {
		var r [3]string
		if err := rows.Scan(&r[0], &r[1], &r[2]); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	if len(got) != len(mau) {
		t.Fatalf("seed kriteria = %v, mau %v", got, mau)
	}
	for i := range mau {
		if got[i] != mau[i] {
			t.Errorf("kriteria %d = %v, mau %v", i, got[i], mau[i])
		}
	}
	var total string
	if err := db.QueryRow("SELECT CAST(SUM(bobot) AS CHAR) FROM kriteria").Scan(&total); err != nil || total != "1.0000" {
		t.Errorf("Σ bobot = %q (err %v), mau 1.0000", total, err)
	}
}

// Constraint di level database adalah garis pertahanan terakhir di belakang validasi aplikasi.
func TestMigrasiAsli_Constraint(t *testing.T) {
	db := buka(t, testdb.Migrated(t))
	exec := func(q string, args ...any) error { _, err := db.Exec(q, args...); return err }

	if err := exec("INSERT INTO users (id, username, password_hash, nama, role) VALUES ('u1','adm','x','Admin','admin')"); err != nil {
		t.Fatal(err)
	}
	if err := exec("INSERT INTO mustahik (id, nama, nik_enc, nik_hash, alamat, asnaf) VALUES ('m1','A','e','h1','jl','miskin')"); err != nil {
		t.Fatal(err)
	}
	if err := exec("INSERT INTO periode (id, nama, jenis, dana_tersedia, nominal_per_penerima, created_by) VALUES ('p1','P','zakat',3000000,1000000,'u1')"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		nama  string
		sql   string
		nomor uint16
	}{
		{"CHECK dana_tersedia > 0", "INSERT INTO periode (nama, jenis, dana_tersedia, nominal_per_penerima, created_by) VALUES ('P2','zakat',0,1000000,'u1')", 3819},
		{"CHECK bobot <= 1", "INSERT INTO kriteria (kode, nama, tipe, bobot, urutan) VALUES ('C9','x','cost',1.5,9)", 3819},
		{"UNIQUE nik_hash", "INSERT INTO mustahik (nama, nik_enc, nik_hash, alamat, asnaf) VALUES ('B','e','h1','jl','fakir')", 1062},
		{"UNIQUE kode kriteria", "INSERT INTO kriteria (kode, nama, tipe, bobot, urutan) VALUES ('C1','dobel','cost',0.1,9)", 1062},
		{"FK penilaian harus kandidat", "INSERT INTO penilaian (periode_id, mustahik_id, kriteria_id, nilai) SELECT 'p1','m1',id,5 FROM kriteria LIMIT 1", 1452},
		{"ENUM tipe", "INSERT INTO kriteria (kode, nama, tipe, bobot, urutan) VALUES ('C8','x','lainnya',0.1,8)", 1265},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			err := exec(tt.sql)
			if got := nomorError(err); got != tt.nomor {
				t.Fatalf("error MySQL = %d (%v), mau %d", got, err, tt.nomor)
			}
		})
	}
}
