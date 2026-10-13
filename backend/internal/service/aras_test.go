package service

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	"spkzakat/internal/aras"
	"spkzakat/internal/domain"
)

func namaUrut(items []HasilArasItem) []string {
	n := make([]string, len(items))
	for i, it := range items {
		n[i] = it.Mustahik.Nama
	}
	return n
}

// jumlahRun membaca langsung dari tabel (bukan lewat service) supaya tidak
// ikut menanggung bug yang sedang diuji.
func (c *contoh) jumlahRun(t *testing.T) int {
	t.Helper()
	var n int
	if err := c.e.db.QueryRow(`SELECT COUNT(*) FROM aras_run WHERE periode_id = ?`, c.periode.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (c *contoh) statusPeriode(t *testing.T) domain.StatusPeriode {
	t.Helper()
	p, err := c.e.svc.AmbilPeriode(bg, c.periode.ID)
	if err != nil {
		t.Fatal(err)
	}
	return p.Status
}

// INI TEST UTAMANYA: contoh hitung di docs/02 dijalankan dari data mentah
// lewat service + store sampai tersimpan di MySQL, lalu dibaca kembali dan
// dicocokkan dengan angka di dokumen.
func TestHitungAras_ContohDokumen02_DariDataSampaiTersimpan(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)

	hasil, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user)
	if err != nil {
		t.Fatal(err)
	}

	mau := []struct {
		kunci  string
		k      float64
		status domain.StatusHasil
	}{
		{"A2", 0.8939, domain.HasilUsulan}, {"A4", 0.7327, domain.HasilUsulan}, {"A1", 0.6888, domain.HasilUsulan},
		{"A5", 0.5611, domain.HasilCadangan}, {"A3", 0.4099, domain.HasilCadangan},
	}
	cek := func(t *testing.T, nama string, items []HasilArasItem, toleransi float64) {
		t.Helper()
		if len(items) != len(mau) {
			t.Fatalf("%s: %d baris, mau %d", nama, len(items), len(mau))
		}
		for i, m := range mau {
			it := items[i]
			if it.Mustahik.Nama != c.m[m.kunci].Nama || it.Ranking != i+1 || it.Status != m.status || math.Abs(it.K-m.k) > toleransi {
				t.Errorf("%s baris %d = {%s rank %d K %.8f %s}, mau {%s rank %d K %.4f %s}",
					nama, i, it.Mustahik.Nama, it.Ranking, it.K, it.Status, c.m[m.kunci].Nama, i+1, m.k, m.status)
			}
			if !strings.HasPrefix(it.Mustahik.NIKTersamar, "•••• ") {
				t.Errorf("%s baris %d: NIK tidak tersamar: %q", nama, i, it.Mustahik.NIKTersamar)
			}
		}
	}

	// 1) Hasil yang dikembalikan langsung (toleransi pembulatan dokumen 4 desimal).
	cek(t, "langsung", hasil.Items, 5e-5)
	if hasil.Kuota != 3 || hasil.Periode.Status != domain.StatusDihitung || hasil.Run.Versi != 1 || !hasil.Run.IsCurrent {
		t.Errorf("kuota=%d status=%s versi=%d current=%v", hasil.Kuota, hasil.Periode.Status, hasil.Run.Versi, hasil.Run.IsCurrent)
	}
	if hasil.Langkah == nil || !reflect.DeepEqual(hasil.Langkah.A0, []float64{500_000, 6, 4, 4, 4}) ||
		!reflect.DeepEqual(hasil.Langkah.Kriteria, []string{"C1", "C2", "C3", "C4", "C5"}) || len(hasil.Langkah.Alternatif) != 5 {
		t.Errorf("langkah = %+v", hasil.Langkah)
	}

	// 2) Dibaca ULANG dari database (jalur yang dipakai halaman Hasil).
	dari, err := c.e.svc.AmbilHasilAras(bg, c.periode.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	cek(t, "dari database", dari.Items, 5e-5)
	if dari.Periode.Status != domain.StatusDihitung || dari.Kuota != 3 || dari.Run.Versi != 1 {
		t.Errorf("dari database: status=%s kuota=%d versi=%d", dari.Periode.Status, dari.Kuota, dari.Run.Versi)
	}
	if dari.Langkah == nil || !reflect.DeepEqual(dari.Langkah.A0, []float64{500_000, 6, 4, 4, 4}) {
		t.Errorf("langkah dari database = %+v", dari.Langkah)
	}
	if math.Abs(dari.Run.S0-0.2333) > 5e-5 {
		t.Errorf("S0 = %v, mau 0,2333", dari.Run.S0)
	}
	want := []domain.BobotKriteria{
		{Kode: "C1", Tipe: domain.TipeCost, Bobot: 0.30}, {Kode: "C2", Tipe: domain.TipeBenefit, Bobot: 0.20},
		{Kode: "C3", Tipe: domain.TipeBenefit, Bobot: 0.20}, {Kode: "C4", Tipe: domain.TipeBenefit, Bobot: 0.15},
		{Kode: "C5", Tipe: domain.TipeBenefit, Bobot: 0.15},
	}
	if !reflect.DeepEqual(dari.Run.Bobot, want) {
		t.Errorf("potret bobot = %+v", dari.Run.Bobot)
	}

	// 3) Status periode di database dan riwayat.
	if got := c.statusPeriode(t); got != domain.StatusDihitung {
		t.Errorf("status periode di database = %s", got)
	}
	runs, err := c.e.svc.DaftarRun(bg, c.periode.ID)
	if err != nil || len(runs) != 1 {
		t.Fatalf("riwayat = %+v, %v", runs, err)
	}
	if r := runs[0]; r.Versi != 1 || !r.IsCurrent || r.JumlahKandidat != 5 || r.Teratas != "Mustahik 2" || math.Abs(r.TeratasK-0.8939) > 5e-5 {
		t.Errorf("riwayat[0] = %+v", r)
	}
}

// Menjaga agar service meneruskan nilai ke paket aras dengan urutan kolom,
// tipe, dan bobot yang benar: hasilnya harus sama dengan memanggil aras.Hitung langsung.
func TestHitungAras_SamaDenganPaketArasLangsung(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)
	hasil, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user)
	if err != nil {
		t.Fatal(err)
	}

	kr := []aras.Kriteria{{Kode: "C1", Tipe: aras.Cost, Bobot: 0.30}, {Kode: "C2", Tipe: aras.Benefit, Bobot: 0.20},
		{Kode: "C3", Tipe: aras.Benefit, Bobot: 0.20}, {Kode: "C4", Tipe: aras.Benefit, Bobot: 0.15}, {Kode: "C5", Tipe: aras.Benefit, Bobot: 0.15}}
	var alt []aras.Alternatif
	for _, kunci := range urutKunci {
		n := dataContoh[kunci].nilai
		alt = append(alt, aras.Alternatif{ID: kunci, Nilai: []float64{n["C1"], n["C2"], n["C3"], n["C4"], n["C5"]}})
	}
	langsung, err := aras.Hitung(kr, alt)
	if err != nil {
		t.Fatal(err)
	}
	for i, h := range langsung.Alternatif {
		it := hasil.Items[i]
		if it.Mustahik.Nama != c.m[h.ID].Nama || math.Abs(it.K-h.K) > 1e-12 || math.Abs(it.S-h.S) > 1e-12 {
			t.Errorf("baris %d: service {%s S=%v K=%v}, aras langsung {%s S=%v K=%v}", i, it.Mustahik.Nama, it.S, it.K, c.m[h.ID].Nama, h.S, h.K)
		}
	}
}

