package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/payment"
)

// jam tetap: 16/10/2026 10.00 WIB, supaya tahun nomor kuitansi bisa diuji.
var jamUji = time.Date(2026, 10, 16, 3, 0, 0, 0, time.UTC)

// donasiEnv = lingkungan uji dengan penyedia tiruan terpasang dan jam tetap.
type donasiEnv struct {
	*lingkungan
	pay *payment.Tiruan
	now time.Time
}

func baruDonasi(t *testing.T) *donasiEnv {
	t.Helper()
	e := &donasiEnv{lingkungan: baru(t), pay: payment.NewTiruan(), now: jamUji}
	e.svc.AturGerbang(e.pay, "https://masjid.example/")
	e.svc.sekarang = func() time.Time { return e.now }
	return e
}

func inputWeb(ubah ...func(*DonasiWebInput)) DonasiWebInput {
	in := DonasiWebInput{Nama: "Hamba Allah", NoHP: "081234567890", Jenis: domain.DonasiZakatFitrah, Jumlah: 50_000, Keterangan: "semoga berkah"}
	for _, f := range ubah {
		f(&in)
	}
	return in
}

func inputManual(ubah ...func(*DonasiManualInput)) DonasiManualInput {
	in := DonasiManualInput{Nama: "Pak Haji", NoHP: "08111222333", Jenis: domain.DonasiInfak, Jumlah: 100_000, Metode: "tunai"}
	for _, f := range ubah {
		f(&in)
	}
	return in
}

// bayar mensimulasikan notifikasi gateway untuk satu donasi.
func (e *donasiEnv) bayar(t *testing.T, id string, jumlah int64, status string) (HasilNotifikasi, error) {
	t.Helper()
	return e.svc.TerimaNotifikasi(bg, e.pay.Muatan(id, jumlah, status))
}

// idDariKode mencari ID donasi dari kode publik (order_id gateway = ID).
func (e *donasiEnv) idDariKode(t *testing.T, kode string) string {
	t.Helper()
	d, err := e.st.GetDonasiByKode(bg, kode)
	if err != nil {
		t.Fatalf("GetDonasiByKode: %v", err)
	}
	return d.ID
}

func TestBuatDonasiWeb_Validasi(t *testing.T) {
	e := baruDonasi(t)
	tests := []struct {
		nama  string
		ubah  func(*DonasiWebInput)
		field string
	}{
		{"nama kosong", func(i *DonasiWebInput) { i.Nama = " " }, "nama"},
		{"nama 1 huruf", func(i *DonasiWebInput) { i.Nama = "A" }, "nama"},
		{"nama terlalu panjang", func(i *DonasiWebInput) { i.Nama = strings.Repeat("a", 101) }, "nama"},
		{"jenis tidak dikenal", func(i *DonasiWebInput) { i.Jenis = "sedekah" }, "jenis"},
		{"jumlah di bawah batas", func(i *DonasiWebInput) { i.Jumlah = 9_999 }, "jumlah"},
		{"jumlah nol", func(i *DonasiWebInput) { i.Jumlah = 0 }, "jumlah"},
		{"jumlah negatif", func(i *DonasiWebInput) { i.Jumlah = -50_000 }, "jumlah"},
		{"jumlah di atas batas", func(i *DonasiWebInput) { i.Jumlah = 100_000_001 }, "jumlah"},
		{"HP tidak valid", func(i *DonasiWebInput) { i.NoHP = "abc" }, "no_hp"},
		{"keterangan terlalu panjang", func(i *DonasiWebInput) { i.Keterangan = strings.Repeat("x", 201) }, "keterangan"},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			_, err := e.svc.BuatDonasiWeb(bg, inputWeb(tt.ubah))
			var ve *domain.ValidationError
			if !errors.As(err, &ve) || ve.Fields[tt.field] == "" {
				t.Fatalf("err = %v, mau ValidationError pada %q", err, tt.field)
			}
		})
	}
	// Batas tepat diterima.
	for _, j := range []int64{donasiMin, donasiMaks} {
		if _, err := e.svc.BuatDonasiWeb(bg, inputWeb(func(i *DonasiWebInput) { i.Jumlah = j })); err != nil {
			t.Errorf("jumlah %d (batas) ditolak: %v", j, err)
		}
	}
}

