# 02 — Perancangan SPK dengan Metode ARAS

ARAS (*Additive Ratio Assessment*) menilai tiap alternatif dengan **membandingkannya terhadap alternatif ideal (A0)**. Hasil akhirnya adalah derajat utilitas `K` (0–1): makin dekat ke 1, makin mendekati ideal.

## 1. Kriteria, Tipe, Bobot, Skala
> ⚠️ **Bobot di bawah adalah usulan awal.** Wajib divalidasi lewat wawancara dengan pengurus masjid/dosen pembimbing dan dicatat sebagai dasar di Bab Metodologi. Aplikasi menyimpan bobot di database (bisa diubah tanpa ubah kode), dengan syarat Σ bobot = 1 (FR-05). Bobot diubah dalam **kelipatan 0,05** (di UI lewat tombol − / +, atau diketik lalu divalidasi) dan dibandingkan sebagai bilangan bulat persen di backend, supaya Σ = 100 tepat tanpa galat pecahan desimal (mis. 0,1 + 0,2 ≠ 0,3 pada floating point).

| Kode | Kriteria | Tipe | Bobot (w) | Skala nilai |
|------|----------|------|-----------|-------------|
| C1 | Penghasilan per bulan | **Cost** | 0,30 | Rupiah (angka asli) |
| C2 | Jumlah tanggungan | Benefit | 0,20 | Jumlah jiwa (angka asli) |
| C3 | Kondisi tempat tinggal | Benefit | 0,20 | 1 = layak milik sendiri · 2 = layak sewa/menumpang · 3 = kurang layak · 4 = tidak layak/tanpa tempat tinggal |
| C4 | Kondisi kesehatan / disabilitas | Benefit | 0,15 | 1 = sehat · 2 = sakit ringan · 3 = sakit kronis/lansia · 4 = disabilitas berat/sakit parah |
| C5 | Status pekerjaan | Benefit | 0,15 | 1 = tetap · 2 = serabutan teratur · 3 = serabutan tidak teratur · 4 = tidak bekerja |
| | **Total** | | **1,00** | |

Catatan desain:
- **Cost vs Benefit**: *Benefit* = makin besar makin layak dibantu; *Cost* = makin kecil makin layak. Hanya penghasilan yang cost; kriteria kualitatif sudah dibuat searah (4 = paling membutuhkan) sehingga semuanya benefit.
- Nilai mentah harus > 0 karena cost dipakai sebagai `1/x`. Penghasilan 0 diinput sebagai Rp 1 (aturan validasi di aplikasi).
- Nilai skala 1–4 diisi petugas berdasarkan survei/observasi; definisi tiap skala ditampilkan di form agar konsisten.

## 2. Rumus ARAS (6 Langkah)
Misalkan ada `m` alternatif (mustahik) dan `n` kriteria; `x_ij` nilai alternatif i pada kriteria j.

1. **Matriks keputusan + alternatif ideal A0** (baris ke-0):
   - Benefit: `x_0j = max_i x_ij`
   - Cost: `x_0j = min_i x_ij`
2. **Cost dibalik**: untuk kriteria cost, `x_ij ← 1 / x_ij` (termasuk `x_0j`).
3. **Normalisasi per kolom** (A0 ikut dijumlah): `r_ij = x_ij / Σ_{i=0..m} x_ij`
4. **Matriks terbobot**: `d_ij = w_j · r_ij`
5. **Fungsi optimalitas**: `S_i = Σ_j d_ij`  (untuk i = 0..m)
6. **Derajat utilitas**: `K_i = S_i / S_0`, urutkan menurun → ranking.

## 3. Contoh Perhitungan (5 Alternatif, data sintetis)
Data mentah:

| Alt. | C1 Penghasilan (Rp) | C2 Tanggungan | C3 Tempat tinggal | C4 Kesehatan | C5 Pekerjaan |
|------|--------------------:|--------------:|------------------:|-------------:|-------------:|
| A1 | 800.000 | 5 | 3 | 2 | 3 |
| A2 | 500.000 | 4 | 4 | 3 | 4 |
| A3 | 1.500.000 | 3 | 2 | 1 | 2 |
| A4 | 600.000 | 2 | 3 | 4 | 3 |
| A5 | 1.000.000 | 6 | 2 | 2 | 1 |

**Langkah 1 — A0**: C1 = min = 500.000 · C2 = max = 6 · C3 = max = 4 · C4 = max = 4 · C5 = max = 4.

**Langkah 2 — cost dibalik (C1 = 1/x)**:

| Alt. | C1 (1/x) | C2 | C3 | C4 | C5 |
|------|---------:|---:|---:|---:|---:|
| A0 | 2,000×10⁻⁶ | 6 | 4 | 4 | 4 |
| A1 | 1,250×10⁻⁶ | 5 | 3 | 2 | 3 |
| A2 | 2,000×10⁻⁶ | 4 | 4 | 3 | 4 |
| A3 | 0,667×10⁻⁶ | 3 | 2 | 1 | 2 |
| A4 | 1,667×10⁻⁶ | 2 | 3 | 4 | 3 |
| A5 | 1,000×10⁻⁶ | 6 | 2 | 2 | 1 |
| **Σ kolom** | 8,583×10⁻⁶ | 26 | 18 | 16 | 17 |

