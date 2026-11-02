package httpapi

import (
	"errors"
	"io"
	"net/http"

	"spkzakat/internal/domain"
	"spkzakat/internal/service"
)

// CMS konten kegiatan keagamaan. Pengurus (Admin dan Ketua) mengelola; publik
// hanya membaca konten yang terbit. Audit mencatat jenis konten dan ID, tidak
// pernah isi tulisannya.

// ---------- publik (tanpa sesi) ----------

func (a *API) daftarKegiatanPublik(w http.ResponseWriter, r *http.Request) {
	d, err := a.svc.DaftarKegiatanPublik(r.Context(), r.URL.Query().Get("kategori"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, d)
}

func (a *API) kegiatanPublik(w http.ResponseWriter, r *http.Request) {
	k, err := a.svc.AmbilKegiatanPublik(r.Context(), r.PathValue("slug"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, k)
}

func (a *API) gambarKegiatanPublik(w http.ResponseWriter, r *http.Request) {
	isi, tipe, err := a.svc.AmbilGambarKegiatanPublik(r.Context(), r.PathValue("slug"))
	a.tulisGambar(w, r, isi, tipe, err)
}

func (a *API) jadwalPublik(w http.ResponseWriter, r *http.Request) {
	js, err := a.svc.DaftarJadwalPublik(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, js)
}

func (a *API) pengumumanPublik(w http.ResponseWriter, r *http.Request) {
	ps, err := a.svc.DaftarPengumumanPublik(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, ps)
}

// ---------- pengurus: kegiatan ----------

type masukanKegiatan struct {
	Judul        string                `json:"judul"`
	Kategori     string                `json:"kategori"`
	Ringkasan    string                `json:"ringkasan"`
	Isi          string                `json:"isi"`
	TanggalAcara string                `json:"tanggal_acara"`
	JamAcara     string                `json:"jam_acara"`
	Lokasi       string                `json:"lokasi"`
	Status       domain.StatusKegiatan `json:"status"`
}

func (m masukanKegiatan) input() service.KegiatanInput {
	return service.KegiatanInput{
		Judul: m.Judul, Kategori: m.Kategori, Ringkasan: m.Ringkasan, Isi: m.Isi, TanggalAcara: m.TanggalAcara,
		JamAcara: m.JamAcara, Lokasi: m.Lokasi, Status: m.Status,
	}
}

func (a *API) daftarKegiatan(w http.ResponseWriter, r *http.Request) {
	d, err := a.svc.DaftarKegiatan(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, d)
}

func (a *API) kategoriKegiatan(w http.ResponseWriter, _ *http.Request) {
	a.tulisJSON(w, http.StatusOK, service.DaftarKategoriKegiatan())
}

func (a *API) buatKegiatan(w http.ResponseWriter, r *http.Request) {
	var in masukanKegiatan
	if !a.bacaJSON(w, r, &in) {
		return
	}
	k, err := a.svc.BuatKegiatan(r.Context(), userDari(r.Context()).ID, in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenBuat, "kegiatan", k.ID, map[string]any{"kategori": k.Kategori, "status": k.Status})
	a.tulisJSON(w, http.StatusCreated, k)
}

func (a *API) ambilKegiatan(w http.ResponseWriter, r *http.Request) {
	k, err := a.svc.AmbilKegiatan(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, k)
}

func (a *API) ubahKegiatan(w http.ResponseWriter, r *http.Request) {
	var in masukanKegiatan
	if !a.bacaJSON(w, r, &in) {
		return
	}
	k, err := a.svc.UbahKegiatan(r.Context(), r.PathValue("id"), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenUbah, "kegiatan", k.ID, map[string]any{"kategori": k.Kategori, "status": k.Status})
	a.tulisJSON(w, http.StatusOK, k)
}

func (a *API) hapusKegiatan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.svc.HapusKegiatan(r.Context(), id); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenHapus, "kegiatan", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// unggahGambarKegiatan menerima isi berkas mentah (bukan multipart), sama seperti
// gambar program donasi: SvelteKit membongkar formulirnya, Go menentukan jenis
// berkas dari isinya.
func (a *API) unggahGambarKegiatan(w http.ResponseWriter, r *http.Request) {
	isi, err := io.ReadAll(http.MaxBytesReader(w, r.Body, service.GambarMaks+1))
	var terlalu *http.MaxBytesError
	switch {
	case errors.As(err, &terlalu):
		isi = nil // service membalas pesan ukuran yang sama dengan berkas kosong
	case err != nil:
		a.tulisError(w, r, http.StatusBadRequest, "Format permintaan tidak valid", nil)
		return
	}
	id := r.PathValue("id")
	if err := a.svc.SimpanGambarKegiatan(r.Context(), id, isi); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenUbah, "kegiatan", id, map[string]any{"gambar_bytes": len(isi)})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) gambarKegiatan(w http.ResponseWriter, r *http.Request) {
	isi, tipe, err := a.svc.AmbilGambarKegiatan(r.Context(), r.PathValue("id"))
	a.tulisGambar(w, r, isi, tipe, err)
}

// ---------- pengurus: jadwal rutin ----------

type masukanJadwal struct {
	Judul      string `json:"judul"`
	Pemateri   string `json:"pemateri"`
	Hari       int    `json:"hari"`
	Jam        string `json:"jam"`
	Lokasi     string `json:"lokasi"`
	Keterangan string `json:"keterangan"`
	Aktif      bool   `json:"aktif"`
}

func (m masukanJadwal) input() service.JadwalInput {
	return service.JadwalInput{
		Judul: m.Judul, Pemateri: m.Pemateri, Hari: m.Hari, Jam: m.Jam, Lokasi: m.Lokasi, Keterangan: m.Keterangan, Aktif: m.Aktif,
	}
}

func (a *API) daftarJadwal(w http.ResponseWriter, r *http.Request) {
	js, err := a.svc.DaftarJadwal(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, js)
}

func (a *API) buatJadwal(w http.ResponseWriter, r *http.Request) {
	var in masukanJadwal
	if !a.bacaJSON(w, r, &in) {
		return
	}
	j, err := a.svc.BuatJadwal(r.Context(), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenBuat, "jadwal_rutin", j.ID, nil)
	a.tulisJSON(w, http.StatusCreated, j)
}

func (a *API) ubahJadwal(w http.ResponseWriter, r *http.Request) {
	var in masukanJadwal
	if !a.bacaJSON(w, r, &in) {
		return
	}
	j, err := a.svc.UbahJadwal(r.Context(), r.PathValue("id"), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenUbah, "jadwal_rutin", j.ID, map[string]any{"aktif": j.Aktif})
	a.tulisJSON(w, http.StatusOK, j)
}

func (a *API) hapusJadwal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.svc.HapusJadwal(r.Context(), id); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenHapus, "jadwal_rutin", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// ---------- pengurus: pengumuman ----------

type masukanPengumuman struct {
	Judul        string `json:"judul"`
	Isi          string `json:"isi"`
	TampilSampai string `json:"tampil_sampai"`
	Terbit       bool   `json:"terbit"`
}

func (m masukanPengumuman) input() service.PengumumanInput {
	return service.PengumumanInput{Judul: m.Judul, Isi: m.Isi, TampilSampai: m.TampilSampai, Terbit: m.Terbit}
}

func (a *API) daftarPengumuman(w http.ResponseWriter, r *http.Request) {
	ps, err := a.svc.DaftarPengumuman(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, ps)
}

func (a *API) buatPengumuman(w http.ResponseWriter, r *http.Request) {
	var in masukanPengumuman
	if !a.bacaJSON(w, r, &in) {
		return
	}
	p, err := a.svc.BuatPengumuman(r.Context(), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenBuat, "pengumuman", p.ID, nil)
	a.tulisJSON(w, http.StatusCreated, p)
}

func (a *API) ubahPengumuman(w http.ResponseWriter, r *http.Request) {
	var in masukanPengumuman
	if !a.bacaJSON(w, r, &in) {
		return
	}
	p, err := a.svc.UbahPengumuman(r.Context(), r.PathValue("id"), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenUbah, "pengumuman", p.ID, map[string]any{"terbit": p.Terbit})
	a.tulisJSON(w, http.StatusOK, p)
}

func (a *API) hapusPengumuman(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.svc.HapusPengumuman(r.Context(), id); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKontenHapus, "pengumuman", id, nil)
	w.WriteHeader(http.StatusNoContent)
}
