package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Sanity CMS konten: Admin dan Ketua sama-sama boleh mengelola, publik hanya
// membaca yang terbit, gambar aman, dan audit tidak memuat isi tulisan.
func TestKonten_PengurusPublikDanGambar(t *testing.T) {
	srv, _, log := siapkanServerGerbang(t, false)
	admin, ketua, publik := klienBaru(t, srv), klienBaru(t, srv), klienBaru(t, srv)
	admin.login("admin")
	ketua.login("ketua")

	kegiatan := map[string]any{
		"judul": "Peringatan Maulid Nabi", "kategori": "maulid_nabi",
		"ringkasan":     "Pengajian akbar dan santunan anak yatim di aula masjid.",
		"isi":           "RAHASIA-ISI-ARTIKEL pengajian akbar bersama ustaz tamu, dilanjutkan santunan.",
		"tanggal_acara": "2099-10-25", "jam_acara": "19:30", "lokasi": "Aula", "status": "terbit",
	}
	publik.minta("GET", "/api/kegiatan", nil, 401, nil)
	publik.minta("POST", "/api/kegiatan", kegiatan, 401, nil)

	var k struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	admin.minta("POST", "/api/kegiatan", kegiatan, 201, &k) // Admin boleh
	kedua := map[string]any{}
	for a, b := range kegiatan {
		kedua[a] = b
	}
	kedua["judul"], kedua["status"] = "Kajian Draf Ketua", "draf"
	var d struct {
		Slug string `json:"slug"`
	}
	ketua.minta("POST", "/api/kegiatan", kedua, 201, &d) // Ketua juga boleh

	var pub struct {
		Mendatang []struct{ Slug string } `json:"mendatang"`
		Terdahulu []struct{ Slug string } `json:"terdahulu"`
		Kategori  []struct{ Kode string } `json:"kategori"`
	}
	publik.minta("GET", "/api/publik/kegiatan", nil, 200, &pub)
	if len(pub.Mendatang) != 1 || pub.Mendatang[0].Slug != k.Slug || len(pub.Terdahulu) != 0 || len(pub.Kategori) == 0 {
		t.Fatalf("daftar publik = %+v", pub)
	}
	publik.minta("GET", "/api/publik/kegiatan?kategori=konser", nil, 422, nil)
	publik.minta("GET", "/api/publik/kegiatan/"+k.Slug, nil, 200, nil)
	publik.minta("GET", "/api/publik/kegiatan/"+d.Slug, nil, 404, nil) // draf

	// Gambar: bukan gambar / terlalu besar ditolak; PNG diterima dan disajikan aman.
	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 64))
	if c := admin.unggah("/api/kegiatan/"+k.ID+"/gambar", []byte("<svg onload=alert(1)>")); c != 422 {
		t.Errorf("svg = %d", c)
	}
	if c := admin.unggah("/api/kegiatan/"+k.ID+"/gambar", make([]byte, 3<<20)); c != 422 {
		t.Errorf("3 MB = %d", c)
	}
	if c := ketua.unggah("/api/kegiatan/"+k.ID+"/gambar", png); c != 204 {
		t.Fatalf("png = %d", c)
	}
	res, err := http.Get(srv.URL + "/api/publik/kegiatan/" + k.Slug + "/gambar")
	if err != nil {
		t.Fatal(err)
	}
	isi, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/png" || !bytes.Equal(isi, png) ||
		!strings.Contains(res.Header.Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("gambar publik = %d %v", res.StatusCode, res.Header)
	}

	// Jadwal rutin dan pengumuman: pengurus mengelola, publik hanya yang aktif/tayang.
	var j struct {
		ID string `json:"id"`
	}
	jadwal := map[string]any{"judul": "Kajian Subuh", "hari": 0, "jam": "05:30", "aktif": true}
	admin.minta("POST", "/api/jadwal-rutin", jadwal, 201, &j)
	admin.minta("POST", "/api/jadwal-rutin", map[string]any{"judul": "Kajian", "hari": 9, "jam": "05:30"}, 422, nil)
	var jp []struct{ Judul string }
	publik.minta("GET", "/api/publik/jadwal-rutin", nil, 200, &jp)
	if len(jp) != 1 {
		t.Fatalf("jadwal publik = %+v", jp)
	}
	jadwal["aktif"] = false
	ketua.minta("PUT", "/api/jadwal-rutin/"+j.ID, jadwal, 200, nil)
	publik.minta("GET", "/api/publik/jadwal-rutin", nil, 200, &jp)
	if len(jp) != 0 {
		t.Errorf("jadwal nonaktif tampil di publik: %+v", jp)
	}

	var p struct {
		ID string `json:"id"`
	}
	admin.minta("POST", "/api/pengumuman", map[string]any{"judul": "Kerja Bakti", "isi": "Kerja bakti Ahad pagi, mohon hadir semua.", "terbit": true}, 201, &p)
	var pp []struct{ Judul string }
	publik.minta("GET", "/api/publik/pengumuman", nil, 200, &pp)
	if len(pp) != 1 {
		t.Fatalf("pengumuman publik = %+v", pp)
	}
	ketua.minta("DELETE", "/api/pengumuman/"+p.ID, nil, 204, nil)
	publik.minta("GET", "/api/publik/pengumuman", nil, 200, &pp)
	if len(pp) != 0 {
		t.Errorf("pengumuman terhapus masih tampil: %+v", pp)
	}

	// Hapus kegiatan → hilang dari publik.
	admin.minta("DELETE", "/api/kegiatan/"+k.ID, nil, 204, nil)
	publik.minta("GET", "/api/publik/kegiatan/"+k.Slug, nil, 404, nil)

	// Audit mencatat jenis konten dan ID, tidak pernah isi tulisan.
	var audit struct {
		Items []struct {
			Action   string `json:"action"`
			Resource string `json:"resource"`
		} `json:"items"`
	}
	ketua.minta("GET", "/api/audit?kategori=konten", nil, 200, &audit)
	if len(audit.Items) < 8 {
		t.Fatalf("audit konten = %+v", audit.Items)
	}
	if strings.Contains(log.String(), "RAHASIA-ISI-ARTIKEL") {
		t.Error("isi artikel tercatat di log server")
	}
}
