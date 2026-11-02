package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

// kegiatanSah = isian kegiatan yang valid; ubah lewat fungsi opsional.
func kegiatanSah(ubah ...func(*KegiatanInput)) KegiatanInput {
	in := KegiatanInput{
		Judul: "Peringatan Maulid Nabi 1448 H", Kategori: "maulid_nabi",
		Ringkasan:    "Pengajian akbar dan santunan anak yatim dalam rangka Maulid Nabi.",
		Isi:          "Insya Allah diadakan pengajian akbar bersama ustaz tamu, dilanjutkan santunan anak yatim.",
		TanggalAcara: "2026-10-25", JamAcara: "19:30", Lokasi: "Aula Masjid", Status: domain.KegiatanTerbit,
	}
	for _, f := range ubah {
		f(&in)
	}
	return in
}

func fieldGalat(err error) string {
	var ve *domain.ValidationError
	if errors.As(err, &ve) {
		for f := range ve.Fields {
			return f
		}
	}
	return ""
}

func TestKegiatan_Validasi(t *testing.T) {
	e := baruDonasi(t) // jam tetap 16/10/2026 WIB
	admin := e.user(t)
	tests := []struct {
		nama  string
		ubah  func(*KegiatanInput)
		field string
	}{
		{"judul pendek", func(i *KegiatanInput) { i.Judul = "Maul" }, "judul"},
		{"kategori asing", func(i *KegiatanInput) { i.Kategori = "konser" }, "kategori"},
		{"kategori kosong", func(i *KegiatanInput) { i.Kategori = "" }, "kategori"},
		{"ringkasan pendek", func(i *KegiatanInput) { i.Ringkasan = "singkat" }, "ringkasan"},
		{"isi pendek", func(i *KegiatanInput) { i.Isi = "terlalu pendek" }, "isi"},
		{"isi terlalu panjang", func(i *KegiatanInput) { i.Isi = strings.Repeat("a", 10_001) }, "isi"},
		{"tanggal salah", func(i *KegiatanInput) { i.TanggalAcara = "25-10-2026" }, "tanggal_acara"},
		{"jam salah", func(i *KegiatanInput) { i.JamAcara = "25:99" }, "jam_acara"},
		{"jam tanpa tanggal", func(i *KegiatanInput) { i.TanggalAcara = "" }, "jam_acara"},
		{"lokasi terlalu panjang", func(i *KegiatanInput) { i.Lokasi = strings.Repeat("x", 121) }, "lokasi"},
		{"status asing", func(i *KegiatanInput) { i.Status = "arsip" }, "status"},
	}
	for _, tt := range tests {
		if _, err := e.svc.BuatKegiatan(bg, admin, kegiatanSah(tt.ubah)); fieldGalat(err) != tt.field {
			t.Errorf("%s: err = %v, mau galat field %q", tt.nama, err, tt.field)
		}
	}
	// Artikel tanpa tanggal dan jam, serta tanggal lampau, sah.
	for _, ubah := range []func(*KegiatanInput){
		func(i *KegiatanInput) { i.TanggalAcara, i.JamAcara = "", "" },
		func(i *KegiatanInput) { i.TanggalAcara = "2025-01-01" },
	} {
		if _, err := e.svc.BuatKegiatan(bg, admin, kegiatanSah(ubah)); err != nil {
			t.Errorf("isian sah ditolak: %v", err)
		}
	}
}

