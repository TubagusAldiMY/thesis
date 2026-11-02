import { describe, expect, it } from 'vitest';
import { sisaWaktu } from './format';
import { isianProgram } from './program';

function fd(isi: Record<string, string>) {
	const f = new FormData();
	for (const [k, v] of Object.entries(isi)) f.set(k, v);
	return f;
}

describe('isianProgram', () => {
	it('mengubah target rupiah bertitik menjadi angka dan memangkas spasi', () => {
		const { body, nilai } = isianProgram(
			fd({ judul: '  Renovasi Atap  ', ringkasan: 'r', cerita: 'c', target: 'Rp 5.000.000', batas_waktu: '2026-12-31', status: 'aktif' })
		);
		expect(body).toEqual({ judul: 'Renovasi Atap', ringkasan: 'r', cerita: 'c', target: 5_000_000, batas_waktu: '2026-12-31', status: 'aktif' });
		expect(nilai.target).toBe('Rp 5.000.000');
	});

	it('target tanpa digit menjadi 0 (ditolak Go) dan field hilang menjadi string kosong', () => {
		const { body } = isianProgram(fd({ target: 'lima juta' }));
		expect(body.target).toBe(0);
		expect(body.judul).toBe('');
		expect(body.batas_waktu).toBe('');
	});
});

describe('sisaWaktu', () => {
	it.each([
		[{ status: 'aktif', terbuka: true, sisa_hari: 12 }, '12 hari lagi'],
		[{ status: 'aktif', terbuka: true, sisa_hari: 0 }, 'Hari terakhir'],
		[{ status: 'aktif', terbuka: false, sisa_hari: 0 }, 'Berakhir'],
		[{ status: 'aktif', terbuka: true, sisa_hari: null }, 'Tanpa batas waktu'],
		[{ status: 'selesai', terbuka: false, sisa_hari: 5 }, 'Selesai']
	] as const)('%o → %s', (p, mau) => {
		expect(sisaWaktu(p)).toBe(mau);
	});
});