func TestHitungAras_HitungUlangMenyimpanRiwayat(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)
	v1, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user)
	if err != nil {
		t.Fatalf("hitung ulang dari status dihitung harus boleh: %v", err)
	}
	if v1.Run.Versi != 1 || v2.Run.Versi != 2 || !v2.Run.IsCurrent {
		t.Errorf("versi = %d lalu %d, current v2 = %v", v1.Run.Versi, v2.Run.Versi, v2.Run.IsCurrent)
	}
	if c.statusPeriode(t) != domain.StatusDihitung {
		t.Error("status periode berubah setelah hitung ulang")
	}

	lama, err := c.e.svc.AmbilHasilAras(bg, c.periode.ID, 1)
	if err != nil || lama.Run.IsCurrent || lama.Run.Versi != 1 || !reflect.DeepEqual(namaUrut(lama.Items), namaUrut(v1.Items)) {
		t.Errorf("run v1 = %+v, %v", lama.Run, err)
	}
	if lama.Langkah == nil {
		t.Error("langkah run lama hilang")
	}
	terkini, _ := c.e.svc.AmbilHasilAras(bg, c.periode.ID, 0)
	if terkini.Run.Versi != 2 {
		t.Errorf("terkini = v%d, mau v2", terkini.Run.Versi)
	}
	runs, _ := c.e.svc.DaftarRun(bg, c.periode.ID)
	if len(runs) != 2 || runs[0].Versi != 2 || !runs[0].IsCurrent || runs[1].Versi != 1 || runs[1].IsCurrent {
		t.Errorf("riwayat = %+v", runs)
	}
}

