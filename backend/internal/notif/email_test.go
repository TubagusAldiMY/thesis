package notif

import (
	"bufio"
	"context"
	"io"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"testing"
	"time"
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

	raw, err := bangunPesan(*dari, *ke, subjek, isi, time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC))
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
	if err := s.email(t).Kirim(context.Background(), "Donasi baru", "Isi pesan"); err != nil {
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
	if err := s.email(t).Kirim(ctx, "x", "y"); err == nil {
		t.Fatal("Kirim ke server diam harus galat")
	}
	if d := time.Since(mulai); d > 1500*time.Millisecond {
		t.Errorf("Kirim menunggu %v, tenggat 300ms tidak ditegakkan", d)
	}
}
