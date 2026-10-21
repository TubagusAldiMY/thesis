// Package kuitansi membuat PDF kuitansi penerimaan zakat/infak (A5 lanskap, siap
// cetak) dan helper tampilannya dalam bahasa Indonesia. Desainnya sengaja
// menyerupai halaman /kuitansi/<kode> di frontend; terbilang dan label metode
// adalah cermin dari frontend/src/lib/format.ts (ubah keduanya bersamaan).
// Paket ini tidak mengimpor paket project lain.
package kuitansi

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// Logo = lambang masjid untuk kop PDF dan email. Salinan frontend/static/logo-masjid.jpg;
// bila logo diganti di frontend, ganti berkas ini juga.
//
//go:embed logo-masjid.jpg
var Logo []byte

// wib: tanggal kuitansi mengikuti kalender Indonesia, bukan UTC. Zona tetap dipakai
// supaya tidak bergantung pada tzdata di image Docker.
var wib = time.FixedZone("WIB", 7*3600)

// Identitas = kop kuitansi. Alamat dan Kontak boleh kosong (tidak dicetak).
type Identitas struct {
	Nama, Alamat, Kontak string
}

// Data = isi satu kuitansi. Metode sudah berupa label ramah (lihat NamaMetode).
type Data struct {
	Masjid  Identitas
	Judul   string // mis. "KUITANSI ZAKAT"
	Nomor   string
	Donatur string
	Jenis   string // mis. "Zakat Fitrah"
	Jumlah  int64  // rupiah utuh, ≥ 0
	Metode  string
	Tanggal time.Time
}

var satuan = [...]string{"", "satu", "dua", "tiga", "empat", "lima", "enam", "tujuh", "delapan", "sembilan", "sepuluh", "sebelas"}

var skala = [...]struct {
	nilai int64
	nama  string
}{{1e12, "triliun"}, {1e9, "miliar"}, {1e6, "juta"}, {1e3, "ribu"}}

// bilang mengeja bilangan bulat > 0 ("seratus dua puluh tiga").
func bilang(n int64) string {
	switch {
	case n < 12:
		return satuan[n]
	case n < 20:
		return satuan[n-10] + " belas"
	case n < 100:
		return satuan[n/10] + " puluh" + sambung(n%10)
	case n < 200:
		return "seratus" + sambung(n-100)
	case n < 1000:
		return satuan[n/100] + " ratus" + sambung(n%100)
	case n < 2000:
		return "seribu" + sambung(n-1000)
	}
	for _, s := range skala {
		if n >= s.nilai {
			return bilang(n/s.nilai) + " " + s.nama + sambung(n%s.nilai)
		}
	}
	return ""
}

// sambung = " " + ejaan sisa, atau kosong bila sisanya 0.
func sambung(n int64) string {
	if n == 0 {
		return ""
	}
	return " " + bilang(n)
}

// Terbilang: 125000 → "Seratus dua puluh lima ribu rupiah". Bilangan negatif
// (tidak mungkin pada donasi, yang divalidasi di service) dikembalikan kosong.
func Terbilang(n int64) string {
	if n < 0 {
		return ""
	}
	t := "nol"
	if n > 0 {
		t = bilang(n)
	}
	return strings.ToUpper(t[:1]) + t[1:] + " rupiah"
}

// Rupiah: 50000 → "Rp 50.000" (n ≥ 0).
func Rupiah(n int64) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "Rp " + b.String()
}

var labelMetode = map[string]string{
	"tunai":         "Tunai",
	"transfer":      "Transfer bank",
	"qris":          "QRIS",
	"gopay":         "GoPay",
	"shopeepay":     "ShopeePay",
	"bank_transfer": "Transfer bank (virtual account)",
	"echannel":      "Mandiri Bill",
	"credit_card":   "Kartu kredit/debit",
	"cstore":        "Gerai retail",
}

// NamaMetode: metode petugas atau payment_type gateway → label ramah; nilai tak
// dikenal ditampilkan apa adanya, kosong → "-".
func NamaMetode(m string) string {
	if l, ok := labelMetode[m]; ok {
		return l
	}
	if m == "" {
		return "-"
	}
	return m
}

var namaBulan = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// TanggalPanjang: waktu apa pun → "21 Oktober 2026" menurut WIB.
func TanggalPanjang(t time.Time) string {
	t = t.In(wib)
	return fmt.Sprintf("%d %s %d", t.Day(), namaBulan[t.Month()-1], t.Year())
}

