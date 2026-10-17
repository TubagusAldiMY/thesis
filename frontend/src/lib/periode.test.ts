import { describe, expect, it } from 'vitest';
import { hitungPembagian } from './periode';

describe('hitungPembagian — cara nominal per penerima', () => {
	it.each([
		[3_000_000, 1_000_000, 3, 0],
		[3_500_000, 1_000_000, 3, 500_000], // sisa dana tidak cukup untuk satu orang lagi
		[1_000_000, 1_000_000, 1, 0],
		[900_000, 1_000_000, 0, 0] // nominal melebihi dana → tidak ada penerima
	])('dana %i, nominal %i → kuota %i, sisa %i', (dana, nominal, kuota, sisa) => {
		const h = hitungPembagian('nominal', dana, nominal);
		expect(h.kuota).toBe(kuota);
		expect(h.sisa).toBe(sisa);
		expect(h.nominal).toBe(nominal);
	});
});

describe('hitungPembagian — cara jumlah penerima', () => {
	it('3 jt ÷ 3 orang = Rp 1 jt per orang, tanpa sisa', () => {
		expect(hitungPembagian('jumlah', 3_000_000, 3)).toMatchObject({ nominal: 1_000_000, kuota: 3, terpakai: 3_000_000, sisa: 0, peringatan: '' });
	});

	it('tidak habis dibagi: 1 jt ÷ 3 → 333.333 per orang, sisa Rp 1', () => {
		expect(hitungPembagian('jumlah', 1_000_000, 3)).toMatchObject({ nominal: 333_333, kuota: 3, sisa: 1 });
	});

	it('jumlah melebihi dana ditolak dengan peringatan', () => {
		const h = hitungPembagian('jumlah', 100, 101);
		expect(h.kuota).toBe(0);
		expect(h.peringatan).toMatch(/melebihi dana/);
	});

	it('dana terlalu kecil agar kuota tepat (10 ÷ 4) ditolak seperti server', () => {
		const h = hitungPembagian('jumlah', 10, 4);
		expect(h.kuota).toBe(0);
		expect(h.peringatan).toMatch(/terlalu kecil/);
	});

	it('kuota yang lolos selalu tepat sama dengan jumlah yang diminta', () => {
		for (const dana of [10, 99, 1_000_000, 1_000_003, 3_500_000]) {
			for (let n = 1; n <= 12; n++) {
				const h = hitungPembagian('jumlah', dana, n);
				if (h.peringatan === '') expect(h.kuota).toBe(n);
				else expect(h.kuota).toBe(0);
			}
		}
	});

	it('pecahan dibulatkan ke bawah (jumlah orang selalu bulat)', () => {
		expect(hitungPembagian('jumlah', 3_000_000, 3.9).kuota).toBe(3);
	});
});

describe('hitungPembagian — isian belum lengkap', () => {
	it.each([
		['nominal', NaN, 1_000_000],
		['nominal', 3_000_000, 0],
		['jumlah', 0, 3],
		['jumlah', 3_000_000, NaN],
		['jumlah', 3_000_000, -2]
	] as const)('%s dana=%s isian=%s → semua nol', (cara, dana, isian) => {
		expect(hitungPembagian(cara, dana, isian)).toEqual({ nominal: 0, kuota: 0, terpakai: 0, sisa: 0, peringatan: '' });
	});
});
