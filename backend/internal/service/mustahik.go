package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

var (
	reNIK = regexp.MustCompile(`^[0-9]{16}$`)
	reHP  = regexp.MustCompile(`^\+?[0-9]{8,15}$`)
)

// MustahikInput = isian formulir mustahik. NIK dan NoHP berupa PLAINTEXT di
// sini — satu-satunya tempat plaintext hidup; begitu masuk service langsung
// dienkripsi dan tidak pernah dikembalikan lagi.
//
// Saat mengubah, NIK/NoHP kosong berarti "biarkan seperti semula" (UI tidak
// pernah menampilkan nilai aslinya, jadi kolom kosong ≠ ingin menghapus).
type MustahikInput struct {
	Nama   string
	NIK    string
	NoHP   string
	Alamat string
	Asnaf  domain.Asnaf
	Aktif  bool
}

// MustahikView = bentuk yang aman ditampilkan: NIK/No. HP hanya tersamar.
type MustahikView struct {
	ID           string       `json:"id"`
	Nama         string       `json:"nama"`
	NIKTersamar  string       `json:"nik_tersamar"`   // "•••• •••• •••• 8901"
	NoHPTersamar string       `json:"no_hp_tersamar"` // "•••• 5678"; kosong bila tidak ada
	Alamat       string       `json:"alamat"`
	Asnaf        domain.Asnaf `json:"asnaf"`
	Aktif        bool         `json:"aktif"`
	DibuatPada   time.Time    `json:"dibuat_pada"`
}

// MustahikQuery = pencarian/penyaringan daftar mustahik.
type MustahikQuery struct {
	Q       string
	Asnaf   domain.Asnaf
	Aktif   *bool
	Page    int
	PerPage int
}

// MustahikHalaman = satu halaman daftar + total + jumlah mustahik aktif per golongan.
type MustahikHalaman struct {
	Items       []MustahikView       `json:"items"`
	Total       int                  `json:"total"`
	AsnafCounts map[domain.Asnaf]int `json:"asnaf_counts"`
}

func normalNIK(s string) string { return strings.ReplaceAll(strings.TrimSpace(s), " ", "") }

func normalHP(s string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(s))
}

func panjangRune(s string) int { return utf8.RuneCountInString(s) }

// validasiMustahik memeriksa isian; nik/hp sudah dinormalkan. wajibNIK=true
// saat membuat baru.
func validasiMustahik(in MustahikInput, nik, hp string, wajibNIK bool) galat {
	g := galat{}
	if n := panjangRune(strings.TrimSpace(in.Nama)); n < 2 || n > 100 {
		g.tambah("nama", "Nama harus 2–100 karakter")
	}
	if (wajibNIK || nik != "") && !reNIK.MatchString(nik) {
		g.tambah("nik", "NIK harus 16 digit angka")
	}
	if hp != "" && !reHP.MatchString(hp) {
		g.tambah("no_hp", "No. HP tidak valid (8–15 digit, boleh diawali +)")
	}
	if n := panjangRune(strings.TrimSpace(in.Alamat)); n < 5 || n > 500 {
		g.tambah("alamat", "Alamat harus 5–500 karakter")
	}
	if !asnafSah(in.Asnaf) {
		g.tambah("asnaf", "Golongan asnaf tidak dikenal")
	}
	return g
}

// samarkan menampilkan hanya empat karakter terakhir: "•••• •••• •••• 8901".
func samarkan(nilai string, kelompok int) string {
	r := []rune(nilai)
	akhir := r
	if len(r) > 4 {
		akhir = r[len(r)-4:]
	}
	return strings.Repeat("•••• ", kelompok) + string(akhir)
}

