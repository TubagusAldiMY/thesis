package domain

import "time"

// Role = peran pengguna. Hanya dua: Admin (petugas amil, mengelola data) dan
// Ketua DKM (memutuskan usulan). Hak tiap peran dicek di lapisan HTTP.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleKetua Role = "ketua"
)

// User = akun aplikasi. PasswordHash berformat PHC Argon2id dan tidak pernah
// ikut dikirim ke klien (tag json "-").
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Nama         string    `json:"nama"`
	Role         Role      `json:"role"`
	Aktif        bool      `json:"-"`
	CreatedAt    time.Time `json:"-"`
}

// Kode aksi audit log. Dikelompokkan ke kategori filter UI di
// service.KategoriAudit (docs/04 bagian 5).
const (
	AksiLogin        = "LOGIN"
	AksiLoginGagal   = "LOGIN_GAGAL"
	AksiLogout       = "LOGOUT"
	AksiPasswordUbah = "PASSWORD_UBAH"
	AksiMustahikBuat = "MUSTAHIK_BUAT"
	AksiMustahikUbah = "MUSTAHIK_UBAH"
	AksiKriteriaUbah = "KRITERIA_UBAH"
	AksiPeriodeBuat  = "PERIODE_BUAT"
	AksiKandidat     = "KANDIDAT"
	AksiPenilaian    = "PENILAIAN"
	AksiHitung       = "HITUNG"
	AksiSetuju       = "SETUJU"
	AksiTolak        = "TOLAK"
	AksiBukaKembali  = "BUKA_KEMBALI"
	AksiSalur        = "SALUR"
	AksiTutup        = "TUTUP"

	// Kelola pengguna oleh Ketua. Detail audit hanya ID akun sasaran dan
	// peran; tidak pernah password (NFR-03).
	AksiPenggunaBuat     = "USER_BUAT"
	AksiPenggunaUbah     = "USER_UBAH"
	AksiPenggunaNonaktif = "USER_NONAKTIF"
	AksiPenggunaAktif    = "USER_AKTIF"
	AksiPenggunaReset    = "USER_RESET_PASSWORD"
	AksiSetupAwal        = "SETUP_AWAL" // akun Ketua pertama dibuat lewat halaman /setup
)

// Audit = satu baris audit log. Detail TIDAK BOLEH memuat NIK, No. HP,
// password, atau token (NFR-03); cukup ID dan angka yang menjelaskan aksi.
type Audit struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`   // kosong untuk login gagal dengan username tak dikenal
	ActorNama  string         `json:"actor_nama"` // diisi saat dibaca (join users)
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	Detail     map[string]any `json:"detail"`
	IP         string         `json:"ip"`
	CreatedAt  time.Time      `json:"created_at"`
}
