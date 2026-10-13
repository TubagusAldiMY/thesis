package service

import (
	"context"
	"fmt"
	"strings"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// maksUang = batas atas kolom DECIMAL(15,2) dalam rupiah utuh (< 10^13).
const maksUang int64 = 9_999_999_999_999

// maksKandidat membatasi ukuran satu permintaan (mencegah query IN raksasa).
const maksKandidat = 500

// PeriodeInput = isian formulir "Buat periode".
type PeriodeInput struct {
	Nama               string
	Jenis              domain.JenisPeriode
	DanaTersedia       int64
	NominalPerPenerima int64
	AsnafDiizinkan     []domain.Asnaf // dipakai untuk zakat; diabaikan untuk infak
}

var semuaStatus = []domain.StatusPeriode{
	domain.StatusDraft, domain.StatusDihitung, domain.StatusDisetujui, domain.StatusDitolak, domain.StatusSelesai,
}

func statusSah(st domain.StatusPeriode) bool {
	for _, s := range semuaStatus {
		if st == s {
			return true
		}
	}
	return false
}

// BuatPeriode memvalidasi lalu membuat periode berstatus draft (FR-06).
// Aturan: nama 3–100 karakter; dana dan nominal > 0; nominal ≤ dana (kuota
// minimal 1); zakat wajib punya ≥ 1 golongan asnaf yang sah, infak tidak
// membatasi golongan (isian asnaf dibuang).
func (s *Service) BuatPeriode(ctx context.Context, userID string, in PeriodeInput) (domain.Periode, error) {
	g := galat{}
	nama := strings.TrimSpace(in.Nama)
	if n := panjangRune(nama); n < 3 || n > 100 {
		g.tambah("nama", "Nama periode minimal 3 karakter (maksimal 100)")
	}
	if in.Jenis != domain.JenisZakat && in.Jenis != domain.JenisInfak {
		g.tambah("jenis", "Jenis dana harus zakat atau infak")
	}
	if in.DanaTersedia <= 0 || in.DanaTersedia > maksUang {
		g.tambah("dana_tersedia", "Dana tersedia harus lebih dari 0 dan tidak melebihi batas")
	}
	if in.NominalPerPenerima <= 0 || in.NominalPerPenerima > maksUang {
		g.tambah("nominal_per_penerima", "Nominal per penerima harus lebih dari 0 dan tidak melebihi batas")
	}
	if in.NominalPerPenerima > in.DanaTersedia && in.DanaTersedia > 0 {
		g.tambah("nominal_per_penerima", "Nominal per penerima tidak boleh melebihi dana tersedia")
	}
	if strings.TrimSpace(userID) == "" {
		g.tambah("pembuat", "Pembuat periode tidak diketahui")
	}

	var asnaf []domain.Asnaf
	if in.Jenis == domain.JenisZakat {
		lihat := map[domain.Asnaf]bool{}
		for _, a := range in.AsnafDiizinkan {
			if !asnafSah(a) {
				g.tambah("asnaf_diizinkan", "Ada golongan asnaf yang tidak dikenal")
				continue
			}
			if !lihat[a] {
				lihat[a] = true
				asnaf = append(asnaf, a)
			}
		}
		if len(asnaf) == 0 {
			g.tambah("asnaf_diizinkan", "Periode zakat harus memilih minimal satu golongan asnaf")
		}
	}
	if err := g.err(); err != nil {
		return domain.Periode{}, err
	}

	p := domain.Periode{
		Nama: nama, Jenis: in.Jenis, DanaTersedia: in.DanaTersedia,
		NominalPerPenerima: in.NominalPerPenerima, AsnafDiizinkan: asnaf, CreatedBy: userID,
	}
	if err := s.st.CreatePeriode(ctx, &p); err != nil {
		return domain.Periode{}, fmt.Errorf("service.BuatPeriode: %w", err)
	}
	return p, nil
}

// DaftarPeriode mengembalikan periode terbaru dulu; status "" = semua.
func (s *Service) DaftarPeriode(ctx context.Context, status domain.StatusPeriode) ([]domain.Periode, error) {
	if status != "" && !statusSah(status) {
		return nil, (galat{"status": "Status periode tidak dikenal"}).err()
	}
	ps, err := s.st.ListPeriode(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarPeriode: %w", err)
	}
	return ps, nil
}

// AmbilPeriode mengembalikan satu periode.
func (s *Service) AmbilPeriode(ctx context.Context, id string) (domain.Periode, error) {
	p, err := s.st.GetPeriode(ctx, id)
	if err != nil {
		return domain.Periode{}, fmt.Errorf("service.AmbilPeriode: %w", err)
	}
	return p, nil
}

// AturKandidat menetapkan daftar kandidat periode (FR-07), atomik.
// Aturan:
//   - periode harus berstatus draft (selain itu → ErrConflict);
//   - semua mustahik harus ada dan aktif;
//   - periode zakat: golongan asnaf mustahik harus termasuk yang diizinkan;
//   - ID dobel dibuang diam-diam (urutan pertama dipertahankan).
//
// Baris periode dikunci (FOR UPDATE) selama transaksi, sehingga status tidak
// bisa berpindah ke "dihitung" tepat di antara pemeriksaan dan penulisan.
func (s *Service) AturKandidat(ctx context.Context, periodeID string, mustahikIDs []string) error {
	ids := buangDobel(mustahikIDs)
	if len(ids) > maksKandidat {
		return (galat{"kandidat": fmt.Sprintf("Kandidat maksimal %d per periode", maksKandidat)}).err()
	}

	return s.st.WithTx(ctx, func(tx *store.Store) error {
		p, err := tx.GetPeriodeForUpdate(ctx, periodeID)
		if err != nil {
			return fmt.Errorf("service.AturKandidat: %w", err)
		}
		if p.Status != domain.StatusDraft {
			return fmt.Errorf("service.AturKandidat: %w", domain.Konflik("Periode berstatus %s; kandidat hanya bisa diubah saat draft", p.Status))
		}

		ditemukan, err := tx.GetMustahikByIDs(ctx, ids)
		if err != nil {
			return fmt.Errorf("service.AturKandidat: %w", err)
		}
		adaDi := make(map[string]domain.Mustahik, len(ditemukan))
		for _, m := range ditemukan {
			adaDi[m.ID] = m
		}

		if pesan := periksaKandidat(p, ids, adaDi); pesan != "" {
			return (galat{"kandidat": pesan}).err()
		}

		if err := tx.ReplaceKandidat(ctx, periodeID, ids); err != nil {
			return fmt.Errorf("service.AturKandidat: %w", err)
		}
		return nil
	})
}

// periksaKandidat memeriksa calon kandidat terhadap aturan periode dan
// mengembalikan pesan masalah (kosong = semua sah): mustahik tidak ditemukan,
// tidak aktif, atau (periode zakat) golongan asnaf tidak diizinkan.
// ada = mustahik yang berhasil ditemukan, dikunci menurut ID.
//
// Dipakai di dua tempat dengan alasan yang sama: AturKandidat (saat memilih)
// dan HitungAras (saat menghitung) — mustahik bisa saja dinonaktifkan atau
// berganti golongan di antara keduanya.
func periksaKandidat(p domain.Periode, ids []string, ada map[string]domain.Mustahik) string {
	diizinkan := map[domain.Asnaf]bool{}
	for _, a := range p.AsnafDiizinkan {
		diizinkan[a] = true
	}
	hilang := 0
	var nonaktif, salahGolongan []string
	for _, id := range ids {
		m, ok := ada[id]
		switch {
		case !ok:
			hilang++
		case !m.Aktif:
			nonaktif = append(nonaktif, m.Nama)
		case p.Jenis == domain.JenisZakat && !diizinkan[m.Asnaf]:
			salahGolongan = append(salahGolongan, fmt.Sprintf("%s (%s)", m.Nama, m.Asnaf))
		}
	}
	var pesan []string
	if hilang > 0 {
		pesan = append(pesan, fmt.Sprintf("%d mustahik tidak ditemukan", hilang))
	}
	if len(nonaktif) > 0 {
		pesan = append(pesan, "mustahik tidak aktif: "+strings.Join(nonaktif, ", "))
	}
	if len(salahGolongan) > 0 {
		pesan = append(pesan, "golongan tidak diizinkan untuk periode ini: "+strings.Join(salahGolongan, ", "))
	}
	return strings.Join(pesan, "; ")
}

// DaftarKandidat mengembalikan kandidat periode (tersamar), urut nama.
func (s *Service) DaftarKandidat(ctx context.Context, periodeID string) ([]MustahikView, error) {
	if _, err := s.st.GetPeriode(ctx, periodeID); err != nil {
		return nil, fmt.Errorf("service.DaftarKandidat: %w", err)
	}
	ms, err := s.st.ListKandidat(ctx, periodeID)
	if err != nil {
		return nil, fmt.Errorf("service.DaftarKandidat: %w", err)
	}
	hasil := make([]MustahikView, 0, len(ms))
	for _, m := range ms {
		v, err := s.view(m)
		if err != nil {
			return nil, fmt.Errorf("service.DaftarKandidat: %w", err)
		}
		hasil = append(hasil, v)
	}
	return hasil, nil
}

// buangDobel membuang ID kembar dan kosong dengan mempertahankan urutan pertama.
func buangDobel(ids []string) []string {
	lihat := make(map[string]bool, len(ids))
	hasil := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || lihat[id] {
			continue
		}
		lihat[id] = true
		hasil = append(hasil, id)
	}
	return hasil
}