// NamaBerkas: nomor kuitansi → nama berkas aman, "KWT/2026/0007" → "Kuitansi-KWT-2026-0007.pdf".
func NamaBerkas(nomor string) string {
	var b strings.Builder
	for _, r := range nomor {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return "Kuitansi-" + strings.Trim(b.String(), "-") + ".pdf"
}

// Warna (RGB) sama dengan token CSS frontend: --hijau-tua, --hijau, --hijau-muda, --teks-redup, --teks.
var (
	hijauTua  = [3]int{23, 71, 58}
	hijau     = [3]int{31, 90, 69}
	hijauMuda = [3]int{227, 239, 232}
	redup     = [3]int{107, 106, 104}
	teks      = [3]int{31, 30, 29}
	titik     = [3]int{183, 179, 166}
)

// PDF membuat berkas PDF kuitansi (A5 lanskap, satu halaman).
func PDF(d Data) ([]byte, error) { return buat(d, true) }

// buat dipisah dari PDF supaya test bisa mematikan kompresi dan membaca teksnya.
func buat(d Data, kompres bool) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A5", "")
	pdf.SetCompression(kompres)
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle(d.Judul+" "+d.Nomor, true)
	pdf.SetAuthor(d.Masjid.Nama, true)
	pdf.SetCreator("SPK Zakat", true)
	pdf.AddPage()

	// Font inti PDF (Helvetica) memakai cp1252: huruf di luar itu (mis. aksara Arab
	// pada nama donatur) tampil sebagai pengganti, bukan galat. Yang utuh ada di web.
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	warna := func(c [3]int) { pdf.SetTextColor(c[0], c[1], c[2]) }
	garis := func(c [3]int) { pdf.SetDrawColor(c[0], c[1], c[2]) }
	isi := func(c [3]int) { pdf.SetFillColor(c[0], c[1], c[2]) }
	// muat memotong s dengan "..." agar tidak lebih lebar dari lebar mm pada font saat ini.
	muat := func(s string, lebar float64) string {
		s = strings.Join(strings.Fields(s), " ")
		if pdf.GetStringWidth(tr(s)) <= lebar {
			return s
		}
		r := []rune(s)
		for len(r) > 1 && pdf.GetStringWidth(tr(string(r)+"...")) > lebar {
			r = r[:len(r)-1]
		}
		return string(r) + "..."
	}

	// bungkus memecah s per kata menjadi baris selebar maksimal lebar mm pada font saat ini.
	bungkus := func(s string, lebar float64) []string {
		var baris []string
		cur := ""
		for _, kata := range strings.Fields(s) {
			coba := kata
			if cur != "" {
				coba = cur + " " + kata
			}
			if cur != "" && pdf.GetStringWidth(tr(coba)) > lebar {
				baris = append(baris, cur)
				cur = kata
			} else {
				cur = coba
			}
		}
		if cur != "" {
			baris = append(baris, cur)
		}
		return baris
	}

	const kiri, kanan = 16.0, 194.0

	// Bingkai dan pita hijau di atas.
	isi(hijauTua)
	pdf.RoundedRect(8, 8, 194, 6, 3, "12", "F")
	garis(hijauTua)
	pdf.SetLineWidth(0.6)
	pdf.RoundedRect(8, 8, 194, 132, 3, "1234", "D")

	// Kop: logo + identitas di kiri, judul + nomor di kanan.
	if len(Logo) > 0 {
		opt := fpdf.ImageOptions{ImageType: "JPG"}
		pdf.RegisterImageOptionsReader("logo", opt, bytes.NewReader(Logo))
		pdf.ImageOptions("logo", kiri, 19, 21, 21, false, opt, 0, "")
	}
	pdf.SetFont("Helvetica", "B", 15)
	warna(hijauTua)
	pdf.Text(41, 27, tr(muat(d.Masjid.Nama, 100)))
	pdf.SetFont("Helvetica", "", 8.5)
	warna(redup)
	if alamat := strings.Join(nonKosong(d.Masjid.Alamat, d.Masjid.Kontak), " · "); alamat != "" {
		// Dibungkus manual per kata (MultiCell kembali ke margin kiri tiap baris, dan
		// SplitText fpdf panik pada teks cp1252); maksimal dua baris, sisanya dipotong.
		baris := bungkus(alamat, 98)
		if len(baris) > 2 {
			baris = []string{baris[0], baris[1] + " ..."}
		}
		for i, b := range baris {
			pdf.Text(41, 32.5+float64(i)*4.2, tr(muat(b, 98)))
		}
	}

	pdf.SetFont("Helvetica", "B", 12)
	warna(hijauTua)
	pdf.Text(kanan-pdf.GetStringWidth(tr(d.Judul)), 25, tr(d.Judul))
	// Nomor kuitansi dalam pil, rata kanan.
	pdf.SetFont("Helvetica", "B", 9.5)
	nomor := "No. " + d.Nomor
	lebarPil := pdf.GetStringWidth(tr(nomor)) + 8
	pdf.SetLineWidth(0.3)
	pdf.RoundedRect(kanan-lebarPil, 29, lebarPil, 6.5, 3.25, "1234", "D")
	warna(teks)
	pdf.Text(kanan-lebarPil+4, 33.4, tr(nomor))

	garis(hijauTua)
	pdf.SetLineWidth(0.6)
	pdf.Line(kiri, 46, kanan, 46)

	// Isian bergaris titik-titik.
	const (
		nilaiX = 58.0
		baris0 = 56.0
		jarak  = 10.0
	)
	rows := []struct{ label, nilai, gaya string }{
		{"Telah terima dari", d.Donatur, "B"},
		{"Terbilang", Terbilang(d.Jumlah), "I"},
		{"Untuk pembayaran", d.Jenis, ""},
		{"Metode", d.Metode, ""},
		{"Tanggal", TanggalPanjang(d.Tanggal), ""},
	}
	pdf.SetLineCapStyle("round") // garis putus-putus berujung bulat = titik
	for i, r := range rows {
		y := baris0 + float64(i)*jarak
		pdf.SetFont("Helvetica", "", 9.5)
		warna(redup)
		pdf.Text(kiri, y, tr(r.label))
		pdf.SetFont("Helvetica", r.gaya, 11)
		warna(teks)
		pdf.Text(nilaiX, y, tr(muat(r.nilai, kanan-nilaiX)))
		garis(titik)
		pdf.SetLineWidth(0.3)
		pdf.SetDashPattern([]float64{0.05, 1}, 0)
		pdf.Line(nilaiX, y+1.8, kanan, y+1.8)
		pdf.SetDashPattern([]float64{}, 0)
	}
	pdf.SetLineCapStyle("butt")

	// Kotak jumlah: font mengecil bila nominalnya sangat besar.
	isi(hijauMuda)
	garis(hijauTua)
	pdf.SetLineWidth(0.5)
	pdf.RoundedRect(kiri, 108, 66, 21, 3, "1234", "FD")
	pdf.SetFont("Helvetica", "B", 7.5)
	warna(hijau)
	pdf.Text(kiri+5, 113.5, "JUMLAH")
	rp := Rupiah(d.Jumlah)
	ukuran := 21.0
	for ukuran > 10 {
		pdf.SetFont("Helvetica", "B", ukuran)
		if pdf.GetStringWidth(rp) <= 56 {
			break
		}
		ukuran--
	}
	pdf.SetFont("Helvetica", "B", ukuran)
	warna(hijauTua)
	pdf.Text(kiri+5, 124.5, rp)

	// Penerima: ruang tanda tangan di atas garis, nama pengurus di bawahnya.
	const ttdKiri, ttdKanan = 138.0, kanan
	tengah := (ttdKiri + ttdKanan) / 2
	pdf.SetFont("Helvetica", "", 9.5)
	warna(teks)
	pdf.Text(tengah-pdf.GetStringWidth("Penerima,")/2, 110, "Penerima,")
	garis(teks)
	pdf.SetLineWidth(0.3)
	pdf.Line(ttdKiri, 125, ttdKanan, 125)
	pdf.SetFont("Helvetica", "B", 9)
	pengurus := muat("Pengurus "+d.Masjid.Nama, ttdKanan-ttdKiri+8)
	pdf.Text(tengah-pdf.GetStringWidth(tr(pengurus))/2, 129.5, tr(pengurus))

	pdf.SetFont("Helvetica", "", 7.5)
	warna(redup)
	pdf.Text(kiri, 135, tr(muat("Kuitansi ini diterbitkan oleh sistem "+d.Masjid.Nama+".", 118)))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("kuitansi: buat PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// nonKosong membuang string kosong (atau hanya spasi) dari daftar.
func nonKosong(s ...string) []string {
	var out []string
	for _, x := range s {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}