// view membuat tampilan aman dari mustahik; mendekripsi hanya untuk
// mengambil 4 karakter terakhir. Gagal dekripsi = kunci salah / data rusak
// dan dikembalikan sebagai error (bukan disembunyikan), tanpa memuat isi data.
func (s *Service) view(m domain.Mustahik) (MustahikView, error) {
	nik, err := s.enc.Decrypt(m.NIKEnc, []byte(aadNIK))
	if err != nil {
		return MustahikView{}, fmt.Errorf("service: NIK mustahik %s tidak terbaca (kunci enkripsi salah atau data rusak): %w", m.ID, err)
	}
	v := MustahikView{
		ID: m.ID, Nama: m.Nama, NIKTersamar: samarkan(string(nik), 3),
		Alamat: m.Alamat, Asnaf: m.Asnaf, Aktif: m.Aktif, DibuatPada: m.CreatedAt,
	}
	if len(m.NoHPEnc) > 0 {
		hp, err := s.enc.Decrypt(m.NoHPEnc, []byte(aadNoHP))
		if err != nil {
			return MustahikView{}, fmt.Errorf("service: No. HP mustahik %s tidak terbaca: %w", m.ID, err)
		}
		v.NoHPTersamar = samarkan(string(hp), 1)
	}
	return v, nil
}

// BuatMustahik memvalidasi, memeriksa duplikat NIK lewat HMAC, mengenkripsi
// NIK/No. HP, lalu menyimpan.
func (s *Service) BuatMustahik(ctx context.Context, in MustahikInput) (MustahikView, error) {
	nik, hp := normalNIK(in.NIK), normalHP(in.NoHP)
	g := validasiMustahik(in, nik, hp, true)
	if err := g.err(); err != nil {
		return MustahikView{}, err
	}

	hash := s.idx.Sum(nik)
	if _, err := s.st.FindMustahikByNIKHash(ctx, hash); err == nil {
		g.tambah("nik", "NIK sudah terdaftar")
		return MustahikView{}, g.err()
	} else if !errors.Is(err, domain.ErrNotFound) {
		return MustahikView{}, fmt.Errorf("service.BuatMustahik: %w", err)
	}

	nikEnc, err := s.enc.Encrypt([]byte(nik), []byte(aadNIK))
	if err != nil {
		return MustahikView{}, fmt.Errorf("service.BuatMustahik: %w", err)
	}
	m := domain.Mustahik{
		Nama: strings.TrimSpace(in.Nama), NIKEnc: nikEnc, NIKHash: hash,
		Alamat: strings.TrimSpace(in.Alamat), Asnaf: in.Asnaf, Aktif: in.Aktif,
	}
	if hp != "" {
		if m.NoHPEnc, err = s.enc.Encrypt([]byte(hp), []byte(aadNoHP)); err != nil {
			return MustahikView{}, fmt.Errorf("service.BuatMustahik: %w", err)
		}
	}
	if err := s.st.CreateMustahik(ctx, &m); err != nil {
		if errors.Is(err, domain.ErrConflict) { // dua permintaan serentak dengan NIK sama
			g.tambah("nik", "NIK sudah terdaftar")
			return MustahikView{}, g.err()
		}
		return MustahikView{}, fmt.Errorf("service.BuatMustahik: %w", err)
	}
	return s.view(m)
}