func TestBuatDonasiWeb_TanpaGerbangNonaktif(t *testing.T) {
	e := baru(t) // tanpa AturGerbang
	if _, err := e.svc.BuatDonasiWeb(bg, inputWeb()); !errors.Is(err, domain.ErrNonaktif) {
		t.Errorf("err = %v, mau ErrNonaktif", err)
	}
	if _, err := e.svc.TerimaNotifikasi(bg, []byte("{}")); !errors.Is(err, domain.ErrNonaktif) {
		t.Errorf("TerimaNotifikasi err = %v, mau ErrNonaktif", err)
	}
}

// Alur utama (sanity): form → menunggu → notifikasi bayar → diterima + kuitansi.
func TestDonasiWeb_AlurDariFormSampaiKuitansi(t *testing.T) {
	e := baruDonasi(t)
	p, err := e.svc.BuatDonasiWeb(bg, inputWeb())
	if err != nil {
		t.Fatalf("BuatDonasiWeb: %v", err)
	}
	if p.Status != domain.DonasiMenunggu || p.NomorKuitansi != "" || len(p.Kode) != 32 {
		t.Errorf("publik awal = %+v", p)
	}
	if p.BayarURL != "https://masjid.example/donasi/tiruan?kode="+p.Kode {
		t.Errorf("BayarURL = %q (URL web harus tanpa garis miring ganda)", p.BayarURL)
	}

	// No. HP tersimpan terenkripsi, bukan plaintext.
	var enc []byte
	if err := e.db.QueryRow(`SELECT no_hp_enc FROM donasi WHERE kode_publik = ?`, p.Kode).Scan(&enc); err != nil {
		t.Fatal(err)
	}
	if len(enc) == 0 || strings.Contains(string(enc), "081234567890") {
		t.Error("No. HP donasi tidak terenkripsi")
	}

	id := e.idDariKode(t, p.Kode)
	h, err := e.bayar(t, id, 50_000, "settlement")
	if err != nil {
		t.Fatalf("TerimaNotifikasi: %v", err)
	}
	if !h.Berubah || h.Status != domain.DonasiDiterima {
		t.Errorf("hasil = %+v", h)
	}

	p2, err := e.svc.AmbilDonasiPublik(bg, p.Kode)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Status != domain.DonasiDiterima || p2.NomorKuitansi != "KWT/2026/0001" || p2.BayarURL != "" || p2.DibayarPada == nil || p2.Metode != "qris" {
		t.Errorf("publik setelah bayar = %+v", p2)
	}
}

// Notifikasi dari gateway bisa datang ganda atau terlambat: tidak boleh
// menerbitkan kuitansi kedua atau menimpa status akhir (regresi idempotensi).
func TestTerimaNotifikasi_Idempoten(t *testing.T) {
	e := baruDonasi(t)
	p, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	id := e.idDariKode(t, p.Kode)

	if h, err := e.bayar(t, id, 50_000, "settlement"); err != nil || !h.Berubah {
		t.Fatalf("settlement pertama: %+v, %v", h, err)
	}
	for _, status := range []string{"settlement", "expire", "cancel", "pending"} {
		h, err := e.bayar(t, id, 50_000, status)
		if err != nil || h.Berubah || h.Status != domain.DonasiDiterima {
			t.Errorf("notifikasi %q setelah diterima: %+v, %v (harus diabaikan)", status, h, err)
		}
	}
	var n int
	_ = e.db.QueryRow(`SELECT COUNT(*) FROM donasi WHERE nomor_kuitansi IS NOT NULL`).Scan(&n)
	if n != 1 {
		t.Errorf("kuitansi terbit %d, mau 1", n)
	}
}

