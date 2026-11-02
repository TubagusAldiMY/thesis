package httpapi

import "net/http"

// Setup awal (publik, tanpa sesi): membuat akun Ketua pertama selama tabel users
// masih kosong dan kode setup dari log server diketahui. Lihat service/setup.go.

// setupStatus memberi tahu halaman publik apakah /setup masih terbuka.
func (a *API) setupStatus(w http.ResponseWriter, r *http.Request) {
	aktif, err := a.svc.SetupAktif(r.Context())
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusOK, map[string]bool{"aktif": aktif})
}

// setupBuat membuat akun Ketua pertama. Setup tidak aktif → 404 (sama dengan
// rute yang tidak ada). Password dan kode tidak pernah masuk log maupun audit.
func (a *API) setupBuat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Nama     string `json:"nama"`
		Password string `json:"password"`
	}
	if !a.bacaJSON(w, r, &in) {
		return
	}
	u, err := a.svc.BuatAkunPertama(r.Context(), ipKlien(r), in.Token, in.Username, in.Nama, in.Password)
	if err != nil {
		a.gagal(w, r, err)
		return
	}
	a.tulisJSON(w, http.StatusCreated, u)
}
