package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"spkzakat/internal/aras"
	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// LangkahTersimpan = langkah hitung sebuah run, disimpan sebagai JSON di
// aras_run.langkah agar "Lihat langkah hitung" tetap bisa menampilkan run
// lama walau penilaian atau bobot berubah sesudahnya. Setiap matriks
// berukuran (m+1)×n: baris 0 = alternatif ideal A0, baris 1..m = Alternatif
// menurut urutan di bawah; kolom = Kriteria.
type LangkahTersimpan struct {
	Kriteria    []string    `json:"kriteria"`   // kode kriteria, urutan kolom
	Alternatif  []string    `json:"alternatif"` // ID mustahik, urutan baris 1..m
	A0          []float64   `json:"a0"`
	Matriks     [][]float64 `json:"matriks"`
	Normalisasi [][]float64 `json:"normalisasi"`
	Terbobot    [][]float64 `json:"terbobot"`
}

// HasilArasItem = satu baris tabel ranking.
type HasilArasItem struct {
	Mustahik MustahikView       `json:"mustahik"`
	S        float64            `json:"s"`
	K        float64            `json:"k"`
	Ranking  int                `json:"ranking"`
	Status   domain.StatusHasil `json:"status"`
}

// HasilAras = satu run lengkap untuk ditampilkan: tabel ranking menurut
// urutan, kuota, dan langkah hitung.
type HasilAras struct {
	Periode domain.Periode    `json:"periode"`
	Run     domain.ArasRun    `json:"run"` // Run.Langkah sengaja kosong; pakai field Langkah di bawah
	Kuota   int               `json:"kuota"`
	Items   []HasilArasItem   `json:"items"`
	Langkah *LangkahTersimpan `json:"langkah"` // nil untuk run yang dibuat sebelum V003
}

// RunRingkas = satu baris "Riwayat perhitungan".
type RunRingkas struct {
	Versi          int       `json:"versi"`
	IsCurrent      bool      `json:"is_current"`
	DibuatPada     time.Time `json:"dibuat_pada"`
	DijalankanOleh string    `json:"dijalankan_oleh"`
	JumlahKandidat int       `json:"jumlah_kandidat"`
	Teratas        string    `json:"teratas"` // nama kandidat peringkat 1
	TeratasK       float64   `json:"teratas_k"`
}