func TestTerimaNotifikasi_StatusLain(t *testing.T) {
	e := baruDonasi(t)
	tests := []struct {
		status string
		mau    domain.StatusDonasi
	}{
		{"expire", domain.DonasiKedaluwarsa},
		{"deny", domain.DonasiGagal},
		{"cancel", domain.DonasiGagal},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			p, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
			if _, err := e.bayar(t, e.idDariKode(t, p.Kode), 50_000, tt.status); err != nil {
				t.Fatal(err)
			}
			got, _ := e.svc.AmbilDonasiPublik(bg, p.Kode)
			if got.Status != tt.mau || got.NomorKuitansi != "" || got.BayarURL != "" {
				t.Errorf("publik = %+v, mau %s tanpa kuitansi", got, tt.mau)
			}
		})
	}
	// pending: status tetap menunggu, metode yang dipilih muzaki tercatat.
	p, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	h, err := e.bayar(t, e.idDariKode(t, p.Kode), 50_000, "pending")
	if err != nil || h.Berubah {
		t.Fatalf("pending: %+v, %v", h, err)
	}
	got, _ := e.svc.AmbilDonasiPublik(bg, p.Kode)
	if got.Status != domain.DonasiMenunggu || got.Metode != "qris" || got.BayarURL == "" {
		t.Errorf("setelah pending = %+v", got)
	}
}

// Keamanan: jumlah notifikasi yang tidak sama dengan donasi ditolak dan tidak
// menerbitkan kuitansi; order tak dikenal dan tanda tangan salah juga ditolak.
func TestTerimaNotifikasi_Ditolak(t *testing.T) {
	e := baruDonasi(t)
	p, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	id := e.idDariKode(t, p.Kode)

	if _, err := e.bayar(t, id, 10_000, "settlement"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("jumlah beda: err = %v, mau ErrConflict", err)
	}
	if got, _ := e.svc.AmbilDonasiPublik(bg, p.Kode); got.Status != domain.DonasiMenunggu || got.NomorKuitansi != "" {
		t.Errorf("jumlah beda tidak boleh mengubah donasi: %+v", got)
	}
	if _, err := e.bayar(t, "00000000-0000-4000-8000-000000000000", 50_000, "settlement"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("order tak dikenal: err = %v, mau ErrNotFound", err)
	}
	if _, err := e.svc.TerimaNotifikasi(bg, []byte(`{"order_id":"x"}`)); !errors.Is(err, domain.ErrTandaTanganTidakSah) {
		t.Errorf("muatan tanpa tanda tangan: err = %v", err)
	}
	// Donasi petugas tidak boleh bisa "dibayar" lewat notifikasi gateway.
	dm, err := e.svc.CatatDonasiManual(bg, e.user(t), inputManual())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.bayar(t, dm.ID, 100_000, "settlement"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("notifikasi untuk donasi petugas: err = %v, mau ErrNotFound", err)
	}
}

type gerbangRusak struct{ *payment.Tiruan }

func (gerbangRusak) Buat(context.Context, domain.PermintaanBayar) (domain.HasilBayar, error) {
	return domain.HasilBayar{}, errors.New("koneksi ditolak")
}

func TestBuatDonasiWeb_GerbangGagal(t *testing.T) {
	e := baruDonasi(t)
	e.svc.AturGerbang(gerbangRusak{e.pay}, "https://masjid.example")
	_, err := e.svc.BuatDonasiWeb(bg, inputWeb())
	if !errors.Is(err, domain.ErrGerbang) {
		t.Fatalf("err = %v, mau ErrGerbang", err)
	}
	// Donasi tidak boleh menggantung "menunggu": ditandai gagal.
	var status string
	if err := e.db.QueryRow(`SELECT status FROM donasi`).Scan(&status); err != nil || status != "gagal" {
		t.Errorf("status = %q, %v; mau gagal", status, err)
	}
}

