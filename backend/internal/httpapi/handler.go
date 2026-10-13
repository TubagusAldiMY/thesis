package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/service"
)

// Handler hanya: baca input → panggil service → tulis JSON (+ audit bila
// aksi mengubah data). Validasi isi ada di service.

// healthz = smoke check: proses hidup dan MySQL menjawab.
func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.ping(ctx); err != nil {
		a.logger(r).Error("healthz: database tidak menjawab", "error", err.Error())
		a.tulisJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false})
		return
	}
	a.tulisJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- auth ----------

func (a *API) setCookie(w http.ResponseWriter, token string, umur time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: NamaCookie, Value: token, Path: "/", MaxAge: int(umur.Seconds()),
		HttpOnly: true, Secure: a.cookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	hasil, err := a.svc.Login(r.Context(), in.Username, in.Password, ipKlien(r))
	var lg *service.LoginGagalError
	if errors.As(err, &lg) {
		a.tulisError(w, r, http.StatusUnauthorized, lg.Error(), map[string]any{"sisa_percobaan": lg.SisaPercobaan})
		return
	}
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.setCookie(w, hasil.Token, service.DurasiSesi)
	a.tulisJSON(w, http.StatusOK, map[string]any{"user": hasil.User})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	if err := a.svc.Logout(r.Context(), userDari(r.Context()), tokenDari(r.Context()), ipKlien(r)); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.setCookie(w, "", -time.Second) // MaxAge < 0 = hapus cookie
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	a.tulisJSON(w, http.StatusOK, userDari(r.Context()))
}

func (a *API) gantiPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Lama string `json:"password_lama"`
		Baru string `json:"password_baru"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	err := a.svc.GantiPassword(r.Context(), userDari(r.Context()), tokenDari(r.Context()), in.Lama, in.Baru, ipKlien(r))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := a.svc.AmbilDashboard(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, d)
}

// ---------- mustahik ----------

// masukanMustahik = body POST/PUT mustahik. Aktif berupa pointer supaya
// "tidak dikirim" berarti aktif (bawaan formulir tambah).
type masukanMustahik struct {
	Nama   string       `json:"nama"`
	NIK    string       `json:"nik"`
	NoHP   string       `json:"no_hp"`
	Alamat string       `json:"alamat"`
	Asnaf  domain.Asnaf `json:"asnaf"`
	Aktif  *bool        `json:"aktif"`
}

func (m masukanMustahik) input() service.MustahikInput {
	aktif := m.Aktif == nil || *m.Aktif
	return service.MustahikInput{Nama: m.Nama, NIK: m.NIK, NoHP: m.NoHP, Alamat: m.Alamat, Asnaf: m.Asnaf, Aktif: aktif}
}

// angkaQuery membaca parameter angka; kosong/tidak valid → bawaan.
func angkaQuery(r *http.Request, nama string, bawaan int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(nama))
	if err != nil {
		return bawaan
	}
	return n
}

func (a *API) daftarMustahik(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mq := service.MustahikQuery{
		Q: q.Get("q"), Asnaf: domain.Asnaf(q.Get("asnaf")),
		Page: angkaQuery(r, "page", 1), PerPage: angkaQuery(r, "per_page", 20),
	}
	if v := q.Get("aktif"); v == "true" || v == "false" {
		aktif := v == "true"
		mq.Aktif = &aktif
	}
	h, err := a.svc.DaftarMustahik(r.Context(), mq)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, h)
}

func (a *API) buatMustahik(w http.ResponseWriter, r *http.Request) {
	var in masukanMustahik
	if !a.bacaJSON(w, r, &in) {
		return
	}
	m, err := a.svc.BuatMustahik(r.Context(), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiMustahikBuat, "mustahik", m.ID, map[string]any{"asnaf": m.Asnaf})
	a.tulisJSON(w, http.StatusCreated, m)
}

func (a *API) ambilMustahik(w http.ResponseWriter, r *http.Request) {
	m, err := a.svc.AmbilMustahik(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, m)
}

func (a *API) ubahMustahik(w http.ResponseWriter, r *http.Request) {
	var in masukanMustahik
	if !a.bacaJSON(w, r, &in) {
		return
	}
	m, err := a.svc.UbahMustahik(r.Context(), r.PathValue("id"), in.input())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiMustahikUbah, "mustahik", m.ID, map[string]any{"aktif": m.Aktif, "asnaf": m.Asnaf})
	a.tulisJSON(w, http.StatusOK, m)
}

// ---------- kriteria ----------

func (a *API) daftarKriteria(w http.ResponseWriter, r *http.Request) {
	k, err := a.svc.DaftarKriteria(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, k)
}

// ubahKriteria menerima SEMUA perubahan sekaligus karena syarat Σ bobot = 100%
// hanya bisa diperiksa atas satu paket utuh.
func (a *API) ubahKriteria(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []struct {
			ID    string              `json:"id"`
			Tipe  domain.TipeKriteria `json:"tipe"`
			Bobot float64             `json:"bobot"`
		} `json:"items"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	ubah := make([]service.KriteriaUbah, len(in.Items))
	detail := map[string]any{}
	for i, it := range in.Items {
		ubah[i] = service.KriteriaUbah{ID: it.ID, Tipe: it.Tipe, Bobot: it.Bobot}
		detail[it.ID] = map[string]any{"tipe": it.Tipe, "bobot": it.Bobot}
	}
	if err := a.svc.UbahKriteria(r.Context(), ubah); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKriteriaUbah, "kriteria", "", detail)
	a.daftarKriteria(w, r)
}

// ---------- periode ----------

