package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// KriteriaDaftar = semua kriteria + total bobot aktif dalam persen (untuk
// indikator "Total bobot: 100% — siap disimpan" di UI).
type KriteriaDaftar struct {
	Items       []domain.Kriteria `json:"items"`
	TotalPersen int               `json:"total_persen"`
}

// KriteriaUbah = satu perubahan dari formulir Kriteria & Bobot.
type KriteriaUbah struct {
	ID    string
	Tipe  domain.TipeKriteria
	Bobot float64 // pecahan 0–1, kelipatan 0,05
}

// persen mengubah bobot pecahan ke persen bulat (0,15 → 15). Bobot dari
// database ber-4-desimal, jadi pembulatan ini tepat.
func persen(bobot float64) int { return int(math.Round(bobot * 100)) }

// bobotSah: kelipatan 0,05 dalam rentang 0,05–1,00 (FR-04). Pemeriksaan
// memakai persen bulat, bukan perbandingan float langsung, karena
// 0,15*100 = 15,000000000000002 di floating point.
func bobotSah(b float64) bool {
	if math.IsNaN(b) || math.IsInf(b, 0) {
		return false
	}
	p := math.Round(b * 100)
	return math.Abs(b*100-p) < 1e-6 && p >= 5 && p <= 100 && int(p)%5 == 0
}

// DaftarKriteria mengembalikan kriteria menurut urutan dan total bobot aktif.
func (s *Service) DaftarKriteria(ctx context.Context) (KriteriaDaftar, error) {
	ks, err := s.st.ListKriteria(ctx, false)
	if err != nil {
		return KriteriaDaftar{}, fmt.Errorf("service.DaftarKriteria: %w", err)
	}
	total := 0
	for _, k := range ks {
		if k.Aktif {
			total += persen(k.Bobot)
		}
	}
	return KriteriaDaftar{Items: ks, TotalPersen: total}, nil
}

// UbahKriteria menerapkan semua perubahan sekaligus dalam SATU transaksi.
// Setelah diterapkan, total bobot kriteria AKTIF harus tepat 100%; bila tidak,
// seluruh perubahan dibatalkan (FR-05) dan database tetap seperti semula.
//
// ponytail: dua Admin yang menyimpan serentak pada baris kriteria berbeda
// secara teori bisa sama-sama lolos cek total. Admin hanya satu-dua orang;
// kunci semua baris kriteria (SELECT ... FOR UPDATE) bila itu jadi masalah.
func (s *Service) UbahKriteria(ctx context.Context, ubah []KriteriaUbah) error {
	g := galat{}
	if len(ubah) == 0 {
		g.tambah("bobot", "Tidak ada perubahan untuk disimpan")
		return g.err()
	}
	lihat := make(map[string]bool, len(ubah))
	for _, u := range ubah {
		if lihat[u.ID] {
			g.tambah("kriteria", "Satu kriteria dikirim lebih dari sekali")
		}
		lihat[u.ID] = true
		if u.Tipe != domain.TipeBenefit && u.Tipe != domain.TipeCost {
			g.tambah("tipe", "Tipe kriteria harus benefit atau cost")
		}
		if !bobotSah(u.Bobot) {
			g.tambah("bobot", "Bobot harus kelipatan 0,05 antara 0,05 dan 1,00")
		}
	}
	if err := g.err(); err != nil {
		return err
	}

	return s.st.WithTx(ctx, func(tx *store.Store) error {
		for _, u := range ubah {
			if err := tx.UpdateKriteria(ctx, u.ID, u.Tipe, u.Bobot); err != nil {
				return fmt.Errorf("service.UbahKriteria: %w", err)
			}
		}
		aktif, err := tx.ListKriteria(ctx, true)
		if err != nil {
			return fmt.Errorf("service.UbahKriteria: %w", err)
		}
		total := 0
		for _, k := range aktif {
			total += persen(k.Bobot)
		}
		if total != 100 {
			g.tambah("bobot", fmt.Sprintf("Total bobot harus 100%% (sekarang %d%%)", total))
			return g.err() // memicu rollback
		}
		return nil
	})
}

