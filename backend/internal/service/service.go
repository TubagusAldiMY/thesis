// Package service berisi aturan bisnis: validasi input, enkripsi data pribadi,
// dan pengaturan transaksi. Ia memanggil store (data) dan crypto (keamanan),
// tetapi tidak tahu apa pun tentang HTTP.
//
// Konvensi error: kesalahan input pengguna dikembalikan sebagai
// *domain.ValidationError (errors.Is(err, domain.ErrValidation)); data yang
// tidak ada → domain.ErrNotFound; bentrok status/duplikat → domain.ErrConflict.
// Lapisan HTTP menerjemahkannya ke 422 / 404 / 409.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"spkzakat/internal/crypto"
	"spkzakat/internal/domain"
	"spkzakat/internal/store"
)

// AAD (konteks terautentikasi) tiap kolom terenkripsi. Dengan AAD berbeda,
// ciphertext NIK yang dipindah ke kolom No. HP akan ditolak saat didekripsi.
const (
	aadNIK  = "mustahik.nik"
	aadNoHP = "mustahik.no_hp"
)

// Service menyatukan semua use case.
type Service struct {
	st  *store.Store
	enc *crypto.Cipher
	idx *crypto.Hasher
	// batas = penghitung login/ganti password gagal (lihat auth.go).
	batas *pembatas
}

// New membuat Service. enc mengenkripsi NIK/No. HP; idx menghitung HMAC NIK
// untuk deteksi duplikat. Keduanya HARUS memakai kunci berbeda.
func New(st *store.Store, enc *crypto.Cipher, idx *crypto.Hasher) *Service {
	return &Service{st: st, enc: enc, idx: idx, batas: pembatasBaru()}
}

// galat mengumpulkan pesan validasi per field; pesan PERTAMA tiap field yang
// dipertahankan (yang paling mendasar biasanya dicek lebih dulu).
type galat map[string]string

func (g galat) tambah(field, pesan string) {
	if _, ada := g[field]; !ada {
		g[field] = pesan
	}
}

// err mengembalikan nil bila tidak ada pesan.
func (g galat) err() error {
	if len(g) == 0 {
		return nil
	}
	return &domain.ValidationError{Fields: g}
}

// semuaAsnaf = golongan yang sah (urutan = urutan ENUM di database).
var semuaAsnaf = []domain.Asnaf{
	domain.AsnafFakir, domain.AsnafMiskin, domain.AsnafAmil, domain.AsnafMuallaf,
	domain.AsnafRiqab, domain.AsnafGharim, domain.AsnafFisabilillah, domain.AsnafIbnuSabil,
}

func asnafSah(a domain.Asnaf) bool {
	for _, s := range semuaAsnaf {
		if a == s {
			return true
		}
	}
	return false
}

// Catat menulis satu baris audit log (FR-15). Dipanggil lapisan HTTP setelah
// aksi berhasil, karena di sanalah pelaku dan IP diketahui.
//
// ponytail: audit ditulis SETELAH transaksi aksi selesai, bukan di dalamnya.
// Bila penulisan audit gagal, aksinya tetap tersimpan dan kegagalan dicatat ke
// log error. Pindahkan ke dalam transaksi bila audit harus dijamin 100%.
func (s *Service) Catat(ctx context.Context, a domain.Audit) error {
	if err := s.st.InsertAudit(ctx, a); err != nil {
		return fmt.Errorf("service.Catat: %w", err)
	}
	return nil
}

// catatDiam = Catat yang hanya melaporkan kegagalan ke log (dipakai di alur
// login, yang tidak boleh gagal hanya karena audit gagal).
func (s *Service) catatDiam(ctx context.Context, a domain.Audit) {
	if err := s.Catat(ctx, a); err != nil {
		slog.ErrorContext(ctx, "audit gagal ditulis", "action", a.Action, "error", err.Error())
	}
}

// KategoriAudit memetakan chip filter di halaman Audit ke kode aksi (docs/04 bagian 5).
var KategoriAudit = map[string][]string{
	"login": {domain.AksiLogin, domain.AksiLoginGagal, domain.AksiLogout, domain.AksiPasswordUbah},
	"data": {domain.AksiMustahikBuat, domain.AksiMustahikUbah, domain.AksiKriteriaUbah,
		domain.AksiPeriodeBuat, domain.AksiKandidat, domain.AksiPenilaian},
	"hitung":     {domain.AksiHitung},
	"keputusan":  {domain.AksiSetuju, domain.AksiTolak, domain.AksiBukaKembali},
	"penyaluran": {domain.AksiSalur, domain.AksiTutup},
}

// AuditHalaman = satu halaman audit log.
type AuditHalaman struct {
	Items   []domain.Audit `json:"items"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
}

// DaftarAudit mengembalikan audit log terbaru dulu (FR-20); kategori "" = semua.
func (s *Service) DaftarAudit(ctx context.Context, kategori string, page int) (AuditHalaman, error) {
	const perPage = 20
	var aksi []string
	if kategori != "" {
		var ada bool
		if aksi, ada = KategoriAudit[kategori]; !ada {
			return AuditHalaman{}, (galat{"kategori": "Kategori audit tidak dikenal"}).err()
		}
	}
	if page < 1 {
		page = 1
	}
	items, total, err := s.st.ListAudit(ctx, aksi, page, perPage)
	if err != nil {
		return AuditHalaman{}, fmt.Errorf("service.DaftarAudit: %w", err)
	}
	return AuditHalaman{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}
