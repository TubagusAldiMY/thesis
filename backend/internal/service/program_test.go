package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

func TestBuatSlug(t *testing.T) {
	tests := []struct{ judul, mau string }{
		{"Renovasi Atap Masjid", "renovasi-atap-masjid"},
		{"  Santunan Yatim — Ramadan 1448 H!  ", "santunan-yatim-ramadan-1448-h"},
		{"Wakaf   Al-Qur'an", "wakaf-al-qur-an"},
		{"!!!", "program"},
		{strings.Repeat("ab ", 50), strings.TrimRight(strings.Repeat("ab-", 27), "-")},
	}
	for _, tt := range tests {
		got := buatSlug(tt.judul)
		if got != tt.mau {
			t.Errorf("buatSlug(%q) = %q, mau %q", tt.judul, got, tt.mau)
		}
		if !reSlug.MatchString(got) {
			t.Errorf("buatSlug(%q) = %q tidak lolos reSlug", tt.judul, got)
		}
	}
}

func TestTipeGambar(t *testing.T) {
	tests := []struct {
		nama string
		isi  []byte
		mau  string
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0}, "image/jpeg"},
		{"png", []byte("\x89PNG\r\n\x1a\nIHDR"), "image/png"},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "image/webp"},
		{"svg ditolak", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), ""},
		{"gif ditolak", []byte("GIF89a......"), ""},
		{"riff bukan webp", []byte("RIFF\x00\x00\x00\x00WAVEfmt "), ""},
		{"kosong", nil, ""},
	}
	for _, tt := range tests {
		if got := tipeGambar(tt.isi); got != tt.mau {
			t.Errorf("%s: tipeGambar = %q, mau %q", tt.nama, got, tt.mau)
		}
	}
}

// programSah = isian program yang valid; ubah lewat fungsi opsional.
func programSah(ubah ...func(*ProgramInput)) ProgramInput {
	in := ProgramInput{
		Judul: "Renovasi Atap Masjid", Ringkasan: "Atap bocor di sisi utara perlu diganti sebelum musim hujan.",
		Cerita: "Atap masjid bocor di beberapa titik sehingga jamaah terganggu saat hujan.", Target: 5_000_000,
		BatasWaktu: "2026-10-31", Status: domain.ProgramAktif,
	}
	for _, f := range ubah {
		f(&in)
	}
	return in
}

func TestProgram_Validasi(t *testing.T) {
	e := baruDonasi(t) // jam tetap 16/10/2026
	admin := e.user(t)
	tests := []struct {
		nama  string
		ubah  func(*ProgramInput)
		field string
	}{
		{"judul pendek", func(i *ProgramInput) { i.Judul = "Atap" }, "judul"},
		{"ringkasan kosong", func(i *ProgramInput) { i.Ringkasan = " " }, "ringkasan"},
		{"cerita pendek", func(i *ProgramInput) { i.Cerita = "terlalu pendek" }, "cerita"},
		{"target di bawah batas", func(i *ProgramInput) { i.Target = 99_999 }, "target"},
		{"target di atas batas", func(i *ProgramInput) { i.Target = 10_000_000_001 }, "target"},
		{"status asing", func(i *ProgramInput) { i.Status = "arsip" }, "status"},
		{"tanggal salah", func(i *ProgramInput) { i.BatasWaktu = "31-10-2026" }, "batas_waktu"},
		{"batas sudah lewat", func(i *ProgramInput) { i.BatasWaktu = "2026-10-15" }, "batas_waktu"},
	}
	for _, tt := range tests {
		_, err := e.svc.BuatProgram(bg, admin, programSah(tt.ubah))
		var ve *domain.ValidationError
		if !errors.As(err, &ve) || ve.Fields[tt.field] == "" {
			t.Errorf("%s: err = %v, mau galat field %q", tt.nama, err, tt.field)
		}
	}
	// Batas waktu hari ini masih sah (hari terakhir), dan tanpa batas juga sah.
	for _, b := range []string{"2026-10-16", ""} {
		if _, err := e.svc.BuatProgram(bg, admin, programSah(func(i *ProgramInput) { i.BatasWaktu = b })); err != nil {
			t.Errorf("batas %q: %v", b, err)
		}
	}
}