func (a *API) daftarPeriode(w http.ResponseWriter, r *http.Request) {
	ps, err := a.svc.DaftarPeriode(r.Context(), domain.StatusPeriode(r.URL.Query().Get("status")))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	type baris struct {
		domain.Periode
		Kuota int `json:"kuota"`
	}
	out := make([]baris, len(ps))
	for i, p := range ps {
		out[i] = baris{p, p.Kuota()}
	}
	a.tulisJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (a *API) buatPeriode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nama               string              `json:"nama"`
		Jenis              domain.JenisPeriode `json:"jenis"`
		DanaTersedia       int64               `json:"dana_tersedia"`
		NominalPerPenerima int64               `json:"nominal_per_penerima"`
		AsnafDiizinkan     []domain.Asnaf      `json:"asnaf_diizinkan"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	p, err := a.svc.BuatPeriode(r.Context(), userDari(r.Context()).ID, service.PeriodeInput{
		Nama: in.Nama, Jenis: in.Jenis, DanaTersedia: in.DanaTersedia,
		NominalPerPenerima: in.NominalPerPenerima, AsnafDiizinkan: in.AsnafDiizinkan,
	})
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiPeriodeBuat, "periode", p.ID, map[string]any{"jenis": p.Jenis, "dana": p.DanaTersedia})
	a.tulisJSON(w, http.StatusCreated, p)
}

func (a *API) ambilPeriode(w http.ResponseWriter, r *http.Request) {
	d, err := a.svc.AmbilDetailPeriode(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, d)
}

func (a *API) daftarKandidat(w http.ResponseWriter, r *http.Request) {
	ks, err := a.svc.DaftarKandidat(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, map[string]any{"items": ks})
}

func (a *API) aturKandidat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MustahikIDs []string `json:"mustahik_ids"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	if err := a.svc.AturKandidat(r.Context(), id, in.MustahikIDs); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiKandidat, "periode", id, map[string]any{"jumlah": len(in.MustahikIDs)})
	a.daftarKandidat(w, r)
}

func (a *API) ambilPenilaian(w http.ResponseWriter, r *http.Request) {
	m, err := a.svc.AmbilPenilaian(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, m)
}

func (a *API) simpanPenilaian(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []struct {
			MustahikID string             `json:"mustahik_id"`
			Nilai      map[string]float64 `json:"nilai"`
		} `json:"items"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	masuk := make([]service.PenilaianInput, len(in.Items))
	for i, it := range in.Items {
		masuk[i] = service.PenilaianInput{MustahikID: it.MustahikID, Nilai: it.Nilai}
	}
	id := r.PathValue("id")
	if err := a.svc.SimpanPenilaian(r.Context(), id, masuk); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiPenilaian, "periode", id, map[string]any{"kandidat": len(masuk)})
	a.ambilPenilaian(w, r)
}

func (a *API) hitung(w http.ResponseWriter, r *http.Request) {
	h, err := a.svc.HitungAras(r.Context(), r.PathValue("id"), userDari(r.Context()).ID)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiHitung, "periode", h.Periode.ID, map[string]any{"versi": h.Run.Versi, "kandidat": len(h.Items), "kuota": h.Kuota})
	a.tulisJSON(w, http.StatusOK, h)
}

func (a *API) hasil(w http.ResponseWriter, r *http.Request) {
	h, err := a.svc.AmbilHasilAras(r.Context(), r.PathValue("id"), angkaQuery(r, "versi", 0))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, h)
}

func (a *API) keputusan(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Keputusan domain.Keputusan `json:"keputusan"`
		Catatan   string           `json:"catatan"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	p, err := a.svc.Putuskan(r.Context(), id, userDari(r.Context()).ID, in.Keputusan, in.Catatan)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	aksi := domain.AksiSetuju
	if p.Keputusan == domain.KeputusanTolak {
		aksi = domain.AksiTolak
	}
	a.catat(r, aksi, "periode", id, map[string]any{"versi": p.Versi})
	a.tulisJSON(w, http.StatusOK, p)
}

func (a *API) bukaKembali(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.svc.BukaKembali(r.Context(), id); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiBukaKembali, "periode", id, nil)
	a.ambilPeriode(w, r)
}

func (a *API) daftarPenyaluran(w http.ResponseWriter, r *http.Request) {
	rek, err := a.svc.DaftarPenyaluran(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, rek)
}

func (a *API) catatPenyaluran(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MustahikID string `json:"mustahik_id"`
		Nominal    int64  `json:"nominal"`
		Tanggal    string `json:"tanggal"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	p, err := a.svc.CatatPenyaluran(r.Context(), id, userDari(r.Context()).ID, service.PenyaluranInput{
		MustahikID: in.MustahikID, Nominal: in.Nominal, Tanggal: in.Tanggal,
	})
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiSalur, "periode", id, map[string]any{"mustahik_id": p.MustahikID, "nominal": p.Nominal})
	a.tulisJSON(w, http.StatusCreated, p)
}

func (a *API) tutup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.svc.TutupPeriode(r.Context(), id); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiTutup, "periode", id, nil)
	a.ambilPeriode(w, r)
}

func (a *API) laporan(w http.ResponseWriter, r *http.Request) {
	l, err := a.svc.AmbilLaporan(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	if r.URL.Query().Get("format") != "csv" {
		a.tulisJSON(w, http.StatusOK, l)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="laporan-`+l.Periode.ID+`.csv"`)
	if _, err := w.Write([]byte("\xEF\xBB\xBF")); err != nil { // BOM: Excel membaca UTF-8 dengan benar
		return
	}
	if err := l.TulisCSV(w); err != nil {
		a.logger(r).Error("tulis CSV gagal", "error", err.Error())
	}
}

func (a *API) audit(w http.ResponseWriter, r *http.Request) {
	h, err := a.svc.DaftarAudit(r.Context(), r.URL.Query().Get("kategori"), angkaQuery(r, "page", 1))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, h)
}
