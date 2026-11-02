package domain

import "time"

// StatusKegiatan: draf (hanya pengurus yang melihat) atau terbit (tampil publik).
type StatusKegiatan string

const (
	KegiatanDraf   StatusKegiatan = "draf"
	KegiatanTerbit StatusKegiatan = "terbit"
)

// Kegiatan = satu artikel/acara keagamaan (Maulid Nabi, kajian, dll.).
// TanggalAcara nil = artikel tanpa tanggal acara. Tanggal disimpan sebagai
// DATE (00.00 UTC); jam sebagai teks "HH:MM" waktu setempat.
type Kegiatan struct {
	ID           string
	Slug         string
	Judul        string
	Kategori     string
	Ringkasan    string
	Isi          string
	TanggalAcara *time.Time
	JamAcara     string
	Lokasi       string
	Status       StatusKegiatan
	AdaGambar    bool
	DibuatOleh   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// JadwalRutin = kegiatan mingguan yang berulang (mis. Kajian Subuh tiap Ahad).
// Hari: 0 = Ahad ... 6 = Sabtu.
type JadwalRutin struct {
	ID         string
	Judul      string
	Pemateri   string
	Hari       int
	Jam        string
	Lokasi     string
	Keterangan string
	Aktif      bool
	CreatedAt  time.Time
}

// Pengumuman = pengumuman singkat dengan masa tayang. TampilSampai nil =
// tayang terus; batasnya termasuk (hari itu masih tayang).
type Pengumuman struct {
	ID           string
	Judul        string
	Isi          string
	TampilSampai *time.Time
	Terbit       bool
	CreatedAt    time.Time
}

// Kode aksi audit untuk konten; jenis kontennya ada di kolom resource
// ("kegiatan" | "jadwal_rutin" | "pengumuman"). Detail tanpa isi artikel.
const (
	AksiKontenBuat  = "KONTEN_BUAT"
	AksiKontenUbah  = "KONTEN_UBAH" // isian, status, atau gambar sampul
	AksiKontenHapus = "KONTEN_HAPUS"
)
