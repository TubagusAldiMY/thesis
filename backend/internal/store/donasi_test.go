package store

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

// kodeUji membuat kode_publik unik 32 karakter heksadesimal (seperti yang dibuat service).
func kodeUji() string { return strings.ReplaceAll(newID(), "-", "") }

// buatDonasi menyimpan donasi web berstatus menunggu; ubah lewat fungsi opsional.
func buatDonasi(t *testing.T, s *Store, ubah ...func(*domain.Donasi)) domain.Donasi {
	t.Helper()
	d := domain.Donasi{
		KodePublik: kodeUji(), Nama: "Hamba Allah", Jenis: domain.DonasiZakatFitrah, Jumlah: 50_000,
		Sumber: domain.SumberWeb, Status: domain.DonasiMenunggu,
	}
	for _, f := range ubah {
		f(&d)
	}
	if err := s.CreateDonasi(bg, &d); err != nil {
		t.Fatalf("CreateDonasi: %v", err)
	}
	return d
}

func TestDonasi_CreateGet(t *testing.T) {
	s := newStore(t)

	t.Run("donasi web minimal: kolom opsional NULL", func(t *testing.T) {
		d := buatDonasi(t, s)
		got, err := s.GetDonasi(bg, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.ID) != 36 || got.Nama != "Hamba Allah" || got.Jumlah != 50_000 || got.Status != domain.DonasiMenunggu || got.Sumber != domain.SumberWeb {
			t.Errorf("got = %+v", got)
		}
		if got.NoHPEnc != nil || got.NomorKuitansi != "" || got.Metode != "" || got.BayarURL != "" || got.DibayarPada != nil || got.DilihatPada != nil || got.DicatatOleh != "" {
			t.Errorf("kolom opsional tidak NULL: %+v", got)
		}
		if got.CreatedAt.IsZero() || got.CreatedAt.Location() != time.UTC {
			t.Errorf("CreatedAt = %v (loc %v), mau waktu UTC terisi", got.CreatedAt, got.CreatedAt.Location())
		}
		if dt := time.Since(got.CreatedAt); dt < -time.Minute || dt > time.Minute {
			t.Errorf("CreatedAt menyimpang %v dari sekarang (zona waktu salah?)", dt)
		}
	})

	t.Run("donasi petugas lengkap: biner, waktu, dan uang pulang-pergi utuh", func(t *testing.T) {
		petugas := seedUser(t, s)
		bayar := time.Date(2026, 10, 16, 5, 0, 0, 0, time.UTC)
		d := buatDonasi(t, s, func(d *domain.Donasi) {
			d.NoHPEnc = []byte{0x00, 0xff, 0x10}
			d.Sumber, d.Metode, d.Status = domain.SumberPetugas, "tunai", domain.DonasiDiterima
			d.NomorKuitansi, d.Keterangan, d.DicatatOleh = "KWT/2026/0001", "untuk anak yatim", petugas
			d.DibayarPada, d.DilihatPada = &bayar, &bayar
			d.Jumlah = 99_999_999_999 // jauh di atas batas int32; harus utuh sebagai int64
		})
		got, err := s.GetDonasi(bg, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.NoHPEnc, []byte{0x00, 0xff, 0x10}) {
			t.Errorf("data biner NoHPEnc rusak: %v", got.NoHPEnc)
		}
		if got.Jumlah != 99_999_999_999 || got.NomorKuitansi != "KWT/2026/0001" || got.Metode != "tunai" || got.Keterangan != "untuk anak yatim" || got.DicatatOleh != petugas {
			t.Errorf("got = %+v", got)
		}
		if got.DibayarPada == nil || !got.DibayarPada.Equal(bayar) || got.DibayarPada.Location() != time.UTC {
			t.Errorf("DibayarPada = %v, mau %v (UTC)", got.DibayarPada, bayar)
		}
	})

	t.Run("jumlah nol atau negatif ditolak CHECK database", func(t *testing.T) {
		for _, j := range []int64{0, -1} {
			d := domain.Donasi{KodePublik: kodeUji(), Nama: "X Y", Jenis: domain.DonasiInfak, Jumlah: j, Sumber: domain.SumberWeb, Status: domain.DonasiMenunggu}
			if err := s.CreateDonasi(bg, &d); err == nil {
				t.Errorf("jumlah %d diterima database", j)
			}
		}
	})
}

