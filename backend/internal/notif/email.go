// Package notif mengirim pemberitahuan ke pengurus lewat email (SMTP). Hanya
// memakai stdlib (net/smtp): satu penerima tetap (Ketua), teks + HTML + lampiran.
// Satu-satunya paket project yang diimpor adalah domain (tipe Pesan).
package notif

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"spkzakat/internal/domain"
)

// batasKirim = batas total satu pengiriman bila ctx pemanggil tidak punya
// tenggat. Tanpa batas, server SMTP yang diam membuat goroutine menggantung.
const batasKirim = 30 * time.Second

// Email mengirim pesan dari satu pengirim ke satu penerima lewat server SMTP.
// Mengimplementasikan domain.Pemberitahu.
type Email struct {
	host, port  string
	user, pass  string
	dari, ke    mail.Address
	tlsLangsung bool // port 465 (SMTPS): TLS sejak sambungan pertama, bukan STARTTLS
}

// NewEmail memvalidasi isian dan membuat pengirim. user boleh kosong (relay
// lokal tanpa login). Alamat dari/ke harus alamat email tunggal yang valid:
// ini juga yang menutup celah header injection lewat konfigurasi.
func NewEmail(host, port, user, pass, dari, ke string) (*Email, error) {
	if host == "" {
		return nil, errors.New("notif: SMTP_HOST kosong")
	}
	if port == "" {
		port = "587"
	}
	d, err := alamat(dari)
	if err != nil {
		return nil, fmt.Errorf("notif: alamat pengirim: %w", err)
	}
	k, err := alamat(ke)
	if err != nil {
		return nil, fmt.Errorf("notif: alamat penerima: %w", err)
	}
	return &Email{host: host, port: port, user: user, pass: pass, dari: d, ke: k, tlsLangsung: port == "465"}, nil
}

// alamat menerima tepat satu alamat email. Karakter kontrol ditolak eksplisit
// (CR/LF di header = injection) walau parser net/mail umumnya sudah menolaknya.
func alamat(s string) (mail.Address, error) {
	if strings.ContainsAny(s, "\r\n\x00") {
		return mail.Address{}, errors.New("mengandung karakter kontrol")
	}
	a, err := mail.ParseAddress(strings.TrimSpace(s))
	if err != nil {
		return mail.Address{}, errors.New("bukan alamat email yang valid")
	}
	return *a, nil
}

