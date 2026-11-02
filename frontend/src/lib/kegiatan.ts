// Pembantu tampilan dan formulir untuk konten kegiatan keagamaan (CMS). Validasi
// sebenarnya ada di Go; di sini hanya format tampilan dan perakitan isian.
import type { StatusKegiatan } from './types';

export const labelStatusKegiatan: Record<StatusKegiatan, string> = { draf: 'Draf', terbit: 'Terbit' };
/** Kelas .badge untuk status kegiatan. */
export const warnaStatusKegiatan: Record<StatusKegiatan, string> = { draf: '', terbit: 'hijau' };

export const namaHari = ['Ahad', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];

const fmtHari = new Intl.DateTimeFormat('id-ID', {
	weekday: 'long',
	day: 'numeric',
	month: 'long',
	year: 'numeric',
	timeZone: 'UTC' // tanggal acara adalah tanggal kalender (DATE), bukan saat; jangan digeser zona waktu
});

/** "2026-10-25" + "19:30" → "Ahad, 25 Oktober 2026 · 19.30 WIB". Tanpa tanggal → "". */
export function tanggalAcara(tanggal: string, jam = ''): string {
	if (!tanggal) return '';
	// Masjid menyebut hari pertama "Ahad", bukan "Minggu" (sama dengan jadwal rutin di Go).
	const hari = fmtHari.format(new Date(tanggal.slice(0, 10) + 'T00:00:00Z')).replace(/^Minggu/, 'Ahad');
	return jam ? `${hari} · ${jam.replace(':', '.')} WIB` : hari;
}

/** Teks bebas → paragraf (dipisah baris kosong). Svelte meng-escape isinya. */
export function paragraf(s: string): string[] {
	return s
		.split(/\n\s*\n/)
		.map((x) => x.trim())
		.filter(Boolean);
}

export interface IsianKegiatan {
	judul: string;
	kategori: string;
	ringkasan: string;
	isi: string;
	tanggal_acara: string;
	jam_acara: string;
	lokasi: string;
	status: string;
}

/** Merakit isian formulir kegiatan; nilai dipantulkan ke formulir bila Go menolak. */
export function isianKegiatan(fd: FormData): IsianKegiatan {
	const ambil = (k: string) => {
		const v = fd.get(k);
		return typeof v === 'string' ? v.trim() : '';
	};
	return {
		judul: ambil('judul'),
		kategori: ambil('kategori'),
		ringkasan: ambil('ringkasan'),
		isi: ambil('isi'),
		tanggal_acara: ambil('tanggal_acara'),
		jam_acara: ambil('jam_acara'),
		lokasi: ambil('lokasi'),
		status: ambil('status')
	};
}
