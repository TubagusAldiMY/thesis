package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// aadNoHPDonasi memisahkan konteks enkripsi No. HP donatur dari No. HP mustahik.
const aadNoHPDonasi = "donasi.no_hp"

const (
	// Batas jumlah donasi web. Bawah: mencegah transaksi iseng yang hanya
	// memenuhi tabel. Atas: batas wajar satu transaksi lewat gateway.
	donasiMin  int64 = 10_000
	donasiMaks int64 = 100_000_000

	// Tautan bayar berlaku 24 jam; lewat itu gateway mengirim status expire.
	kedaluwarsaMenit = 24 * 60
)

// wib: nomor kuitansi dan tanggalnya mengikuti kalender Indonesia, bukan UTC
// (donasi pukul 23.30 WIB tanggal 31 Desember tetap bertahun lama). Zona tetap
// dipakai supaya tidak bergantung pada database tzdata di image Docker.
var wib = time.FixedZone("WIB", 7*3600)

var reKodePublik = regexp.MustCompile(`^[0-9a-f]{32}$`)

var namaJenisDonasi = map[domain.JenisDonasi]string{
	domain.DonasiZakatFitrah:  "Zakat Fitrah",
	domain.DonasiZakatMal:     "Zakat Mal",
	domain.DonasiZakatProfesi: "Zakat Profesi",
	domain.DonasiInfak:        "Infak",
}

func jenisDonasiSah(j domain.JenisDonasi) bool { _, ok := namaJenisDonasi[j]; return ok }

// AturGerbang memasang payment gateway untuk donasi web. Tanpa gateway,
// BuatDonasiWeb mengembalikan domain.ErrNonaktif (halaman publik tidak
// menyala), sedangkan pencatatan oleh petugas tetap bekerja. urlWeb = alamat
// publik frontend (tanpa garis miring di akhir) untuk tujuan pengalihan.
func (s *Service) AturGerbang(g domain.GerbangBayar, urlWeb string) {
	s.gerbang = g
	s.urlWeb = strings.TrimRight(urlWeb, "/")
}

// DonasiWebAktif = apakah halaman donasi publik menyala (gateway terpasang).
func (s *Service) DonasiWebAktif() bool { return s.gerbang != nil }

// DonasiWebInput = isian formulir publik. NoHP berupa plaintext hanya di sini;
// begitu masuk service langsung dienkripsi.
type DonasiWebInput struct {
	Nama       string
	NoHP       string
	Jenis      domain.JenisDonasi
	Jumlah     int64
	Keterangan string
}

// DonasiManualInput = isian petugas saat mencatat penerimaan langsung di masjid.
type DonasiManualInput struct {
	Nama       string
	NoHP       string
	Jenis      domain.JenisDonasi
	Jumlah     int64
	Metode     string // "tunai" | "transfer"
	Keterangan string
	Tanggal    string // yyyy-mm-dd, kosong = hari ini
}

// DonasiView = bentuk untuk pengurus. No. HP lengkap hanya diisi bila
// pemanggil berhak (Admin, pada detail); selain itu tersamar atau kosong.
type DonasiView struct {
	ID            string              `json:"id"`
	KodePublik    string              `json:"kode_publik"`
	Nama          string              `json:"nama"`
	NoHP          string              `json:"no_hp,omitempty"`
	NoHPTersamar  string              `json:"no_hp_tersamar,omitempty"`
	Jenis         domain.JenisDonasi  `json:"jenis"`
	Jumlah        int64               `json:"jumlah"`
	Sumber        domain.SumberDonasi `json:"sumber"`
	Metode        string              `json:"metode"`
	Status        domain.StatusDonasi `json:"status"`
	NomorKuitansi string              `json:"nomor_kuitansi"`
	Keterangan    string              `json:"keterangan"`
	AlasanBatal   string              `json:"alasan_batal"`
	DibayarPada   *time.Time          `json:"dibayar_pada"`
	DibuatPada    time.Time           `json:"dibuat_pada"`
	Baru          bool                `json:"baru"` // donasi web diterima yang belum dilihat pengurus
}