// Bobot yang dipakai tiap run harus tersimpan bersama run itu, sehingga hasil
// lama tetap bisa dijelaskan walau bobot diubah kemudian.
func TestHitungAras_PotretBobotMengikutiSaatDihitung(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)
	if _, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user); err != nil {
		t.Fatal(err)
	}

	id := idKriteria(t, c.e)
	if err := c.e.svc.UbahKriteria(bg, []KriteriaUbah{{id["C1"], domain.TipeCost, 0.35}, {id["C2"], domain.TipeBenefit, 0.15}}); err != nil {
		t.Fatal(err)
	}
	v2, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user)
	if err != nil {
		t.Fatal(err)
	}

	lama, _ := c.e.svc.AmbilHasilAras(bg, c.periode.ID, 1)
	if lama.Run.Bobot[0].Bobot != 0.30 || lama.Run.Bobot[1].Bobot != 0.20 {
		t.Errorf("v1 memakai bobot %+v, mau 0,30/0,20 (potret saat dihitung)", lama.Run.Bobot[:2])
	}
	if v2.Run.Bobot[0].Bobot != 0.35 || v2.Run.Bobot[1].Bobot != 0.15 {
		t.Errorf("v2 memakai bobot %+v, mau 0,35/0,15", v2.Run.Bobot[:2])
	}
	if lama.Items[0].K == v2.Items[0].K {
		t.Error("K v1 dan v2 identik padahal bobot berbeda")
	}
}