// Batas isian kriteria yang dapat diedit Admin.
const (
	namaKriteriaMin   = 3
	namaKriteriaMaks  = 100
	kriteriaAktifMin  = 2  // ARAS membandingkan alternatif lintas kriteria; satu kriteria tak bermakna
	kriteriaAktifMaks = 10 // kode "C10" masih muat di kolom kode dan tabel penilaian tetap terbaca
)

// KriteriaEdit = isi LENGKAP satu kriteria yang sudah ada. Formulir selalu
// mengirim semua kolom, jadi tidak ada arti "tidak diubah".
type KriteriaEdit struct {
	ID    string
	Nama  string
	Tipe  domain.TipeKriteria
	Bobot float64
	Skala map[string]string // nil/kosong = bernilai asli (tanpa skala)
	Aktif bool
}

// KriteriaBaru = kriteria yang ditambahkan; kode (C6, C7, ...) dan urutan dibuat store.
type KriteriaBaru struct {
	Nama  string
	Tipe  domain.TipeKriteria
	Bobot float64
	Skala map[string]string
}

// KriteriaSimpan = satu paket perubahan dari halaman Kriteria.
type KriteriaSimpan struct {
	Ubah []KriteriaEdit
	Baru []KriteriaBaru
}

// periksaIsiKriteria memvalidasi satu baris isian dan mengembalikan nama dan skala
// yang sudah dirapikan (spasi dipangkas, skala kosong → nil). label dipakai di
// pesan galat ("C3", "baru 1") supaya Admin tahu baris mana yang salah.
func periksaIsiKriteria(g galat, label, nama string, tipe domain.TipeKriteria, bobot float64, skala map[string]string) (string, map[string]string) {
	nama = strings.TrimSpace(nama)
	if n := utf8.RuneCountInString(nama); n < namaKriteriaMin || n > namaKriteriaMaks {
		g.tambah("nama", fmt.Sprintf("Kriteria %s: nama harus %d–%d huruf", label, namaKriteriaMin, namaKriteriaMaks))
	}
	if tipe != domain.TipeBenefit && tipe != domain.TipeCost {
		g.tambah("tipe", fmt.Sprintf("Kriteria %s: tipe harus benefit atau cost", label))
	}
	if !bobotSah(bobot) {
		g.tambah("bobot", fmt.Sprintf("Kriteria %s: bobot harus kelipatan 0,05 antara 0,05 dan 1,00", label))
	}
	bersih, pesan := bersihkanSkala(skala)
	if pesan != "" {
		g.tambah("skala", fmt.Sprintf("Kriteria %s: %s", label, pesan))
	}
	return nama, bersih
}

// bersihkanSkala memeriksa skala nilai: kosong = nil (nilai asli); selain itu 2–9
// pilihan dengan kunci angka bulat 1–9 yang ditulis kanonik ("1", bukan "01")
// dan keterangan 1–100 huruf. Kunci harus angka karena penilaian menyimpan nilai
// sebagai angka dan pesanNilai mencocokkannya lewat strconv.Itoa.
func bersihkanSkala(skala map[string]string) (map[string]string, string) {
	if len(skala) == 0 {
		return nil, ""
	}
	if len(skala) < 2 || len(skala) > 9 {
		return nil, "skala butuh 2–9 pilihan (atau kosongkan untuk nilai angka bebas)"
	}
	hasil := make(map[string]string, len(skala))
	for k, v := range skala {
		n, err := strconv.Atoi(k)
		if err != nil || n < 1 || n > 9 || strconv.Itoa(n) != k {
			return nil, fmt.Sprintf("nilai skala %q harus angka bulat 1–9", k)
		}
		v = strings.TrimSpace(v)
		if l := utf8.RuneCountInString(v); l < 1 || l > 100 {
			return nil, fmt.Sprintf("keterangan skala %s harus 1–100 huruf", k)
		}
		hasil[k] = v
	}
	return hasil, ""
}

