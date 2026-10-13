package service

import (
	"context"
	"fmt"
	"math"

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