func TestDonasi_Unik(t *testing.T) {
	s := newStore(t)
	a := buatDonasi(t, s, func(d *domain.Donasi) { d.NomorKuitansi = "KWT/2026/0001"; d.Status = domain.DonasiDiterima })

	// kode_publik ganda
	d := domain.Donasi{KodePublik: a.KodePublik, Nama: "Lain", Jenis: domain.DonasiInfak, Jumlah: 10_000, Sumber: domain.SumberWeb, Status: domain.DonasiMenunggu}
	if err := s.CreateDonasi(bg, &d); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("kode_publik ganda: err = %v, mau ErrConflict", err)
	}
	// nomor kuitansi ganda
	d = domain.Donasi{KodePublik: kodeUji(), Nama: "Lain", Jenis: domain.DonasiInfak, Jumlah: 10_000, Sumber: domain.SumberWeb, Status: domain.DonasiDiterima, NomorKuitansi: "KWT/2026/0001"}
	if err := s.CreateDonasi(bg, &d); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("nomor kuitansi ganda: err = %v, mau ErrConflict", err)
	}
	// banyak donasi tanpa nomor (NULL) boleh
	buatDonasi(t, s)
	buatDonasi(t, s)
}

func TestDonasi_NotFound(t *testing.T) {
	s := newStore(t)
	if _, err := s.GetDonasi(bg, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetDonasi: err = %v", err)
	}
	if _, err := s.GetDonasiByKode(bg, kodeUji()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetDonasiByKode: err = %v", err)
	}
	if err := s.UpdateDonasi(bg, domain.Donasi{ID: "00000000-0000-4000-8000-000000000000", Status: domain.DonasiDiterima}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("UpdateDonasi: err = %v", err)
	}
	d := buatDonasi(t, s)
	got, err := s.GetDonasiByKode(bg, d.KodePublik)
	if err != nil || got.ID != d.ID {
		t.Errorf("GetDonasiByKode = %+v, %v", got, err)
	}
}

// Kuitansi adalah bukti keuangan: UpdateDonasi hanya boleh mengubah kolom status,
// bukan data donatur atau jumlah, walaupun pemanggil mengubah struct-nya.
func TestDonasi_UpdateHanyaKolomStatus(t *testing.T) {
	s := newStore(t)
	d := buatDonasi(t, s, func(d *domain.Donasi) { d.BayarURL = "https://bayar.example/x" })

	bayar := time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC)
	d.Status, d.Metode, d.NomorKuitansi, d.DibayarPada, d.BayarURL = domain.DonasiDiterima, "qris", "KWT/2026/0001", &bayar, ""
	d.Nama, d.Jumlah, d.Jenis = "DIUBAH", 1, domain.DonasiInfak // tidak boleh tersimpan
	if err := s.UpdateDonasi(bg, d); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetDonasi(bg, d.ID)
	if got.Status != domain.DonasiDiterima || got.Metode != "qris" || got.NomorKuitansi != "KWT/2026/0001" || got.BayarURL != "" || got.DibayarPada == nil {
		t.Errorf("kolom status tidak tersimpan: %+v", got)
	}
	if got.Nama != "Hamba Allah" || got.Jumlah != 50_000 || got.Jenis != domain.DonasiZakatFitrah {
		t.Errorf("data donatur/jumlah ikut berubah: %+v", got)
	}
}

