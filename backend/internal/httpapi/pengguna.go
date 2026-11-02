package httpapi

import (
	"net/http"

	"spkzakat/internal/domain"
)

// Kelola pengguna: khusus Ketua (dipasang dengan peran(ketua) di server.go).
// Password hanya diterima di body JSON dan tidak pernah masuk audit maupun log.

func (a *API) daftarPengguna(w http.ResponseWriter, r *http.Request) {
	us, err := a.svc.DaftarPengguna(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, us)
}

func (a *API) ambilPengguna(w http.ResponseWriter, r *http.Request) {
	u, err := a.svc.AmbilPengguna(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, u)
}

func (a *API) buatPengguna(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string      `json:"username"`
		Nama     string      `json:"nama"`
		Role     domain.Role `json:"role"`
		Password string      `json:"password"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	u, err := a.svc.BuatPengguna(r.Context(), in.Username, in.Nama, in.Role, in.Password)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiPenggunaBuat, "users", u.ID, map[string]any{"role": u.Role})
	a.tulisJSON(w, http.StatusCreated, u)
}

func (a *API) ubahPengguna(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nama string      `json:"nama"`
		Role domain.Role `json:"role"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	u, err := a.svc.UbahPengguna(r.Context(), userDari(r.Context()), r.PathValue("id"), in.Nama, in.Role)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiPenggunaUbah, "users", u.ID, map[string]any{"role": u.Role})
	a.tulisJSON(w, http.StatusOK, u)
}

func (a *API) nonaktifkanPengguna(w http.ResponseWriter, r *http.Request) {
	u, err := a.svc.NonaktifkanPengguna(r.Context(), userDari(r.Context()), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiPenggunaNonaktif, "users", u.ID, map[string]any{"role": u.Role})
	a.tulisJSON(w, http.StatusOK, u)
}

func (a *API) aktifkanPengguna(w http.ResponseWriter, r *http.Request) {
	u, err := a.svc.AktifkanPengguna(r.Context(), r.PathValue("id"))
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiPenggunaAktif, "users", u.ID, map[string]any{"role": u.Role})
	a.tulisJSON(w, http.StatusOK, u)
}

func (a *API) resetPasswordPengguna(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	id := r.PathValue("id")
	if err := a.svc.ResetPasswordPengguna(r.Context(), userDari(r.Context()), id, in.Password); err != nil {
		a.gagal(w, r, err)
		return
	}
	a.catat(r, domain.AksiPenggunaReset, "users", id, nil)
	w.WriteHeader(http.StatusNoContent)
}