func TestHitungAras_Validasi(t *testing.T) {
	tests := []struct {
		nama  string
		siap  func(t *testing.T, c *contoh)
		user  string
		field string
		pesan string
	}{
		{"penilaian belum lengkap (A5 kosong)", func(t *testing.T, c *contoh) { c.isi(t, "A1", "A2", "A3", "A4") }, "", "penilaian", "5 dari 25"},
		{"belum ada penilaian sama sekali", func(t *testing.T, c *contoh) {}, "", "penilaian", "25 dari 25"},
		{"kandidat dinonaktifkan setelah dipilih", func(t *testing.T, c *contoh) {
			c.isi(t)
			in := masukanSah(func(in *MustahikInput) {
				in.NIK, in.NoHP, in.Nama, in.Asnaf, in.Aktif = "", "", "Mustahik 3", domain.AsnafMiskin, false
			})
			if _, err := c.e.svc.UbahMustahik(bg, c.m["A3"].ID, in); err != nil {
				t.Fatal(err)
			}
		}, "", "kandidat", "tidak aktif: Mustahik 3"},
		{"golongan kandidat berubah di luar golongan periode", func(t *testing.T, c *contoh) {
			c.isi(t)
			in := masukanSah(func(in *MustahikInput) { in.NIK, in.NoHP, in.Nama, in.Asnaf = "", "", "Mustahik 1", domain.AsnafGharim })
			if _, err := c.e.svc.UbahMustahik(bg, c.m["A1"].ID, in); err != nil {
				t.Fatal(err)
			}
		}, "", "kandidat", "gharim"},
		{"total bobot aktif bukan 100%", func(t *testing.T, c *contoh) {
			c.isi(t)
			c.e.exec(t, `UPDATE kriteria SET bobot = 0.25 WHERE kode = 'C1'`)
		}, "", "bobot", "95%"},
		{"pengguna tidak diketahui", func(t *testing.T, c *contoh) { c.isi(t) }, "   ", "pengguna", ""},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			c := siapkanContoh(t)
			tt.siap(t, c)
			user := c.user
			if tt.user != "" {
				user = tt.user
			}
			_, err := c.e.svc.HitungAras(bg, c.periode.ID, user)
			ve := cekValidasi(t, err, tt.field)
			if tt.pesan != "" && !strings.Contains(ve.Fields[tt.field], tt.pesan) {
				t.Errorf("pesan = %q, harus memuat %q", ve.Fields[tt.field], tt.pesan)
			}
			// Gagal = tidak ada jejak: tak ada run, status tetap draft.
			if n := c.jumlahRun(t); n != 0 {
				t.Errorf("tersimpan %d run dari perhitungan yang gagal", n)
			}
			if got := c.statusPeriode(t); got != domain.StatusDraft {
				t.Errorf("status = %s, mau draft", got)
			}
		})
	}

	t.Run("periode tanpa kandidat", func(t *testing.T) {
		c := siapkanContoh(t)
		kosong, err := c.e.svc.BuatPeriode(bg, c.user, zakat())
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.e.svc.HitungAras(bg, kosong.ID, c.user)
		cekValidasi(t, err, "kandidat")
	})
}

func TestHitungAras_StatusYangTidakBolehDihitung(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)
	for _, status := range []string{"disetujui", "ditolak", "selesai"} {
		t.Run(status, func(t *testing.T) {
			c.e.exec(t, `UPDATE periode SET status = ? WHERE id = ?`, status, c.periode.ID)
			_, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user)
			if !errors.Is(err, domain.ErrConflict) {
				t.Errorf("err = %v, mau ErrConflict", err)
			}
			if n := c.jumlahRun(t); n != 0 {
				t.Errorf("tersimpan %d run", n)
			}
		})
	}
}

func TestHitungAras_TidakDitemukan(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)

	t.Run("periode tidak ada", func(t *testing.T) {
		if _, err := c.e.svc.HitungAras(bg, idHantu, c.user); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})
	t.Run("pengguna tidak ada: batal total, status tetap draft", func(t *testing.T) {
		if _, err := c.e.svc.HitungAras(bg, c.periode.ID, idHantu); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, mau ErrNotFound", err)
		}
		if n := c.jumlahRun(t); n != 0 {
			t.Errorf("tersimpan %d run", n)
		}
		if got := c.statusPeriode(t); got != domain.StatusDraft {
			t.Errorf("status = %s, mau draft (transaksi harus dibatalkan)", got)
		}
	})
}

