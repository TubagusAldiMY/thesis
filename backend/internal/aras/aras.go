// Package aras menghitung ranking alternatif dengan metode ARAS
// (Additive Ratio Assessment). Paket ini murni: tanpa I/O, database, atau
// logging, sehingga mudah diuji dan hasilnya selalu deterministik.
//
// Langkah hitung mengikuti docs/02-perancangan-aras.md:
//  1. tambah alternatif ideal A0 (max untuk benefit, min untuk cost)
//  2. kriteria cost dibalik: x ← 1/x
//  3. normalisasi per kolom: r = x / Σx (A0 ikut dijumlah)
//  4. terbobot: d = w·r
//  5. fungsi optimalitas: S = Σd
//  6. derajat utilitas: K = S_i / S_0, urut menurun → ranking
package aras

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Tipe menentukan arah "lebih baik" sebuah kriteria.
type Tipe string

const (
	// Benefit: makin besar nilainya makin layak.
	Benefit Tipe = "benefit"
	// Cost: makin kecil nilainya makin layak (mis. penghasilan).
	Cost Tipe = "cost"
)

// ErrInvalid dibungkus oleh semua kesalahan validasi input Hitung,
// sehingga pemanggil cukup memeriksa errors.Is(err, aras.ErrInvalid).
var ErrInvalid = errors.New("aras: input tidak valid")

// Kriteria adalah satu kolom penilaian. Bobot berupa pecahan 0–1 dan
// total seluruh bobot harus tepat 1 (lihat validasi persen bulat di Hitung).
type Kriteria struct {
	Kode  string
	Tipe  Tipe
	Bobot float64
}

// Alternatif adalah satu calon penerima. Nilai diurutkan sama persis
// dengan slice Kriteria dan semuanya harus > 0 (cost dipakai sebagai 1/x).
type Alternatif struct {
	ID    string
	Nilai []float64
}

// HasilAlternatif adalah hasil satu alternatif. Ranking dimulai dari 1
// dan tidak pernah dipakai bersama (seri sudah dipecahkan, lihat Hitung).
type HasilAlternatif struct {
	ID      string
	S       float64
	K       float64
	Ranking int
}

// Langkah menyimpan matriks antara untuk ditampilkan di UI ("Lihat langkah
// hitung"). Setiap matriks berukuran (m+1)×n: baris 0 = A0, baris 1..m =
// alternatif sesuai URUTAN INPUT (bukan urutan ranking).
type Langkah struct {
	A0          []float64   // nilai mentah alternatif ideal per kriteria
	Matriks     [][]float64 // setelah cost dibalik (1/x)
	Normalisasi [][]float64 // r = x / Σx
	Terbobot    [][]float64 // d = w·r
}

// Hasil adalah keluaran Hitung. Alternatif sudah terurut menurut ranking.
type Hasil struct {
	S0         float64
	Alternatif []HasilAlternatif
	Langkah    Langkah
}

// toleransiPersen: bobot dianggap persen bulat bila selisihnya kurang dari
// ini. Cukup longgar untuk galat floating point (0,15*100 = 15,000000000000002)
// tetapi menolak bobot seperti 0,125.
const toleransiPersen = 1e-6