func TestDonasi_List(t *testing.T) {
	s := newStore(t)
	buatDonasi(t, s, func(d *domain.Donasi) {
		d.Nama = "Budi Santoso"
		d.Jenis = domain.DonasiZakatMal
		d.Status = domain.DonasiDiterima
		d.NomorKuitansi = "A1"
	})
	time.Sleep(5 * time.Millisecond) // created_at berbeda supaya urutan terbaru-dulu pasti
	buatDonasi(t, s, func(d *domain.Donasi) {
		d.Nama = "Siti_Aminah"
		d.Jenis = domain.DonasiInfak
		d.Status = domain.DonasiDiterima
		d.NomorKuitansi = "A2"
	})
	time.Sleep(5 * time.Millisecond)
	buatDonasi(t, s, func(d *domain.Donasi) { d.Nama = "Candra 100%"; d.Jenis = domain.DonasiInfak })

	tests := []struct {
		nama    string
		f       DonasiFilter
		total   int
		pertama string
	}{
		{"semua, terbaru dulu", DonasiFilter{}, 3, "Candra 100%"},
		{"status diterima", DonasiFilter{Status: domain.DonasiDiterima}, 2, "Siti_Aminah"},
		{"jenis infak", DonasiFilter{Jenis: domain.DonasiInfak}, 2, "Candra 100%"},
		{"status + jenis", DonasiFilter{Status: domain.DonasiDiterima, Jenis: domain.DonasiInfak}, 1, "Siti_Aminah"},
		{"cari potongan nama tak peka huruf besar", DonasiFilter{Q: "SANTOSO"}, 1, "Budi Santoso"},
		{"LIKE % dibaca harfiah", DonasiFilter{Q: "%"}, 1, "Candra 100%"},
		{"LIKE _ dibaca harfiah", DonasiFilter{Q: "_"}, 1, "Siti_Aminah"},
		{"tidak ada yang cocok", DonasiFilter{Q: "zzz"}, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			l, err := s.ListDonasi(bg, tt.f)
			if err != nil {
				t.Fatal(err)
			}
			if l.Total != tt.total || len(l.Items) != tt.total {
				t.Fatalf("total = %d, items = %d, mau %d", l.Total, len(l.Items), tt.total)
			}
			if tt.total > 0 && l.Items[0].Nama != tt.pertama {
				t.Errorf("pertama = %q, mau %q", l.Items[0].Nama, tt.pertama)
			}
		})
	}

	// Paging: halaman 2 dengan 1 baris/halaman; per_page dibatasi 100, bawaan 20.
	l, _ := s.ListDonasi(bg, DonasiFilter{Page: 2, PerPage: 1})
	if len(l.Items) != 1 || l.Total != 3 || l.Items[0].Nama != "Siti_Aminah" {
		t.Errorf("halaman 2 = %+v (total %d)", l.Items, l.Total)
	}
	if l, _ := s.ListDonasi(bg, DonasiFilter{Page: -5, PerPage: 100000}); len(l.Items) != 3 {
		t.Errorf("page/per_page tidak valid harus dinormalkan, items = %d", len(l.Items))
	}
}

