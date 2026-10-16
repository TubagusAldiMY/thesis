package domain

import "time"

// StatusHasil = nasib seorang kandidat pada satu run: N teratas (N = kuota)
// menjadi usulan penerima, sisanya cadangan.
type StatusHasil string

const (
	HasilUsulan   StatusHasil = "usulan"
	HasilCadangan StatusHasil = "cadangan"
)

// Penilaian = nilai mentah satu kandidat pada satu kriteria dalam satu periode.
type Penilaian struct {
	MustahikID string  `json:"mustahik_id"`
	KriteriaID string  `json:"kriteria_id"`
	Nilai      float64 `json:"nilai"`
}

// BobotKriteria = potret kriteria saat sebuah run dihitung. Disimpan bersama
// run (aras_run.bobot_snapshot) supaya hasil lama tetap bisa dijelaskan
// walau bobot kriteria diubah kemudian.
type BobotKriteria struct {
	Kode string `json:"kode"`
	// Nama ikut dipotret supaya laporan lama tetap memakai nama saat itu bila kriteria
	// diganti namanya. Kosong pada run lama (sebelum nama bisa diedit): laporan jatuh
	// ke nama kriteria saat ini.
	Nama  string       `json:"nama,omitempty"`
	Tipe  TipeKriteria `json:"tipe"`
	Bobot float64      `json:"bobot"`
}

// ArasRun = satu kali "Hitung ARAS" pada sebuah periode. Versi naik 1 tiap
// perhitungan ulang; hanya satu run per periode yang IsCurrent.
type ArasRun struct {
	ID             string          `json:"id"`
	PeriodeID      string          `json:"periode_id"`
	Versi          int             `json:"versi"`
	Bobot          []BobotKriteria `json:"bobot"`
	S0             float64         `json:"s0"`
	Langkah        []byte          `json:"-"` // JSON langkah hitung; nil bila tidak ada. Isinya milik service
	IsCurrent      bool            `json:"is_current"`
	DijalankanOleh string          `json:"dijalankan_oleh"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ArasHasil = hasil satu kandidat pada satu run. Ranking mulai dari 1.
type ArasHasil struct {
	MustahikID string      `json:"mustahik_id"`
	S          float64     `json:"s"`
	K          float64     `json:"k"`
	Ranking    int         `json:"ranking"`
	Status     StatusHasil `json:"status"`
}

// Keputusan Ketua DKM atas usulan penerima satu run.
type Keputusan string

const (
	KeputusanSetuju Keputusan = "setuju"
	KeputusanTolak  Keputusan = "tolak"
)

// Persetujuan = keputusan Ketua atas satu run (satu run hanya bisa diputuskan
// sekali; setelah ditolak dan dibuka kembali, perlu hitung ulang = run baru).
type Persetujuan struct {
	RunID     string    `json:"run_id"`
	Versi     int       `json:"versi"`
	Keputusan Keputusan `json:"keputusan"`
	Catatan   string    `json:"catatan"`
	Oleh      string    `json:"oleh"`
	OlehNama  string    `json:"oleh_nama"`
	CreatedAt time.Time `json:"created_at"`
}

// Penyaluran = realisasi dana untuk satu penerima pada satu periode. Tanggal
// hanya bagian tanggal (UTC 00:00); uang dalam rupiah utuh.
type Penyaluran struct {
	ID          string    `json:"id"`
	PeriodeID   string    `json:"periode_id"`
	MustahikID  string    `json:"mustahik_id"`
	Nominal     int64     `json:"nominal"`
	Tanggal     time.Time `json:"tanggal"`
	DicatatOleh string    `json:"dicatat_oleh"`
	CreatedAt   time.Time `json:"created_at"`
}
