package notif

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

func TestNewEmail_Validasi(t *testing.T) {
	tests := []struct {
		nama           string
		host, dari, ke string
		galat          bool
		keterangan     string
	}{
		{"valid", "smtp.example.com", "sistem@masjid.id", "ketua@masjid.id", false, ""},
		{"valid dengan nama", "smtp.example.com", `"SPK Zakat" <sistem@masjid.id>`, "ketua@masjid.id", false, ""},
		{"host kosong", "", "sistem@masjid.id", "ketua@masjid.id", true, "SMTP_HOST"},
		{"pengirim kosong", "smtp.example.com", "", "ketua@masjid.id", true, "pengirim"},
		{"penerima bukan email", "smtp.example.com", "sistem@masjid.id", "ketua", true, "penerima"},
		{"penerima disusupi header", "smtp.example.com", "sistem@masjid.id", "ketua@masjid.id\r\nBcc: x@y.z", true, "penerima"},
		{"penerima dua alamat", "smtp.example.com", "sistem@masjid.id", "a@masjid.id, b@masjid.id", true, "penerima"},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			_, err := NewEmail(tt.host, "", "", "", tt.dari, tt.ke)
			if (err != nil) != tt.galat {
				t.Fatalf("err = %v, galat? %v", err, tt.galat)
			}
			if err != nil && !strings.Contains(err.Error(), tt.keterangan) {
				t.Errorf("pesan galat %q tidak menyebut %q", err, tt.keterangan)
			}
		})
	}
}

func TestNewEmail_PortBawaanDanSMTPS(t *testing.T) {
	e, err := NewEmail("h", "", "", "", "a@b.id", "c@d.id")
	if err != nil || e.port != "587" || e.tlsLangsung {
		t.Errorf("bawaan = %+v, %v (mau port 587, STARTTLS)", e, err)
	}
	if e, _ = NewEmail("h", "465", "", "", "a@b.id", "c@d.id"); !e.tlsLangsung {
		t.Error("port 465 harus TLS langsung")
	}
}

// Subjek berisi CR/LF tidak boleh menjadi header baru; huruf non-ASCII harus
// utuh setelah dibaca balik.
func TestBangunPesan_AmanDanBisaDibacaBalik(t *testing.T) {
	dari, _ := mail.ParseAddress("sistem@masjid.id")
	ke, _ := mail.ParseAddress("ketua@masjid.id")
	subjek := "Donasi baru Rp 50.000 – Jamʻiyyah\r\nBcc: penyusup@evil.id"
	isi := "Baris satu\nBaris dua — é\n" + strings.Repeat("panjang ", 40)

	raw, err := bangunPesan(*dari, *ke, domain.Pesan{Subjek: subjek, Teks: isi}, time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	m, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("pesan tidak terbaca: %v", err)
	}
	if m.Header.Get("Bcc") != "" {
		t.Fatal("CR/LF di subjek lolos menjadi header Bcc")
	}
	gotSubjek, err := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
	if err != nil || !strings.Contains(gotSubjek, "Jamʻiyyah") || !strings.Contains(gotSubjek, "Rp 50.000") {
		t.Errorf("subjek terbaca balik = %q, %v", gotSubjek, err)
	}
	if m.Header.Get("To") != "<ketua@masjid.id>" || !strings.HasPrefix(m.Header.Get("Message-Id"), "<") {
		t.Errorf("header To/Message-ID = %q / %q", m.Header.Get("To"), m.Header.Get("Message-Id"))
	}
	body, err := io.ReadAll(quotedprintable.NewReader(m.Body))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.ReplaceAll(string(body), "\r\n", "\n"); got != isi {
		t.Errorf("isi terbaca balik:\n%q\nmau:\n%q", got, isi)
	}
}

// bacaBagian membaca satu bagian multipart mentah (tanpa dekode otomatis) dan
// mendekode isinya sesuai Content-Transfer-Encoding.
func bacaBagian(t *testing.T, p *multipart.Part) []byte {
	t.Helper()
	var r io.Reader = p
	switch cte := p.Header.Get("Content-Transfer-Encoding"); cte {
	case "quoted-printable":
		r = quotedprintable.NewReader(p)
	case "base64":
		r = base64.NewDecoder(base64.StdEncoding, p)
	default:
		t.Fatalf("Content-Transfer-Encoding tak terduga: %q", cte)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("dekode bagian: %v", err)
	}
	return b
}

// bagianBerikut mengambil bagian berikutnya dan memastikan tipenya; mengembalikan
// bagian + parameter tipenya (boundary, name).
func bagianBerikut(t *testing.T, mr *multipart.Reader, tipe string) (*multipart.Part, map[string]string) {
	t.Helper()
	p, err := mr.NextRawPart()
	if err != nil {
		t.Fatalf("bagian %s tidak ada: %v", tipe, err)
	}
	mt, params, err := mime.ParseMediaType(p.Header.Get("Content-Type"))
	if err != nil || mt != tipe {
		t.Fatalf("tipe bagian = %q (%v), mau %q", mt, err, tipe)
	}
	return p, params
}