// Penomoran: urut per tahun, tanpa celah, reset tiap tahun, format 4 digit.
func TestNomorKuitansi_UrutPerTahun(t *testing.T) {
	e := baruDonasi(t)
	petugas := e.user(t)
	catat := func() string {
		t.Helper()
		v, err := e.svc.CatatDonasiManual(bg, petugas, inputManual())
		if err != nil {
			t.Fatal(err)
		}
		return v.NomorKuitansi
	}
	if a, b := catat(), catat(); a != "KWT/2026/0001" || b != "KWT/2026/0002" {
		t.Errorf("nomor = %s, %s", a, b)
	}
	// Pukul 23.30 WIB 31/12/2026 masih tahun 2026; 00.30 WIB 1/1/2027 sudah 2027.
	e.now = time.Date(2026, 12, 31, 16, 30, 0, 0, time.UTC)
	if n := catat(); n != "KWT/2026/0003" {
		t.Errorf("malam 31 Desember WIB = %s, mau KWT/2026/0003", n)
	}
	e.now = time.Date(2026, 12, 31, 17, 30, 0, 0, time.UTC)
	if n := catat(); n != "KWT/2027/0001" {
		t.Errorf("1 Januari WIB = %s, mau KWT/2027/0001", n)
	}
}

// Regresi balapan: banyak pembayaran dikonfirmasi bersamaan harus mendapat nomor
// unik dan berurutan 1..N (tanpa ganda, tanpa celah).
func TestNomorKuitansi_PembayaranBersamaan(t *testing.T) {
	e := baruDonasi(t)
	const n = 12
	ids := make([]string, n)
	for i := range ids {
		p, err := e.svc.BuatDonasiWeb(bg, inputWeb())
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = e.idDariKode(t, p.Kode)
	}
	var wg sync.WaitGroup
	galat := make(chan error, n)
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.bayar(t, id, 50_000, "settlement"); err != nil {
				galat <- err
			}
		}()
	}
	wg.Wait()
	close(galat)
	for err := range galat {
		t.Errorf("notifikasi bersamaan: %v", err)
	}

	rows, err := e.db.Query(`SELECT nomor_kuitansi FROM donasi`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	sort.Strings(got)
	if len(got) != n {
		t.Fatalf("terbit %d nomor, mau %d: %v", len(got), n, got)
	}
	for i, g := range got {
		if mau := fmt.Sprintf("KWT/2026/%04d", i+1); g != mau {
			t.Fatalf("nomor ke-%d = %s, mau %s (semua: %v)", i+1, g, mau, got)
		}
	}
}

func TestCatatDonasiManual_Validasi(t *testing.T) {
	e := baruDonasi(t)
	petugas := e.user(t)
	tests := []struct {
		nama  string
		ubah  func(*DonasiManualInput)
		field string
	}{
		{"metode salah", func(i *DonasiManualInput) { i.Metode = "kartu" }, "metode"},
		{"metode kosong", func(i *DonasiManualInput) { i.Metode = "" }, "metode"},
		{"tanggal bukan tanggal", func(i *DonasiManualInput) { i.Tanggal = "kemarin" }, "tanggal"},
		{"tanggal masa depan", func(i *DonasiManualInput) { i.Tanggal = "2026-10-17" }, "tanggal"},
		{"jumlah terlalu kecil", func(i *DonasiManualInput) { i.Jumlah = 500 }, "jumlah"},
		{"nama kosong", func(i *DonasiManualInput) { i.Nama = "" }, "nama"},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			_, err := e.svc.CatatDonasiManual(bg, petugas, inputManual(tt.ubah))
			var ve *domain.ValidationError
			if !errors.As(err, &ve) || ve.Fields[tt.field] == "" {
				t.Fatalf("err = %v, mau ValidationError pada %q", err, tt.field)
			}
		})
	}
	var n int
	_ = e.db.QueryRow(`SELECT COUNT(*) FROM donasi`).Scan(&n)
	if n != 0 {
		t.Errorf("%d donasi tersimpan padahal semua isian salah", n)
	}
	// Tanggal hari ini dan tanggal lampau diterima.
	for _, tgl := range []string{"2026-10-16", "2026-10-01", ""} {
		if _, err := e.svc.CatatDonasiManual(bg, petugas, inputManual(func(i *DonasiManualInput) { i.Tanggal = tgl })); err != nil {
			t.Errorf("tanggal %q ditolak: %v", tgl, err)
		}
	}
}

