// Package notif mengirim pemberitahuan ke pengurus lewat email (SMTP). Hanya
// memakai stdlib (net/smtp): satu penerima tetap (Ketua), teks polos.
package notif

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
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
func (e *Email) Kirim(ctx context.Context, subjek, isi string) error {
	pesan, err := bangunPesan(e.dari, e.ke, subjek, isi, time.Now())
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

// bangunPesan merakit email teks polos UTF-8. Subjek di-encode RFC 2047 dan
// isi quoted-printable supaya aman untuk huruf non-ASCII dan baris panjang;
// encoder subjek juga meng-encode CR/LF, jadi subjek tak bisa menyisipkan header.
func bangunPesan(dari, ke mail.Address, subjek, isi string, sekarang time.Time) ([]byte, error) {
	var id [12]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, fmt.Errorf("notif: crypto/rand gagal: %w", err)
	}
	// Message-ID wajib unik; domainnya diambil dari alamat pengirim.
	domain := dari.Address[strings.LastIndex(dari.Address, "@")+1:]

	var b bytes.Buffer
	for _, h := range [][2]string{
		{"From", dari.String()},
		{"To", ke.String()},
		{"Subject", mime.QEncoding.Encode("utf-8", subjek)},
		{"Date", sekarang.Format(time.RFC1123Z)},
		{"Message-ID", "<" + hex.EncodeToString(id[:]) + "@" + domain + ">"},
		{"MIME-Version", "1.0"},
		{"Content-Type", "text/plain; charset=UTF-8"},
		{"Content-Transfer-Encoding", "quoted-printable"},
	} {
		b.WriteString(h[0] + ": " + h[1] + "\r\n")
	}
	b.WriteString("\r\n")
	w := quotedprintable.NewWriter(&b)
	if _, err := w.Write([]byte(isi)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
