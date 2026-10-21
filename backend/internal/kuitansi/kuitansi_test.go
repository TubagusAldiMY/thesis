package kuitansi

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Kasus sama dengan terbilang di frontend/src/lib/format.test.ts: keduanya cermin,
// jadi salah satu berubah tanpa yang lain harus langsung terlihat.
func TestTerbilang(t *testing.T) {
	tests := []struct {
		n   int64
		mau string
	}{
		{0, "Nol rupiah"},
		{1, "Satu rupiah"},
		{10, "Sepuluh rupiah"},
		{11, "Sebelas rupiah"},
		{15, "Lima belas rupiah"},
		{20, "Dua puluh rupiah"},
		{21, "Dua puluh satu rupiah"},
		{100, "Seratus rupiah"},
		{111, "Seratus sebelas rupiah"},
		{200, "Dua ratus rupiah"},
		{999, "Sembilan ratus sembilan puluh sembilan rupiah"},
		{1000, "Seribu rupiah"},
		{1001, "Seribu satu rupiah"},
		{1500, "Seribu lima ratus rupiah"},
		{2000, "Dua ribu rupiah"},
		{10_000, "Sepuluh ribu rupiah"},
		{11_000, "Sebelas ribu rupiah"},
		{50_000, "Lima puluh ribu rupiah"},
		{100_000, "Seratus ribu rupiah"},
		{125_000, "Seratus dua puluh lima ribu rupiah"},
		{1_000_000, "Satu juta rupiah"},
		{1_001_000, "Satu juta seribu rupiah"},
		{2_500_000, "Dua juta lima ratus ribu rupiah"},
		{100_000_000, "Seratus juta rupiah"},
		{1_000_000_000, "Satu miliar rupiah"},
		{1_234_567_891, "Satu miliar dua ratus tiga puluh empat juta lima ratus enam puluh tujuh ribu delapan ratus sembilan puluh satu rupiah"},
		{-1, ""}, // tidak mungkin pada donasi; tidak boleh panik
	}
	for _, tt := range tests {
		if got := Terbilang(tt.n); got != tt.mau {
			t.Errorf("Terbilang(%d) = %q, mau %q", tt.n, got, tt.mau)
		}
	}
}

func TestRupiah(t *testing.T) {
	for n, mau := range map[int64]string{0: "Rp 0", 999: "Rp 999", 1000: "Rp 1.000", 50_000: "Rp 50.000", 100_000_000: "Rp 100.000.000", 1_234_567: "Rp 1.234.567"} {
		if got := Rupiah(n); got != mau {
			t.Errorf("Rupiah(%d) = %q, mau %q", n, got, mau)
		}
	}
}

func TestNamaMetode(t *testing.T) {
	for m, mau := range map[string]string{
		"tunai": "Tunai", "qris": "QRIS", "bank_transfer": "Transfer bank (virtual account)",
		"metode_baru_gateway": "metode_baru_gateway", "": "-",
	} {
		if got := NamaMetode(m); got != mau {
			t.Errorf("NamaMetode(%q) = %q, mau %q", m, got, mau)
		}
	}
}

func TestTanggalPanjang(t *testing.T) {
	tests := []struct {
		waktu time.Time
		mau   string
	}{
		{time.Date(2026, 10, 21, 2, 14, 0, 0, time.UTC), "21 Oktober 2026"},
		{time.Date(2026, 12, 31, 17, 30, 0, 0, time.UTC), "1 Januari 2027"}, // lewat tengah malam WIB
		{time.Date(2026, 3, 5, 0, 0, 0, 0, wib), "5 Maret 2026"},
	}
	for _, tt := range tests {
		if got := TanggalPanjang(tt.waktu); got != tt.mau {
			t.Errorf("TanggalPanjang(%v) = %q, mau %q", tt.waktu, got, tt.mau)
		}
	}
}

func TestNamaBerkas(t *testing.T) {
	for nomor, mau := range map[string]string{
		"KWT/2026/0007":        "Kuitansi-KWT-2026-0007.pdf",
		"../../etc/passwd":     "Kuitansi-etc-passwd.pdf", // tanpa jalur: hanya huruf, angka, strip
		"KWT/2026/0007\r\nBcc": "Kuitansi-KWT-2026-0007--Bcc.pdf",
	} {
		if got := NamaBerkas(nomor); got != mau {
			t.Errorf("NamaBerkas(%q) = %q, mau %q", nomor, got, mau)
		}
	}
}