// HitungAras menjalankan ARAS untuk periode (FR-09) dan menyimpan hasilnya
// sebagai run baru. Boleh dari status draft (→ dihitung) maupun dihitung
// (hitung ulang → run baru, versi naik, run lama tetap tersimpan sebagai riwayat).
//
// Syarat (semuanya diperiksa di dalam satu transaksi, baris periode dikunci):
//   - total bobot kriteria AKTIF tepat 100%;
//   - ada kandidat, semuanya masih aktif dan (zakat) golongannya masih diizinkan;
//   - penilaian lengkap: setiap kandidat × setiap kriteria aktif terisi.
//
// Kandidat peringkat 1..kuota berstatus usulan, sisanya cadangan (kuota =
// ⌊dana ÷ nominal⌋). Bila langkah mana pun gagal, tidak ada yang berubah:
// tidak ada run tersimpan dan status periode tetap.
func (s *Service) HitungAras(ctx context.Context, periodeID, userID string) (HasilAras, error) {
	if strings.TrimSpace(userID) == "" {
		return HasilAras{}, (galat{"pengguna": "Pengguna yang menjalankan perhitungan tidak diketahui"}).err()
	}

	var out HasilAras
	err := s.st.WithTx(ctx, func(tx *store.Store) error {
		p, err := tx.GetPeriodeForUpdate(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.HitungAras: %w", err)
		}
		if p.Status != domain.StatusDraft && p.Status != domain.StatusDihitung {
			return fmt.Errorf("service.HitungAras: %w", domain.Konflik("Periode berstatus %s; perhitungan hanya bisa dari draft atau dihitung", p.Status))
		}

		ks, err := tx.ListKriteria(ctx, true)
		if err != nil {
			return fmt.Errorf("service.HitungAras: %w", err)
		}
		g := galat{}
		total := 0
		for _, k := range ks {
			total += persen(k.Bobot)
		}
		if len(ks) == 0 || total != 100 {
			g.tambah("bobot", fmt.Sprintf("Total bobot kriteria aktif harus 100%% (sekarang %d%%)", total))
		}

		kandidat, err := tx.ListKandidat(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.HitungAras: %w", err)
		}
		if len(kandidat) == 0 {
			g.tambah("kandidat", "Belum ada kandidat pada periode ini")
		} else {
			ids := make([]string, len(kandidat))
			ada := make(map[string]domain.Mustahik, len(kandidat))
			for i, m := range kandidat {
				ids[i], ada[m.ID] = m.ID, m
			}
			if pesan := periksaKandidat(p, ids, ada); pesan != "" {
				g.tambah("kandidat", pesan)
			}
		}
		if err := g.err(); err != nil {
			return err
		}

		pen, err := tx.ListPenilaian(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.HitungAras: %w", err)
		}
		nilai := map[string]map[string]float64{} // mustahikID → kriteriaID → nilai
		for _, x := range pen {
			if nilai[x.MustahikID] == nil {
				nilai[x.MustahikID] = map[string]float64{}
			}
			nilai[x.MustahikID][x.KriteriaID] = x.Nilai
		}

		kriteria := make([]aras.Kriteria, len(ks))
		kodeKolom := make([]string, len(ks))
		bobot := make([]domain.BobotKriteria, len(ks))
		for j, k := range ks {
			kriteria[j] = aras.Kriteria{Kode: k.Kode, Tipe: aras.Tipe(k.Tipe), Bobot: k.Bobot}
			kodeKolom[j] = k.Kode
			bobot[j] = domain.BobotKriteria{Kode: k.Kode, Tipe: k.Tipe, Bobot: k.Bobot}
		}
		alternatif := make([]aras.Alternatif, len(kandidat))
		idBaris := make([]string, len(kandidat))
		kosong, belumLengkap := 0, []string{}
		for i, m := range kandidat {
			baris := make([]float64, len(ks))
			kurang := 0
			for j, k := range ks {
				v, ada := nilai[m.ID][k.ID]
				if !ada {
					kurang++
					continue
				}
				baris[j] = v
			}
			if kurang > 0 {
				kosong += kurang
				if len(belumLengkap) < 5 { // cukup untuk petunjuk; daftar penuh ada di grid penilaian
					belumLengkap = append(belumLengkap, m.Nama)
				}
			}
			alternatif[i] = aras.Alternatif{ID: m.ID, Nilai: baris}
			idBaris[i] = m.ID
		}
		if kosong > 0 {
			g.tambah("penilaian", fmt.Sprintf("Penilaian belum lengkap: %d dari %d sel masih kosong (mis. %s)",
				kosong, len(kandidat)*len(ks), strings.Join(belumLengkap, ", ")))
			return g.err()
		}

		hasil, err := aras.Hitung(kriteria, alternatif)
		if err != nil {
			if errors.Is(err, aras.ErrInvalid) { // data tersimpan tidak layak hitung (mestinya sudah ditolak saat disimpan)
				g.tambah("penilaian", err.Error())
				return g.err()
			}
			return fmt.Errorf("service.HitungAras: %w", err)
		}

		kuota := p.Kuota()
		baris := make([]domain.ArasHasil, len(hasil.Alternatif))
		for i, h := range hasil.Alternatif {
			status := domain.HasilCadangan
			if h.Ranking <= kuota {
				status = domain.HasilUsulan
			}
			baris[i] = domain.ArasHasil{MustahikID: h.ID, S: h.S, K: h.K, Ranking: h.Ranking, Status: status}
		}

		langkah := LangkahTersimpan{
			Kriteria: kodeKolom, Alternatif: idBaris, A0: hasil.Langkah.A0, Matriks: hasil.Langkah.Matriks,
			Normalisasi: hasil.Langkah.Normalisasi, Terbobot: hasil.Langkah.Terbobot,
		}
		mentah, err := json.Marshal(langkah)
		if err != nil {
			return fmt.Errorf("service.HitungAras: langkah: %w", err)
		}
		run := domain.ArasRun{PeriodeID: periodeID, Bobot: bobot, S0: hasil.S0, Langkah: mentah, DijalankanOleh: userID}
		if err := tx.SimpanRun(ctx, &run, baris); err != nil {
			return fmt.Errorf("service.HitungAras: %w", err)
		}
		if p.Status == domain.StatusDraft {
			if err := tx.SetStatusPeriode(ctx, periodeID, domain.StatusDraft, domain.StatusDihitung); err != nil {
				return fmt.Errorf("service.HitungAras: %w", err)
			}
			p.Status = domain.StatusDihitung
		}

		mustahik := make(map[string]domain.Mustahik, len(kandidat))
		for _, m := range kandidat {
			mustahik[m.ID] = m
		}
		items := make([]HasilArasItem, len(baris))
		for i, h := range hasil.Alternatif {
			v, err := s.view(mustahik[h.ID])
			if err != nil {
				return fmt.Errorf("service.HitungAras: %w", err)
			}
			items[i] = HasilArasItem{Mustahik: v, S: h.S, K: h.K, Ranking: h.Ranking, Status: baris[i].Status}
		}
		run.Langkah = nil
		out = HasilAras{Periode: p, Run: run, Kuota: kuota, Items: items, Langkah: &langkah}
		return nil
	})
	if err != nil {
		return HasilAras{}, err
	}
	return out, nil
}