// Alur utama: program aktif menerima infak, angka kemajuan dan daftar donatur
// publik ikut berubah, donatur anonim tidak terlihat namanya.
func TestProgram_AlurDonasiDanTampilanPublik(t *testing.T) {
	e := baruDonasi(t)
	admin := e.user(t)
	p, err := e.svc.BuatProgram(bg, admin, programSah())
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "renovasi-atap-masjid" || !p.Terbuka || p.SisaHari == nil || *p.SisaHari != 15 {
		t.Fatalf("program baru = %+v", p)
	}

	// Judul sama → slug tetap unik (akhiran acak), program pertama tidak terganggu.
	p2, err := e.svc.BuatProgram(bg, admin, programSah())
	if err != nil || p2.Slug == p.Slug || !strings.HasPrefix(p2.Slug, p.Slug+"-") {
		t.Fatalf("slug kedua = %q, err %v", p2.Slug, err)
	}

	bayarProgram := func(nama string, jumlah int64, anonim bool) {
		t.Helper()
		pub, err := e.svc.BuatDonasiWeb(bg, inputWeb(func(i *DonasiWebInput) {
			i.Nama, i.Jenis, i.Jumlah, i.Program, i.Anonim = nama, domain.DonasiInfak, jumlah, p.Slug, anonim
		}))
		if err != nil {
			t.Fatalf("donasi %s: %v", nama, err)
		}
		if pub.ProgramJudul != p.Judul {
			t.Errorf("ProgramJudul publik = %q", pub.ProgramJudul)
		}
		e.now = e.now.Add(time.Minute) // urutan donatur ditentukan waktu bayar
		if _, err := e.bayar(t, e.idDariKode(t, pub.Kode), jumlah, "settlement"); err != nil {
			t.Fatal(err)
		}
	}
	bayarProgram("Bu Siti", 1_000_000, false)
	bayarProgram("Pak Rahasia", 1_500_000, true)
	// Donasi umum (tanpa program) tidak ikut terhitung di program.
	if _, err := e.svc.BuatDonasiWeb(bg, inputWeb()); err != nil {
		t.Fatal(err)
	}

	d, err := e.svc.AmbilProgramPublik(bg, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if d.Terkumpul != 2_500_000 || d.ProgramView.Donatur != 2 || d.Persen != 50 {
		t.Fatalf("kemajuan = terkumpul %d, donatur %d, persen %d", d.Terkumpul, d.ProgramView.Donatur, d.Persen)
	}
	var nama []string
	for _, x := range d.Donatur {
		nama = append(nama, x.Nama)
	}
	if got := strings.Join(nama, ","); got != "Hamba Allah,Bu Siti" {
		t.Fatalf("donatur publik = %s (nama asli donatur anonim tidak boleh tampil)", got)
	}

	// Pengurus tetap melihat nama asli dan bisa menyaring donasi per program.
	h, err := e.svc.DaftarDonasi(bg, DonasiQuery{Program: p.ID})
	if err != nil || h.Total != 2 {
		t.Fatalf("filter program: total %d, err %v", h.Total, err)
	}
	for _, v := range h.Items {
		if v.ProgramJudul != p.Judul || (v.Anonim && v.Nama != "Pak Rahasia") {
			t.Errorf("view pengurus = %+v", v)
		}
	}

	// Sudah ada donatur → tidak boleh kembali ke draf.
	if _, err := e.svc.UbahProgram(bg, p.ID, programSah(func(i *ProgramInput) { i.Status = domain.ProgramDraf })); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("kembali ke draf: err = %v, mau konflik", err)
	}
}

func TestProgram_AturanDonasiKeProgram(t *testing.T) {
	e := baruDonasi(t)
	admin := e.user(t)
	aktif, _ := e.svc.BuatProgram(bg, admin, programSah(func(i *ProgramInput) { i.BatasWaktu = "2026-10-20" }))
	draf, _ := e.svc.BuatProgram(bg, admin, programSah(func(i *ProgramInput) { i.Judul, i.Status = "Program Masih Draf", domain.ProgramDraf }))

	web := func(slug string, jenis domain.JenisDonasi) error {
		_, err := e.svc.BuatDonasiWeb(bg, inputWeb(func(i *DonasiWebInput) { i.Program, i.Jenis = slug, jenis }))
		return err
	}
	manual := func(id string) error {
		_, err := e.svc.CatatDonasiManual(bg, admin, inputManual(func(i *DonasiManualInput) { i.ProgramID = id }))
		return err
	}
	field := func(err error) string {
		var ve *domain.ValidationError
		if errors.As(err, &ve) {
			for f := range ve.Fields {
				return f
			}
		}
		return ""
	}

	tests := []struct {
		nama  string
		err   error
		field string
	}{
		{"zakat ke program ditolak", web(aktif.Slug, domain.DonasiZakatFitrah), "jenis"},
		{"program draf ditolak (web)", web(draf.Slug, domain.DonasiInfak), "program"},
		{"slug tidak ada", web("tidak-ada", domain.DonasiInfak), "program"},
		{"slug format asing", web("../../etc", domain.DonasiInfak), "program"},
		{"program draf ditolak (petugas)", manual(draf.ID), "program"},
		{"ID program tidak ada (petugas)", manual("00000000-0000-4000-8000-000000000000"), "program"},
	}
	for _, tt := range tests {
		if got := field(tt.err); got != tt.field {
			t.Errorf("%s: err = %v, mau galat field %q", tt.nama, tt.err, tt.field)
		}
	}
	if err := web(aktif.Slug, domain.DonasiInfak); err != nil {
		t.Errorf("infak ke program aktif: %v", err)
	}

	// Lewat batas waktu (21/10 WIB): muzaki ditolak, petugas tetap boleh mencatat setoran.
	e.now = time.Date(2026, 10, 20, 17, 30, 0, 0, time.UTC) // 21/10 00.30 WIB
	if field(web(aktif.Slug, domain.DonasiInfak)) != "program" {
		t.Error("donasi web setelah batas waktu seharusnya ditolak")
	}
	if err := manual(aktif.ID); err != nil {
		t.Errorf("petugas setelah batas waktu: %v", err)
	}
	if p, _ := e.svc.AmbilProgram(bg, aktif.ID); p.Terbuka || p.SisaHari == nil || *p.SisaHari != 0 {
		t.Errorf("setelah batas: terbuka %v, sisa %v", p.Terbuka, p.SisaHari)
	}
}