// SimpanKriteria menerapkan semua perubahan dan penambahan kriteria dalam SATU
// transaksi (FR-04/05). Sesudahnya, atas SEMUA kriteria: nama unik (tanpa
// membedakan huruf besar-kecil), jumlah kriteria aktif 2–10, dan total bobot
// aktif tepat 100%. Satu pelanggaran membatalkan seluruh paket (rollback).
//
// Kriteria tidak dihapus, hanya dinonaktifkan: penilaian dan potret run lama
// menunjuk kriteria itu, dan menghapusnya membuat riwayat tak bisa dijelaskan.
func (s *Service) SimpanKriteria(ctx context.Context, in KriteriaSimpan) error {
	g := galat{}
	if len(in.Ubah)+len(in.Baru) == 0 {
		g.tambah("bobot", "Tidak ada perubahan untuk disimpan")
		return g.err()
	}

	// Salinan yang sudah dirapikan; masukan asli tidak dimodifikasi.
	ubah := make([]KriteriaEdit, len(in.Ubah))
	lihat := make(map[string]bool, len(in.Ubah))
	for i, u := range in.Ubah {
		if lihat[u.ID] {
			g.tambah("kriteria", "Satu kriteria dikirim lebih dari sekali")
		}
		lihat[u.ID] = true
		u.Nama, u.Skala = periksaIsiKriteria(g, u.ID, u.Nama, u.Tipe, u.Bobot, u.Skala)
		ubah[i] = u
	}
	baru := make([]KriteriaBaru, len(in.Baru))
	for i, b := range in.Baru {
		b.Nama, b.Skala = periksaIsiKriteria(g, "baru "+strconv.Itoa(i+1), b.Nama, b.Tipe, b.Bobot, b.Skala)
		baru[i] = b
	}
	if err := g.err(); err != nil {
		return err
	}

	return s.st.WithTx(ctx, func(tx *store.Store) error {
		for _, u := range ubah {
			if err := tx.UpdateKriteriaPenuh(ctx, u.ID, u.Nama, u.Tipe, u.Bobot, u.Skala, u.Aktif); err != nil {
				return fmt.Errorf("service.SimpanKriteria: %w", err)
			}
		}
		for _, b := range baru {
			if _, err := tx.CreateKriteria(ctx, b.Nama, b.Tipe, b.Bobot, b.Skala); err != nil {
				return fmt.Errorf("service.SimpanKriteria: %w", err)
			}
		}

		semua, err := tx.ListKriteria(ctx, false)
		if err != nil {
			return fmt.Errorf("service.SimpanKriteria: %w", err)
		}
		namaDipakai := make(map[string]bool, len(semua))
		aktif, total := 0, 0
		for _, k := range semua {
			kunci := strings.ToLower(k.Nama)
			if namaDipakai[kunci] {
				g.tambah("nama", fmt.Sprintf("Nama kriteria %q dipakai lebih dari satu kriteria", k.Nama))
			}
			namaDipakai[kunci] = true
			if k.Aktif {
				aktif++
				total += persen(k.Bobot)
			}
		}
		if aktif < kriteriaAktifMin || aktif > kriteriaAktifMaks {
			g.tambah("kriteria", fmt.Sprintf("Jumlah kriteria aktif harus %d–%d (sekarang %d)", kriteriaAktifMin, kriteriaAktifMaks, aktif))
		}
		if total != 100 {
			g.tambah("bobot", fmt.Sprintf("Total bobot harus 100%% (sekarang %d%%)", total))
		}
		return g.err() // bila ada galat, WithTx melakukan rollback
	})
}