func tidakAdaLagi(t *testing.T, mr *multipart.Reader) {
	t.Helper()
	if p, err := mr.NextRawPart(); err != io.EOF {
		t.Fatalf("masih ada bagian lagi: %v, %v", p, err)
	}
}

// Struktur penuh: mixed [ alternative [ plain, related [ html, logo ] ], pdf ].
// Dibaca balik dengan parser stdlib; semua isi harus utuh, termasuk biner dan huruf non-ASCII.
func TestBangunPesan_HTMLInlineDanLampiran(t *testing.T) {
	dari, _ := mail.ParseAddress("sistem@masjid.id")
	ke, _ := mail.ParseAddress("ketua@masjid.id")
	pdf := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte{0, 1, 2, 0xff, 0xfe}, 100)...) // biner, jauh > 76 kolom base64
	logo := []byte("\xff\xd8\xff\xe0LOGO-PALSU")
	p := domain.Pesan{
		Subjek:   "Donasi baru",
		Teks:     "versi teks é\nbaris dua",
		HTML:     `<p style="color:#17473a">versi <b>html</b> é</p><img src="cid:logo-masjid">`,
		Inline:   []domain.Berkas{{Nama: "logo.jpg", Tipe: "image/jpeg", CID: "logo-masjid", Isi: logo}},
		Lampiran: []domain.Berkas{{Nama: "Kuitansi-KWT-2026-0007.pdf", Tipe: "application/pdf", Isi: pdf}},
	}
	raw, err := bangunPesan(*dari, *ke, p, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for i, baris := range strings.Split(string(raw), "\r\n") {
		if len(baris) > 998 {
			t.Fatalf("baris %d sepanjang %d (batas RFC 998)", i, len(baris))
		}
	}

	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	mt, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/mixed" {
		t.Fatalf("tipe teratas = %q (%v)", mt, err)
	}
	mixed := multipart.NewReader(m.Body, params["boundary"])

	// 1. badan: alternative
	badan, bp := bagianBerikut(t, mixed, "multipart/alternative")
	alt := multipart.NewReader(badan, bp["boundary"])
	teks, _ := bagianBerikut(t, alt, "text/plain")
	if got := strings.ReplaceAll(string(bacaBagian(t, teks)), "\r\n", "\n"); got != p.Teks {
		t.Errorf("teks = %q, mau %q", got, p.Teks)
	}
	rel, rp := bagianBerikut(t, alt, "multipart/related") // HTML terakhir = diutamakan klien
	related := multipart.NewReader(rel, rp["boundary"])
	html, _ := bagianBerikut(t, related, "text/html")
	if got := string(bacaBagian(t, html)); got != p.HTML {
		t.Errorf("html = %q, mau %q", got, p.HTML)
	}
	gambar, _ := bagianBerikut(t, related, "image/jpeg")
	if gambar.Header.Get("Content-ID") != "<logo-masjid>" || !strings.HasPrefix(gambar.Header.Get("Content-Disposition"), "inline") {
		t.Errorf("gambar inline: Content-ID=%q Disposition=%q", gambar.Header.Get("Content-ID"), gambar.Header.Get("Content-Disposition"))
	}
	if got := bacaBagian(t, gambar); !bytes.Equal(got, logo) {
		t.Error("isi logo berubah setelah encode/decode")
	}
	tidakAdaLagi(t, related)
	tidakAdaLagi(t, alt)

	// 2. lampiran PDF
	lamp, lp := bagianBerikut(t, mixed, "application/pdf")
	if lp["name"] != "Kuitansi-KWT-2026-0007.pdf" {
		t.Errorf("name = %q", lp["name"])
	}
	dispo, dp, err := mime.ParseMediaType(lamp.Header.Get("Content-Disposition"))
	if err != nil || dispo != "attachment" || dp["filename"] != "Kuitansi-KWT-2026-0007.pdf" {
		t.Errorf("Content-Disposition = %q %v (%v)", dispo, dp, err)
	}
	if got := bacaBagian(t, lamp); !bytes.Equal(got, pdf) {
		t.Error("isi PDF berubah setelah encode/decode")
	}
	tidakAdaLagi(t, mixed)
}