// Alur utama: yang terbit tampil di publik, dipisah mendatang (terdekat dulu) dan
// terdahulu (terbaru dulu); draf dan yang dihapus tidak pernah tampil.
func TestKegiatan_TampilanPublik(t *testing.T) {
	e := baruDonasi(t) // hari ini 16/10/2026
	admin := e.user(t)
	buat := func(judul, tanggal, kategori string, status domain.StatusKegiatan) KegiatanView {
		t.Helper()
		v, err := e.svc.BuatKegiatan(bg, admin, kegiatanSah(func(i *KegiatanInput) {
			i.Judul, i.TanggalAcara, i.Kategori, i.Status = judul, tanggal, kategori, status
			if tanggal == "" {
				i.JamAcara = ""
			}
		}))
		if err != nil {
			t.Fatalf("buat %s: %v", judul, err)
		}
		return v
	}
	jauh := buat("Isra Miraj Besar", "2026-12-01", "isra_miraj", domain.KegiatanTerbit)
	dekat := buat("Maulid Nabi Pekan Depan", "2026-10-20", "maulid_nabi", domain.KegiatanTerbit)
	hariIni := buat("Kajian Hari Ini", "2026-10-16", "kajian", domain.KegiatanTerbit)
	lampauBaru := buat("Santunan Bulan Lalu", "2026-09-10", "santunan", domain.KegiatanTerbit)
	lampauLama := buat("Ramadan Tahun Lalu", "2026-03-01", "ramadan", domain.KegiatanTerbit)
	tanpaTanggal := buat("Artikel Tanpa Tanggal", "", "lainnya", domain.KegiatanTerbit)
	draf := buat("Masih Draf Rahasia", "2026-10-30", "kajian", domain.KegiatanDraf)

	nama := func(vs []KegiatanView) string {
		var s []string
		for _, v := range vs {
			s = append(s, v.Judul)
		}
		return strings.Join(s, " | ")
	}
	d, err := e.svc.DaftarKegiatanPublik(bg, "")
	if err != nil {
		t.Fatal(err)
	}
	if got, mau := nama(d.Mendatang), hariIni.Judul+" | "+dekat.Judul+" | "+jauh.Judul; got != mau {
		t.Errorf("mendatang = %q\nmau = %q", got, mau)
	}
	// Terdahulu: menurut tanggal acara (atau tanggal dibuat bila tanpa tanggal), terbaru dulu.
	// Artikel tanpa tanggal memakai tanggal dibuat (hari ini sungguhan, jauh setelah 2026-10), jadi paling atas.
	if got := nama(d.Terdahulu); !strings.Contains(got, lampauBaru.Judul+" | "+lampauLama.Judul) || !strings.Contains(got, tanpaTanggal.Judul) {
		t.Errorf("terdahulu = %q", got)
	}
	if strings.Contains(nama(d.Mendatang)+nama(d.Terdahulu), draf.Judul) {
		t.Error("draf bocor ke daftar publik")
	}
	if len(d.Kategori) != len(kategoriKegiatan) {
		t.Errorf("kategori = %d", len(d.Kategori))
	}

	// Filter kategori; kategori asing ditolak.
	if f, _ := e.svc.DaftarKegiatanPublik(bg, "kajian"); len(f.Mendatang) != 1 || len(f.Terdahulu) != 0 {
		t.Errorf("filter kajian = %d mendatang, %d terdahulu (draf tidak boleh ikut)", len(f.Mendatang), len(f.Terdahulu))
	}
	if _, err := e.svc.DaftarKegiatanPublik(bg, "konser"); fieldGalat(err) != "kategori" {
		t.Errorf("kategori asing: err = %v", err)
	}

	// Draf: halaman dan gambar 404 bagi publik; terbitkan → tampil; sembunyikan lagi → hilang.
	if _, err := e.svc.AmbilKegiatanPublik(bg, draf.Slug); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("draf publik: err = %v", err)
	}
	if _, err := e.svc.AmbilKegiatanPublik(bg, "../etc"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("slug asing: err = %v", err)
	}
	if _, err := e.svc.UbahKegiatan(bg, draf.ID, kegiatanSah(func(i *KegiatanInput) { i.Judul = draf.Judul })); err != nil {
		t.Fatal(err)
	}
	if v, err := e.svc.AmbilKegiatanPublik(bg, draf.Slug); err != nil || v.Judul != draf.Judul || v.KategoriLabel != "Maulid Nabi" {
		t.Errorf("setelah terbit: %+v, %v", v, err)
	}
	if _, err := e.svc.UbahKegiatan(bg, draf.ID, kegiatanSah(func(i *KegiatanInput) { i.Judul, i.Status = draf.Judul, domain.KegiatanDraf })); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.AmbilKegiatanPublik(bg, draf.Slug); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("setelah disembunyikan: err = %v", err)
	}

	// Slug tetap walau judul diubah; judul sama menghasilkan slug lain.
	v, _ := e.svc.UbahKegiatan(bg, dekat.ID, kegiatanSah(func(i *KegiatanInput) { i.Judul = "Judul Sudah Diganti Total" }))
	if v.Slug != dekat.Slug {
		t.Errorf("slug berubah: %q → %q", dekat.Slug, v.Slug)
	}
	kembar, err := e.svc.BuatKegiatan(bg, admin, kegiatanSah(func(i *KegiatanInput) { i.Judul = jauh.Judul }))
	if err != nil || kembar.Slug == jauh.Slug || !strings.HasPrefix(kembar.Slug, jauh.Slug+"-") {
		t.Errorf("slug kembar = %q, err %v", kembar.Slug, err)
	}

	// Hapus: hilang dari publik dan pengurus; hapus lagi → not found.
	if err := e.svc.HapusKegiatan(bg, jauh.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.AmbilKegiatanPublik(bg, jauh.Slug); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("setelah dihapus: err = %v", err)
	}
	if err := e.svc.HapusKegiatan(bg, jauh.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("hapus dua kali: err = %v", err)
	}
	if semua, _ := e.svc.DaftarKegiatan(bg); len(semua.Items) != 7 { // 7 awal + kembar - jauh
		t.Errorf("daftar pengurus = %d", len(semua.Items))
	}
}

