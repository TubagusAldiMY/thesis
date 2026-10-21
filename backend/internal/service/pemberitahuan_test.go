package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/kuitansi"
)

// pemberiPalsu merekam pesan yang "dikirim"; err != nil membuat Kirim gagal
// (pesan tetap terekam, supaya terlihat bahwa pengiriman memang dicoba).
type pemberiPalsu struct {
	mu    sync.Mutex
	surat []domain.Pesan
	err   error
}

func (p *pemberiPalsu) Kirim(_ context.Context, pesan domain.Pesan) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.surat = append(p.surat, pesan)
	return p.err
}

func (p *pemberiPalsu) jumlah() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.surat)
}

func TestPesanDonasi(t *testing.T) {
	bayar := time.Date(2026, 12, 31, 17, 30, 0, 0, time.UTC) // 1 Januari 2027 00.30 WIB
	dasar := domain.Donasi{
		ID: "id-donasi-1", Nama: "Budi", Jenis: domain.DonasiInfak, Jumlah: 1_250_000,
		NomorKuitansi: "KWT/2026/0007", DibayarPada: &bayar,
		KodePublik: "kode-rahasia-muzaki", Keterangan: "keterangan bebas",
	}
	masjid := kuitansi.Identitas{Nama: "Masjid Uji"}
	tests := []struct {
		nama       string
		ubah       func(*domain.Donasi, *kuitansi.Identitas)
		urlWeb     string
		adaPDF     bool
		subjek     string
		teks, html []string // harus ada
		tidak      []string // tidak boleh ada di subjek, teks, maupun HTML
		tidakHTML  []string // tidak boleh ada di HTML saja
	}{
		{
			nama: "semua data tampil, waktu dalam WIB", urlWeb: "https://masjid.example", adaPDF: true,
			subjek: "Donasi baru Rp 1.250.000 - KWT/2026/0007",
			teks: []string{
				"Jumlah: Rp 1.250.000", "No. kuitansi: KWT/2026/0007", "Donatur: Budi", "Jenis: Infak",
				"Diterima: 1 Januari 2027, 00.30 WIB", "terlampir: Kuitansi-KWT-2026-0007.pdf",
				"https://masjid.example/penerimaan/id-donasi-1", "sistem Masjid Uji",
			},
			html: []string{
				"Rp 1.250.000", "KWT/2026/0007", ">Budi<", "Infak", "1 Januari 2027, 00.30 WIB",
				"Kuitansi PDF terlampir", "Kuitansi-KWT-2026-0007.pdf", `href="https://masjid.example/penerimaan/id-donasi-1"`,
				"cid:logo-masjid", "Masjid Uji",
			},
			// kode publik = rahasia muzaki; keterangan bebas bisa memuat data pribadi
			tidak: []string{"kode-rahasia-muzaki", "keterangan bebas", "/kuitansi/"},
		},
		{
			nama: "tanpa alamat web tidak ada tautan", urlWeb: "", adaPDF: true,
			teks:  []string{"Donatur: Budi"},
			html:  []string{">Budi<"},
			tidak: []string{"http", "/penerimaan/", "Lihat detail"},
		},
		{
			nama: "tanpa PDF tidak menyebut lampiran", urlWeb: "https://masjid.example", adaPDF: false,
			teks:  []string{"Donatur: Budi"},
			html:  []string{">Budi<"},
			tidak: []string{"terlampir", "Kuitansi-KWT"},
		},
		{
			nama: "nama berbaris-baris dipadatkan jadi satu baris", urlWeb: "https://masjid.example", adaPDF: true,
			ubah: func(d *domain.Donasi, _ *kuitansi.Identitas) {
				d.Nama = "Budi\n\nKlik http://evil.example\r\n  sekarang"
			},
			teks:  []string{"Donatur: Budi Klik http://evil.example sekarang\n"},
			html:  []string{">Budi Klik http://evil.example sekarang<"},
			tidak: []string{"\nKlik http://evil.example"},
		},
		{
			nama: "HTML pada nama donatur di-escape", urlWeb: "https://masjid.example", adaPDF: true,
			ubah: func(d *domain.Donasi, _ *kuitansi.Identitas) {
				d.Nama = `<script>alert(1)</script><img src=x onerror=alert(2)>`
			},
			html: []string{"&lt;script&gt;alert(1)&lt;/script&gt;", "&lt;img src=x onerror=alert(2)&gt;"},
			// Hanya HTML yang harus ter-escape; bagian text/plain tidak menafsirkan tag.
			tidakHTML: []string{"<script>", "<img src=x"},
		},
		{
			nama: "nama masjid ber-& di-escape di HTML", urlWeb: "https://masjid.example", adaPDF: true,
			ubah: func(_ *domain.Donasi, m *kuitansi.Identitas) { m.Nama = "Masjid & Mushala <Al-Barokah>" },
			teks: []string{"sistem Masjid & Mushala <Al-Barokah>"},
			html: []string{"Masjid &amp; Mushala &lt;Al-Barokah&gt;"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			d, m := dasar, masjid
			if tt.ubah != nil {
				tt.ubah(&d, &m)
			}
			p := pesanDonasi(d, tt.urlWeb, m, tt.adaPDF)
			if tt.subjek != "" && p.Subjek != tt.subjek {
				t.Errorf("subjek = %q, mau %q", p.Subjek, tt.subjek)
			}
			if strings.ContainsAny(p.Subjek, "\r\n") {
				t.Errorf("subjek berbaris-baris: %q", p.Subjek)
			}
			for _, s := range tt.teks {
				if !strings.Contains(p.Teks, s) {
					t.Errorf("teks tidak memuat %q:\n%s", s, p.Teks)
				}
			}
			for _, s := range tt.html {
				if !strings.Contains(p.HTML, s) {
					t.Errorf("HTML tidak memuat %q", s)
				}
			}
			for _, s := range tt.tidak {
				for nama, isi := range map[string]string{"subjek": p.Subjek, "teks": p.Teks, "HTML": p.HTML} {
					if strings.Contains(isi, s) {
						t.Errorf("%s tidak boleh memuat %q", nama, s)
					}
				}
			}
			for _, s := range tt.tidakHTML {
				if strings.Contains(p.HTML, s) {
					t.Errorf("HTML tidak boleh memuat %q", s)
				}
			}
			if len(p.Inline) != 1 || p.Inline[0].CID != "logo-masjid" || p.Inline[0].Tipe != "image/jpeg" || len(p.Inline[0].Isi) == 0 {
				t.Errorf("logo inline = %+v", p.Inline)
			}
			if len(p.Lampiran) != 0 {
				t.Error("pesanDonasi tidak boleh melampirkan apa pun; PDF dilampirkan susunPesan")
			}
		})
	}
}