// Notifikasi: hanya donasi WEB yang sudah diterima dan belum dilihat yang
// menaikkan penanda; catatan petugas tidak (yang mencatat sudah tahu).
func TestNotifikasiDonasiBaru(t *testing.T) {
	e := baruDonasi(t)
	ringkas := func() DonasiRingkasView {
		t.Helper()
		r, err := e.svc.RingkasDonasi(bg)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if r := ringkas(); r.Baru != 0 || r.TotalDiterima != 0 {
		t.Fatalf("awal = %+v", r)
	}

	if _, err := e.svc.CatatDonasiManual(bg, e.user(t), inputManual()); err != nil {
		t.Fatal(err)
	}
	if r := ringkas(); r.Baru != 0 || r.JumlahDiterima != 1 {
		t.Errorf("setelah catatan petugas = %+v (tidak boleh jadi notifikasi)", r)
	}

	p, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	if r := ringkas(); r.Baru != 0 {
		t.Errorf("donasi masih menunggu bayar tidak boleh jadi notifikasi: %+v", r)
	}
	id := e.idDariKode(t, p.Kode)
	_, _ = e.bayar(t, id, 50_000, "settlement")
	if r := ringkas(); r.Baru != 1 || r.JumlahDiterima != 2 || r.TotalDiterima != 150_000 {
		t.Errorf("setelah bayar web = %+v", r)
	}
	v, _ := e.svc.AmbilDonasi(bg, id, false)
	if !v.Baru {
		t.Error("donasi web yang baru diterima harus bertanda Baru")
	}

	if n, err := e.svc.TandaiDonasiDilihat(bg); err != nil || n != 1 {
		t.Errorf("TandaiDonasiDilihat = %d, %v", n, err)
	}
	if r := ringkas(); r.Baru != 0 {
		t.Errorf("setelah ditandai dilihat = %+v", r)
	}
	if n, _ := e.svc.TandaiDonasiDilihat(bg); n != 0 {
		t.Errorf("tandai ulang mengubah %d baris", n)
	}
}

func TestBatalkanDonasi(t *testing.T) {
	e := baruDonasi(t)
	dm, err := e.svc.CatatDonasiManual(bg, e.user(t), inputManual())
	if err != nil {
		t.Fatal(err)
	}
	for _, alasan := range []string{"", "  ", "x", strings.Repeat("a", 201)} {
		var ve *domain.ValidationError
		if _, err := e.svc.BatalkanDonasi(bg, dm.ID, alasan); !errors.As(err, &ve) || ve.Fields["alasan"] == "" {
			t.Errorf("alasan %q: err = %v", alasan, err)
		}
	}
	v, err := e.svc.BatalkanDonasi(bg, dm.ID, "salah ketik jumlah")
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != domain.DonasiDibatalkan || v.NomorKuitansi != dm.NomorKuitansi || v.AlasanBatal != "salah ketik jumlah" {
		t.Errorf("setelah batal = %+v (nomor harus tetap tercatat)", v)
	}
	if _, err := e.svc.BatalkanDonasi(bg, dm.ID, "dua kali"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("batal dua kali: err = %v, mau ErrConflict", err)
	}
	// Donasi yang masih menunggu bayar tidak bisa dibatalkan petugas.
	p, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	if _, err := e.svc.BatalkanDonasi(bg, e.idDariKode(t, p.Kode), "coba batal"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("batal donasi menunggu: err = %v, mau ErrConflict", err)
	}
	if _, err := e.svc.BatalkanDonasi(bg, idHantu, "tidak ada"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("batal ID tak dikenal: err = %v", err)
	}
	// Jumlah donasi dibatalkan tidak dihitung sebagai penerimaan.
	if r, _ := e.svc.RingkasDonasi(bg); r.JumlahDiterima != 0 || r.TotalDiterima != 0 {
		t.Errorf("ringkas setelah batal = %+v", r)
	}
}

// Privasi: No. HP penuh hanya untuk Admin pada detail; daftar dan publik tidak
// pernah memuatnya.
func TestDonasi_PrivasiNoHP(t *testing.T) {
	e := baruDonasi(t)
	p, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	id := e.idDariKode(t, p.Kode)
	_, _ = e.bayar(t, id, 50_000, "settlement")

	admin, err := e.svc.AmbilDonasi(bg, id, true)
	if err != nil || admin.NoHP != "081234567890" || admin.NoHPTersamar != "" {
		t.Errorf("admin = %+v, %v", admin, err)
	}
	ketua, _ := e.svc.AmbilDonasi(bg, id, false)
	if ketua.NoHP != "" || ketua.NoHPTersamar != "•••• 7890" {
		t.Errorf("ketua = %+v", ketua)
	}
	h, err := e.svc.DaftarDonasi(bg, DonasiQuery{})
	if err != nil || len(h.Items) != 1 {
		t.Fatalf("daftar = %+v, %v", h, err)
	}
	if h.Items[0].NoHP != "" || h.Items[0].NoHPTersamar != "" {
		t.Errorf("daftar memuat No. HP: %+v", h.Items[0])
	}
}

func TestAmbilDonasiPublik_KodeSalahSamaDenganTidakAda(t *testing.T) {
	e := baruDonasi(t)
	for _, kode := range []string{"", "abc", strings.Repeat("g", 32), strings.Repeat("a", 31), strings.Repeat("A", 32), strings.Repeat("0", 32), "' OR '1'='1"} {
		if _, err := e.svc.AmbilDonasiPublik(bg, kode); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("kode %q: err = %v, mau ErrNotFound", kode, err)
		}
	}
}