// AmbilHasilAras membaca satu run yang sudah tersimpan. versi ≤ 0 = run
// terkini. ErrNotFound bila periode belum pernah dihitung atau versi tidak ada.
func (s *Service) AmbilHasilAras(ctx context.Context, periodeID string, versi int) (HasilAras, error) {
	p, err := s.st.GetPeriode(ctx, periodeID)
	if err != nil {
		return HasilAras{}, fmt.Errorf("service.AmbilHasilAras: %w", err)
	}
	run, err := s.st.GetRun(ctx, periodeID, versi)
	if err != nil {
		return HasilAras{}, fmt.Errorf("service.AmbilHasilAras: %w", err)
	}
	hasil, err := s.st.ListHasil(ctx, run.ID)
	if err != nil {
		return HasilAras{}, fmt.Errorf("service.AmbilHasilAras: %w", err)
	}

	ids := make([]string, len(hasil))
	for i, h := range hasil {
		ids[i] = h.MustahikID
	}
	ms, err := s.st.GetMustahikByIDs(ctx, ids)
	if err != nil {
		return HasilAras{}, fmt.Errorf("service.AmbilHasilAras: %w", err)
	}
	mustahik := make(map[string]domain.Mustahik, len(ms))
	for _, m := range ms {
		mustahik[m.ID] = m
	}

	items := make([]HasilArasItem, len(hasil))
	for i, h := range hasil {
		m, ada := mustahik[h.MustahikID]
		if !ada {
			return HasilAras{}, fmt.Errorf("service.AmbilHasilAras: hasil run v%d merujuk mustahik %s yang tidak ada", run.Versi, h.MustahikID)
		}
		v, err := s.view(m)
		if err != nil {
			return HasilAras{}, fmt.Errorf("service.AmbilHasilAras: %w", err)
		}
		items[i] = HasilArasItem{Mustahik: v, S: h.S, K: h.K, Ranking: h.Ranking, Status: h.Status}
	}

	var langkah *LangkahTersimpan
	if len(run.Langkah) > 0 {
		langkah = &LangkahTersimpan{}
		if err := json.Unmarshal(run.Langkah, langkah); err != nil {
			return HasilAras{}, fmt.Errorf("service.AmbilHasilAras: langkah run v%d rusak: %w", run.Versi, err)
		}
	}
	run.Langkah = nil
	return HasilAras{Periode: p, Run: run, Kuota: p.Kuota(), Items: items, Langkah: langkah}, nil
}

// DaftarRun mengembalikan riwayat perhitungan periode, terbaru dulu:
// "Run v1 · 5 kandidat · teratas A2 (K 0,8939)".
func (s *Service) DaftarRun(ctx context.Context, periodeID string) ([]RunRingkas, error) {
	if _, err := s.st.GetPeriode(ctx, periodeID); err != nil {
		return nil, fmt.Errorf("service.DaftarRun: %w", err)
	}
	runs, err := s.st.ListRun(ctx, periodeID)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarRun: %w", err)
	}

	hasilRun := make([][]domain.ArasHasil, len(runs))
	var idTeratas []string
	for i, r := range runs {
		hs, err := s.st.ListHasil(ctx, r.ID)
		if err != nil {
			return nil, fmt.Errorf("service.DaftarRun: %w", err)
		}
		hasilRun[i] = hs
		if len(hs) > 0 {
			idTeratas = append(idTeratas, hs[0].MustahikID)
		}
	}
	ms, err := s.st.GetMustahikByIDs(ctx, idTeratas)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarRun: %w", err)
	}
	nama := make(map[string]string, len(ms))
	for _, m := range ms {
		nama[m.ID] = m.Nama
	}

	ringkas := make([]RunRingkas, len(runs))
	for i, r := range runs {
		rr := RunRingkas{
			Versi: r.Versi, IsCurrent: r.IsCurrent, DibuatPada: r.CreatedAt,
			DijalankanOleh: r.DijalankanOleh, JumlahKandidat: len(hasilRun[i]),
		}
		if len(hasilRun[i]) > 0 {
			rr.Teratas, rr.TeratasK = nama[hasilRun[i][0].MustahikID], hasilRun[i][0].K
		}
		ringkas[i] = rr
	}
	return ringkas, nil
}
