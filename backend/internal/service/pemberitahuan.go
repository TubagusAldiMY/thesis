package service

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"log/slog"
	"strings"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/kuitansi"
)

// batasPemberitahuan = waktu maksimal satu email "donasi baru"; setelah itu
// dianggap gagal dan hanya dicatat ke log.
const batasPemberitahuan = 30 * time.Second

// templates/donasi_email.html di-embed ke binary. html/template meng-escape semua
// nilai yang disisipkan (nama donatur ditulis muzaki sendiri), jadi HTML dari
// pengguna tidak bisa masuk ke email.
//
//go:embed templates/donasi_email.html
var sumberEmailDonasi string

var tmplEmailDonasi = template.Must(template.New("donasi_email").Parse(sumberEmailDonasi))

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

// AturMasjid memasang identitas untuk kop email dan PDF kuitansi.
func (s *Service) AturMasjid(i kuitansi.Identitas) { s.masjid = i }

// identitasMasjid = identitas terpasang, dengan nama cadangan supaya kop tidak pernah kosong.
func (s *Service) identitasMasjid() kuitansi.Identitas {
	i := s.masjid
	if strings.TrimSpace(i.Nama) == "" {
		i.Nama = "Masjid"
	}
	return i
}

// TungguPemberitahuan memblokir sampai semua email yang sedang dikirim selesai
// (berhasil atau gagal). Dipanggil saat shutdown supaya email tidak terpotong,
// dan oleh test sebagai pengganti sleep.
func (s *Service) TungguPemberitahuan() { s.latar.Wait() }

// beritahuDonasi mengirim email donasi yang BARU SAJA diterima, di goroutine
// sendiri: webhook gateway harus dijawab cepat, dan SMTP yang lambat atau mati
// tidak boleh menggagalkan donasi yang sudah tersimpan. Pembuatan PDF kuitansi
// juga di goroutine ini. Kegagalan hanya dicatat ke log (tanpa alamat email,
// nama, atau isi pesan).
//
// ponytail: kirim-lalu-lupa, tanpa antrean/retry. Bila server mati tepat setelah
// commit atau SMTP sedang down, email itu hilang (donasinya tetap tercatat dan
// terlihat di menu Penerimaan). Pindah ke tabel outbox + worker bila pemberitahuan
// harus dijamin terkirim.
func (s *Service) beritahuDonasi(ctx context.Context, d domain.Donasi) {
	if s.pemberi == nil {
		return
	}
	masjid, urlWeb := s.identitasMasjid(), s.urlWeb
	// WithoutCancel: ctx permintaan dibatalkan begitu respons ke gateway terkirim,
	// padahal pengiriman email baru berjalan; nilai di ctx (mis. untuk log) tetap terbawa.
	ctx = context.WithoutCancel(ctx)
	s.latar.Add(1)
	go func() {
		defer s.latar.Done()
		ctx, batal := context.WithTimeout(ctx, batasPemberitahuan)
		defer batal()
		if err := s.pemberi.Kirim(ctx, susunPesan(ctx, d, urlWeb, masjid)); err != nil {
			slog.ErrorContext(ctx, "pemberitahuan donasi ke Ketua gagal terkirim", "donasi_id", d.ID, "error", err.Error())
			return
		}
		slog.InfoContext(ctx, "pemberitahuan donasi ke Ketua terkirim", "donasi_id", d.ID)
	}()
}

// susunPesan membuat PDF kuitansi lalu merakit email beserta lampirannya. PDF yang
// gagal dibuat tidak menggagalkan pemberitahuan: email tetap terkirim tanpa lampiran.
func susunPesan(ctx context.Context, d domain.Donasi, urlWeb string, masjid kuitansi.Identitas) domain.Pesan {
	pdf, err := kuitansi.PDF(dataKuitansi(d, masjid))
	if err != nil {
		slog.WarnContext(ctx, "PDF kuitansi gagal dibuat; email dikirim tanpa lampiran", "donasi_id", d.ID, "error", err.Error())
		pdf = nil
	}
	p := pesanDonasi(d, urlWeb, masjid, len(pdf) > 0)
	if len(pdf) > 0 {
		p.Lampiran = []domain.Berkas{{Nama: kuitansi.NamaBerkas(d.NomorKuitansi), Tipe: "application/pdf", Isi: pdf}}
	}
	return p
}

