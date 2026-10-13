package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// PeriodeDetail = isi kepala halaman /periode/[id]: periode + kuota, riwayat
// run, dan keputusan Ketua (catatan penolakan terbaru ditampilkan bila ditolak).
type PeriodeDetail struct {
	Periode          domain.Periode       `json:"periode"`
	Kuota            int                  `json:"kuota"`
	Runs             []RunRingkas         `json:"runs"`
	Keputusan        []domain.Persetujuan `json:"keputusan"`
	CatatanPenolakan string               `json:"catatan_penolakan"`
}

// AmbilDetailPeriode menyusun PeriodeDetail.
func (s *Service) AmbilDetailPeriode(ctx context.Context, id string) (PeriodeDetail, error) {
	p, err := s.AmbilPeriode(ctx, id)
	if err != nil {
		return PeriodeDetail{}, err
	}
	runs, err := s.DaftarRun(ctx, id)
	if err != nil {
		return PeriodeDetail{}, err
	}
	kep, err := s.st.ListPersetujuan(ctx, id)
	if err != nil {
		return PeriodeDetail{}, fmt.Errorf("service.AmbilDetailPeriode: %w", err)
	}
	d := PeriodeDetail{Periode: p, Kuota: p.Kuota(), Runs: runs, Keputusan: kep}
	if p.Status == domain.StatusDitolak && len(kep) > 0 && kep[0].Keputusan == domain.KeputusanTolak {
		d.CatatanPenolakan = kep[0].Catatan
	}
	return d, nil
}

// Putuskan mencatat keputusan Ketua DKM atas run terkini (FR-12):
// setuju → status disetujui; tolak (catatan wajib) → status ditolak.
// Hanya dari status dihitung; baris periode dikunci selama proses sehingga
// Admin tidak bisa menghitung ulang di tengah keputusan.
func (s *Service) Putuskan(ctx context.Context, periodeID, ketuaID string, kep domain.Keputusan, catatan string) (domain.Persetujuan, error) {
	catatan = strings.TrimSpace(catatan)
	g := galat{}
	if kep != domain.KeputusanSetuju && kep != domain.KeputusanTolak {
		g.tambah("keputusan", "Keputusan harus setuju atau tolak")
	}
	if kep == domain.KeputusanTolak && panjangRune(catatan) < 3 {
		g.tambah("catatan", "Catatan penolakan wajib diisi (minimal 3 karakter)")
	}
	if panjangRune(catatan) > 1000 {
		g.tambah("catatan", "Catatan maksimal 1000 karakter")
	}
	if err := g.err(); err != nil {
		return domain.Persetujuan{}, err
	}

	var hasil domain.Persetujuan
	err := s.st.WithTx(ctx, func(tx *store.Store) error {
		p, err := tx.GetPeriodeForUpdate(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.Putuskan: %w", err)
		}
		if p.Status != domain.StatusDihitung {
			return fmt.Errorf("service.Putuskan: %w", domain.Konflik("Periode berstatus %s; keputusan hanya bisa diberikan saat dihitung", p.Status))
		}
		run, err := tx.GetRun(ctx, periodeID, 0)
		if err != nil {
			return fmt.Errorf("service.Putuskan: %w", err)
		}
		hasil = domain.Persetujuan{RunID: run.ID, Versi: run.Versi, Keputusan: kep, Catatan: catatan, Oleh: ketuaID}
		if err := tx.CreatePersetujuan(ctx, hasil); err != nil {
			return fmt.Errorf("service.Putuskan: %w", err)
		}
		ke := domain.StatusDisetujui
		if kep == domain.KeputusanTolak {
			ke = domain.StatusDitolak
		}
		return tx.SetStatusPeriode(ctx, periodeID, domain.StatusDihitung, ke)
	})
	if err != nil {
		return domain.Persetujuan{}, err
	}
	return hasil, nil
}

