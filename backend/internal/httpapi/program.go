package httpapi

import (
	"errors"
	"io"
	"net/http"

	"spkzakat/internal/domain"
	"spkzakat/internal/service"
)

// ---------- publik (tanpa sesi) ----------

// daftarProgramPublik = program aktif dan selesai untuk halaman /program.
func (a *API) daftarProgramPublik(w http.ResponseWriter, r *http.Request) {
	ps, err := a.svc.DaftarProgramPublik(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, ps)
}

// programPublik = satu program beserta kabar dan donatur terbaru (nama anonim sudah disamarkan).
func (a *API) programPublik(w http.ResponseWriter, r *http.Request) {
	p, err := a.svc.AmbilProgramPublik(r.Context(), r.PathValue("slug"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, p)
}

func (a *API) gambarProgramPublik(w http.ResponseWriter, r *http.Request) {
	isi, tipe, err := a.svc.AmbilGambarProgramPublik(r.Context(), r.PathValue("slug"))
	a.tulisGambar(w, r, isi, tipe, err)
}

// ---------- pengurus (butuh sesi) ----------

// masukanProgram = body JSON buat/ubah program.
type masukanProgram struct {
	Judul      string               `json:"judul"`
	Ringkasan  string               `json:"ringkasan"`
	Cerita     string               `json:"cerita"`
	Target     int64                `json:"target"`
	BatasWaktu string               `json:"batas_waktu"`
	Status     domain.StatusProgram `json:"status"`
}

func (m masukanProgram) input() service.ProgramInput {
	return service.ProgramInput{
		Judul: m.Judul, Ringkasan: m.Ringkasan, Cerita: m.Cerita, Target: m.Target, BatasWaktu: m.BatasWaktu, Status: m.Status,
	}
}

func (a *API) daftarProgram(w http.ResponseWriter, r *http.Request) {
	ps, err := a.svc.DaftarProgram(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, ps)
}

func (a *API) buatProgram(w http.ResponseWriter, r *http.Request) {
	var in masukanProgram
	if !a.bacaJSON(w, r, &in) {
		return
	}
	p, err := a.svc.BuatProgram(r.Context(), userDari(r.Context()).ID, in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiProgramBuat, "program_donasi", p.ID, map[string]any{"judul": p.Judul, "status": p.Status})
	a.tulisJSON(w, http.StatusCreated, p)
}

func (a *API) ambilProgram(w http.ResponseWriter, r *http.Request) {
	p, err := a.svc.AmbilProgram(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, p)
}

func (a *API) ubahProgram(w http.ResponseWriter, r *http.Request) {
	var in masukanProgram
	if !a.bacaJSON(w, r, &in) {
		return
	}
	p, err := a.svc.UbahProgram(r.Context(), r.PathValue("id"), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiProgramUbah, "program_donasi", p.ID, map[string]any{"status": p.Status, "target": p.Target})
	a.tulisJSON(w, http.StatusOK, p)
}

// unggahGambarProgram menerima isi berkas mentah (bukan multipart): SvelteKit
// sudah membongkar formulir dan meneruskan bytes gambarnya saja. Jenis berkas
// ditentukan service dari isinya, Content-Type kiriman diabaikan.
func (a *API) unggahGambarProgram(w http.ResponseWriter, r *http.Request) {
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
	if err := a.svc.SimpanGambarProgram(r.Context(), id, isi); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiProgramUbah, "program_donasi", id, map[string]any{"gambar_bytes": len(isi)})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) gambarProgram(w http.ResponseWriter, r *http.Request) {
	isi, tipe, err := a.svc.AmbilGambarProgram(r.Context(), r.PathValue("id"))
	a.tulisGambar(w, r, isi, tipe, err)
}

// tulisGambar mengirim gambar sampul. Boleh di-cache (alamatnya membawa
// ?v=<versi> yang berubah setiap program diubah), dan diberi CSP paling ketat
// supaya berkas yang lolos validasi pun tidak bisa menjalankan apa pun.
func (a *API) tulisGambar(w http.ResponseWriter, r *http.Request, isi []byte, tipe string, err error) {
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", tipe)
	h.Set("Cache-Control", "public, max-age=86400") // menimpa no-store bawaan middleware
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	_, _ = w.Write(isi)
}

func (a *API) tambahKabar(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Judul string `json:"judul"`
		Isi   string `json:"isi"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	k, err := a.svc.TambahKabar(r.Context(), userDari(r.Context()).ID, id, in.Judul, in.Isi)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiProgramKabar, "program_donasi", id, map[string]any{"kabar": "tambah", "kabar_id": k.ID})
	a.tulisJSON(w, http.StatusCreated, k)
}

func (a *API) hapusKabar(w http.ResponseWriter, r *http.Request) {
	id, kabar := r.PathValue("id"), r.PathValue("kabar")
	if err := a.svc.HapusKabar(r.Context(), id, kabar); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiProgramKabar, "program_donasi", id, map[string]any{"kabar": "hapus", "kabar_id": kabar})
	w.WriteHeader(http.StatusNoContent)
}
