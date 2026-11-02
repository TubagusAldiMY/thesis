// Merakit isian formulir program donasi menjadi body JSON untuk Go. Dipakai
// action "buat" dan "ubah". Validasi sebenarnya di Go; di sini hanya mengubah
// teks rupiah ("5.000.000") menjadi angka dan memangkas spasi.
import { parseAngka } from './format';

export interface IsianProgram {
	judul: string;
	ringkasan: string;
	cerita: string;
	target: number;
	batas_waktu: string;
	status: string;
}

/** Nilai yang dipantulkan ke formulir saat Go menolak isian (teks apa adanya). */
export type NilaiProgram = Omit<IsianProgram, 'target'> & { target: string };

export function isianProgram(fd: FormData): { body: IsianProgram; nilai: NilaiProgram } {
	const ambil = (k: string) => {
		const v = fd.get(k);
		return typeof v === 'string' ? v.trim() : '';
	};
	const targetTeks = ambil('target');
	const target = parseAngka(targetTeks);
	const nilai = {
		judul: ambil('judul'),
		ringkasan: ambil('ringkasan'),
		cerita: ambil('cerita'),
		target: targetTeks,
		batas_waktu: ambil('batas_waktu'),
		status: ambil('status')
	};
	// 0 → Go menolak dengan pesan "Target harus antara …".
	return { body: { ...nilai, target: Number.isFinite(target) ? target : 0 }, nilai };
}