// DonasiPublik = yang boleh dilihat muzaki lewat kode rahasianya: status
// pembayaran dan (setelah diterima) data kuitansi miliknya sendiri.
type DonasiPublik struct {
	ID            string              `json:"-"` // internal (audit, order_id gateway); tidak pernah ikut ke JSON publik
	Kode          string              `json:"kode"`
	Status        domain.StatusDonasi `json:"status"`
	Nama          string              `json:"nama"`
	Jenis         domain.JenisDonasi  `json:"jenis"`
	Jumlah        int64               `json:"jumlah"`
	Metode        string              `json:"metode"`
	NomorKuitansi string              `json:"nomor_kuitansi,omitempty"`
	DibayarPada   *time.Time          `json:"dibayar_pada,omitempty"`
	BayarURL      string              `json:"bayar_url,omitempty"` // hanya selama menunggu
}

// DonasiQuery = penyaringan daftar donasi.
type DonasiQuery struct {
	Status  domain.StatusDonasi
	Jenis   domain.JenisDonasi
	Q       string
	Page    int
	PerPage int
}

// DonasiRingkasView = penanda notifikasi dan total penerimaan.
type DonasiRingkasView struct {
	Baru           int   `json:"baru"`
	JumlahDiterima int   `json:"jumlah_diterima"`
	TotalDiterima  int64 `json:"total_diterima"`
}

// DonasiHalaman = satu halaman daftar + total baris + ringkasan.
type DonasiHalaman struct {
	Items   []DonasiView      `json:"items"`
	Total   int               `json:"total"`
	Ringkas DonasiRingkasView `json:"ringkas"`
}

// HasilNotifikasi = akibat satu notifikasi gateway, untuk audit oleh lapisan HTTP.
type HasilNotifikasi struct {
	DonasiID string
	Status   domain.StatusDonasi
	Berubah  bool // false = notifikasi ulang/terlambat yang diabaikan (idempoten)
	Jumlah   int64
}