// HTML tanpa lampiran = alternative di tingkat teratas (tanpa pembungkus mixed).
func TestBangunPesan_HTMLTanpaLampiran(t *testing.T) {
	dari, _ := mail.ParseAddress("sistem@masjid.id")
	ke, _ := mail.ParseAddress("ketua@masjid.id")
	raw, err := bangunPesan(*dari, *ke, domain.Pesan{Subjek: "x", Teks: "teks", HTML: "<p>html</p>"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	m, _ := mail.ReadMessage(bytes.NewReader(raw))
	mt, params, _ := mime.ParseMediaType(m.Header.Get("Content-Type"))
	if mt != "multipart/alternative" {
		t.Fatalf("tipe teratas = %q", mt)
	}
	alt := multipart.NewReader(m.Body, params["boundary"])
	// Isi dibaca sebelum pindah ke bagian berikutnya: multipart.Reader membuang sisa bagian lama.
	teks, _ := bagianBerikut(t, alt, "text/plain")
	if got := string(bacaBagian(t, teks)); got != "teks" {
		t.Errorf("teks = %q", got)
	}
	html, _ := bagianBerikut(t, alt, "text/html")
	if got := string(bacaBagian(t, html)); got != "<p>html</p>" {
		t.Errorf("html = %q", got)
	}
	tidakAdaLagi(t, alt)
}

// Nama/tipe/CID berkas dari pemanggil tidak boleh menyisipkan header atau merusak struktur.
func TestBangunPesan_BerkasTidakValidDitolak(t *testing.T) {
	dari, _ := mail.ParseAddress("sistem@masjid.id")
	ke, _ := mail.ParseAddress("ketua@masjid.id")
	tests := []struct {
		nama string
		f    domain.Berkas
	}{
		{"nama disusupi header", domain.Berkas{Nama: "a.pdf\r\nBcc: x@y.z", Tipe: "application/pdf"}},
		{"tipe disusupi header", domain.Berkas{Nama: "a.pdf", Tipe: "application/pdf\r\nBcc: x@y.z"}},
		{"nama kosong", domain.Berkas{Nama: "", Tipe: "application/pdf"}},
		{"tipe kosong", domain.Berkas{Nama: "a.pdf"}},
		{"CID memuat kurung sudut", domain.Berkas{Nama: "a.jpg", Tipe: "image/jpeg", CID: "x>\r\nBcc: y"}},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			if _, err := bangunPesan(*dari, *ke, domain.Pesan{Subjek: "x", Teks: "y", Lampiran: []domain.Berkas{tt.f}}, time.Now()); err == nil {
				t.Error("harus galat")
			}
		})
	}
}

// smtpPalsu = server SMTP minimal di localhost: cukup untuk memeriksa urutan
// perintah dan isi pesan tanpa jaringan luar.
type smtpPalsu struct {
	ln    net.Listener
	hasil chan hasilSMTP
}

type hasilSMTP struct{ dari, ke, data string }

func jalankanSMTPPalsu(t *testing.T, diam bool) *smtpPalsu {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &smtpPalsu{ln: ln, hasil: make(chan hasilSMTP, 1)}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if diam {
			time.Sleep(2 * time.Second) // tidak pernah menyapa
			return
		}
		r, w := bufio.NewReader(conn), func(l string) { _, _ = conn.Write([]byte(l + "\r\n")) }
		var h hasilSMTP
		w("220 palsu siap")
		for {
			l, err := r.ReadString('\n')
			if err != nil {
				return
			}
			l = strings.TrimSpace(l)
			switch {
			case strings.HasPrefix(l, "EHLO"):
				w("250 palsu")
			case strings.HasPrefix(l, "MAIL FROM:"):
				h.dari = l
				w("250 ok")
			case strings.HasPrefix(l, "RCPT TO:"):
				h.ke = l
				w("250 ok")
			case l == "DATA":
				w("354 lanjut")
				var sb strings.Builder
				for {
					b, err := r.ReadString('\n')
					if err != nil || b == ".\r\n" {
						break
					}
					sb.WriteString(b)
				}
				h.data = sb.String()
				w("250 diterima")
			case l == "QUIT":
				w("221 sampai jumpa")
				s.hasil <- h
				return
			}
		}
	}()
	return s
}

func (s *smtpPalsu) email(t *testing.T) *Email {
	t.Helper()
	host, port, _ := net.SplitHostPort(s.ln.Addr().String())
	e, err := NewEmail(host, port, "", "", "sistem@masjid.id", "ketua@masjid.id")
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestKirim_PercakapanSMTP(t *testing.T) {
	s := jalankanSMTPPalsu(t, false)
	if err := s.email(t).Kirim(context.Background(), domain.Pesan{Subjek: "Donasi baru", Teks: "Isi pesan"}); err != nil {
		t.Fatalf("Kirim: %v", err)
	}
	select {
	case h := <-s.hasil:
		if h.dari != "MAIL FROM:<sistem@masjid.id>" || h.ke != "RCPT TO:<ketua@masjid.id>" {
			t.Errorf("amplop = %q / %q", h.dari, h.ke)
		}
		if !strings.Contains(h.data, "Subject: Donasi baru") || !strings.Contains(h.data, "Isi pesan") {
			t.Errorf("data pesan = %q", h.data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server palsu tidak menerima pesan sampai QUIT")
	}
}

// Server yang tidak menjawab tidak boleh membuat Kirim menggantung.
func TestKirim_ServerDiamKenaTenggat(t *testing.T) {
	s := jalankanSMTPPalsu(t, true)
	ctx, batal := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer batal()
	mulai := time.Now()
	if err := s.email(t).Kirim(ctx, domain.Pesan{Subjek: "x", Teks: "y"}); err == nil {
		t.Fatal("Kirim ke server diam harus galat")
	}
	if d := time.Since(mulai); d > 1500*time.Millisecond {
		t.Errorf("Kirim menunggu %v, tenggat 300ms tidak ditegakkan", d)
	}
}
