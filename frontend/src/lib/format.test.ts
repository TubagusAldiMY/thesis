import { describe, expect, it } from 'vitest';
import {
	angka,
	desimal,
	parseAngka,
	parseDesimal,
	parseNilai,
	rupiah,
	namaMetode,
	nomorWA,
	rupiahRingkas,
	tanggal,
	tanggalSaja,
	tautanWA,
	terbilang
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

// Kuitansi adalah bukti keuangan: salah eja jumlah (mis. "seribu ribu") merusak
// kepercayaan donatur, jadi bentuk-bentuk tak beraturan bahasa Indonesia diuji satu per satu.
describe('terbilang', () => {
	it.each([
		[0, 'Nol rupiah'],
		[1, 'Satu rupiah'],
		[10, 'Sepuluh rupiah'],
		[11, 'Sebelas rupiah'],
		[15, 'Lima belas rupiah'],
		[20, 'Dua puluh rupiah'],
		[21, 'Dua puluh satu rupiah'],
		[100, 'Seratus rupiah'],
		[111, 'Seratus sebelas rupiah'],
		[200, 'Dua ratus rupiah'],
		[999, 'Sembilan ratus sembilan puluh sembilan rupiah'],
		[1000, 'Seribu rupiah'],
		[1001, 'Seribu satu rupiah'],
		[1500, 'Seribu lima ratus rupiah'],
		[2000, 'Dua ribu rupiah'],
		[10_000, 'Sepuluh ribu rupiah'],
		[11_000, 'Sebelas ribu rupiah'],
		[50_000, 'Lima puluh ribu rupiah'],
		[100_000, 'Seratus ribu rupiah'],
		[125_000, 'Seratus dua puluh lima ribu rupiah'],
		[1_000_000, 'Satu juta rupiah'],
		[1_001_000, 'Satu juta seribu rupiah'],
		[2_500_000, 'Dua juta lima ratus ribu rupiah'],
		[100_000_000, 'Seratus juta rupiah'],
		[1_000_000_000, 'Satu miliar rupiah'],
		[1_234_567_891, 'Satu miliar dua ratus tiga puluh empat juta lima ratus enam puluh tujuh ribu delapan ratus sembilan puluh satu rupiah']
	])('terbilang(%d)', (n, mau) => expect(terbilang(n)).toBe(mau));

	it.each([[-1], [1.5], [NaN], [Infinity]])('menolak %d', (n) => expect(() => terbilang(n)).toThrow(RangeError));
});

describe('namaMetode', () => {
	it.each([
		['tunai', 'Tunai'],
		['qris', 'QRIS'],
		['bank_transfer', 'Transfer bank (virtual account)'],
		['metode_baru_gateway', 'metode_baru_gateway'], // tak dikenal: tampil apa adanya
		['', '-']
	])('namaMetode(%j) = %s', (m, mau) => expect(namaMetode(m)).toBe(mau));
});

describe('nomor WhatsApp', () => {
	it.each([
		['081234567890', '6281234567890'],
		['0812-3456-7890', '6281234567890'],
		['+62 812 3456 7890', '6281234567890'],
		['6281234567890', '6281234567890'],
		['+1 555 123 4567', '15551234567']
	])('nomorWA(%j) = %s', (hp, mau) => expect(nomorWA(hp)).toBe(mau));

	it.each([[''], ['abc'], ['12345'], ['8123456789'], ['0']])('nomorWA(%j) = null', (hp) => expect(nomorWA(hp)).toBeNull());

	it('tautanWA mengenkode teks dan null untuk nomor tak valid', () => {
		expect(tautanWA('081234567890', 'Halo & terima kasih')).toBe('https://wa.me/6281234567890?text=Halo%20%26%20terima%20kasih');
		expect(tautanWA('abc', 'x')).toBeNull();
	});
});