// kodeAcak membuat 128 bit acak heksadesimal: kunci tautan muzaki. Tidak bisa
// ditebak, jadi siapa pun yang tidak diberi tautannya tidak bisa membukanya.
func kodeAcak() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("service: crypto/rand gagal: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func formatNomorKuitansi(tahun, urut int) string { return fmt.Sprintf("KWT/%d/%04d", tahun, urut) }

// validasiDonatur memeriksa isian yang sama untuk donasi web dan manual.
func validasiDonatur(g galat, nama, hp string, jenis domain.JenisDonasi, jumlah int64, ket string) {
	if n := panjangRune(strings.TrimSpace(nama)); n < 2 || n > 100 {
		g.tambah("nama", "Nama harus 2–100 karakter")
	}
	if hp != "" && !reHP.MatchString(hp) {
		g.tambah("no_hp", "No. HP tidak valid (8–15 digit, boleh diawali +)")
	}
	if !jenisDonasiSah(jenis) {
		g.tambah("jenis", "Jenis donasi tidak dikenal")
	}
	if jumlah < donasiMin || jumlah > donasiMaks {
		g.tambah("jumlah", fmt.Sprintf("Jumlah harus antara Rp %s dan Rp %s", titik(donasiMin), titik(donasiMaks)))
	}
	if panjangRune(strings.TrimSpace(ket)) > 200 {
		g.tambah("keterangan", "Keterangan maksimal 200 karakter")
	}
}

// titik memformat angka dengan pemisah ribuan titik (pesan galat berbahasa Indonesia).
func titik(n int64) string {
	s := fmt.Sprint(n)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	return string(out)
}

// enkripsiHP mengenkripsi No. HP yang sudah dinormalkan; kosong → nil.
func (s *Service) enkripsiHP(hp string) ([]byte, error) {
	if hp == "" {
		return nil, nil
	}
	b, err := s.enc.Encrypt([]byte(hp), []byte(aadNoHPDonasi))
	if err != nil {
		return nil, fmt.Errorf("service: enkripsi No. HP donasi: %w", err)
	}
	return b, nil
}

type modeHP int

const (
	hpTanpa modeHP = iota
	hpTersamar
	hpPenuh
)

func (s *Service) viewDonasi(d domain.Donasi, mode modeHP) (DonasiView, error) {
	v := DonasiView{
		ID: d.ID, KodePublik: d.KodePublik, Nama: d.Nama, Jenis: d.Jenis, Jumlah: d.Jumlah,
		Sumber: d.Sumber, Metode: d.Metode, Status: d.Status, NomorKuitansi: d.NomorKuitansi,
		Keterangan: d.Keterangan, AlasanBatal: d.AlasanBatal, DibayarPada: d.DibayarPada,
		DibuatPada: d.CreatedAt,
		Baru:       d.Sumber == domain.SumberWeb && d.Status == domain.DonasiDiterima && d.DilihatPada == nil,
	}
	if mode != hpTanpa && len(d.NoHPEnc) > 0 {
		hp, err := s.enc.Decrypt(d.NoHPEnc, []byte(aadNoHPDonasi))
		if err != nil {
			return DonasiView{}, fmt.Errorf("service: No. HP donasi %s tidak terbaca (kunci enkripsi salah atau data rusak): %w", d.ID, err)
		}
		if mode == hpPenuh {
			v.NoHP = string(hp)
		} else {
			v.NoHPTersamar = samarkan(string(hp), 1)
		}
	}
	return v, nil
}

func publik(d domain.Donasi) DonasiPublik {
	p := DonasiPublik{
		ID: d.ID, Kode: d.KodePublik, Status: d.Status, Nama: d.Nama, Jenis: d.Jenis, Jumlah: d.Jumlah,
		Metode: d.Metode, NomorKuitansi: d.NomorKuitansi, DibayarPada: d.DibayarPada,
	}
	if d.Status == domain.DonasiMenunggu {
		p.BayarURL = d.BayarURL
	}
	return p
}

// BuatDonasiWeb menerima isian muzaki dari halaman publik: menyimpan donasi
// berstatus menunggu, membuat transaksi di gateway, lalu mengembalikan tautan
// halaman bayar. Donasi baru dianggap diterima saat gateway mengirim notifikasi
// pembayaran berhasil (TerimaNotifikasi), bukan saat form dikirim.
func (s *Service) BuatDonasiWeb(ctx context.Context, in DonasiWebInput) (DonasiPublik, error) {
	if s.gerbang == nil {
		return DonasiPublik{}, domain.ErrNonaktif
	}
	hp := normalHP(in.NoHP)
	g := galat{}
	validasiDonatur(g, in.Nama, hp, in.Jenis, in.Jumlah, in.Keterangan)
	if err := g.err(); err != nil {
		return DonasiPublik{}, err
	}
	hpEnc, err := s.enkripsiHP(hp)
	if err != nil {
		return DonasiPublik{}, err
	}
	kode, err := kodeAcak()
	if err != nil {
		return DonasiPublik{}, err
	}

	d := domain.Donasi{
		KodePublik: kode, Nama: strings.TrimSpace(in.Nama), NoHPEnc: hpEnc, Jenis: in.Jenis,
		Jumlah: in.Jumlah, Sumber: domain.SumberWeb, Status: domain.DonasiMenunggu,
		Keterangan: strings.TrimSpace(in.Keterangan),
	}
	if err := s.st.CreateDonasi(ctx, &d); err != nil {
		return DonasiPublik{}, fmt.Errorf("service.BuatDonasiWeb: %w", err)
	}

	hasil, err := s.gerbang.Buat(ctx, domain.PermintaanBayar{
		OrderID: d.ID, Jumlah: d.Jumlah, NamaDonatur: d.Nama, NamaItem: namaJenisDonasi[d.Jenis],
		SelesaiURL: s.urlWeb + "/donasi/selesai?kode=" + kode, KedaluwarsaMenit: kedaluwarsaMenit,
	})
	if err != nil {
		// Transaksi gagal dibuat: tandai gagal agar tidak menggantung "menunggu" selamanya.
		d.Status = domain.DonasiGagal
		if uerr := s.st.UpdateDonasi(ctx, d); uerr != nil {
			slog.ErrorContext(ctx, "gagal menandai donasi gagal", "donasi_id", d.ID, "error", uerr.Error())
		}
		return DonasiPublik{}, fmt.Errorf("service.BuatDonasiWeb: %w", errors.Join(domain.ErrGerbang, err))
	}
	d.BayarURL = hasil.RedirectURL
	if err := s.st.UpdateDonasi(ctx, d); err != nil {
		return DonasiPublik{}, fmt.Errorf("service.BuatDonasiWeb: simpan tautan bayar: %w", err)
	}
	return publik(d), nil
}

// TerimaNotifikasi memproses notifikasi gateway: verifikasi tanda tangan,
// cocokkan jumlah, lalu ubah status dalam satu transaksi dengan baris donasi
// terkunci. Idempoten: gateway boleh mengirim notifikasi yang sama berkali-kali,
// dan notifikasi terlambat (mis. expire setelah settlement) tidak menimpa
// status akhir. Kuitansi hanya terbit dari status menunggu → diterima, dan
// hanya perubahan itu yang memicu email ke Ketua (pemberitahuan.go).
func (s *Service) TerimaNotifikasi(ctx context.Context, body []byte) (HasilNotifikasi, error) {
	if s.gerbang == nil {
		return HasilNotifikasi{}, domain.ErrNonaktif
	}
	n, err := s.gerbang.Verifikasi(body)
	if err != nil {
		return HasilNotifikasi{}, err
	}

	var hasil HasilNotifikasi
	var baruDiterima *domain.Donasi // terisi hanya bila notifikasi ini yang mengubahnya jadi diterima
	err = s.st.WithTx(ctx, func(tx *store.Store) error {
		d, err := tx.GetDonasiForUpdate(ctx, n.OrderID)
		if err != nil {
			return err
		}
		if d.Sumber != domain.SumberWeb {
			return fmt.Errorf("service.TerimaNotifikasi: donasi %s bukan donasi web: %w", d.ID, domain.ErrNotFound)
		}
		hasil = HasilNotifikasi{DonasiID: d.ID, Status: d.Status, Jumlah: d.Jumlah}
		if n.Jumlah != d.Jumlah {
			slog.WarnContext(ctx, "jumlah notifikasi gateway tidak cocok", "donasi_id", d.ID, "jumlah_donasi", d.Jumlah, "jumlah_notifikasi", n.Jumlah)
			return domain.Konflik("Jumlah pada notifikasi tidak sama dengan donasi")
		}
		if d.Status != domain.DonasiMenunggu {
			return nil // sudah final: abaikan notifikasi ulang/terlambat
		}

		switch n.Status {
		case domain.DonasiDiterima:
			sekarang := s.sekarang()
			tahun := sekarang.In(wib).Year()
			urut, err := tx.NextNomorKuitansi(ctx, tahun)
			if err != nil {
				return err
			}
			d.Status = domain.DonasiDiterima
			d.NomorKuitansi = formatNomorKuitansi(tahun, urut)
			d.DibayarPada = &sekarang
			d.Metode = n.Metode
			d.BayarURL = ""
		case domain.DonasiGagal, domain.DonasiKedaluwarsa:
			d.Status = n.Status
			d.BayarURL = ""
		default: // menunggu: hanya simpan metode yang dipilih muzaki, status tetap
			if n.Metode == "" || n.Metode == d.Metode {
				return nil
			}
			d.Metode = n.Metode
			return tx.UpdateDonasi(ctx, d)
		}
		if err := tx.UpdateDonasi(ctx, d); err != nil {
			return err
		}
		hasil.Status, hasil.Berubah = d.Status, true
		if d.Status == domain.DonasiDiterima {
			salinan := d
			baruDiterima = &salinan
		}
		return nil
	})
	if err != nil {
		return HasilNotifikasi{}, fmt.Errorf("service.TerimaNotifikasi: %w", err)
	}
	// Setelah commit, bukan di dalam transaksi: email tidak boleh terkirim untuk
	// donasi yang akhirnya batal tersimpan, dan SMTP tidak boleh menahan kunci baris.
	// Notifikasi ulang dari gateway tidak sampai sini (Berubah=false) → tidak ada email ganda.
	if baruDiterima != nil {
		s.beritahuDonasi(ctx, *baruDiterima)
	}
	return hasil, nil
}

// CatatDonasiManual dipakai petugas untuk penerimaan langsung di masjid
// (tunai atau transfer). Langsung berstatus diterima dan mendapat kuitansi.
// Tidak memicu penanda "baru" maupun email ke Ketua: yang mencatat sudah
// pengurus itu sendiri, dan tanggal bisa mundur (mengisi dari buku kas lama).
func (s *Service) CatatDonasiManual(ctx context.Context, petugasID string, in DonasiManualInput) (DonasiView, error) {
	hp := normalHP(in.NoHP)
	g := galat{}
	validasiDonatur(g, in.Nama, hp, in.Jenis, in.Jumlah, in.Keterangan)
	if in.Metode != "tunai" && in.Metode != "transfer" {
		g.tambah("metode", "Pilih tunai atau transfer")
	}
	sekarang := s.sekarang()
	tanggal := sekarang
	if t := strings.TrimSpace(in.Tanggal); t != "" {
		w, err := time.ParseInLocation("2006-01-02", t, wib)
		if err != nil {
			g.tambah("tanggal", "Tanggal tidak valid")
		} else if w.After(sekarang) {
			g.tambah("tanggal", "Tanggal tidak boleh di masa depan")
		} else {
			tanggal = w.Add(12 * time.Hour) // tengah hari WIB: tanggal tetap benar di zona mana pun
		}
	}
	if err := g.err(); err != nil {
		return DonasiView{}, err
	}
	hpEnc, err := s.enkripsiHP(hp)
	if err != nil {
		return DonasiView{}, err
	}
	kode, err := kodeAcak()
	if err != nil {
		return DonasiView{}, err
	}

	var d domain.Donasi
	err = s.st.WithTx(ctx, func(tx *store.Store) error {
		tahun := sekarang.In(wib).Year()
		urut, err := tx.NextNomorKuitansi(ctx, tahun)
		if err != nil {
			return err
		}
		d = domain.Donasi{
			KodePublik: kode, Nama: strings.TrimSpace(in.Nama), NoHPEnc: hpEnc, Jenis: in.Jenis,
			Jumlah: in.Jumlah, Sumber: domain.SumberPetugas, Metode: in.Metode, Status: domain.DonasiDiterima,
			NomorKuitansi: formatNomorKuitansi(tahun, urut),
			Keterangan:    strings.TrimSpace(in.Keterangan), DibayarPada: &tanggal, DilihatPada: &sekarang,
			DicatatOleh: petugasID,
		}
		return tx.CreateDonasi(ctx, &d)
	})
	if err != nil {
		return DonasiView{}, fmt.Errorf("service.CatatDonasiManual: %w", err)
	}
	return s.viewDonasi(d, hpTanpa)
}

// DaftarDonasi mengembalikan satu halaman daftar (terbaru dulu) beserta ringkasan.
func (s *Service) DaftarDonasi(ctx context.Context, q DonasiQuery) (DonasiHalaman, error) {
	g := galat{}
	switch q.Status {
	case "", domain.DonasiMenunggu, domain.DonasiDiterima, domain.DonasiGagal, domain.DonasiKedaluwarsa, domain.DonasiDibatalkan:
	default:
		g.tambah("status", "Status tidak dikenal")
	}
	if q.Jenis != "" && !jenisDonasiSah(q.Jenis) {
		g.tambah("jenis", "Jenis donasi tidak dikenal")
	}
	if panjangRune(q.Q) > 100 {
		g.tambah("q", "Kata kunci maksimal 100 karakter")
	}
	if err := g.err(); err != nil {
		return DonasiHalaman{}, err
	}
	l, err := s.st.ListDonasi(ctx, store.DonasiFilter{
		Status: q.Status, Jenis: q.Jenis, Q: strings.TrimSpace(q.Q), Page: q.Page, PerPage: q.PerPage,
	})
	if err != nil {
		return DonasiHalaman{}, fmt.Errorf("service.DaftarDonasi: %w", err)
	}
	r, err := s.st.RingkasDonasi(ctx)
	if err != nil {
		return DonasiHalaman{}, fmt.Errorf("service.DaftarDonasi: %w", err)
	}
	h := DonasiHalaman{Items: make([]DonasiView, 0, len(l.Items)), Total: l.Total, Ringkas: DonasiRingkasView(r)}
	for _, d := range l.Items {
		v, err := s.viewDonasi(d, hpTanpa)
		if err != nil {
			return DonasiHalaman{}, err
		}
		h.Items = append(h.Items, v)
	}
	return h, nil
}

// AmbilDonasi mengambil satu donasi. lihatHP=true (Admin) menyertakan No. HP
// lengkap untuk tombol WhatsApp; selain itu hanya tersamar.
func (s *Service) AmbilDonasi(ctx context.Context, id string, lihatHP bool) (DonasiView, error) {
	d, err := s.st.GetDonasi(ctx, id)
	if err != nil {
		return DonasiView{}, fmt.Errorf("service.AmbilDonasi: %w", err)
	}
	mode := hpTersamar
	if lihatHP {
		mode = hpPenuh
	}
	return s.viewDonasi(d, mode)
}

// BatalkanDonasi membatalkan kuitansi yang sudah terbit (mis. salah catat).
// Nomornya tetap tercatat dan tampil "dibatalkan"; kuitansi tidak pernah dihapus.
func (s *Service) BatalkanDonasi(ctx context.Context, id, alasan string) (DonasiView, error) {
	alasan = strings.TrimSpace(alasan)
	if n := panjangRune(alasan); n < 3 || n > 200 {
		return DonasiView{}, (galat{"alasan": "Alasan pembatalan 3–200 karakter"}).err()
	}
	var d domain.Donasi
	err := s.st.WithTx(ctx, func(tx *store.Store) error {
		var err error
		if d, err = tx.GetDonasiForUpdate(ctx, id); err != nil {
			return err
		}
		if d.Status != domain.DonasiDiterima {
			return domain.Konflik("Hanya donasi yang sudah diterima yang bisa dibatalkan")
		}
		d.Status, d.AlasanBatal = domain.DonasiDibatalkan, alasan
		return tx.UpdateDonasi(ctx, d)
	})
	if err != nil {
		return DonasiView{}, fmt.Errorf("service.BatalkanDonasi: %w", err)
	}
	return s.viewDonasi(d, hpTanpa)
}

// RingkasDonasi = jumlah donasi web baru (penanda menu) dan total penerimaan.
func (s *Service) RingkasDonasi(ctx context.Context) (DonasiRingkasView, error) {
	r, err := s.st.RingkasDonasi(ctx)
	if err != nil {
		return DonasiRingkasView{}, fmt.Errorf("service.RingkasDonasi: %w", err)
	}
	return DonasiRingkasView(r), nil
}

// TandaiDonasiDilihat menghapus penanda "baru" untuk semua donasi web yang sudah diterima.
func (s *Service) TandaiDonasiDilihat(ctx context.Context) (int64, error) {
	n, err := s.st.TandaiDonasiDilihat(ctx)
	if err != nil {
		return 0, fmt.Errorf("service.TandaiDonasiDilihat: %w", err)
	}
	return n, nil
}

// AmbilDonasiPublik dipakai muzaki (tanpa login) dengan kode rahasia tautannya.
// Format kode salah diperlakukan sama dengan kode yang tidak ada (404), supaya
// tidak ada petunjuk bagi penebak.
func (s *Service) AmbilDonasiPublik(ctx context.Context, kode string) (DonasiPublik, error) {
	if !reKodePublik.MatchString(kode) {
		return DonasiPublik{}, domain.ErrNotFound
	}
	d, err := s.st.GetDonasiByKode(ctx, kode)
	if err != nil {
		return DonasiPublik{}, fmt.Errorf("service.AmbilDonasiPublik: %w", err)
	}
	return publik(d), nil
}