func TestDonasi_RingkasDanDilihat(t *testing.T) {
	s := newStore(t)
	r, err := s.RingkasDonasi(bg)
	if err != nil || r != (DonasiRingkas{}) {
		t.Fatalf("ringkas kosong = %+v, %v", r, err)
	}

	sekarang := time.Now().UTC()
	buatDonasi(t, s, func(d *domain.Donasi) { d.Status = domain.DonasiDiterima; d.NomorKuitansi = "R1"; d.Jumlah = 100_000 }) // web, belum dilihat
	buatDonasi(t, s, func(d *domain.Donasi) { d.Status = domain.DonasiDiterima; d.NomorKuitansi = "R2"; d.Jumlah = 50_000 })  // web, belum dilihat
	buatDonasi(t, s, func(d *domain.Donasi) {
		d.Status = domain.DonasiDiterima
		d.NomorKuitansi = "R3"
		d.Jumlah = 25_000
		d.DilihatPada = &sekarang
	}) // web, sudah dilihat
	buatDonasi(t, s, func(d *domain.Donasi) {
		d.Sumber = domain.SumberPetugas
		d.Status = domain.DonasiDiterima
		d.NomorKuitansi = "R4"
		d.Jumlah = 10_000
	}) // petugas: bukan notifikasi
	buatDonasi(t, s, func(d *domain.Donasi) { d.Status = domain.DonasiMenunggu; d.Jumlah = 999_999 })                           // belum dibayar: tidak dihitung
	buatDonasi(t, s, func(d *domain.Donasi) { d.Status = domain.DonasiDibatalkan; d.NomorKuitansi = "R5"; d.Jumlah = 777_777 }) // dibatalkan: tidak dihitung

	r, err = s.RingkasDonasi(bg)
	if err != nil {
		t.Fatal(err)
	}
	if r.Baru != 2 || r.JumlahDiterima != 4 || r.TotalDiterima != 185_000 {
		t.Errorf("ringkas = %+v, mau {Baru:2 JumlahDiterima:4 TotalDiterima:185000}", r)
	}

	n, err := s.TandaiDonasiDilihat(bg)
	if err != nil || n != 2 {
		t.Errorf("TandaiDonasiDilihat = %d, %v; mau 2", n, err)
	}
	if r, _ = s.RingkasDonasi(bg); r.Baru != 0 {
		t.Errorf("setelah ditandai: %+v", r)
	}
	if n, _ = s.TandaiDonasiDilihat(bg); n != 0 {
		t.Errorf("tandai ulang mengubah %d baris", n)
	}
}

func TestNextNomorKuitansi(t *testing.T) {
	s := newStore(t)

	t.Run("wajib di dalam transaksi", func(t *testing.T) {
		if _, err := s.NextNomorKuitansi(bg, 2026); err == nil {
			t.Error("di luar transaksi seharusnya galat (kunci baris tidak ditahan)")
		}
	})

	ambil := func(t *testing.T, tahun int) int {
		t.Helper()
		var n int
		if err := s.WithTx(bg, func(tx *Store) error {
			var err error
			n, err = tx.NextNomorKuitansi(bg, tahun)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("urut per tahun dan reset tiap tahun", func(t *testing.T) {
		got := []int{ambil(t, 2026), ambil(t, 2026), ambil(t, 2027), ambil(t, 2026), ambil(t, 2027)}
		want := []int{1, 2, 1, 3, 2}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("urutan = %v, mau %v", got, want)
			}
		}
	})

	t.Run("rollback tidak meninggalkan celah", func(t *testing.T) {
		errBatal := errors.New("batal")
		var diambil int
		err := s.WithTx(bg, func(tx *Store) error {
			var e error
			diambil, e = tx.NextNomorKuitansi(bg, 2030)
			if e != nil {
				return e
			}
			return errBatal
		})
		if !errors.Is(err, errBatal) || diambil != 1 {
			t.Fatalf("err = %v, diambil = %d", err, diambil)
		}
		if n := ambil(t, 2030); n != 1 {
			t.Errorf("setelah rollback nomor berikutnya = %d, mau 1 (tanpa celah)", n)
		}
	})
}

// GetDonasiForUpdate harus bisa dipakai di dalam transaksi dan membaca data yang sama.
func TestDonasi_GetForUpdateDalamTransaksi(t *testing.T) {
	s := newStore(t)
	d := buatDonasi(t, s)
	err := s.WithTx(bg, func(tx *Store) error {
		got, err := tx.GetDonasiForUpdate(bg, d.ID)
		if err != nil {
			return err
		}
		if got.ID != d.ID || got.Status != domain.DonasiMenunggu {
			t.Errorf("got = %+v", got)
		}
		got.Status = domain.DonasiGagal
		return tx.UpdateDonasi(bg, got)
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetDonasi(bg, d.ID); got.Status != domain.DonasiGagal {
		t.Errorf("status = %q setelah commit", got.Status)
	}
	if _, err := s.GetDonasiForUpdate(bg, d.ID); err != nil {
		t.Errorf("GetDonasiForUpdate di luar transaksi tetap harus bisa membaca: %v", err)
	}
}
