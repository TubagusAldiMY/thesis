import { describe, expect, it } from 'vitest';
import { alternatifContoh, hitungContoh, kriteriaContoh, kuota } from './aras-contoh';

// Acuan: Tabel 3.8–3.11 di Paper/BAB-III dan ranking seed demo (CLAUDE.md). Float dibandingkan dengan toleransi.
const hasil = hitungContoh(kriteriaContoh, alternatifContoh);
const dekat = (a: number, b: number, tol = 5e-5) => expect(Math.abs(a - b)).toBeLessThan(tol);

describe('hitungContoh', () => {
	it('alternatif ideal A0: cost diambil terkecil, benefit terbesar', () => {
		expect(hasil.a0).toEqual([500_000, 6, 4, 4, 4]);
	});

	it('jumlah kolom setelah cost dibalik cocok dengan Tabel 3.8', () => {
		dekat(hasil.jumlah[0] * 1e6, 8.583, 5e-4);
		expect(hasil.jumlah.slice(1)).toEqual([26, 18, 16, 17]);
	});

	it('S0 dan S tiap calon cocok dengan Tabel 3.10', () => {
		dekat(hasil.baris[0].s, 0.2333);
		const s = Object.fromEntries(hasil.baris.map((b) => [b.id, b.s]));
		dekat(s.A1, 0.1607);
		dekat(s.A2, 0.2085);
		dekat(s.A3, 0.0956);
		dekat(s.A4, 0.1709);
		dekat(s.A5, 0.1309);
	});

	it('ranking K sama dengan seed demo: A2 0,8939 · A4 0,7327 · A1 0,6888 · A5 0,5611 · A3 0,4099', () => {
		expect(hasil.ranking.map((b) => b.id)).toEqual(['A2', 'A4', 'A1', 'A5', 'A3']);
		[0.8939, 0.7327, 0.6888, 0.5611, 0.4099].forEach((k, i) => dekat(hasil.ranking[i].k, k));
	});

	it('A0 selalu K = 1 dan setiap kolom ternormalisasi berjumlah 1', () => {
		dekat(hasil.baris[0].k, 1, 1e-12);
		for (let j = 0; j < kriteriaContoh.length; j++) {
			dekat(hasil.baris.reduce((acc, b) => acc + b.r[j], 0), 1, 1e-12);
		}
	});
});

describe('kuota', () => {
	it.each([
		[3_000_000, 1_000_000, 3],
		[3_500_000, 1_000_000, 3], // sisa dana tidak cukup untuk satu penerima lagi → dibulatkan ke bawah
		[900_000, 1_000_000, 0] // dana kurang dari satu nominal → tidak ada penerima
	])('dana %i, nominal %i → %i penerima', (dana, nominal, harapan) => {
		expect(kuota(dana, nominal)).toBe(harapan);
	});
});
