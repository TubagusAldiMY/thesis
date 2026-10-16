package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// Batas nilai untuk kriteria TANPA skala (mis. penghasilan, jumlah tanggungan).
// Minimal 1 karena paket aras tidak menerima nilai ≤ 0 (cost dibagi 0); UI
// sudah merancang "isi 1 bila tidak berpenghasilan". Kriteria BERSKALA (yang
// Admin beri skala di halaman Kriteria) hanya menerima kunci skalanya.
const (
	nilaiMin  = 1.0
	nilaiMaks = 9_999_999_999.0
)

// PenilaianInput = nilai satu kandidat: kunci peta = KODE kriteria ("C1").
type PenilaianInput struct {
	MustahikID string
	Nilai      map[string]float64
}

// BarisPenilaian = satu baris grid penilaian. Nilai hanya berisi sel yang
// sudah terisi.
type BarisPenilaian struct {
	Mustahik MustahikView       `json:"mustahik"`
	Nilai    map[string]float64 `json:"nilai"`
	Lengkap  bool               `json:"lengkap"`
}

// PenilaianMatriks = grid kandidat × kriteria aktif beserta progresnya
// (untuk "Lengkap 4/5 kandidat" dan tombol Hitung ARAS yang nonaktif).
type PenilaianMatriks struct {
	Kriteria []domain.Kriteria `json:"kriteria"`
	Baris    []BarisPenilaian  `json:"baris"`
	Lengkap  int               `json:"lengkap"` // jumlah kandidat yang semua selnya terisi
	Total    int               `json:"total"`   // jumlah kandidat
}

// pesanNilai memeriksa satu nilai terhadap kriterianya; "" = sah.
func pesanNilai(k domain.Kriteria, v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Sprintf("Nilai %s tidak valid", k.Kode)
	}
	if k.Skala != nil { // kriteria berskala: hanya kunci skala yang sah
		if v == math.Trunc(v) {
			if _, ok := k.Skala[strconv.Itoa(int(v))]; ok {
				return ""
			}
		}
		kunci := make([]string, 0, len(k.Skala))
		for kk := range k.Skala {
			kunci = append(kunci, kk)
		}
		sort.Strings(kunci)
		return fmt.Sprintf("Nilai %s harus salah satu dari: %s", k.Kode, strings.Join(kunci, ", "))
	}
	if v < nilaiMin || v > nilaiMaks {
		return fmt.Sprintf("Nilai %s harus antara 1 dan %.0f (isi 1 bila tidak ada)", k.Kode, nilaiMaks)
	}
	return ""
}

