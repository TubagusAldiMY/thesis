import { describe, expect, it } from 'vitest';
import { jadwalSalat, jamMenit } from './salat';

// Pembanding independen: layanan jadwal salat berparameter Kemenag RI (subuh 20°,
// isya 18°) untuk titik Pamulang, Tangerang Selatan (-6,3436; 106,7365), WIB.
// Toleransi 1 menit karena pembulatan tiap penyedia berbeda (selisih nyata: 0–1 menit).
const lintang = -6.3436;
const bujur = 106.7365;
const hm = (s: string) => Number(s.slice(0, 2)) * 60 + Number(s.slice(3));

const pembanding = [
	{ tgl: [2026, 10, 16], subuh: '04:13', terbit: '05:31', dzuhur: '11:39', ashar: '14:46', maghrib: '17:46', isya: '18:56' },
	{ tgl: [2026, 12, 21], subuh: '04:11', terbit: '05:36', dzuhur: '11:51', ashar: '15:18', maghrib: '18:06', isya: '19:22' },
	{ tgl: [2026, 6, 21], subuh: '04:38', terbit: '06:02', dzuhur: '11:55', ashar: '15:17', maghrib: '17:47', isya: '19:02' }
] as const;

describe('jadwalSalat', () => {
	it.each(pembanding)('cocok dengan pembanding Kemenag untuk $tgl', (p) => {
		const j = jadwalSalat(p.tgl[0], p.tgl[1], p.tgl[2], lintang, bujur);
		for (const k of ['subuh', 'terbit', 'dzuhur', 'ashar', 'maghrib', 'isya'] as const) {
			expect(Math.abs(j[k] - hm(p[k])), `${k}: hitung ${jamMenit(j[k])}, pembanding ${p[k]}`).toBeLessThanOrEqual(1);
		}
	});

	it('urutan waktu dalam sehari selalu menaik', () => {
		for (const [y, m, d] of [[2026, 1, 1], [2026, 3, 21], [2026, 9, 23], [2027, 2, 28]]) {
			const j = jadwalSalat(y, m, d, lintang, bujur);
			const urut = [j.subuh, j.terbit, j.dzuhur, j.ashar, j.maghrib, j.isya];
			expect(urut).toEqual([...urut].sort((a, b) => a - b));
		}
	});

	it('titik yang lebih timur salat lebih awal (bujur +1° ≈ 4 menit)', () => {
		const barat = jadwalSalat(2026, 10, 16, lintang, 106);
		const timur = jadwalSalat(2026, 10, 16, lintang, 107);
		expect(barat.dzuhur - timur.dzuhur).toBeGreaterThanOrEqual(3);
		expect(barat.dzuhur - timur.dzuhur).toBeLessThanOrEqual(5);
	});
});

describe('jamMenit', () => {
	it.each([
		[0, '00:00'],
		[699, '11:39'],
		[1439, '23:59'],
		[1440, '00:00'],
		[-1, '23:59']
	])('jamMenit(%i) = %s', (m, mau) => expect(jamMenit(m)).toBe(mau));
});