// Hitung menjalankan ARAS dan mengembalikan ranking. Input tidak diubah.
//
// Seri: dua alternatif dengan K sama sampai 4 desimal dipecahkan dengan
// membandingkan nilai mentah kriteria mulai dari bobot terbesar (kriteria
// berbobot sama: urutan kriteria); benefit → nilai lebih besar menang, cost
// → lebih kecil menang. Dengan bobot bawaan ini berarti penghasilan terkecil
// dulu, lalu tanggungan terbanyak (docs/02 bagian 4). Bila masih sama, urut
// ID naik. Jadi hasil tidak bergantung pada urutan input.
func Hitung(kriteria []Kriteria, alternatif []Alternatif) (Hasil, error) {
	if err := validasi(kriteria, alternatif); err != nil {
		return Hasil{}, err
	}
	n, m := len(kriteria), len(alternatif)

	// Langkah 1: alternatif ideal A0.
	a0 := make([]float64, n)
	for j, k := range kriteria {
		a0[j] = alternatif[0].Nilai[j]
		for _, a := range alternatif[1:] {
			if (k.Tipe == Benefit && a.Nilai[j] > a0[j]) ||
				(k.Tipe == Cost && a.Nilai[j] < a0[j]) {
				a0[j] = a.Nilai[j]
			}
		}
	}

	// Langkah 2: matriks (m+1)×n, cost dibalik. Baris 0 = A0.
	x := make([][]float64, m+1)
	for i := range x {
		sumber := a0
		if i > 0 {
			sumber = alternatif[i-1].Nilai
		}
		x[i] = make([]float64, n)
		for j, k := range kriteria {
			if k.Tipe == Cost {
				x[i][j] = 1 / sumber[j]
			} else {
				x[i][j] = sumber[j]
			}
		}
	}

	// Langkah 3–5: normalisasi per kolom, bobot, lalu jumlah per baris.
	jumlah := make([]float64, n)
	for _, baris := range x {
		for j, v := range baris {
			jumlah[j] += v
		}
	}
	r := make([][]float64, m+1)
	d := make([][]float64, m+1)
	s := make([]float64, m+1)
	for i := range x {
		r[i] = make([]float64, n)
		d[i] = make([]float64, n)
		for j := range kriteria {
			r[i][j] = x[i][j] / jumlah[j]
			d[i][j] = kriteria[j].Bobot * r[i][j]
			s[i] += d[i][j]
		}
	}

	// Langkah 6: K = S_i / S_0, lalu urutkan.
	hasil := make([]HasilAlternatif, m)
	for i, a := range alternatif {
		hasil[i] = HasilAlternatif{ID: a.ID, S: s[i+1], K: s[i+1] / s[0]}
	}
	nilaiByID := make(map[string][]float64, m)
	for _, a := range alternatif {
		nilaiByID[a.ID] = a.Nilai
	}
	urutan := indeksBobot(kriteria)
	sort.SliceStable(hasil, func(p, q int) bool {
		kp, kq := bulat4(hasil[p].K), bulat4(hasil[q].K)
		if kp != kq {
			return kp > kq
		}
		return menang(kriteria, urutan, nilaiByID[hasil[p].ID], nilaiByID[hasil[q].ID], hasil[p].ID, hasil[q].ID)
	})
	for i := range hasil {
		hasil[i].Ranking = i + 1
	}

	return Hasil{
		S0:         s[0],
		Alternatif: hasil,
		Langkah:    Langkah{A0: a0, Matriks: x, Normalisasi: r, Terbobot: d},
	}, nil
}

// bulat4 membulatkan K ke 4 desimal; dua K dianggap seri bila hasilnya sama.
func bulat4(k float64) float64 { return math.Round(k*1e4) / 1e4 }

// indeksBobot mengurutkan indeks kriteria dari bobot terbesar; bobot sama
// tetap menurut urutan kriteria (sort stabil).
func indeksBobot(kriteria []Kriteria) []int {
	idx := make([]int, len(kriteria))
	for j := range idx {
		idx[j] = j
	}
	sort.SliceStable(idx, func(p, q int) bool {
		return kriteria[idx[p]].Bobot > kriteria[idx[q]].Bobot
	})
	return idx
}

// menang melaporkan apakah alternatif p harus di atas q saat K seri.
func menang(kriteria []Kriteria, urutan []int, p, q []float64, idP, idQ string) bool {
	for _, j := range urutan {
		if p[j] == q[j] {
			continue
		}
		if kriteria[j].Tipe == Cost {
			return p[j] < q[j]
		}
		return p[j] > q[j]
	}
	return idP < idQ
}

// validasi memastikan input aman dipakai: tanpa ini pembagian 1/x atau
// normalisasi bisa menghasilkan NaN/Inf diam-diam (validasi di batas
// kepercayaan, bukan mengandalkan pemanggil).
func validasi(kriteria []Kriteria, alternatif []Alternatif) error {
	if len(kriteria) == 0 {
		return fmt.Errorf("%w: kriteria kosong", ErrInvalid)
	}
	if len(alternatif) == 0 {
		return fmt.Errorf("%w: alternatif kosong", ErrInvalid)
	}

	totalPersen := 0.0
	for _, k := range kriteria {
		if k.Tipe != Benefit && k.Tipe != Cost {
			return fmt.Errorf("%w: tipe kriteria %q tidak dikenal", ErrInvalid, k.Kode)
		}
		persen := math.Round(k.Bobot * 100)
		if math.IsNaN(k.Bobot) || k.Bobot <= 0 || math.Abs(k.Bobot*100-persen) > toleransiPersen {
			return fmt.Errorf("%w: bobot %q harus persen bulat > 0", ErrInvalid, k.Kode)
		}
		totalPersen += persen
	}
	if totalPersen != 100 {
		return fmt.Errorf("%w: total bobot %.0f%%, harus 100%%", ErrInvalid, totalPersen)
	}

	id := make(map[string]struct{}, len(alternatif))
	for _, a := range alternatif {
		if _, ada := id[a.ID]; ada {
			return fmt.Errorf("%w: ID alternatif %q duplikat", ErrInvalid, a.ID)
		}
		id[a.ID] = struct{}{}
		if len(a.Nilai) != len(kriteria) {
			return fmt.Errorf("%w: alternatif %q punya %d nilai, kriteria %d", ErrInvalid, a.ID, len(a.Nilai), len(kriteria))
		}
		for j, v := range a.Nilai {
			if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
				return fmt.Errorf("%w: nilai %q pada %q harus > 0", ErrInvalid, a.ID, kriteria[j].Kode)
			}
		}
	}
	return nil
}
