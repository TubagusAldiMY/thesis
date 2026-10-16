package httpapi

import (
	"errors"
	"io"
	"net/http"

	"spkzakat/internal/domain"
	"spkzakat/internal/service"
)

// maksNotifikasi = batas ukuran body notifikasi gateway (aslinya < 2 KiB).
const maksNotifikasi = 64 << 10

// ---------- publik (tanpa sesi) ----------

// donasiAktif memberi tahu halaman publik apakah donasi web menyala, supaya
// muzaki tidak mengisi formulir lalu baru mendapat galat.
func (a *API) donasiAktif(w http.ResponseWriter, _ *http.Request) {
	a.tulisJSON(w, http.StatusOK, map[string]bool{"aktif": a.svc.DonasiWebAktif()})
}

// ringkasanPublik membalas angka transparansi untuk landing (tanpa login):
// total terkumpul, tersalur, jumlah donasi, dan penerima manfaat. Hanya agregat.
func (a *API) ringkasanPublik(w http.ResponseWriter, r *http.Request) {
	s, err := a.svc.AmbilRingkasanPublik(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, s)
}

// buatDonasiWeb menerima isian muzaki dari halaman publik, membuat donasi
// berstatus menunggu + transaksi di gateway, dan membalas tautan halaman bayar.
func (a *API) buatDonasiWeb(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nama       string             `json:"nama"`
		NoHP       string             `json:"no_hp"`
		Jenis      domain.JenisDonasi `json:"jenis"`
		Jumlah     int64              `json:"jumlah"`
		Keterangan string             `json:"keterangan"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	p, err := a.svc.BuatDonasiWeb(r.Context(), service.DonasiWebInput{
		Nama: in.Nama, NoHP: in.NoHP, Jenis: in.Jenis, Jumlah: in.Jumlah, Keterangan: in.Keterangan,
	})
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusCreated, p)
}

// donasiPublik = status pembayaran dan data kuitansi untuk pemegang kode rahasia.
func (a *API) donasiPublik(w http.ResponseWriter, r *http.Request) {
	p, err := a.svc.AmbilDonasiPublik(r.Context(), r.PathValue("kode"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, p)
}

// notifikasiGerbang menerima notifikasi pembayaran dari payment gateway. Tanpa
// sesi: keasliannya dijamin tanda tangan SHA512 yang diperiksa service. Balasan
// 2xx berarti "sudah kami terima"; selain itu gateway akan mengirim ulang.
func (a *API) notifikasiGerbang(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maksNotifikasi))
	if err != nil {
		a.tulisError(w, r, http.StatusBadRequest, "Format permintaan tidak valid", nil)
		return
	}
	h, err := a.svc.TerimaNotifikasi(r.Context(), body)
	switch {
	case errors.Is(err, domain.ErrTandaTanganTidakSah):
		a.logger(r).Warn("notifikasi gateway ditolak: tanda tangan tidak sah")
		a.tulisError(w, r, http.StatusUnauthorized, "Notifikasi tidak sah", nil)
		return
	case err != nil:
		a.gagal(w, r, err)
		return
	}
	a.catatTerima(r, h)
	a.tulisJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// catatTerima menulis audit hanya untuk perubahan menjadi diterima (kejadian
// yang berarti bagi pengurus); status gagal/kedaluwarsa cukup di tabel donasi.
func (a *API) catatTerima(r *http.Request, h service.HasilNotifikasi) {
	if h.Berubah && h.Status == domain.DonasiDiterima {
		a.catat(r, domain.AksiDonasiTerima, "donasi", h.DonasiID, map[string]any{"jumlah": h.Jumlah})
	}
}

// simulasikanBayar hanya terpasang bila penyedia tiruan menyala (dev/demo):
// menandai donasi lunas lewat jalur notifikasi yang sama dengan gateway asli.
func (a *API) simulasikanBayar(w http.ResponseWriter, r *http.Request) {
	p, err := a.svc.AmbilDonasiPublik(r.Context(), r.PathValue("kode"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	if p.Status != domain.DonasiMenunggu {
		a.tulisJSON(w, http.StatusOK, p)
		return
	}
	h, err := a.svc.TerimaNotifikasi(r.Context(), a.simulasiBayar.Muatan(p.ID, p.Jumlah, "settlement"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catatTerima(r, h)
	p, err = a.svc.AmbilDonasiPublik(r.Context(), r.PathValue("kode"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, p)
}

// ---------- pengurus (butuh sesi) ----------

func (a *API) daftarDonasi(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	h, err := a.svc.DaftarDonasi(r.Context(), service.DonasiQuery{
		Status: domain.StatusDonasi(q.Get("status")), Jenis: domain.JenisDonasi(q.Get("jenis")), Q: q.Get("q"),
		Page: angkaQuery(r, "page", 1), PerPage: angkaQuery(r, "per_page", 20),
	})
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, h)
}

// catatDonasi: petugas mencatat penerimaan langsung di masjid (tunai/transfer).
func (a *API) catatDonasi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nama       string             `json:"nama"`
		NoHP       string             `json:"no_hp"`
		Jenis      domain.JenisDonasi `json:"jenis"`
		Jumlah     int64              `json:"jumlah"`
		Metode     string             `json:"metode"`
		Keterangan string             `json:"keterangan"`
		Tanggal    string             `json:"tanggal"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	v, err := a.svc.CatatDonasiManual(r.Context(), userDari(r.Context()).ID, service.DonasiManualInput{
		Nama: in.Nama, NoHP: in.NoHP, Jenis: in.Jenis, Jumlah: in.Jumlah, Metode: in.Metode,
		Keterangan: in.Keterangan, Tanggal: in.Tanggal,
	})
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiDonasiCatat, "donasi", v.ID, map[string]any{
		"jenis": v.Jenis, "jumlah": v.Jumlah, "metode": v.Metode, "nomor_kuitansi": v.NomorKuitansi,
	})
	a.tulisJSON(w, http.StatusCreated, v)
}

func (a *API) ringkasDonasi(w http.ResponseWriter, r *http.Request) {
	h, err := a.svc.RingkasDonasi(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, h)
}

func (a *API) tandaiDonasiDilihat(w http.ResponseWriter, r *http.Request) {
	n, err := a.svc.TandaiDonasiDilihat(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, map[string]int64{"diubah": n})
}

// ambilDonasi: No. HP lengkap hanya untuk Admin (tombol WhatsApp); Ketua hanya tersamar.
func (a *API) ambilDonasi(w http.ResponseWriter, r *http.Request) {
	lihatHP := userDari(r.Context()).Role == domain.RoleAdmin
	v, err := a.svc.AmbilDonasi(r.Context(), r.PathValue("id"), lihatHP)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, v)
}

func (a *API) batalkanDonasi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Alasan string `json:"alasan"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	v, err := a.svc.BatalkanDonasi(r.Context(), r.PathValue("id"), in.Alasan)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	// Alasan sengaja tidak masuk audit: teks bebas bisa memuat data pribadi.
	a.catat(r, domain.AksiDonasiBatal, "donasi", v.ID, map[string]any{"nomor_kuitansi": v.NomorKuitansi})
	a.tulisJSON(w, http.StatusOK, v)
}