func TestProgram_DrafTidakPublikDanKabar(t *testing.T) {
	e := baruDonasi(t)
	admin := e.user(t)
	draf, _ := e.svc.BuatProgram(bg, admin, programSah(func(i *ProgramInput) { i.Status = domain.ProgramDraf }))
	lain, _ := e.svc.BuatProgram(bg, admin, programSah(func(i *ProgramInput) { i.Judul = "Santunan Anak Yatim" }))

	if _, err := e.svc.AmbilProgramPublik(bg, draf.Slug); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("draf publik: err = %v, mau not found", err)
	}
	if _, _, err := e.svc.AmbilGambarProgramPublik(bg, draf.Slug); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("gambar draf publik: err = %v, mau not found", err)
	}
	pub, _ := e.svc.DaftarProgramPublik(bg)
	if len(pub) != 1 || pub[0].ID != lain.ID {
		t.Errorf("daftar publik = %+v", pub)
	}
	if semua, _ := e.svc.DaftarProgram(bg); len(semua) != 2 {
		t.Errorf("daftar pengurus = %d program", len(semua))
	}

	// Kabar: validasi, hanya bisa dihapus lewat programnya sendiri.
	if _, err := e.svc.TambahKabar(bg, admin, lain.ID, "Ok", "isi"); !errors.Is(err, domain.ErrValidation) {
		t.Errorf("kabar pendek: err = %v", err)
	}
	k, err := e.svc.TambahKabar(bg, admin, lain.ID, "Santunan tahap 1", "Santunan diserahkan kepada 20 anak yatim.")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.TambahKabar(bg, admin, "00000000-0000-4000-8000-000000000000", "Judul kabar", "Isi kabar yang cukup."); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("kabar ke program tidak ada: err = %v", err)
	}
	if err := e.svc.HapusKabar(bg, draf.ID, k.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("hapus kabar lewat program lain: err = %v", err)
	}
	if d, _ := e.svc.AmbilProgramPublik(bg, lain.Slug); len(d.Kabar) != 1 || d.Kabar[0].Judul != "Santunan tahap 1" {
		t.Errorf("kabar publik = %+v", d.Kabar)
	}
	if err := e.svc.HapusKabar(bg, lain.ID, k.ID); err != nil {
		t.Fatal(err)
	}
	if d, _ := e.svc.AmbilProgram(bg, lain.ID); len(d.Kabar) != 0 {
		t.Errorf("kabar setelah dihapus = %+v", d.Kabar)
	}
}

func TestProgram_GambarSampul(t *testing.T) {
	e := baruDonasi(t)
	admin := e.user(t)
	p, _ := e.svc.BuatProgram(bg, admin, programSah())
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))

	tests := []struct {
		nama string
		isi  []byte
		sah  bool
	}{
		{"kosong", nil, false},
		{"lebih dari 2 MB", append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, GambarMaks)...), false},
		{"bukan gambar", []byte("<html>bukan gambar</html>"), false},
		{"png", png, true},
	}
	for _, tt := range tests {
		err := e.svc.SimpanGambarProgram(bg, p.ID, tt.isi)
		if (err == nil) != tt.sah {
			t.Errorf("%s: err = %v", tt.nama, err)
		}
	}
	isi, tipe, err := e.svc.AmbilGambarProgramPublik(bg, p.Slug)
	if err != nil || tipe != "image/png" || string(isi) != string(png) {
		t.Fatalf("gambar = %q %d byte, err %v", tipe, len(isi), err)
	}
	if v, _ := e.svc.AmbilProgram(bg, p.ID); !v.AdaGambar {
		t.Error("AdaGambar = false setelah unggah")
	}
	if err := e.svc.SimpanGambarProgram(bg, "00000000-0000-4000-8000-000000000000", png); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("program tidak ada: err = %v", err)
	}
}
