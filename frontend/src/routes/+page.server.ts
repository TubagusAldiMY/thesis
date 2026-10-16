// Landing publik: memuat angka transparansi (agregat tanpa data pribadi) dari Go.
// Bila API gagal, landing tetap tampil tanpa bagian itu: halaman depan masjid
// tidak boleh ikut rusak hanya karena angka tidak termuat.
import { api } from '$lib/server/api';
import type { RingkasanPublik } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	// Tanggal hari ini menurut kalender WIB (zona tetap +7), dasar jadwal salat.
	const w = new Date(Date.now() + 7 * 3600_000);
	const hari = { tahun: w.getUTCFullYear(), bulan: w.getUTCMonth() + 1, tanggal: w.getUTCDate() };
	try {
		return { hari, ringkasan: await api<RingkasanPublik>(event, 'GET', '/api/publik/ringkasan') };
	} catch {
		return { hari, ringkasan: null };
	}
};