// Email hanya untuk perubahan menunggu → diterima pada donasi web: tidak untuk
// donasi yang masih menunggu/gagal/kedaluwarsa, tidak untuk notifikasi ulang,
// dan tidak untuk catatan petugas.
func TestPemberitahuanDonasi_HanyaSaatBaruDiterima(t *testing.T) {
	e := baruDonasi(t)
	p := &pemberiPalsu{}
	e.svc.AturPemberitahu(p, "https://masjid.example/")
	e.svc.AturMasjid(kuitansi.Identitas{Nama: "Masjid Uji", Alamat: "Jl. Uji No. 1"})
	cek := func(langkah string, mau int) {
		t.Helper()
		e.svc.TungguPemberitahuan()
		if n := p.jumlah(); n != mau {
			t.Fatalf("%s: %d email, mau %d", langkah, n, mau)
		}
	}

	if _, err := e.svc.CatatDonasiManual(bg, e.user(t), inputManual()); err != nil {
		t.Fatal(err)
	}
	cek("catatan petugas", 0)

	dibuat, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	cek("donasi web baru dibuat (belum bayar)", 0)
	id := e.idDariKode(t, dibuat.Kode)

	if _, err := e.bayar(t, id, 50_000, "pending"); err != nil {
		t.Fatal(err)
	}
	cek("status pending", 0)

	if h, err := e.bayar(t, id, 50_000, "settlement"); err != nil || !h.Berubah {
		t.Fatalf("settlement: %+v, %v", h, err)
	}
	cek("bayar berhasil", 1)

	p.mu.Lock()
	pesan := p.surat[0]
	p.mu.Unlock()
	for nama, isi := range map[string]string{"teks": pesan.Teks, "HTML": pesan.HTML} {
		for _, s := range []string{"Rp 50.000", "Hamba Allah", "Zakat Fitrah", "16 Oktober 2026, 10.00 WIB", "https://masjid.example/penerimaan/" + id, "Masjid Uji"} {
			if !strings.Contains(isi, s) {
				t.Errorf("%s tidak memuat %q", nama, s)
			}
		}
		for _, rahasia := range []string{dibuat.Kode, "081234567890"} {
			if strings.Contains(isi, rahasia) {
				t.Errorf("%s membocorkan %q", nama, rahasia)
			}
		}
	}
	if len(pesan.Lampiran) != 1 {
		t.Fatalf("lampiran = %d, mau 1 (PDF kuitansi)", len(pesan.Lampiran))
	}
	pdf := pesan.Lampiran[0]
	if pdf.Tipe != "application/pdf" || !strings.HasPrefix(pdf.Nama, "Kuitansi-KWT-2026-") || !strings.HasSuffix(pdf.Nama, ".pdf") {
		t.Errorf("lampiran = %s %s", pdf.Nama, pdf.Tipe)
	}
	if !strings.HasPrefix(string(pdf.Isi), "%PDF-") || len(pdf.Isi) < 2000 {
		t.Errorf("isi lampiran bukan PDF utuh (%d byte)", len(pdf.Isi))
	}
	if !strings.Contains(pesan.Teks, pdf.Nama) {
		t.Errorf("teks tidak menyebut nama lampiran %q", pdf.Nama)
	}

	for _, status := range []string{"settlement", "expire", "cancel"} {
		if _, err := e.bayar(t, id, 50_000, status); err != nil {
			t.Fatal(err)
		}
	}
	cek("notifikasi ulang dari gateway", 1)

	kedua, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	if _, err := e.bayar(t, e.idDariKode(t, kedua.Kode), 50_000, "expire"); err != nil {
		t.Fatal(err)
	}
	cek("donasi kedua kedaluwarsa", 1)
}