// pindahStatus memindahkan status periode dari → ke dengan pesan konflik yang
// ramah (dipakai Buka kembali dan Tutup periode). cek dijalankan di dalam
// transaksi yang sama, setelah baris periode dikunci.
func (s *Service) pindahStatus(ctx context.Context, periodeID string, dari, ke domain.StatusPeriode, aksi string,
	cek func(tx *store.Store, p domain.Periode) error) error {
	return s.st.WithTx(ctx, func(tx *store.Store) error {
		p, err := tx.GetPeriodeForUpdate(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.%s: %w", aksi, err)
		}
		if p.Status != dari {
			return fmt.Errorf("service.%s: %w", aksi, domain.Konflik("Periode berstatus %s; aksi ini hanya untuk periode %s", p.Status, dari))
		}
		if cek != nil {
			if err := cek(tx, p); err != nil {
				return err
			}
		}
		return tx.SetStatusPeriode(ctx, periodeID, dari, ke)
	})
}

// BukaKembali mengembalikan periode ditolak ke draft (FR-19) supaya kandidat
// dan penilaian bisa diperbaiki lalu dihitung ulang. Run lama tetap tersimpan.
func (s *Service) BukaKembali(ctx context.Context, periodeID string) error {
	return s.pindahStatus(ctx, periodeID, domain.StatusDitolak, domain.StatusDraft, "BukaKembali", nil)
}

// PenyaluranInput = isian tombol "Catat" pada tab Penyaluran.
type PenyaluranInput struct {
	MustahikID string
	Nominal    int64
	Tanggal    string // "YYYY-MM-DD"
}