// dataKuitansi memetakan donasi yang sudah diterima ke isi PDF kuitansi.
func dataKuitansi(d domain.Donasi, masjid kuitansi.Identitas) kuitansi.Data {
	judul := "KUITANSI ZAKAT"
	if d.Jenis == domain.DonasiInfak {
		judul = "KUITANSI INFAK"
	}
	tanggal := time.Now()
	if d.DibayarPada != nil {
		tanggal = *d.DibayarPada
	}
	return kuitansi.Data{
		Masjid: masjid, Judul: judul, Nomor: d.NomorKuitansi, Donatur: d.Nama, Jenis: namaJenisDonasi[d.Jenis],
		Jumlah: d.Jumlah, Metode: kuitansi.NamaMetode(d.Metode), Tanggal: tanggal,
	}
}

// dataEmail = nilai yang disisipkan ke templates/donasi_email.html dan versi teks.
type dataEmail struct {
	NamaMasjid, Nomor, Donatur, Jenis, Jumlah, Waktu string
	Tautan, NamaPDF                                  string
	AdaLogo, AdaPDF                                  bool
}

// pesanDonasi menyusun subjek, versi teks, dan HTML email (tanpa lampiran PDF;
// adaPDF hanya menentukan apakah email menyebut lampiran). Sengaja TIDAK memuat
// No. HP, kode publik kuitansi (rahasia muzaki), maupun keterangan bebas: tautannya
// menuju halaman pengurus yang wajib login, bukan ke kuitansi publik.
func pesanDonasi(d domain.Donasi, urlWeb string, masjid kuitansi.Identitas, adaPDF bool) domain.Pesan {
	x := dataEmail{
		NamaMasjid: masjid.Nama,
		Nomor:      d.NomorKuitansi,
		// Nama diisi muzaki: dipadatkan jadi satu baris supaya tidak bisa menyisipkan
		// paragraf/tautan palsu ke dalam email resmi dari sistem.
		Donatur: strings.Join(strings.Fields(d.Nama), " "),
		Jenis:   namaJenisDonasi[d.Jenis],
		Jumlah:  kuitansi.Rupiah(d.Jumlah),
		AdaLogo: len(kuitansi.Logo) > 0,
		AdaPDF:  adaPDF,
		NamaPDF: kuitansi.NamaBerkas(d.NomorKuitansi),
	}
	if d.DibayarPada != nil {
		w := d.DibayarPada.In(wib)
		x.Waktu = kuitansi.TanggalPanjang(w) + ", " + w.Format("15.04") + " WIB"
	}
	if urlWeb != "" {
		x.Tautan = urlWeb + "/penerimaan/" + d.ID
	}

	p := domain.Pesan{Subjek: fmt.Sprintf("Donasi baru %s - %s", x.Jumlah, d.NomorKuitansi), Teks: teksDonasi(x)}
	var b bytes.Buffer
	if err := tmplEmailDonasi.Execute(&b, x); err != nil {
		// Templatenya tetap, jadi ini praktis tak mungkin; versi teks cukup sebagai cadangan.
		slog.Error("render HTML email donasi gagal; dikirim sebagai teks saja", "donasi_id", d.ID, "error", err.Error())
		return p
	}
	p.HTML = b.String()
	if x.AdaLogo {
		p.Inline = []domain.Berkas{{Nama: "logo-masjid.jpg", Tipe: "image/jpeg", CID: "logo-masjid", Isi: kuitansi.Logo}}
	}
	return p
}

// teksDonasi = versi teks polos email (cadangan klien tanpa HTML, juga yang tampil
// di notifikasi ponsel). Format "Label: nilai" tanpa perataan spasi, karena
// perataan hanya rapi di font monospace.
func teksDonasi(x dataEmail) string {
	var b strings.Builder
	b.WriteString("Assalamu'alaikum,\n\nAda donasi baru yang diterima lewat halaman donasi online.\n\n")
	fmt.Fprintf(&b, "Jumlah: %s\nNo. kuitansi: %s\nDonatur: %s\nJenis: %s\n", x.Jumlah, x.Nomor, x.Donatur, x.Jenis)
	if x.Waktu != "" {
		fmt.Fprintf(&b, "Diterima: %s\n", x.Waktu)
	}
	if x.AdaPDF {
		fmt.Fprintf(&b, "\nKuitansi PDF (A5, siap cetak) terlampir: %s\n", x.NamaPDF)
	}
	if x.Tautan != "" {
		fmt.Fprintf(&b, "\nDetail (masuk sebagai pengurus dulu):\n%s\n", x.Tautan)
	}
	fmt.Fprintf(&b, "\nEmail ini dikirim otomatis oleh sistem %s; tidak perlu dibalas.\n", x.NamaMasjid)
	return b.String()
}
