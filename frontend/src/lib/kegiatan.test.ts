import { describe, expect, it } from 'vitest';
import { isianKegiatan, namaHari, paragraf, tanggalAcara } from './kegiatan';

describe('tanggalAcara', () => {
	it('memformat tanggal kalender tanpa bergeser zona waktu', () => {
		expect(tanggalAcara('2026-10-25')).toBe('Ahad, 25 Oktober 2026');
		// 1 Januari tidak boleh jadi 31 Desember (bug klasik DATE → zona waktu lokal).
		expect(tanggalAcara('2026-01-01')).toContain('1 Januari 2026');
	});
	it('menambahkan jam dengan titik dan WIB', () => {
		expect(tanggalAcara('2026-10-25', '19:30')).toBe('Ahad, 25 Oktober 2026 · 19.30 WIB');
	});
	it('menerima timestamp ISO dan tanpa tanggal menghasilkan string kosong', () => {
		expect(tanggalAcara('2026-10-25T00:00:00Z')).toBe('Ahad, 25 Oktober 2026');
		expect(tanggalAcara('', '19:30')).toBe('');
	});
});

describe('paragraf', () => {
	it('memisah paragraf pada baris kosong dan membuang yang kosong', () => {
		expect(paragraf('Satu baris\nlanjut.\n\n  \n\nParagraf dua.\n')).toEqual(['Satu baris\nlanjut.', 'Paragraf dua.']);
		expect(paragraf('   ')).toEqual([]);
	});
});

describe('isianKegiatan', () => {
	it('memangkas spasi dan mengganti field yang hilang menjadi string kosong', () => {
		const fd = new FormData();
		fd.set('judul', '  Maulid Nabi  ');
		fd.set('kategori', 'maulid_nabi');
		const isian = isianKegiatan(fd);
		expect(isian.judul).toBe('Maulid Nabi');
		expect(isian.kategori).toBe('maulid_nabi');
		expect(isian.tanggal_acara).toBe('');
		expect(isian.status).toBe('');
	});
});

describe('namaHari', () => {
	it('berurutan dari Ahad (0) sampai Sabtu (6), sama dengan Go', () => {
		expect(namaHari).toHaveLength(7);
		expect(namaHari[0]).toBe('Ahad');
		expect(namaHari[6]).toBe('Sabtu');
	});
});
