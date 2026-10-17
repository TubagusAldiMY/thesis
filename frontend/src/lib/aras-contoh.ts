// Perhitungan ARAS untuk CONTOH di halaman /cara-kerja.
//
// Kenapa dihitung di sini, bukan angka ketikan tangan: tiap tabel di halaman penjelasan (matriks, normalisasi,
// terbobot, S, K) harus selalu konsisten dengan rumusnya. Dengan menghitung dari data contoh, satu-satunya yang
// diketik tangan adalah data masukan; semua angka turunannya pasti cocok dan dijaga oleh aras-contoh.test.ts.
//
// Ini cermin sederhana dari backend/internal/aras/aras.go (langkah 1–6). Aplikasi yang sebenarnya TIDAK memakai
// berkas ini: perhitungan resmi tetap di backend Go. ponytail: tidak ada aturan seri di sini, karena data contoh
// tidak menghasilkan K yang sama; kalau contoh diganti dan muncul seri, tambahkan aturan seri seperti di Go.

export type TipeKriteria = 'benefit' | 'cost';

export interface KriteriaContoh {
	kode: string;
	nama: string;
	tipe: TipeKriteria;
	/** Bobot 0–1; jumlah semua kriteria = 1. */
	bobot: number;
	/** Cara nilai diisi, untuk tabel penjelasan (rupiah / jiwa / skala 1–4). */
	satuan: string;
}

export interface AlternatifContoh {
	id: string;
	/** Urut sama dengan kriteria. Semua > 0 karena kriteria cost dipakai sebagai 1/x. */
	nilai: number[];
}

/** Satu baris matriks. Baris pertama selalu A0 (alternatif ideal). */
export interface BarisHitung {
	id: string;
	/** Setelah kriteria cost dibalik (1/x). */
	x: number[];
	/** Setelah normalisasi: x ÷ jumlah kolom. */
	r: number[];
	/** Setelah dikali bobot. */
	d: number[];
	/** Nilai fungsi optimalitas: jumlah d pada baris. */
	s: number;
	/** Derajat utilitas: S ÷ S0. Untuk A0 selalu 1. */
	k: number;
}

export interface HasilContoh {
	/** Nilai mentah ideal per kriteria (sebelum dibalik). */
	a0: number[];
	/** Jumlah tiap kolom matriks (sesudah dibalik), termasuk baris A0. */
	jumlah: number[];
	/** Baris 0 = A0, lalu alternatif sesuai urutan masukan. */
	baris: BarisHitung[];
	/** Alternatif (tanpa A0) menurun menurut K. */
	ranking: BarisHitung[];
}

export const kriteriaContoh: KriteriaContoh[] = [
	{ kode: 'C1', nama: 'Penghasilan per bulan', tipe: 'cost', bobot: 0.3, satuan: 'Rupiah' },
	{ kode: 'C2', nama: 'Jumlah tanggungan', tipe: 'benefit', bobot: 0.2, satuan: 'Jiwa' },
	{ kode: 'C3', nama: 'Kondisi tempat tinggal', tipe: 'benefit', bobot: 0.2, satuan: 'Skala 1–4' },
	{ kode: 'C4', nama: 'Kondisi kesehatan / disabilitas', tipe: 'benefit', bobot: 0.15, satuan: 'Skala 1–4' },
	{ kode: 'C5', nama: 'Status pekerjaan', tipe: 'benefit', bobot: 0.15, satuan: 'Skala 1–4' }
];

/** Lima calon SINTETIS (bukan data mustahik asli); sama dengan data seed demo dan Tabel 3.7 di skripsi. */
export const alternatifContoh: AlternatifContoh[] = [
	{ id: 'A1', nilai: [800_000, 5, 3, 2, 3] },
	{ id: 'A2', nilai: [500_000, 4, 4, 3, 4] },
	{ id: 'A3', nilai: [1_500_000, 3, 2, 1, 2] },
	{ id: 'A4', nilai: [600_000, 2, 3, 4, 3] },
	{ id: 'A5', nilai: [1_000_000, 6, 2, 2, 1] }
];

/** Jumlah penerima = dana dibagi nominal per penerima, dibulatkan ke bawah (tidak boleh melebihi dana). */
export function kuota(dana: number, nominalPerPenerima: number): number {
	return Math.floor(dana / nominalPerPenerima);
}

export function hitungContoh(kriteria: KriteriaContoh[], alternatif: AlternatifContoh[]): HasilContoh {
	// Langkah 1: alternatif ideal. Benefit → nilai terbesar, cost → nilai terkecil.
	const a0 = kriteria.map((k, j) => {
		const kolom = alternatif.map((a) => a.nilai[j]);
		return k.tipe === 'cost' ? Math.min(...kolom) : Math.max(...kolom);
	});

	// Langkah 2: matriks keputusan; baris 0 = A0; kriteria cost dibalik jadi 1/x.
	const mentah = [{ id: 'A0', nilai: a0 }, ...alternatif];
	const x = mentah.map((a) => a.nilai.map((v, j) => (kriteria[j].tipe === 'cost' ? 1 / v : v)));

	// Langkah 3: normalisasi per kolom.
	const jumlah = kriteria.map((_, j) => x.reduce((acc, baris) => acc + baris[j], 0));
	const r = x.map((baris) => baris.map((v, j) => v / jumlah[j]));

	// Langkah 4 & 5: kali bobot, lalu jumlahkan per baris.
	const d = r.map((baris) => baris.map((v, j) => v * kriteria[j].bobot));
	const s = d.map((baris) => baris.reduce((acc, v) => acc + v, 0));

	// Langkah 6: derajat utilitas relatif terhadap A0.
	const baris: BarisHitung[] = mentah.map((a, i) => ({ id: a.id, x: x[i], r: r[i], d: d[i], s: s[i], k: s[i] / s[0] }));
	const ranking = baris.slice(1).sort((p, q) => q.k - p.k);
	return { a0, jumlah, baris, ranking };
}