// UbahMustahik memperbarui mustahik. Aturan khusus:
//   - NIK/No. HP kosong = tidak diubah;
//   - NIK baru tidak boleh milik mustahik lain;
//   - menonaktifkan ditolak (ErrConflict) bila mustahik sedang menjadi
//     kandidat periode berstatus dihitung/disetujui.
func (s *Service) UbahMustahik(ctx context.Context, id string, in MustahikInput) (MustahikView, error) {
	nik, hp := normalNIK(in.NIK), normalHP(in.NoHP)
	g := validasiMustahik(in, nik, hp, false)
	if err := g.err(); err != nil {
		return MustahikView{}, err
	}

	var hasil domain.Mustahik
	err := s.st.WithTx(ctx, func(tx *store.Store) error {
		m, err := tx.GetMustahik(ctx, id)
		if err != nil {
			return fmt.Errorf("service.UbahMustahik: %w", err)
		}
		m.Nama, m.Alamat, m.Asnaf = strings.TrimSpace(in.Nama), strings.TrimSpace(in.Alamat), in.Asnaf

		if nik != "" {
			hash := s.idx.Sum(nik)
			if hash != m.NIKHash {
				lain, err := tx.FindMustahikByNIKHash(ctx, hash)
				switch {
				case err == nil && lain.ID != id:
					g.tambah("nik", "NIK sudah terdaftar")
					return g.err()
				case err != nil && !errors.Is(err, domain.ErrNotFound):
					return fmt.Errorf("service.UbahMustahik: %w", err)
				}
				m.NIKHash = hash
			}
			if m.NIKEnc, err = s.enc.Encrypt([]byte(nik), []byte(aadNIK)); err != nil {
				return fmt.Errorf("service.UbahMustahik: %w", err)
			}
		}
		if hp != "" {
			if m.NoHPEnc, err = s.enc.Encrypt([]byte(hp), []byte(aadNoHP)); err != nil {
				return fmt.Errorf("service.UbahMustahik: %w", err)
			}
		}
		if m.Aktif && !in.Aktif {
			dipakai, err := tx.MustahikDalamPeriodeBerjalan(ctx, id)
			if err != nil {
				return fmt.Errorf("service.UbahMustahik: %w", err)
			}
			if dipakai {
				return fmt.Errorf("service.UbahMustahik: %w", domain.Konflik("Mustahik sedang menjadi kandidat periode yang berjalan (dihitung/disetujui), tidak bisa dinonaktifkan"))
			}
		}
		m.Aktif = in.Aktif

		if err := tx.UpdateMustahik(ctx, m); err != nil {
			if errors.Is(err, domain.ErrConflict) { // balapan: NIK baru dipakai orang lain
				g.tambah("nik", "NIK sudah terdaftar")
				return g.err()
			}
			return fmt.Errorf("service.UbahMustahik: %w", err)
		}
		hasil = m
		return nil
	})
	if err != nil {
		return MustahikView{}, err
	}
	return s.view(hasil)
}

// AmbilMustahik mengembalikan satu mustahik (tersamar).
func (s *Service) AmbilMustahik(ctx context.Context, id string) (MustahikView, error) {
	m, err := s.st.GetMustahik(ctx, id)
	if err != nil {
		return MustahikView{}, fmt.Errorf("service.AmbilMustahik: %w", err)
	}
	return s.view(m)
}

// DaftarMustahik mengembalikan satu halaman mustahik (tersamar).
func (s *Service) DaftarMustahik(ctx context.Context, q MustahikQuery) (MustahikHalaman, error) {
	g := galat{}
	q.Q = strings.TrimSpace(q.Q)
	if panjangRune(q.Q) > 100 {
		g.tambah("q", "Kata kunci maksimal 100 karakter")
	}
	if q.Asnaf != "" && !asnafSah(q.Asnaf) {
		g.tambah("asnaf", "Golongan asnaf tidak dikenal")
	}
	if err := g.err(); err != nil {
		return MustahikHalaman{}, err
	}

	hal, err := s.st.ListMustahik(ctx, store.MustahikFilter{
		Q: q.Q, Asnaf: q.Asnaf, Aktif: q.Aktif, Page: q.Page, PerPage: q.PerPage,
	})
	if err != nil {
		return MustahikHalaman{}, fmt.Errorf("service.DaftarMustahik: %w", err)
	}
	hasil := MustahikHalaman{Total: hal.Total, AsnafCounts: hal.AsnafCounts, Items: make([]MustahikView, 0, len(hal.Items))}
	for _, m := range hal.Items {
		v, err := s.view(m)
		if err != nil {
			return MustahikHalaman{}, fmt.Errorf("service.DaftarMustahik: %w", err)
		}
		hasil.Items = append(hasil.Items, v)
	}
	return hasil, nil
}