func TestKegiatan_Gambar(t *testing.T) {
	e := baruDonasi(t)
	admin := e.user(t)
	k, _ := e.svc.BuatKegiatan(bg, admin, kegiatanSah())
	draf, _ := e.svc.BuatKegiatan(bg, admin, kegiatanSah(func(i *KegiatanInput) { i.Judul, i.Status = "Kegiatan Draf Bergambar", domain.KegiatanDraf }))
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100))

	for _, tt := range []struct {
		nama string
		isi  []byte
		sah  bool
	}{
		{"kosong", nil, false},
		{"lebih dari 2 MB", append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, GambarMaks)...), false},
		{"svg", []byte(`<svg onload="alert(1)"></svg>`), false},
		{"png", png, true},
	} {
		if err := e.svc.SimpanGambarKegiatan(bg, k.ID, tt.isi); (err == nil) != tt.sah {
			t.Errorf("%s: err = %v", tt.nama, err)
		}
	}
	isi, tipe, err := e.svc.AmbilGambarKegiatanPublik(bg, k.Slug)
	if err != nil || tipe != "image/png" || string(isi) != string(png) {
		t.Fatalf("gambar publik = %q %d byte, err %v", tipe, len(isi), err)
	}
	if v, _ := e.svc.AmbilKegiatan(bg, k.ID); !v.AdaGambar {
		t.Error("AdaGambar = false setelah unggah")
	}
	_ = e.svc.SimpanGambarKegiatan(bg, draf.ID, png)
	if _, _, err := e.svc.AmbilGambarKegiatanPublik(bg, draf.Slug); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("gambar draf publik: err = %v", err)
	}
	if err := e.svc.SimpanGambarKegiatan(bg, "00000000-0000-4000-8000-000000000000", png); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("kegiatan tidak ada: err = %v", err)
	}
}

func TestJadwalRutin(t *testing.T) {
	e := baruDonasi(t)
	sah := func(ubah ...func(*JadwalInput)) JadwalInput {
		in := JadwalInput{Judul: "Kajian Subuh", Pemateri: "Ustaz Fulan", Hari: 0, Jam: "05:30", Lokasi: "Masjid", Aktif: true}
		for _, f := range ubah {
			f(&in)
		}
		return in
	}
	for _, tt := range []struct {
		nama  string
		ubah  func(*JadwalInput)
		field string
	}{
		{"judul pendek", func(i *JadwalInput) { i.Judul = "Ka" }, "judul"},
		{"hari 7", func(i *JadwalInput) { i.Hari = 7 }, "hari"},
		{"hari negatif", func(i *JadwalInput) { i.Hari = -1 }, "hari"},
		{"jam salah", func(i *JadwalInput) { i.Jam = "5.30" }, "jam"},
		{"jam kosong", func(i *JadwalInput) { i.Jam = "" }, "jam"},
		{"pemateri panjang", func(i *JadwalInput) { i.Pemateri = strings.Repeat("a", 101) }, "pemateri"},
		{"keterangan panjang", func(i *JadwalInput) { i.Keterangan = strings.Repeat("a", 201) }, "keterangan"},
	} {
		if _, err := e.svc.BuatJadwal(bg, sah(tt.ubah)); fieldGalat(err) != tt.field {
			t.Errorf("%s: err = %v, mau galat field %q", tt.nama, err, tt.field)
		}
	}

	// Urutan: menurut hari (Ahad dulu) lalu jam.
	jumat, _ := e.svc.BuatJadwal(bg, sah(func(i *JadwalInput) { i.Judul, i.Hari, i.Jam = "Kajian Jumat Malam", 5, "20:00" }))
	ahadPagi, _ := e.svc.BuatJadwal(bg, sah())
	ahadSiang, _ := e.svc.BuatJadwal(bg, sah(func(i *JadwalInput) { i.Judul, i.Jam = "Kajian Dhuha", "09:00" }))
	if js, _ := e.svc.DaftarJadwalPublik(bg); len(js) != 3 || js[0].ID != ahadPagi.ID || js[1].ID != ahadSiang.ID || js[2].ID != jumat.ID || js[2].HariLabel != "Jumat" {
		t.Errorf("urutan publik = %+v", js)
	}

	// Nonaktif: hilang dari publik tetapi tetap ada di pengurus; hapus: hilang di keduanya.
	if _, err := e.svc.UbahJadwal(bg, ahadSiang.ID, sah(func(i *JadwalInput) { i.Judul, i.Jam, i.Aktif = "Kajian Dhuha", "09:00", false })); err != nil {
		t.Fatal(err)
	}
	if js, _ := e.svc.DaftarJadwalPublik(bg); len(js) != 2 {
		t.Errorf("publik setelah nonaktif = %d", len(js))
	}
	if js, _ := e.svc.DaftarJadwal(bg); len(js) != 3 {
		t.Errorf("pengurus setelah nonaktif = %d", len(js))
	}
	if err := e.svc.HapusJadwal(bg, jumat.ID); err != nil {
		t.Fatal(err)
	}
	if js, _ := e.svc.DaftarJadwal(bg); len(js) != 2 {
		t.Errorf("pengurus setelah hapus = %d", len(js))
	}
	if _, err := e.svc.UbahJadwal(bg, jumat.ID, sah()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("ubah yang sudah dihapus: err = %v", err)
	}
}

