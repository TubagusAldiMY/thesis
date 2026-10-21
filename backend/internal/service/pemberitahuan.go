package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"spkzakat/internal/domain"
)

// batasPemberitahuan = waktu maksimal satu email "donasi baru"; setelah itu
// dianggap gagal dan hanya dicatat ke log.
const batasPemberitahuan = 30 * time.Second

// AturPemberitahu memasang pengirim email "donasi baru" ke Ketua. Tanpa ini
// donasi tetap diproses seperti biasa, hanya tidak ada pemberitahuan. urlWeb
// (alamat publik frontend) dipakai untuk tautan ke halaman detail donasi;
// kosong = email dikirim tanpa tautan.
func (s *Service) AturPemberitahu(p domain.Pemberitahu, urlWeb string) {
	s.pemberi = p
	if urlWeb != "" {
		s.urlWeb = strings.TrimRight(urlWeb, "/")
	}
}

// TungguPemberitahuan memblokir sampai semua email yang sedang dikirim selesai
// (berhasil atau gagal). Dipanggil saat shutdown supaya email tidak terpotong,
// dan oleh test sebagai pengganti sleep.
func (s *Service) TungguPemberitahuan() { s.latar.Wait() }

// beritahuDonasi mengirim email donasi yang BARU SAJA diterima, di goroutine
// sendiri: webhook gateway harus dijawab cepat, dan SMTP yang lambat atau mati
// tidak boleh menggagalkan donasi yang sudah tersimpan. Kegagalan hanya dicatat
// ke log (tanpa alamat email, nama, atau isi pesan).
//
// ponytail: kirim-lalu-lupa, tanpa antrean/retry. Bila server mati tepat setelah
// commit atau SMTP sedang down, email itu hilang (donasinya tetap tercatat dan
// terlihat di menu Penerimaan). Pindah ke tabel outbox + worker bila pemberitahuan
// harus dijamin terkirim.
func (s *Service) beritahuDonasi(ctx context.Context, d domain.Donasi) {
	if s.pemberi == nil {
		return
	}
	subjek, isi := pesanDonasi(d, s.urlWeb)
	// WithoutCancel: ctx permintaan dibatalkan begitu respons ke gateway terkirim,
	// padahal pengiriman email baru berjalan; nilai di ctx (mis. untuk log) tetap terbawa.
	ctx = context.WithoutCancel(ctx)
	s.latar.Add(1)
	go func() {
		defer s.latar.Done()
		ctx, batal := context.WithTimeout(ctx, batasPemberitahuan)
		defer batal()
		if err := s.pemberi.Kirim(ctx, subjek, isi); err != nil {
			slog.ErrorContext(ctx, "pemberitahuan donasi ke Ketua gagal terkirim", "donasi_id", d.ID, "error", err.Error())
			return
		}
		slog.InfoContext(ctx, "pemberitahuan donasi ke Ketua terkirim", "donasi_id", d.ID)
	}()
}

// pesanDonasi menyusun subjek dan isi email. Sengaja TIDAK memuat No. HP, kode
// publik kuitansi (rahasia muzaki), maupun keterangan bebas: tautannya menuju
// halaman pengurus yang wajib login, bukan ke kuitansi publik.
func pesanDonasi(d domain.Donasi, urlWeb string) (subjek, isi string) {
	subjek = fmt.Sprintf("Donasi baru Rp %s - %s", titik(d.Jumlah), d.NomorKuitansi)

	var b strings.Builder
	b.WriteString("Assalamu'alaikum,\n\nAda donasi baru yang diterima lewat halaman donasi online.\n\n")
	baris := func(label, nilai string) { fmt.Fprintf(&b, "%-13s: %s\n", label, nilai) }
	baris("No. kuitansi", d.NomorKuitansi)
	// Nama diisi muzaki: dipadatkan jadi satu baris supaya tidak bisa menyisipkan
	// paragraf/tautan palsu ke dalam email resmi dari sistem.
	baris("Donatur", strings.Join(strings.Fields(d.Nama), " "))
	baris("Jenis", namaJenisDonasi[d.Jenis])
	baris("Jumlah", "Rp "+titik(d.Jumlah))
	if d.DibayarPada != nil {
		baris("Diterima", d.DibayarPada.In(wib).Format("02/01/2006 15.04")+" WIB")
	}
	if urlWeb != "" {
		fmt.Fprintf(&b, "\nDetail (masuk sebagai pengurus dulu):\n%s/penerimaan/%s\n", urlWeb, d.ID)
	}
	b.WriteString("\nEmail ini dikirim otomatis oleh sistem; tidak perlu dibalas.\n")
	return subjek, b.String()
}
