import { describe, expect, it } from 'vitest';
import { alternatifContoh, hitungContoh, kriteriaContoh, kuota, type AlternatifContoh, type KriteriaContoh } from './aras-contoh';

// Acuan: Tabel 3.8–3.11 di Paper/BAB-III dan ranking seed demo (CLAUDE.md). Float dibandingkan dengan toleransi.
const hasil = hitungContoh(kriteriaContoh, alternatifContoh);
const dekat = (a: number, b: number, tol = 5e-5) => expect(Math.abs(a - b)).toBeLessThan(tol);

describe('hitungContoh — konfigurasi final (semua benefit, C1 tingkat 1–4)', () => {
	it('alternatif ideal A0: nilai terbesar tiap kriteria', () => {
		expect(hasil.a0).toEqual([4, 6, 4, 4, 4]);
	});

	it('tanpa kriteria cost, matriks keputusan sama dengan nilai asli dan jumlah kolom cocok Tabel 3.8', () => {
		expect(hasil.baris.slice(1).map((b) => b.x)).toEqual(alternatifContoh.map((a) => a.nilai));
		expect(hasil.jumlah).toEqual([17, 26, 18, 16, 17]);
	});

	it('S0 dan S tiap calon cocok dengan Tabel 3.10', () => {
		dekat(hasil.baris[0].s, 0.2348);
		const s = Object.fromEntries(hasil.baris.map((b) => [b.id, b.s]));
		dekat(s.A1, 0.1725);
		dekat(s.A2, 0.2024);
		dekat(s.A3, 0.0904);
		dekat(s.A4, 0.1567);
		dekat(s.A5, 0.1432);
	});

	it('ranking K sama dengan seed demo: A2 0,8618 · A1 0,7346 · A4 0,6671 · A5 0,6098 · A3 0,3849', () => {
		expect(hasil.ranking.map((b) => b.id)).toEqual(['A2', 'A1', 'A4', 'A5', 'A3']);
		[0.8618, 0.7346, 0.6671, 0.6098, 0.3849].forEach((k, i) => dekat(hasil.ranking[i].k, k));
	});

	it('A0 selalu K = 1 dan setiap kolom ternormalisasi berjumlah 1', () => {
		dekat(hasil.baris[0].k, 1, 1e-12);
		for (let j = 0; j < kriteriaContoh.length; j++) {
			dekat(hasil.baris.reduce((acc, b) => acc + b.r[j], 0), 1, 1e-12);
		}
	});

	it('bobot kriteria berjumlah tepat 1', () => {
		dekat(kriteriaContoh.reduce((acc, k) => acc + k.bobot, 0), 1, 1e-12);
	});
});

// Konfigurasi lama (usulan awal): C1 = penghasilan rupiah bertipe cost. Tetap dites supaya cabang cost
// (A0 = nilai terkecil, nilai dibalik 1/x) tidak rusak diam-diam, sebab pengurus boleh membuat kriteria cost.
describe('hitungContoh — kriteria cost (usulan awal: C1 rupiah)', () => {
	const kriteria: KriteriaContoh[] = [
		{ kode: 'C1', nama: 'Penghasilan', tipe: 'cost', bobot: 0.3, satuan: 'Rupiah' },
		{ kode: 'C2', nama: 'Tanggungan', tipe: 'benefit', bobot: 0.2, satuan: 'Jiwa' },
		{ kode: 'C3', nama: 'Tempat tinggal', tipe: 'benefit', bobot: 0.2, satuan: 'Tingkat' },
		{ kode: 'C4', nama: 'Kesehatan', tipe: 'benefit', bobot: 0.15, satuan: 'Tingkat' },
		{ kode: 'C5', nama: 'Pekerjaan', tipe: 'benefit', bobot: 0.15, satuan: 'Tingkat' }
	];
	const calon: AlternatifContoh[] = [
		{ id: 'A1', nilai: [800_000, 5, 3, 2, 3] },
		{ id: 'A2', nilai: [500_000, 4, 4, 3, 4] },
		{ id: 'A3', nilai: [1_500_000, 3, 2, 1, 2] },
		{ id: 'A4', nilai: [600_000, 2, 3, 4, 3] },
		{ id: 'A5', nilai: [1_000_000, 6, 2, 2, 1] }
	];
	const h = hitungContoh(kriteria, calon);

	it('A0 memakai penghasilan terkecil dan nilai dibalik menjadi 1/x', () => {
		expect(h.a0).toEqual([500_000, 6, 4, 4, 4]);
		dekat(h.jumlah[0] * 1e6, 8.583, 5e-4);
	});

	it('ranking cocok dengan perhitungan manual awal: A2 0,8939 · A4 0,7327 · A1 0,6888 · A5 0,5611 · A3 0,4099', () => {
		expect(h.ranking.map((b) => b.id)).toEqual(['A2', 'A4', 'A1', 'A5', 'A3']);
		[0.8939, 0.7327, 0.6888, 0.5611, 0.4099].forEach((k, i) => dekat(h.ranking[i].k, k));
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