func TestPengumuman_MasaTayang(t *testing.T) {
	e := baruDonasi(t) // 16/10/2026 WIB
	sah := func(ubah ...func(*PengumumanInput)) PengumumanInput {
		in := PengumumanInput{Judul: "Kerja Bakti", Isi: "Kerja bakti membersihkan masjid Ahad pagi, mohon hadir.", Terbit: true}
		for _, f := range ubah {
			f(&in)
		}
		return in
	}
	for _, tt := range []struct {
		nama  string
		ubah  func(*PengumumanInput)
		field string
	}{
		{"judul pendek", func(i *PengumumanInput) { i.Judul = "Ke" }, "judul"},
		{"isi pendek", func(i *PengumumanInput) { i.Isi = "pendek" }, "isi"},
		{"isi panjang", func(i *PengumumanInput) { i.Isi = strings.Repeat("a", 501) }, "isi"},
		{"tanggal salah", func(i *PengumumanInput) { i.TampilSampai = "besok" }, "tampil_sampai"},
		{"masa tayang sudah lewat", func(i *PengumumanInput) { i.TampilSampai = "2026-10-15" }, "tampil_sampai"},
	} {
		if _, err := e.svc.BuatPengumuman(bg, sah(tt.ubah)); fieldGalat(err) != tt.field {
			t.Errorf("%s: err = %v, mau galat field %q", tt.nama, err, tt.field)
		}
	}

	selamanya, _ := e.svc.BuatPengumuman(bg, sah())
	hariIni, _ := e.svc.BuatPengumuman(bg, sah(func(i *PengumumanInput) { i.Judul, i.TampilSampai = "Berakhir Hari Ini", "2026-10-16" }))
	besok, _ := e.svc.BuatPengumuman(bg, sah(func(i *PengumumanInput) { i.Judul, i.TampilSampai = "Berakhir Besok", "2026-10-17" }))
	disembunyikan, _ := e.svc.BuatPengumuman(bg, sah(func(i *PengumumanInput) { i.Judul, i.Terbit = "Disembunyikan", false }))
	if !selamanya.Tayang || !hariIni.Tayang || disembunyikan.Tayang {
		t.Fatalf("tayang: selamanya=%v hariIni=%v disembunyikan=%v", selamanya.Tayang, hariIni.Tayang, disembunyikan.Tayang)
	}

	ids := func(ps []PengumumanView) map[string]bool {
		m := map[string]bool{}
		for _, p := range ps {
			m[p.ID] = true
		}
		return m
	}
	if m := ids(mustPengumuman(e.svc.DaftarPengumumanPublik(bg))); len(m) != 3 || m[disembunyikan.ID] {
		t.Errorf("publik hari ini = %v", m)
	}

	// Hari berganti (17/10 WIB): batas termasuk, jadi "Berakhir Hari Ini" sudah hilang, "Berakhir Besok" masih tayang.
	e.now = time.Date(2026, 10, 16, 17, 30, 0, 0, time.UTC) // 17/10 00.30 WIB
	m := ids(mustPengumuman(e.svc.DaftarPengumumanPublik(bg)))
	if m[hariIni.ID] || !m[besok.ID] || !m[selamanya.ID] || len(m) != 2 {
		t.Errorf("publik 17/10 = %v", m)
	}
	// Pengurus tetap melihat semuanya; mengubah pengumuman lama (masa tayang lampau) boleh.
	if semua, _ := e.svc.DaftarPengumuman(bg); len(semua) != 4 {
		t.Errorf("pengurus = %d", len(semua))
	}
	v, err := e.svc.UbahPengumuman(bg, hariIni.ID, sah(func(i *PengumumanInput) { i.Judul, i.TampilSampai = "Berakhir Hari Ini", "2026-10-16" }))
	if err != nil || v.Tayang {
		t.Errorf("ubah pengumuman berakhir: %+v, %v", v, err)
	}
	if err := e.svc.HapusPengumuman(bg, besok.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.HapusPengumuman(bg, besok.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("hapus dua kali: err = %v", err)
	}
}

func mustPengumuman(ps []PengumumanView, err error) []PengumumanView {
	if err != nil {
		panic(err)
	}
	return ps
}