func TestHitungAras_KuotaMenentukanUsulanDanCadangan(t *testing.T) {
	tests := []struct {
		nama     string
		dana     int64
		usulan   int
		cadangan int
	}{
		{"kuota 1", 1_000_000, 1, 4},
		{"kuota 3 (contoh dokumen)", 3_000_000, 3, 2},
		{"kuota sama dengan jumlah kandidat", 5_000_000, 5, 0},
		{"kuota melebihi jumlah kandidat", 10_000_000, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			c := siapkanContoh(t)
			c.isi(t)
			c.e.exec(t, `UPDATE periode SET dana_tersedia = ? WHERE id = ?`, tt.dana, c.periode.ID)

			hasil, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user)
			if err != nil {
				t.Fatal(err)
			}
			var u, cd int
			for _, it := range hasil.Items {
				if it.Status == domain.HasilUsulan {
					u++
					if it.Ranking > hasil.Kuota {
						t.Errorf("peringkat %d berstatus usulan padahal kuota %d", it.Ranking, hasil.Kuota)
					}
				} else {
					cd++
					if it.Ranking <= hasil.Kuota {
						t.Errorf("peringkat %d cadangan padahal masih dalam kuota %d", it.Ranking, hasil.Kuota)
					}
				}
			}
			if u != tt.usulan || cd != tt.cadangan {
				t.Errorf("usulan=%d cadangan=%d, mau %d dan %d", u, cd, tt.usulan, tt.cadangan)
			}
		})
	}
}

// Dua pengguna menekan "Hitung ARAS" bersamaan: keduanya sukses, berurutan
// (kunci baris periode), versi 1 dan 2, dan tepat satu yang terkini.
func TestHitungAras_SerentakTetapBerurutan(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = c.e.svc.HitungAras(bg, c.periode.ID, c.user)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("perhitungan %d gagal: %v", i, err)
		}
	}

	runs, err := c.e.svc.DaftarRun(bg, c.periode.ID)
	if err != nil || len(runs) != 2 {
		t.Fatalf("riwayat = %+v, %v; mau 2 run", runs, err)
	}
	if runs[0].Versi != 2 || runs[1].Versi != 1 {
		t.Errorf("versi = %d, %d; mau 2, 1 (tanpa nomor kembar)", runs[0].Versi, runs[1].Versi)
	}
	if !runs[0].IsCurrent || runs[1].IsCurrent {
		t.Errorf("terkini = %v, %v; mau tepat satu (v2)", runs[0].IsCurrent, runs[1].IsCurrent)
	}
}

func TestPenilaianDanKandidatTerkunciSetelahDihitung(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)
	if _, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user); err != nil {
		t.Fatal(err)
	}
	if err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, c.masukan("A1")); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("SimpanPenilaian: err = %v, mau ErrConflict", err)
	}
	if err := c.e.svc.AturKandidat(bg, c.periode.ID, []string{c.m["A1"].ID}); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("AturKandidat: err = %v, mau ErrConflict", err)
	}
	// Grid tetap bisa DIBACA (halaman Penilaian read-only).
	if g, err := c.e.svc.AmbilPenilaian(bg, c.periode.ID); err != nil || g.Lengkap != 5 {
		t.Errorf("AmbilPenilaian = lengkap %d, %v", g.Lengkap, err)
	}
}

func TestAmbilHasilAras_TidakAda(t *testing.T) {
	c := siapkanContoh(t)
	if _, err := c.e.svc.AmbilHasilAras(bg, c.periode.ID, 0); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("belum pernah dihitung: err = %v, mau ErrNotFound", err)
	}
	c.isi(t)
	if _, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user); err != nil {
		t.Fatal(err)
	}
	if _, err := c.e.svc.AmbilHasilAras(bg, c.periode.ID, 99); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("versi 99: err = %v, mau ErrNotFound", err)
	}
	if _, err := c.e.svc.AmbilHasilAras(bg, idHantu, 0); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("periode hantu: err = %v, mau ErrNotFound", err)
	}
	if _, err := c.e.svc.DaftarRun(bg, idHantu); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("DaftarRun periode hantu: err = %v, mau ErrNotFound", err)
	}

	belumDihitung, err := c.e.svc.BuatPeriode(bg, c.user, zakat())
	if err != nil {
		t.Fatal(err)
	}
	if runs, err := c.e.svc.DaftarRun(bg, belumDihitung.ID); err != nil || len(runs) != 0 {
		t.Errorf("periode belum dihitung: riwayat = %v, %v; mau kosong", runs, err)
	}
}