**Langkah 3 — normalisasi** `r = x / Σ` (contoh: r(A1,C1) = 1,25×10⁻⁶ / 8,583×10⁻⁶ = 0,1456):

| Alt. | C1 | C2 | C3 | C4 | C5 |
|------|-------:|-------:|-------:|-------:|-------:|
| A0 | 0,2330 | 0,2308 | 0,2222 | 0,2500 | 0,2353 |
| A1 | 0,1456 | 0,1923 | 0,1667 | 0,1250 | 0,1765 |
| A2 | 0,2330 | 0,1538 | 0,2222 | 0,1875 | 0,2353 |
| A3 | 0,0777 | 0,1154 | 0,1111 | 0,0625 | 0,1176 |
| A4 | 0,1942 | 0,0769 | 0,1667 | 0,2500 | 0,1765 |
| A5 | 0,1165 | 0,2308 | 0,1111 | 0,1250 | 0,0588 |

**Langkah 4 & 5 — terbobot `d = w·r` dan `S = Σd`** (contoh A1: 0,30×0,1456 = 0,0437):

| Alt. | C1 | C2 | C3 | C4 | C5 | **S** |
|------|-------:|-------:|-------:|-------:|-------:|-------:|
| A0 | 0,0699 | 0,0462 | 0,0444 | 0,0375 | 0,0353 | **0,2333** |
| A1 | 0,0437 | 0,0385 | 0,0333 | 0,0187 | 0,0265 | **0,1607** |
| A2 | 0,0699 | 0,0308 | 0,0444 | 0,0281 | 0,0353 | **0,2085** |
| A3 | 0,0233 | 0,0231 | 0,0222 | 0,0094 | 0,0176 | **0,0956** |
| A4 | 0,0583 | 0,0154 | 0,0333 | 0,0375 | 0,0265 | **0,1709** |
| A5 | 0,0350 | 0,0462 | 0,0222 | 0,0187 | 0,0088 | **0,1309** |

**Langkah 6 — `K = S_i / S_0`** (S_0 = 0,2333) dan ranking:

| Ranking | Alt. | K |
|:------:|------|-------:|
| 1 | A2 | 0,8939 |
| 2 | A4 | 0,7327 |
| 3 | A1 | 0,6888 |
| 4 | A5 | 0,5611 |
| 5 | A3 | 0,4099 |

Interpretasi: A2 (penghasilan terendah, rumah tidak layak, tidak bekerja) paling layak; A3 (penghasilan tertinggi, sehat, rumah layak) paling akhir. Selaras dengan intuisi — ini jadi *sanity check* manual.

> Angka di atas dihitung ulang lewat skrip dan dipakai sebagai **test fixture** paket `internal/aras` (toleransi 1e-4).

## 4. Aturan Penetapan Penerima dari Ranking
1. **Filter kelayakan** sebelum ranking:
   - Periode **zakat** → hanya mustahik dengan golongan asnaf yang sesuai (default: fakir, miskin; golongan lain bisa diaktifkan Admin per periode).
   - Periode **infak** → semua mustahik aktif boleh jadi kandidat.
2. **Kuota** `N = ⌊dana_tersedia ÷ nominal_per_penerima⌋`.
3. N alternatif ber-`K` tertinggi → status *usulan penerima*; sisanya *cadangan* (urut ranking).
4. **Seri** (K sama sampai 4 desimal): bandingkan nilai mentah mulai dari kriteria **berbobot terbesar** (bobot sama → urutan kriteria); cost: nilai lebih kecil menang, benefit: lebih besar menang; bila masih sama, urut ID naik. Dengan bobot bawaan ini berarti penghasilan terkecil dulu, lalu tanggungan terbanyak. Deterministik dan tidak bergantung urutan input (diimplementasikan di `internal/aras`, dicatat di UI).
5. Usulan baru sah setelah **disetujui Ketua DKM**; jika ditolak, Admin boleh mengubah penilaian dan menghitung ulang (versi hasil lama tetap tersimpan sebagai riwayat).

## 5. Hal yang Perlu Disebut di Bab Pembahasan
- ARAS sensitif pada bobot → lakukan **analisis sensitivitas** (ubah bobot ±10%, lihat apakah ranking berubah) — bisa jadi fitur tambahan opsional.
- Perbandingan dengan metode lain (SAW/TOPSIS) opsional untuk validasi; di luar cakupan implementasi.
- Validasi hasil: bandingkan ranking sistem dengan keputusan manual pengurus pada periode lalu (tingkat kesesuaian).

Lanjut: [03 — Perancangan Sistem](03-perancangan-sistem.md).
