package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

var programBaru = map[string]any{
	"judul": "Renovasi Atap Masjid", "ringkasan": "Atap bocor di sisi utara perlu diganti sebelum musim hujan.",
	"cerita": "Atap masjid bocor di beberapa titik sehingga jamaah terganggu saat hujan.",
	"target": 5000000, "batas_waktu": "", "status": "aktif",
}

// unggah mengirim isi berkas mentah ke endpoint gambar dan mengembalikan status.
func (k *klien) unggah(path string, isi []byte) int {
	k.t.Helper()
	req, _ := http.NewRequest("PUT", k.srv.URL+path, bytes.NewReader(isi))
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := k.c.Do(req)
	if err != nil {
		k.t.Fatal(err)
	}
	res.Body.Close()
	return res.StatusCode
}

// Sanity: Admin mengelola program, Ketua hanya melihat, publik melihat program
// terbit beserta gambar dan donatur (nama anonim tersamar), draf tersembunyi.
func TestProgram_HakAksesPublikDanGambar(t *testing.T) {
	srv, ti, log := siapkanServerGerbang(t, true)
	admin, ketua, publik := klienBaru(t, srv), klienBaru(t, srv), klienBaru(t, srv)
	admin.login("admin")
	ketua.login("ketua")

	var p struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	admin.minta("POST", "/api/program", programBaru, 201, &p)
	ketua.minta("POST", "/api/program", programBaru, 403, nil)
	ketua.minta("GET", "/api/program/"+p.ID, nil, 200, nil)
	publik.minta("GET", "/api/program", nil, 401, nil)

	draf := map[string]any{}
	for k, v := range programBaru {
		draf[k] = v
	}
	draf["judul"], draf["status"] = "Program Rahasia Draf", "draf"
	var d struct {
		Slug string `json:"slug"`
	}
	admin.minta("POST", "/api/program", draf, 201, &d)
	publik.minta("GET", "/api/publik/program/"+d.Slug, nil, 404, nil)
	var daftar []struct {
		Slug string `json:"slug"`
	}
	publik.minta("GET", "/api/publik/program", nil, 200, &daftar)
	if len(daftar) != 1 || daftar[0].Slug != p.Slug {
		t.Fatalf("daftar publik = %+v", daftar)
	}

	// Gambar: ketua tidak boleh mengunggah, berkas bukan gambar/terlalu besar ditolak, PNG diterima.
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 64))
	if c := ketua.unggah("/api/program/"+p.ID+"/gambar", png); c != 403 {
		t.Errorf("ketua unggah = %d", c)
	}
	if c := admin.unggah("/api/program/"+p.ID+"/gambar", []byte("<svg onload=alert(1)>")); c != 422 {
		t.Errorf("svg = %d, mau 422", c)
	}
	if c := admin.unggah("/api/program/"+p.ID+"/gambar", make([]byte, 3<<20)); c != 422 {
		t.Errorf("3 MB = %d, mau 422", c)
	}
	if c := admin.unggah("/api/program/"+p.ID+"/gambar", png); c != 204 {
		t.Fatalf("png = %d", c)
	}
	res, err := http.Get(srv.URL + "/api/publik/program/" + p.Slug + "/gambar")
	if err != nil {
		t.Fatal(err)
	}
	isi, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/png" || !bytes.Equal(isi, png) ||
		!strings.Contains(res.Header.Get("Content-Security-Policy"), "sandbox") || res.Header.Get("Cache-Control") == "no-store" {
		t.Fatalf("gambar publik = %d %v", res.StatusCode, res.Header)
	}

	// Donasi anonim ke program: lunas → publik hanya melihat "Hamba Allah".
	donasi := map[string]any{"nama": "Pak Rahasia Sekali", "jenis": "infak", "jumlah": 250000, "program": p.Slug, "anonim": true}
	publik.minta("POST", "/api/publik/donasi", donasi, 201, nil)
	var semua struct {
		Items []struct {
			ID           string `json:"id"`
			ProgramJudul string `json:"program_judul"`
		} `json:"items"`
	}
	admin.minta("GET", "/api/donasi?program="+p.ID, nil, 200, &semua)
	if len(semua.Items) != 1 || semua.Items[0].ProgramJudul != "Renovasi Atap Masjid" {
		t.Fatalf("donasi program (pengurus) = %+v", semua)
	}
	if code, body := kirimNotifikasi(t, srv, ti.Muatan(semua.Items[0].ID, 250000, "settlement")); code != 200 {
		t.Fatalf("webhook = %d %s", code, body)
	}
	var detail struct {
		Terkumpul int64 `json:"terkumpul"`
		Donatur   int   `json:"donatur"`
		Terbaru   []struct {
			Nama string `json:"nama"`
		} `json:"donatur_terbaru"`
	}
	publik.minta("GET", "/api/publik/program/"+p.Slug, nil, 200, &detail)
	if detail.Terkumpul != 250000 || detail.Donatur != 1 || len(detail.Terbaru) != 1 || detail.Terbaru[0].Nama != "Hamba Allah" {
		t.Fatalf("detail publik = %+v", detail)
	}

	// Zakat ke program ditolak dengan pesan per kolom.
	donasi["jenis"] = "zakat_mal"
	publik.minta("POST", "/api/publik/donasi", donasi, 422, nil)

	// Kabar: tambah lalu hapus, ketua tidak boleh.
	var k struct {
		ID string `json:"id"`
	}
	kabar := map[string]string{"judul": "Material sudah dibeli", "isi": "Genteng dan rangka baja sudah tiba di masjid."}
	ketua.minta("POST", "/api/program/"+p.ID+"/kabar", kabar, 403, nil)
	admin.minta("POST", "/api/program/"+p.ID+"/kabar", kabar, 201, &k)
	admin.minta("DELETE", "/api/program/"+p.ID+"/kabar/"+k.ID, nil, 204, nil)
	admin.minta("DELETE", "/api/program/"+p.ID+"/kabar/"+k.ID, nil, 404, nil)

	if strings.Contains(log.String(), "Pak Rahasia") {
		t.Error("nama donatur tercatat di log server")
	}
}