// CatatPenyaluran mencatat realisasi untuk satu penerima (FR-13). Aturan:
//   - periode berstatus disetujui (dikunci selama proses);
//   - mustahik termasuk USULAN pada run terkini (= run yang disetujui);
//   - nominal > 0, tanggal sah;
//   - satu penerima sekali (ErrConflict);
//   - total tercatat + nominal ≤ dana tersedia (422).
func (s *Service) CatatPenyaluran(ctx context.Context, periodeID, userID string, in PenyaluranInput) (domain.Penyaluran, error) {
	g := galat{}
	if in.Nominal <= 0 || in.Nominal > maksUang {
		g.tambah("nominal", "Nominal harus lebih dari 0")
	}
	tgl, err := time.Parse(time.DateOnly, strings.TrimSpace(in.Tanggal))
	if err != nil {
		g.tambah("tanggal", "Tanggal tidak valid")
	}
	if err := g.err(); err != nil {
		return domain.Penyaluran{}, err
	}

	var hasil domain.Penyaluran
	err = s.st.WithTx(ctx, func(tx *store.Store) error {
		p, err := tx.GetPeriodeForUpdate(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.CatatPenyaluran: %w", err)
		}
		if p.Status != domain.StatusDisetujui {
			return fmt.Errorf("service.CatatPenyaluran: %w", domain.Konflik("Periode berstatus %s; penyaluran hanya bisa dicatat setelah disetujui", p.Status))
		}
		run, err := tx.GetRun(ctx, periodeID, 0)
		if err != nil {
			return fmt.Errorf("service.CatatPenyaluran: %w", err)
		}
		hs, err := tx.ListHasil(ctx, run.ID)
		if err != nil {
			return fmt.Errorf("service.CatatPenyaluran: %w", err)
		}
		usulan := false
		for _, h := range hs {
			if h.MustahikID == in.MustahikID && h.Status == domain.HasilUsulan {
				usulan = true
			}
		}
		if !usulan {
			return (galat{"mustahik_id": "Bukan penerima usulan yang disetujui pada periode ini"}).err()
		}
		sudah, err := tx.ListPenyaluran(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.CatatPenyaluran: %w", err)
		}
		var total int64
		for _, x := range sudah {
			if x.MustahikID == in.MustahikID {
				return fmt.Errorf("service.CatatPenyaluran: %w", domain.Konflik("Penyaluran untuk penerima ini sudah tercatat"))
			}
			total += x.Nominal
		}
		if total+in.Nominal > p.DanaTersedia {
			return (galat{"nominal": fmt.Sprintf("Total penyaluran melebihi dana tersedia (sisa Rp %d)", p.DanaTersedia-total)}).err()
		}
		hasil = domain.Penyaluran{PeriodeID: periodeID, MustahikID: in.MustahikID, Nominal: in.Nominal, Tanggal: tgl, DicatatOleh: userID}
		if err := tx.CreatePenyaluran(ctx, &hasil); err != nil {
			if errors.Is(err, domain.ErrConflict) {
				return fmt.Errorf("service.CatatPenyaluran: %w", domain.Konflik("Penyaluran untuk penerima ini sudah tercatat"))
			}
			return fmt.Errorf("service.CatatPenyaluran: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Penyaluran{}, err
	}
	return hasil, nil
}

// BarisPenyaluran = satu baris tab Penyaluran: penerima (usulan) atau
// cadangan (keterangan saja), beserta realisasinya bila sudah dicatat.
type BarisPenyaluran struct {
	Mustahik   MustahikView       `json:"mustahik"`
	Ranking    int                `json:"ranking"`
	K          float64            `json:"k"`
	Status     domain.StatusHasil `json:"status"`
	Penyaluran *domain.Penyaluran `json:"penyaluran"`
}

// RekapPenyaluran = isi tab Penyaluran + tiga kartu ringkasan.
type RekapPenyaluran struct {
	Periode        domain.Periode    `json:"periode"`
	Baris          []BarisPenyaluran `json:"baris"`
	JumlahPenerima int               `json:"jumlah_penerima"`
	Tercatat       int               `json:"tercatat"`
	TotalTersalur  int64             `json:"total_tersalur"`
	SisaDana       int64             `json:"sisa_dana"`
}

// DaftarPenyaluran menyusun rekap penyaluran berdasarkan run terkini. Periode
// yang belum pernah dihitung menghasilkan rekap kosong (bukan error).
func (s *Service) DaftarPenyaluran(ctx context.Context, periodeID string) (RekapPenyaluran, error) {
	p, err := s.AmbilPeriode(ctx, periodeID)
	if err != nil {
		return RekapPenyaluran{}, err
	}
	rek := RekapPenyaluran{Periode: p, Baris: []BarisPenyaluran{}, SisaDana: p.DanaTersedia}
	hasil, err := s.AmbilHasilAras(ctx, periodeID, 0)
	if errors.Is(err, domain.ErrNotFound) {
		return rek, nil
	} else if err != nil {
		return RekapPenyaluran{}, err
	}
	salur, err := s.st.ListPenyaluran(ctx, periodeID)
	if err != nil {
		return RekapPenyaluran{}, fmt.Errorf("service.DaftarPenyaluran: %w", err)
	}
	per := make(map[string]domain.Penyaluran, len(salur))
	for _, x := range salur {
		per[x.MustahikID] = x
		rek.TotalTersalur += x.Nominal
	}
	for _, h := range hasil.Items {
		b := BarisPenyaluran{Mustahik: h.Mustahik, Ranking: h.Ranking, K: h.K, Status: h.Status}
		if x, ada := per[h.Mustahik.ID]; ada {
			b.Penyaluran = &x
			rek.Tercatat++
		}
		if h.Status == domain.HasilUsulan {
			rek.JumlahPenerima++
		}
		rek.Baris = append(rek.Baris, b)
	}
	rek.SisaDana = p.DanaTersedia - rek.TotalTersalur
	return rek, nil
}

// TutupPeriode menyelesaikan periode disetujui (→ selesai) setelah SEMUA
// penerima usulan tercatat penyalurannya.
func (s *Service) TutupPeriode(ctx context.Context, periodeID string) error {
	return s.pindahStatus(ctx, periodeID, domain.StatusDisetujui, domain.StatusSelesai, "TutupPeriode",
		func(tx *store.Store, _ domain.Periode) error {
			run, err := tx.GetRun(ctx, periodeID, 0)
			if err != nil {
				return fmt.Errorf("service.TutupPeriode: %w", err)
			}
			hs, err := tx.ListHasil(ctx, run.ID)
			if err != nil {
				return fmt.Errorf("service.TutupPeriode: %w", err)
			}
			salur, err := tx.ListPenyaluran(ctx, periodeID)
			if err != nil {
				return fmt.Errorf("service.TutupPeriode: %w", err)
			}
			sudah := make(map[string]bool, len(salur))
			for _, x := range salur {
				sudah[x.MustahikID] = true
			}
			belum := 0
			for _, h := range hs {
				if h.Status == domain.HasilUsulan && !sudah[h.MustahikID] {
					belum++
				}
			}
			if belum > 0 {
				return (galat{"penyaluran": fmt.Sprintf("Masih ada %d penerima yang belum dicatat penyalurannya", belum)}).err()
			}
			return nil
		})
}
