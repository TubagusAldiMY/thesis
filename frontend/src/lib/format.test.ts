import { describe, expect, it } from 'vitest';
import {
	angka,
	desimal,
	parseAngka,
	parseDesimal,
	parseNilai,
	rupiah,
	rupiahRingkas,
	tanggal,
	tanggalSaja
} from './format';

// Format Indonesia (NFR-08) dipakai di semua layar; salah format = angka
// uang/bobot terbaca keliru oleh pengurus, jadi diuji tabel-demi-tabel.
describe('format id-ID', () => {
	it.each([
		[3000000, '3.000.000'],
		[0, '0'],
		[999, '999']
	])('angka(%d) = %s', (n, mau) => expect(angka(n)).toBe(mau));

	it('rupiah & ringkas', () => {
		expect(rupiah(1000000)).toBe('Rp 1.000.000');
		expect(rupiahRingkas(3000000)).toBe('Rp 3,0 jt');
		expect(rupiahRingkas(500000)).toBe('Rp 500.000');
	});

	it('desimal memakai koma', () => {
		expect(desimal(0.89391)).toBe('0,8939');
		expect(desimal(0.3, 2)).toBe('0,30');
	});

	it('tanggal dd/mm/yyyy menurut WIB', () => {
		// 17.30 UTC = 00.30 WIB keesokan harinya
		expect(tanggal('2026-10-11T17:30:00Z')).toBe('12/10/2026');
		expect(tanggalSaja('2026-10-12T00:00:00Z')).toBe('12/10/2026');
		expect(tanggalSaja('2026-01-05')).toBe('05/01/2026');
	});

	it.each([
		['3.000.000', 3000000],
		['Rp 1.500.000', 1500000],
		['250000', 250000],
		['', NaN],
		['abc', NaN]
	])('parseAngka(%j) = %d', (s, mau) => expect(parseAngka(s)).toBe(mau));

	it.each([
		['1.500.000', 1500000],
		['800000', 800000],
		['2,5', 2.5],
		['1', 1],
		['', NaN],
		['1,2,3', NaN],
		['-5', NaN]
	])('parseNilai(%j) = %d', (s, mau) => expect(parseNilai(s)).toBe(mau));

	it.each([
		['0,30', 0.3],
		['0.15', 0.15],
		['1', 1],
		['', NaN],
		['0,3,0', NaN],
		['-0,1', NaN]
	])('parseDesimal(%j) = %d', (s, mau) => expect(parseDesimal(s)).toBe(mau));
});
