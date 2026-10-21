package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

// pemberiPalsu merekam email yang "dikirim"; err != nil membuat Kirim gagal
// (pesan tetap terekam, supaya terlihat bahwa pengiriman memang dicoba).
type pemberiPalsu struct {
	mu    sync.Mutex
	surat []string // subjek + "\n" + isi
	err   error
}

func (p *pemberiPalsu) Kirim(_ context.Context, subjek, isi string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.surat = append(p.surat, subjek+"\n"+isi)
	return p.err
}

func (p *pemberiPalsu) jumlah() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.surat)
}

func TestPesanDonasi(t *testing.T) {
	bayar := time.Date(2026, 12, 31, 17, 30, 0, 0, time.UTC) // 01/01/2027 00.30 WIB
	dasar := domain.Donasi{
		ID: "id-donasi-1", Nama: "Budi", Jenis: domain.DonasiInfak, Jumlah: 1_250_000,
		NomorKuitansi: "KWT/2026/0007", DibayarPada: &bayar,
		KodePublik: "kode-rahasia-muzaki", Keterangan: "keterangan bebas",
	}
	tests := []struct {
		nama       string
		ubah       func(*domain.Donasi)
		urlWeb     string
		mengandung []string
		tidak      []string
	}{
		{
			nama: "semua data tampil, tanggal dalam WIB", urlWeb: "https://masjid.example",
			mengandung: []string{
				"Donasi baru Rp 1.250.000 - KWT/2026/0007", "KWT/2026/0007", "Budi", "Infak", "Rp 1.250.000",
				"01/01/2027 00.30 WIB", "https://masjid.example/penerimaan/id-donasi-1",
			},
			// kode publik = rahasia muzaki; keterangan bebas bisa memuat data pribadi
			tidak: []string{"kode-rahasia-muzaki", "keterangan bebas", "/kuitansi/"},
		},
		{
			nama: "tanpa alamat web tidak ada tautan", urlWeb: "",
			mengandung: []string{"Budi"},
			tidak:      []string{"http", "/penerimaan/"},
		},
		{
			nama:       "nama berbaris-baris dipadatkan jadi satu baris",
			ubah:       func(d *domain.Donasi) { d.Nama = "Budi\n\nKlik http://evil.example\r\n  sekarang" },
			mengandung: []string{"Donatur      : Budi Klik http://evil.example sekarang\n"},
			tidak:      []string{"\nKlik http://evil.example"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			d := dasar
			if tt.ubah != nil {
				tt.ubah(&d)
			}
			subjek, isi := pesanDonasi(d, tt.urlWeb)
			semua := subjek + "\n" + isi
			for _, s := range tt.mengandung {
				if !strings.Contains(semua, s) {
					t.Errorf("pesan tidak memuat %q:\n%s", s, semua)
				}
			}
			for _, s := range tt.tidak {
				if strings.Contains(semua, s) {
					t.Errorf("pesan tidak boleh memuat %q:\n%s", s, semua)
				}
			}
			if strings.ContainsAny(subjek, "\r\n") {
				t.Errorf("subjek berbaris-baris: %q", subjek)
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
	isi := p.surat[0]
	p.mu.Unlock()
	for _, s := range []string{"Rp 50.000", "Hamba Allah", "Zakat Fitrah", "16/10/2026 10.00 WIB", "https://masjid.example/penerimaan/" + id} {
		if !strings.Contains(isi, s) {
			t.Errorf("email tidak memuat %q:\n%s", s, isi)
		}
	}
	for _, rahasia := range []string{dibuat.Kode, "081234567890"} {
		if strings.Contains(isi, rahasia) {
			t.Errorf("email membocorkan %q", rahasia)
		}
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
