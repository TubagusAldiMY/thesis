// Pembantu halaman Kriteria: mengubah skala nilai antara teks isian dan objek.
// Skala = pilihan nilai berketerangan untuk kriteria berskala, mis. kondisi
// kesehatan: 1 = sehat, 2 = sakit ringan. Ditulis satu pilihan per baris.
// Aturan ketat (kunci 1–9, 2–9 pilihan) tetap dijaga Go; di sini hanya bentuknya,
// supaya Admin langsung melihat baris yang salah ketik.

export type HasilSkala = { skala: Record<string, string> | null } | { galat: string };

const baris = /^\s*(\d+)\s*[=:]\s*(.*?)\s*$/;

/**
 * "1 = sehat\n2 = sakit ringan" → {"1":"sehat","2":"sakit ringan"}.
 * Teks kosong → skala null (kriteria bernilai angka asli). Pemisah "=" atau ":".
 */
export function parseSkala(teks: string): HasilSkala {
	const hasil: Record<string, string> = {};
	let no = 0;
	for (const mentah of teks.split(/\r?\n/)) {
		no++;
		if (mentah.trim() === '') continue;
		const m = baris.exec(mentah);
		if (!m) return { galat: `Baris ${no} ("${mentah.trim()}") harus berbentuk "nilai = keterangan", mis. "1 = sehat"` };
		if (m[2] === '') return { galat: `Baris ${no}: keterangan untuk nilai ${m[1]} masih kosong` };
		if (m[1] in hasil) return { galat: `Nilai ${m[1]} ditulis lebih dari sekali (baris ${no})` };
		hasil[m[1]] = m[2];
	}
	return { skala: Object.keys(hasil).length === 0 ? null : hasil };
}

/** Kebalikan parseSkala untuk mengisi kolom teks: urut menurut nilai. null → "". */
export function teksSkala(skala: Record<string, string> | null | undefined): string {
	if (!skala) return '';
	return Object.entries(skala)
		.sort(([a], [b]) => Number(a) - Number(b))
		.map(([n, ket]) => `${n} = ${ket}`)
		.join('\n');
}