func contoh() Data {
	return Data{
		Masjid:  Identitas{Nama: "Masjid Jami Al-Barokah", Alamat: "Jl. Ampera Poncol, Babakan, Kec. Setu, Kota Tangerang Selatan, Banten 15315", Kontak: "0812-0000-0000"},
		Judul:   "KUITANSI ZAKAT",
		Nomor:   "KWT/2026/0007",
		Donatur: "Hamba Allah",
		Jenis:   "Zakat Fitrah",
		Jumlah:  125_000,
		Metode:  NamaMetode("bank_transfer"),
		Tanggal: time.Date(2026, 10, 21, 2, 14, 0, 0, time.UTC),
	}
}

func TestPDF_IsiDanBentuk(t *testing.T) {
	// Tanpa kompresi supaya teks di dalam PDF bisa dicari.
	raw, err := buat(contoh(), false)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.HasPrefix(s, "%PDF-") || !strings.HasSuffix(strings.TrimSpace(s), "%%EOF") {
		t.Fatalf("bukan PDF utuh: awal %q", s[:min(8, len(s))])
	}
	if n := strings.Count(s, "/Type /Page\n"); n != 1 {
		t.Errorf("jumlah halaman = %d, mau 1", n)
	}
	if !strings.Contains(s, "MediaBox [0 0 595.28 420.94]") {
		t.Error("ukuran kertas bukan A5 lanskap (595,28 x 420,94 pt)")
	}
	if !strings.Contains(s, "/Subtype /Image") {
		t.Error("logo tidak tertanam")
	}
	for _, mau := range []string{
		"KUITANSI ZAKAT", "No. KWT/2026/0007", "Hamba Allah", "Seratus dua puluh lima ribu rupiah",
		"Zakat Fitrah", `Transfer bank \(virtual account\)`, "21 Oktober 2026", // kurung di-escape dalam string PDF "Rp 125.000", "Pengurus Masjid Jami Al-Barokah",
		"Jl. Ampera Poncol", "0812-0000-0000",
	} {
		if !strings.Contains(s, mau) {
			t.Errorf("PDF tidak memuat %q", mau)
		}
	}
}

// Isian terburuk tidak boleh membuat PDF gagal atau melebihi satu halaman.
func TestPDF_IsianEkstrem(t *testing.T) {
	tests := []struct {
		nama string
		ubah func(*Data)
	}{
		{"nama 100 huruf", func(d *Data) { d.Donatur = strings.Repeat("Abdurrahman ", 9) }},
		{"jumlah maksimum", func(d *Data) { d.Jumlah = 100_000_000 }},
		{"alamat sangat panjang", func(d *Data) { d.Masjid.Alamat = strings.Repeat("Jalan Panjang Sekali ", 20) }},
		{"tanpa alamat dan kontak", func(d *Data) { d.Masjid.Alamat, d.Masjid.Kontak = "", "" }},
		{"nama masjid panjang", func(d *Data) { d.Masjid.Nama = strings.Repeat("Masjid Agung ", 12) }},
		{"huruf di luar cp1252 (Arab)", func(d *Data) { d.Donatur = "محمد Ahmad" }},
		{"nama berbaris-baris", func(d *Data) { d.Donatur = "Budi\n\nKlik\r\nsekarang" }},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			d := contoh()
			tt.ubah(&d)
			raw, err := buat(d, false)
			if err != nil {
				t.Fatalf("galat: %v", err)
			}
			if n := strings.Count(string(raw), "/Type /Page\n"); n != 1 {
				t.Errorf("jumlah halaman = %d, mau 1", n)
			}
			// Untuk memeriksa tata letak dengan mata: KUITANSI_CONTOH_DIR=/tmp/x go test -run Ekstrem ./internal/kuitansi
			if dir := os.Getenv("KUITANSI_CONTOH_DIR"); dir != "" {
				if err := os.WriteFile(dir+"/ekstrem-"+strings.ReplaceAll(tt.nama, " ", "_")+".pdf", raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPDF_Terkompresi(t *testing.T) {
	kompres, err := PDF(contoh())
	if err != nil {
		t.Fatal(err)
	}
	polos, _ := buat(contoh(), false)
	if len(kompres) >= len(polos) {
		t.Errorf("hasil terkompresi (%d B) tidak lebih kecil dari polos (%d B)", len(kompres), len(polos))
	}
	// Untuk melihat hasilnya: KUITANSI_CONTOH=/tmp/kuitansi.pdf go test -run Terkompresi ./internal/kuitansi
	if p := os.Getenv("KUITANSI_CONTOH"); p != "" {
		if err := os.WriteFile(p, kompres, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
