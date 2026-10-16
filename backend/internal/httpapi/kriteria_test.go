package httpapi

import "testing"

// kriteriaAPI = bentuk balasan GET/PUT /api/kriteria yang dipakai tes ini.
type kriteriaAPI struct {
	Items []struct {
		ID    string            `json:"id"`
		Kode  string            `json:"kode"`
		Nama  string            `json:"nama"`
		Tipe  string            `json:"tipe"`
		Aktif bool              `json:"aktif"`
		Skala map[string]string `json:"skala"`
	} `json:"items"`
	TotalPersen int `json:"total_persen"`
}

// Admin mengedit nama dan menambah kriteria baru lewat satu PUT; Ketua hanya boleh melihat.
func TestKriteria_EditDanTambahLewatAPI(t *testing.T) {
	srv := siapkanServer(t)
	admin := klienBaru(t, srv)
	admin.login("admin")

	var awal kriteriaAPI
	admin.minta("GET", "/api/kriteria", nil, 200, &awal)
	if len(awal.Items) != 5 || awal.TotalPersen != 100 {
		t.Fatalf("awal: %d kriteria, total %d%%", len(awal.Items), awal.TotalPersen)
	}
	c1 := awal.Items[0]

	// 0,20 (C1 turun dari 0,30) + 0,20 + 0,20 + 0,15 + 0,15 + 0,10 (C6 baru) = 1,00
	badan := map[string]any{
		"items": []any{map[string]any{
			"id": c1.ID, "nama": "Pendapatan keluarga per bulan", "tipe": "cost", "bobot": 0.20, "skala": nil, "aktif": true,
		}},
		"baru": []any{map[string]any{
			"nama": "Jumlah anak sekolah", "tipe": "benefit", "bobot": 0.10,
			"skala": map[string]string{"1": "sedikit", "2": "banyak"},
		}},
	}
	var sesudah kriteriaAPI
	admin.minta("PUT", "/api/kriteria", badan, 200, &sesudah)
	if len(sesudah.Items) != 6 || sesudah.TotalPersen != 100 {
		t.Fatalf("sesudah: %d kriteria, total %d%%", len(sesudah.Items), sesudah.TotalPersen)
	}
	if sesudah.Items[0].Nama != "Pendapatan keluarga per bulan" {
		t.Errorf("nama C1 = %q", sesudah.Items[0].Nama)
	}
	if k := sesudah.Items[5]; k.Kode != "C6" || k.Nama != "Jumlah anak sekolah" || !k.Aktif || k.Skala["2"] != "banyak" {
		t.Errorf("kriteria baru = %+v", k)
	}

	// Pelanggaran aturan → 422 dengan pesan, dan tidak ada yang berubah.
	var gagal struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	admin.minta("PUT", "/api/kriteria", map[string]any{
		"baru": []any{map[string]any{"nama": "Beban utang", "tipe": "cost", "bobot": 0.10, "skala": nil}}, // total jadi 110%
	}, 422, &gagal)
	if gagal.Fields["bobot"] == "" {
		t.Errorf("422 tanpa pesan bobot: %+v", gagal)
	}
	var tetap kriteriaAPI
	admin.minta("GET", "/api/kriteria", nil, 200, &tetap)
	if len(tetap.Items) != 6 {
		t.Errorf("jumlah kriteria = %d, mau 6 (PUT yang ditolak tidak boleh menambah)", len(tetap.Items))
	}

	ketua := klienBaru(t, srv)
	ketua.login("ketua")
	ketua.minta("PUT", "/api/kriteria", badan, 403, nil)
}
