package domain

import "time"

// StatusProgram:
//
//	draf ──terbitkan──► aktif ──tutup──► selesai
//
// Hanya program aktif (dan belum lewat batas waktu) yang menerima donasi.
// Draf tidak tampil di halaman publik; selesai tetap tampil sebagai riwayat.
type StatusProgram string

const (
	ProgramDraf    StatusProgram = "draf"
	ProgramAktif   StatusProgram = "aktif"
	ProgramSelesai StatusProgram = "selesai"
)

// Program = satu program penggalangan dana (mis. renovasi atap masjid).
// Terkumpul dan Donatur dihitung dari donasi berstatus diterima, bukan disimpan.
type Program struct {
	ID         string
	Slug       string
	Judul      string
	Ringkasan  string
	Cerita     string
	Target     int64
	BatasWaktu *time.Time // tanggal (00.00 UTC); nil = tanpa batas
	Status     StatusProgram
	AdaGambar  bool
	DibuatOleh string
	Terkumpul  int64
	Donatur    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// KabarProgram = satu kabar terbaru (perkembangan/penyaluran) sebuah program.
type KabarProgram struct {
	ID         string
	ProgramID  string
	Judul      string
	Isi        string
	DibuatOleh string
	CreatedAt  time.Time
}

// DonaturProgram = satu baris daftar donatur publik, langsung dari tabel
// donasi. Service mengganti nama menjadi "Hamba Allah" bila Anonim.
type DonaturProgram struct {
	Nama        string
	Anonim      bool
	Jumlah      int64
	DibayarPada time.Time
}

// Kode aksi audit untuk program donasi (service.KategoriAudit["donasi"]).
const (
	AksiProgramBuat  = "PROGRAM_BUAT"
	AksiProgramUbah  = "PROGRAM_UBAH"  // isian, status, atau gambar sampul
	AksiProgramKabar = "PROGRAM_KABAR" // kabar terbaru ditambah atau dihapus
)