// SimpanPenilaian menyimpan nilai-nilai (FR-08), atomik: satu sel salah
// membatalkan seluruh paket. Boleh sebagian — sel yang tidak disebut dibiarkan,
// jadi formulir yang belum lengkap bisa disimpan. Aturan:
//   - periode harus draft (selain itu ErrConflict) dan dikunci selama proses;
//   - mustahik harus kandidat periode ini;
//   - kode kriteria harus kriteria AKTIF;
//   - nilai sesuai kriteria (lihat pesanNilai);
//   - satu mustahik tidak boleh muncul dua kali dalam satu permintaan.
//
// Pesan error per sel memakai nama field "nilai.<mustahikID>.<kode>" supaya UI
// bisa menandai selnya.
func (s *Service) SimpanPenilaian(ctx context.Context, periodeID string, masuk []PenilaianInput) error {
	if len(masuk) == 0 {
		return (galat{"penilaian": "Tidak ada nilai untuk disimpan"}).err()
	}

	return s.st.WithTx(ctx, func(tx *store.Store) error {
		p, err := tx.GetPeriodeForUpdate(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.SimpanPenilaian: %w", err)
		}
		if p.Status != domain.StatusDraft {
			return fmt.Errorf("service.SimpanPenilaian: %w", domain.Konflik("Periode berstatus %s; penilaian hanya bisa diubah saat draft", p.Status))
		}

		ks, err := tx.ListKriteria(ctx, true)
		if err != nil {
			return fmt.Errorf("service.SimpanPenilaian: %w", err)
		}
		kriteria := make(map[string]domain.Kriteria, len(ks))
		for _, k := range ks {
			kriteria[k.Kode] = k
		}
		kandidat, err := tx.ListKandidat(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.SimpanPenilaian: %w", err)
		}
		adalahKandidat := make(map[string]bool, len(kandidat))
		for _, m := range kandidat {
			adalahKandidat[m.ID] = true
		}

		g := galat{}
		lihat := map[string]bool{}
		var simpan []domain.Penilaian
		for _, in := range masuk {
			if lihat[in.MustahikID] {
				g.tambah("penilaian", "Satu mustahik dikirim lebih dari sekali")
				continue
			}
			lihat[in.MustahikID] = true
			if !adalahKandidat[in.MustahikID] {
				g.tambah("mustahik."+in.MustahikID, "Bukan kandidat periode ini")
				continue
			}
			kode := make([]string, 0, len(in.Nilai))
			for k := range in.Nilai {
				kode = append(kode, k)
			}
			sort.Strings(kode) // urutan tetap → pesan dan penyimpanan deterministik
			for _, kd := range kode {
				field := "nilai." + in.MustahikID + "." + kd
				k, ada := kriteria[kd]
				if !ada {
					g.tambah(field, fmt.Sprintf("Kriteria %q tidak dikenal atau tidak aktif", kd))
					continue
				}
				if pesan := pesanNilai(k, in.Nilai[kd]); pesan != "" {
					g.tambah(field, pesan)
					continue
				}
				simpan = append(simpan, domain.Penilaian{MustahikID: in.MustahikID, KriteriaID: k.ID, Nilai: in.Nilai[kd]})
			}
		}
		if err := g.err(); err != nil {
			return err // transaksi dibatalkan: tidak ada sel yang tersimpan
		}
		if err := tx.UpsertPenilaian(ctx, periodeID, simpan); err != nil {
			return fmt.Errorf("service.SimpanPenilaian: %w", err)
		}
		return nil
	})
}

// AmbilPenilaian mengembalikan grid penilaian periode: kandidat (tersamar) ×
// kriteria aktif, dengan progres kelengkapan.
func (s *Service) AmbilPenilaian(ctx context.Context, periodeID string) (PenilaianMatriks, error) {
	if _, err := s.st.GetPeriode(ctx, periodeID); err != nil {
		return PenilaianMatriks{}, fmt.Errorf("service.AmbilPenilaian: %w", err)
	}
	ks, err := s.st.ListKriteria(ctx, true)
	if err != nil {
		return PenilaianMatriks{}, fmt.Errorf("service.AmbilPenilaian: %w", err)
	}
	kandidat, err := s.st.ListKandidat(ctx, periodeID)
	if err != nil {
		return PenilaianMatriks{}, fmt.Errorf("service.AmbilPenilaian: %w", err)
	}
	pen, err := s.st.ListPenilaian(ctx, periodeID)
	if err != nil {
		return PenilaianMatriks{}, fmt.Errorf("service.AmbilPenilaian: %w", err)
	}

	kodeDariID := make(map[string]string, len(ks))
	for _, k := range ks {
		kodeDariID[k.ID] = k.Kode
	}
	nilai := map[string]map[string]float64{}
	for _, p := range pen {
		kode, aktif := kodeDariID[p.KriteriaID]
		if !aktif {
			continue // nilai untuk kriteria yang kini nonaktif tidak ditampilkan
		}
		if nilai[p.MustahikID] == nil {
			nilai[p.MustahikID] = map[string]float64{}
		}
		nilai[p.MustahikID][kode] = p.Nilai
	}

	hasil := PenilaianMatriks{Kriteria: ks, Total: len(kandidat), Baris: make([]BarisPenilaian, 0, len(kandidat))}
	for _, m := range kandidat {
		v, err := s.view(m)
		if err != nil {
			return PenilaianMatriks{}, fmt.Errorf("service.AmbilPenilaian: %w", err)
		}
		n := nilai[m.ID]
		if n == nil {
			n = map[string]float64{}
		}
		lengkap := len(ks) > 0 && len(n) == len(ks)
		if lengkap {
			hasil.Lengkap++
		}
		hasil.Baris = append(hasil.Baris, BarisPenilaian{Mustahik: v, Nilai: n, Lengkap: lengkap})
	}
	return hasil, nil
}