func TestDaftarDonasi_FilterDanValidasi(t *testing.T) {
	e := baruDonasi(t)
	petugas := e.user(t)
	_, _ = e.svc.CatatDonasiManual(bg, petugas, inputManual(func(i *DonasiManualInput) { i.Nama = "Budi Santoso"; i.Jenis = domain.DonasiZakatMal }))
	_, _ = e.svc.CatatDonasiManual(bg, petugas, inputManual(func(i *DonasiManualInput) { i.Nama = "Siti Aminah"; i.Jenis = domain.DonasiInfak }))
	_, _ = e.svc.BuatDonasiWeb(bg, inputWeb())

	cek := func(nama string, q DonasiQuery, mauTotal int) {
		t.Helper()
		h, err := e.svc.DaftarDonasi(bg, q)
		if err != nil || h.Total != mauTotal || len(h.Items) != mauTotal {
			t.Errorf("%s: total=%d items=%d err=%v, mau %d", nama, h.Total, len(h.Items), err, mauTotal)
		}
	}
	cek("semua", DonasiQuery{}, 3)
	cek("diterima", DonasiQuery{Status: domain.DonasiDiterima}, 2)
	cek("menunggu", DonasiQuery{Status: domain.DonasiMenunggu}, 1)
	cek("infak", DonasiQuery{Jenis: domain.DonasiInfak}, 1)
	cek("cari nama", DonasiQuery{Q: "santoso"}, 1)
	cek("LIKE % dibaca harfiah", DonasiQuery{Q: "%"}, 0)

	for _, q := range []DonasiQuery{{Status: "aneh"}, {Jenis: "aneh"}, {Q: strings.Repeat("a", 101)}} {
		var ve *domain.ValidationError
		if _, err := e.svc.DaftarDonasi(bg, q); !errors.As(err, &ve) {
			t.Errorf("query %+v: err = %v, mau ValidationError", q, err)
		}
	}
	// Paging: per_page=1 → 1 baris, total tetap 3.
	h, _ := e.svc.DaftarDonasi(bg, DonasiQuery{PerPage: 1, Page: 2})
	if len(h.Items) != 1 || h.Total != 3 {
		t.Errorf("paging = %d baris, total %d", len(h.Items), h.Total)
	}
}
