import { describe, expect, it } from 'vitest';
import { parseSkala, teksSkala } from './kriteria';

describe('parseSkala', () => {
	it.each([
		['1 = sehat\n2 = sakit ringan', { '1': 'sehat', '2': 'sakit ringan' }],
		['1: sehat\n2: sakit ringan', { '1': 'sehat', '2': 'sakit ringan' }], // titik dua juga boleh
		['  1=sehat  \r\n\r\n 2 =  sakit ringan  \r\n', { '1': 'sehat', '2': 'sakit ringan' }], // spasi, CRLF, baris kosong
		['3 = layak: milik sendiri', { '3': 'layak: milik sendiri' }], // titik dua kedua tetap bagian keterangan
		['1 = a = b', { '1': 'a = b' }]
	])('mengurai %j', (teks, mau) => expect(parseSkala(teks)).toEqual({ skala: mau }));

	it.each([[''], ['   '], ['\n\n']])('teks kosong %j = tanpa skala (null)', (teks) => {
		expect(parseSkala(teks)).toEqual({ skala: null });
	});

	it.each([
		['sehat', /Baris 1/],
		['1 sehat', /Baris 1/], // tanpa pemisah
		['= sehat', /Baris 1/], // tanpa nilai
		['a = sehat', /Baris 1/], // nilai bukan angka
		['1 = sehat\n2', /Baris 2/],
		['1 =\n2 = ada', /keterangan untuk nilai 1/],
		['1 = sehat\n1 = sakit', /lebih dari sekali/]
	])('menolak %j dengan pesan yang menunjuk baris/penyebab', (teks, pola) => {
		const h = parseSkala(teks);
		expect('galat' in h && pola.test(h.galat)).toBe(true);
	});
});

describe('teksSkala', () => {
	it('mengurutkan menurut nilai (bukan abjad) dan membulat-balik dengan parseSkala', () => {
		const skala = { '10': 'sepuluh', '2': 'dua', '1': 'satu' };
		const teks = teksSkala(skala);
		expect(teks).toBe('1 = satu\n2 = dua\n10 = sepuluh');
		expect(parseSkala(teks)).toEqual({ skala });
	});
	it.each([[null], [undefined], [{}]])('kosong untuk %j', (x) => expect(teksSkala(x as never)).toBe(''));
});