// SMTP mati tidak boleh menggagalkan donasi yang sudah dibayar: gateway akan
// mengirim ulang notifikasi bila kita membalas galat, dan uang sudah masuk.
func TestPemberitahuanDonasi_GagalKirimTidakMenggagalkanDonasi(t *testing.T) {
	e := baruDonasi(t)
	p := &pemberiPalsu{err: errors.New("smtp: koneksi ditolak")}
	e.svc.AturPemberitahu(p, "https://masjid.example")

	dibuat, _ := e.svc.BuatDonasiWeb(bg, inputWeb())
	id := e.idDariKode(t, dibuat.Kode)
	h, err := e.bayar(t, id, 50_000, "settlement")
	if err != nil || !h.Berubah || h.Status != domain.DonasiDiterima {
		t.Fatalf("TerimaNotifikasi = %+v, %v (harus tetap sukses)", h, err)
	}
	e.svc.TungguPemberitahuan()
	if p.jumlah() != 1 {
		t.Errorf("pengiriman dicoba %d kali, mau 1", p.jumlah())
	}
	if v, _ := e.svc.AmbilDonasi(bg, id, false); v.Status != domain.DonasiDiterima || v.NomorKuitansi == "" {
		t.Errorf("donasi = %+v, harus diterima dan berkuitansi", v)
	}
}

// Identitas masjid yang belum dipasang tidak boleh menghasilkan kop kosong.
func TestIdentitasMasjid_NamaCadangan(t *testing.T) {
	s := &Service{}
	if got := s.identitasMasjid().Nama; got != "Masjid" {
		t.Errorf("nama cadangan = %q", got)
	}
	s.AturMasjid(kuitansi.Identitas{Nama: "Masjid Uji", Alamat: "Jl. A"})
	if i := s.identitasMasjid(); i.Nama != "Masjid Uji" || i.Alamat != "Jl. A" {
		t.Errorf("identitas = %+v", i)
	}
}