// Kirim mengirim satu email. Auth PlainAuth milik net/smtp menolak mengirim
// password lewat sambungan tanpa TLS ke server non-lokal, jadi bila server
// tidak menawarkan STARTTLS, login gagal dengan aman (password tidak bocor).
func (e *Email) Kirim(ctx context.Context, p domain.Pesan) error {
	pesan, err := bangunPesan(e.dari, e.ke, p, time.Now())
	if err != nil {
		return err
	}
	tenggat, ada := ctx.Deadline()
	if !ada {
		tenggat = time.Now().Add(batasKirim)
	}

	d := net.Dialer{Deadline: tenggat}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(e.host, e.port))
	if err != nil {
		return fmt.Errorf("notif: sambung ke SMTP: %w", err)
	}
	if e.tlsLangsung {
		conn = tls.Client(conn, &tls.Config{ServerName: e.host, MinVersion: tls.VersionTLS12})
	}
	_ = conn.SetDeadline(tenggat) // satu tenggat untuk seluruh percakapan SMTP

	c, err := smtp.NewClient(conn, e.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("notif: sapaan SMTP: %w", err)
	}
	defer c.Close()

	if !e.tlsLangsung {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: e.host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("notif: STARTTLS: %w", err)
			}
		}
	}
	if e.user != "" {
		if err := c.Auth(smtp.PlainAuth("", e.user, e.pass, e.host)); err != nil {
			return fmt.Errorf("notif: login SMTP: %w", err)
		}
	}
	if err := c.Mail(e.dari.Address); err != nil {
		return fmt.Errorf("notif: MAIL FROM: %w", err)
	}
	if err := c.Rcpt(e.ke.Address); err != nil {
		return fmt.Errorf("notif: RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("notif: DATA: %w", err)
	}
	if _, err := w.Write(pesan); err != nil {
		return fmt.Errorf("notif: tulis pesan: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("notif: pesan ditolak server: %w", err)
	}
	return c.Quit()
}

// bangunPesan merakit email UTF-8. Susunan MIME mengikuti isinya:
//
//	teks saja                → text/plain
//	+ HTML                   → multipart/alternative [text/plain, text/html]
//	  + gambar inline        →   ... text/html dibungkus multipart/related bersama gambarnya
//	+ lampiran               → multipart/mixed [(badan di atas), lampiran...]
//
// Subjek di-encode RFC 2047 dan isi quoted-printable supaya aman untuk huruf
// non-ASCII dan baris panjang; encoder subjek juga meng-encode CR/LF, jadi
// subjek tak bisa menyisipkan header. Nama dan tipe berkas diformat lewat
// mime.FormatMediaType dengan alasan yang sama.
func bangunPesan(dari, ke mail.Address, p domain.Pesan, sekarang time.Time) ([]byte, error) {
	for _, f := range append(append([]domain.Berkas{}, p.Inline...), p.Lampiran...) {
		if f.Nama == "" || f.Tipe == "" || strings.ContainsAny(f.Nama+f.Tipe+f.CID, "\r\n\x00<>") {
			return nil, fmt.Errorf("notif: berkas %q tidak valid", f.Nama)
		}
	}
	var id [12]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, fmt.Errorf("notif: crypto/rand gagal: %w", err)
	}
	// Message-ID wajib unik; domainnya diambil dari alamat pengirim.
	domainPengirim := dari.Address[strings.LastIndex(dari.Address, "@")+1:]

	var b bytes.Buffer
	tulis := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	tulis("From", dari.String())
	tulis("To", ke.String())
	tulis("Subject", mime.QEncoding.Encode("utf-8", p.Subjek))
	tulis("Date", sekarang.Format(time.RFC1123Z))
	tulis("Message-ID", "<"+hex.EncodeToString(id[:])+"@"+domainPengirim+">")
	tulis("MIME-Version", "1.0")

	switch {
	case len(p.Lampiran) > 0:
		mw := multipart.NewWriter(&b)
		tulis("Content-Type", "multipart/mixed; boundary="+mw.Boundary())
		b.WriteString("\r\n")
		if err := isiBadan(mw, p); err != nil {
			return nil, err
		}
		for _, f := range p.Lampiran {
			if err := tulisBerkas(mw, f, false); err != nil {
				return nil, err
			}
		}
		if err := mw.Close(); err != nil {
			return nil, err
		}
	case p.HTML != "":
		mw := multipart.NewWriter(&b)
		tulis("Content-Type", "multipart/alternative; boundary="+mw.Boundary())
		b.WriteString("\r\n")
		if err := isiAlternatif(mw, p); err != nil {
			return nil, err
		}
		if err := mw.Close(); err != nil {
			return nil, err
		}
	default:
		tulis("Content-Type", "text/plain; charset=UTF-8")
		tulis("Content-Transfer-Encoding", "quoted-printable")
		b.WriteString("\r\n")
		if err := tulisQP(&b, p.Teks); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}

// tulisQP menulis s sebagai quoted-printable (LF → CRLF).
func tulisQP(w io.Writer, s string) error {
	q := quotedprintable.NewWriter(w)
	if _, err := q.Write([]byte(s)); err != nil {
		return err
	}
	return q.Close()
}

// isiBadan menulis badan pesan ke mw: teks saja, atau multipart/alternative bila ada HTML.
func isiBadan(mw *multipart.Writer, p domain.Pesan) error {
	if p.HTML == "" {
		return tulisTeks(mw, "text/plain", p.Teks)
	}
	return anak(mw, "alternative", func(a *multipart.Writer) error { return isiAlternatif(a, p) })
}

// isiAlternatif menulis versi teks lalu versi HTML (yang terakhir = yang diutamakan klien).
func isiAlternatif(mw *multipart.Writer, p domain.Pesan) error {
	if err := tulisTeks(mw, "text/plain", p.Teks); err != nil {
		return err
	}
	if len(p.Inline) == 0 {
		return tulisTeks(mw, "text/html", p.HTML)
	}
	return anak(mw, "related", func(r *multipart.Writer) error {
		if err := tulisTeks(r, "text/html", p.HTML); err != nil {
			return err
		}
		for _, f := range p.Inline {
			if err := tulisBerkas(r, f, true); err != nil {
				return err
			}
		}
		return nil
	})
}

// anak membuat bagian bersarang bertipe multipart/<subtipe> di dalam induk.
func anak(induk *multipart.Writer, subtipe string, isi func(*multipart.Writer) error) error {
	batas := multipart.NewWriter(nil).Boundary() // hanya untuk membuat batas acak
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", "multipart/"+subtipe+"; boundary="+batas)
	p, err := induk.CreatePart(h)
	if err != nil {
		return err
	}
	w := multipart.NewWriter(p)
	if err := w.SetBoundary(batas); err != nil {
		return err
	}
	if err := isi(w); err != nil {
		return err
	}
	return w.Close()
}

// tulisTeks menulis satu bagian teks (text/plain atau text/html) berenkode quoted-printable.
func tulisTeks(mw *multipart.Writer, tipe, isi string) error {
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", tipe+"; charset=UTF-8")
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	p, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	return tulisQP(p, isi)
}

// tulisBerkas menulis lampiran (atau gambar inline) berenkode base64 76 kolom per baris.
func tulisBerkas(mw *multipart.Writer, f domain.Berkas, inline bool) error {
	tipe := mime.FormatMediaType(f.Tipe, map[string]string{"name": f.Nama})
	dispo := "attachment"
	if inline {
		dispo = "inline"
	}
	disposisi := mime.FormatMediaType(dispo, map[string]string{"filename": f.Nama})
	if tipe == "" || disposisi == "" {
		return fmt.Errorf("notif: berkas %q tidak valid", f.Nama)
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Type", tipe)
	h.Set("Content-Disposition", disposisi)
	h.Set("Content-Transfer-Encoding", "base64")
	if inline && f.CID != "" {
		h.Set("Content-ID", "<"+f.CID+">")
	}
	p, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	enc := base64.StdEncoding.EncodeToString(f.Isi)
	for len(enc) > 76 {
		if _, err := io.WriteString(p, enc[:76]+"\r\n"); err != nil {
			return err
		}
		enc = enc[76:]
	}
	_, err = io.WriteString(p, enc+"\r\n")
	return err
}
