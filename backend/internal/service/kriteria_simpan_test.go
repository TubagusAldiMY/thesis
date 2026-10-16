package service

import (
	"errors"
	"strings"
	"testing"

	"spkzakat/internal/domain"
)

// kriteriaMenurutKode membaca semua kriteria dari database, kuncinya kode (C1..).
func kriteriaMenurutKode(t *testing.T, e *lingkungan) map[string]domain.Kriteria {
	t.Helper()
	d, err := e.svc.DaftarKriteria(bg)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]domain.Kriteria{}
	for _, k := range d.Items {
		m[k.Kode] = k
	}
	return m
}

// utuh membentuk isian LENGKAP dari baris yang sudah ada (formulir selalu mengirim semua kolom).
func utuh(k domain.Kriteria) KriteriaEdit {
	return KriteriaEdit{ID: k.ID, Nama: k.Nama, Tipe: k.Tipe, Bobot: k.Bobot, Skala: k.Skala, Aktif: k.Aktif}
}

func TestSimpanKriteria(t *testing.T) {
	tests := []struct {
		nama   string
		fn     func(k map[string]domain.Kriteria) KriteriaSimpan
		mauErr string // "" sukses; "validasi:<field>"; "notfound"
		pesan  string // potongan pesan yang harus ada
		cek    func(t *testing.T, k map[string]domain.Kriteria)
	}{
		{"ganti nama", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Nama = "  Jumlah anggota keluarga  "
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "", "", func(t *testing.T, k map[string]domain.Kriteria) {
			if k["C2"].Nama != "Jumlah anggota keluarga" { // spasi dipangkas
				t.Errorf("nama C2 = %q", k["C2"].Nama)
			}
		}},
		{"beri skala pada kriteria bernilai asli", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Skala = map[string]string{"1": "1–2 orang", "2": "3–4 orang", "3": "5 orang atau lebih"}
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "", "", func(t *testing.T, k map[string]domain.Kriteria) {
			if len(k["C2"].Skala) != 3 || k["C2"].Skala["3"] != "5 orang atau lebih" {
				t.Errorf("skala C2 = %v", k["C2"].Skala)
			}
		}},
		{"hapus skala (kembali bernilai asli)", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C3"])
			u.Skala = nil
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "", "", func(t *testing.T, k map[string]domain.Kriteria) {
			if k["C3"].Skala != nil {
				t.Errorf("skala C3 = %v, mau nil", k["C3"].Skala)
			}
		}},
		{"nonaktifkan C5 sambil menyeimbangkan bobot", func(k map[string]domain.Kriteria) KriteriaSimpan {
			c1, c5 := utuh(k["C1"]), utuh(k["C5"])
			c1.Bobot, c5.Aktif = 0.45, false // 0,45 + 0,20 + 0,20 + 0,15 = 1,00
			return KriteriaSimpan{Ubah: []KriteriaEdit{c1, c5}}
		}, "", "", func(t *testing.T, k map[string]domain.Kriteria) {
			if k["C5"].Aktif || !k["C1"].Aktif {
				t.Errorf("aktif C5=%v C1=%v", k["C5"].Aktif, k["C1"].Aktif)
			}
		}},
		{"nonaktifkan tanpa menyeimbangkan → dibatalkan", func(k map[string]domain.Kriteria) KriteriaSimpan {
			c5 := utuh(k["C5"])
			c5.Aktif = false
			return KriteriaSimpan{Ubah: []KriteriaEdit{c5}}
		}, "validasi:bobot", "85%", nil},
		{"tinggal satu kriteria aktif → ditolak", func(k map[string]domain.Kriteria) KriteriaSimpan {
			var ubah []KriteriaEdit
			for kode, kr := range k {
				u := utuh(kr)
				u.Aktif = kode == "C1"
				if kode == "C1" {
					u.Bobot = 1.00
				}
				ubah = append(ubah, u)
			}
			return KriteriaSimpan{Ubah: ubah}
		}, "validasi:kriteria", "2–10", nil},
		{"tambah kriteria baru sambil menyeimbangkan", func(k map[string]domain.Kriteria) KriteriaSimpan {
			c1 := utuh(k["C1"])
			c1.Bobot = 0.20 // 0,20 + 0,20 + 0,20 + 0,15 + 0,15 + 0,10 = 1,00
			return KriteriaSimpan{
				Ubah: []KriteriaEdit{c1},
				Baru: []KriteriaBaru{{Nama: "Jumlah anak sekolah", Tipe: domain.TipeBenefit, Bobot: 0.10}},
			}
		}, "", "", func(t *testing.T, k map[string]domain.Kriteria) {
			b, ada := k["C6"]
			if !ada || b.Nama != "Jumlah anak sekolah" || !b.Aktif || b.Urutan != 6 || b.Tipe != domain.TipeBenefit || b.Skala != nil {
				t.Errorf("kriteria baru = %+v (ada=%v), mau C6 aktif urutan 6", b, ada)
			}
		}},
		{"dua kriteria baru mendapat kode berurutan", func(k map[string]domain.Kriteria) KriteriaSimpan {
			c1, c2 := utuh(k["C1"]), utuh(k["C2"])
			c1.Bobot, c2.Bobot = 0.15, 0.10 // 0,15 + 0,10 + 0,20 + 0,15 + 0,15 = 0,75; dua baru menambah 0,25
			return KriteriaSimpan{
				Ubah: []KriteriaEdit{c1, c2},
				Baru: []KriteriaBaru{
					{Nama: "Beban utang", Tipe: domain.TipeCost, Bobot: 0.15},
					{Nama: "Anak yatim dalam keluarga", Tipe: domain.TipeBenefit, Bobot: 0.10,
						Skala: map[string]string{"1": "tidak ada", "2": "ada"}},
				},
			}
		}, "", "", func(t *testing.T, k map[string]domain.Kriteria) {
			if k["C6"].Nama != "Beban utang" || k["C6"].Tipe != domain.TipeCost || k["C7"].Skala["2"] != "ada" || k["C7"].Urutan != 7 {
				t.Errorf("C6=%+v C7=%+v", k["C6"], k["C7"])
			}
		}},
		{"kriteria baru tanpa menyeimbangkan → dibatalkan, tidak ada baris baru", func(k map[string]domain.Kriteria) KriteriaSimpan {
			return KriteriaSimpan{Baru: []KriteriaBaru{{Nama: "Jumlah anak sekolah", Tipe: domain.TipeBenefit, Bobot: 0.10}}}
		}, "validasi:bobot", "110%", func(t *testing.T, k map[string]domain.Kriteria) {
			if _, ada := k["C6"]; ada {
				t.Error("kriteria baru tetap tersimpan padahal paket dibatalkan (rollback gagal)")
			}
		}},
		{"nama sama dengan kriteria lain (beda huruf besar) → ditolak", func(k map[string]domain.Kriteria) KriteriaSimpan {
			c1 := utuh(k["C1"])
			c1.Bobot = 0.20
			return KriteriaSimpan{
				Ubah: []KriteriaEdit{c1},
				Baru: []KriteriaBaru{{Nama: "JUMLAH TANGGUNGAN", Tipe: domain.TipeBenefit, Bobot: 0.10}},
			}
		}, "validasi:nama", "dipakai", nil},
		{"nama terlalu pendek", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Nama = " ab "
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:nama", "3–100", nil},
		{"nama kosong", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Nama = "   "
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:nama", "", nil},
		{"nama terlalu panjang", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Nama = strings.Repeat("a", 101)
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:nama", "", nil},
		{"skala hanya satu pilihan", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Skala = map[string]string{"1": "satu-satunya"}
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:skala", "2–9", nil},
		{"skala kunci 0", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Skala = map[string]string{"0": "nol", "1": "satu"}
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:skala", "1–9", nil},
		{"skala kunci bukan angka", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Skala = map[string]string{"a": "huruf", "1": "satu"}
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:skala", "angka", nil},
		{"skala kunci tidak kanonik (01)", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Skala = map[string]string{"01": "satu", "2": "dua"}
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:skala", "angka", nil},
		{"skala keterangan kosong", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Skala = map[string]string{"1": "ada", "2": "  "}
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:skala", "keterangan", nil},
		{"tipe tidak dikenal", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Tipe = "lainnya"
			return KriteriaSimpan{Ubah: []KriteriaEdit{u}}
		}, "validasi:tipe", "", nil},
		{"kriteria dikirim dua kali", func(k map[string]domain.Kriteria) KriteriaSimpan {
			return KriteriaSimpan{Ubah: []KriteriaEdit{utuh(k["C1"]), utuh(k["C1"])}}
		}, "validasi:kriteria", "", nil},
		{"tidak ada perubahan", func(k map[string]domain.Kriteria) KriteriaSimpan { return KriteriaSimpan{} }, "validasi:bobot", "", nil},
		{"ID tidak ada: perubahan lain dalam paket ikut dibatalkan", func(k map[string]domain.Kriteria) KriteriaSimpan {
			u := utuh(k["C2"])
			u.Nama = "Nama yang tidak boleh tersimpan"
			hantu := utuh(k["C1"])
			hantu.ID = idHantu
			return KriteriaSimpan{Ubah: []KriteriaEdit{u, hantu}}
		}, "notfound", "", func(t *testing.T, k map[string]domain.Kriteria) {
			if k["C2"].Nama != "Jumlah tanggungan" {
				t.Errorf("nama C2 = %q: perubahan bocor padahal paket dibatalkan", k["C2"].Nama)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			e := baru(t)
			err := e.svc.SimpanKriteria(bg, tt.fn(kriteriaMenurutKode(t, e)))

			switch {
			case tt.mauErr == "":
				if err != nil {
					t.Fatalf("err = %v, mau sukses", err)
				}
			case tt.mauErr == "notfound":
				if !errors.Is(err, domain.ErrNotFound) {
					t.Fatalf("err = %v, mau ErrNotFound", err)
				}
			default:
				ve := cekValidasi(t, err, strings.TrimPrefix(tt.mauErr, "validasi:"))
				if tt.pesan != "" && !strings.Contains(ve.Error(), tt.pesan) {
					t.Errorf("pesan = %q, harus memuat %q", ve.Error(), tt.pesan)
				}
			}
			if tt.cek != nil {
				tt.cek(t, kriteriaMenurutKode(t, e))
			}
		})
	}
}

// Lebih dari 10 kriteria aktif ditolak walau total bobot 100%.
func TestSimpanKriteria_BatasAtas(t *testing.T) {
	e := baru(t)
	k := kriteriaMenurutKode(t, e)
	var ubah []KriteriaEdit
	for _, kode := range []string{"C1", "C2", "C3", "C4", "C5"} {
		u := utuh(k[kode])
		u.Bobot = 0.05 // 5 × 0,05 = 0,25
		ubah = append(ubah, u)
	}
	tambah := []KriteriaBaru{ // 0,50 + 0,25 → total 1,00 dengan 11 kriteria aktif
		{Nama: "Kriteria uji 1", Tipe: domain.TipeBenefit, Bobot: 0.10}, {Nama: "Kriteria uji 2", Tipe: domain.TipeBenefit, Bobot: 0.10},
		{Nama: "Kriteria uji 3", Tipe: domain.TipeBenefit, Bobot: 0.10}, {Nama: "Kriteria uji 4", Tipe: domain.TipeBenefit, Bobot: 0.10},
		{Nama: "Kriteria uji 5", Tipe: domain.TipeBenefit, Bobot: 0.10}, {Nama: "Kriteria uji 6", Tipe: domain.TipeBenefit, Bobot: 0.25},
	}
	err := e.svc.SimpanKriteria(bg, KriteriaSimpan{Ubah: ubah, Baru: tambah})
	ve := cekValidasi(t, err, "kriteria")
	if !strings.Contains(ve.Error(), "sekarang 11") {
		t.Errorf("pesan = %q", ve.Error())
	}
	if _, ada := kriteriaMenurutKode(t, e)["C6"]; ada {
		t.Error("kriteria ke-11 tersimpan padahal ditolak")
	}
}

// Regresi riwayat: mengganti nama kriteria TIDAK mengubah potret run lama, supaya
// laporan yang sudah jadi tetap memakai nama saat dihitung.
func TestSimpanKriteria_PotretRunTidakBerubah(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)
	if _, err := c.e.svc.HitungAras(bg, c.periode.ID, c.user); err != nil {
		t.Fatal(err)
	}

	k := kriteriaMenurutKode(t, c.e)
	u := utuh(k["C1"])
	u.Nama = "Pendapatan bulanan keluarga"
	if err := c.e.svc.SimpanKriteria(bg, KriteriaSimpan{Ubah: []KriteriaEdit{u}}); err != nil {
		t.Fatal(err)
	}

	if got := kriteriaMenurutKode(t, c.e)["C1"].Nama; got != "Pendapatan bulanan keluarga" {
		t.Fatalf("nama C1 sekarang = %q", got)
	}
	hasil, err := c.e.svc.AmbilHasilAras(bg, c.periode.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := hasil.Run.Bobot[0].Nama; got != "Penghasilan per bulan" {
		t.Errorf("potret run C1 = %q, mau nama saat dihitung %q", got, "Penghasilan per bulan")
	}
}

// Menambah kriteria aktif saat penilaian periode sudah lengkap membuat penilaian
// belum lengkap lagi: Admin wajib mengisi kriteria baru sebelum menghitung ulang.
func TestSimpanKriteria_KriteriaBaruMembuatPenilaianBelumLengkap(t *testing.T) {
	c := siapkanContoh(t)
	c.isi(t)
	if d, _ := c.e.svc.AmbilDashboard(bg); d.Aktif == nil || d.Aktif.LangkahAktif != 3 {
		t.Fatalf("sebelum: langkah aktif = %+v, mau 3 (siap hitung)", d.Aktif)
	}

	k := kriteriaMenurutKode(t, c.e)
	c1 := utuh(k["C1"])
	c1.Bobot = 0.20
	err := c.e.svc.SimpanKriteria(bg, KriteriaSimpan{
		Ubah: []KriteriaEdit{c1},
		Baru: []KriteriaBaru{{Nama: "Jumlah anak sekolah", Tipe: domain.TipeBenefit, Bobot: 0.10}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := c.e.svc.AmbilDashboard(bg); d.Aktif == nil || d.Aktif.LangkahAktif != 2 {
		t.Fatalf("sesudah: langkah aktif = %+v, mau 2 (penilaian belum lengkap)", d.Aktif)
	}

	// Kriteria baru dapat dinilai lewat kodenya (C6), lalu periode siap dihitung lagi.
	var in []PenilaianInput
	for _, kunci := range urutKunci {
		nilai := map[string]float64{}
		for kode, v := range dataContoh[kunci].nilai {
			nilai[kode] = v
		}
		nilai["C6"] = 2
		in = append(in, PenilaianInput{MustahikID: c.m[kunci].ID, Nilai: nilai})
	}
	if err := c.e.svc.SimpanPenilaian(bg, c.periode.ID, in); err != nil {
		t.Fatalf("nilai kriteria baru: %v", err)
	}
	if d, _ := c.e.svc.AmbilDashboard(bg); d.Aktif.LangkahAktif != 3 {
		t.Errorf("setelah diisi: langkah aktif = %d, mau 3", d.Aktif.LangkahAktif)
	}
}
