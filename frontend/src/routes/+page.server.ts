// Landing publik: memuat angka transparansi (agregat tanpa data pribadi) dari Go.
// Bila API gagal, landing tetap tampil tanpa bagian itu: halaman depan masjid
// tidak boleh ikut rusak hanya karena angka tidak termuat.
import { api } from '$lib/server/api';
import type { Program, RingkasanPublik } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	// Tanggal hari ini menurut kalender WIB (zona tetap +7), dasar jadwal salat.
	const w = new Date(Date.now() + 7 * 3600_000);
	const hari = { tahun: w.getUTCFullYear(), bulan: w.getUTCMonth() + 1, tanggal: w.getUTCDate() };
	// Dua permintaan paralel; kegagalan salah satu tidak menggagalkan landing.
	const [ringkasan, program] = await Promise.all([
		api<RingkasanPublik>(event, 'GET', '/api/publik/ringkasan').catch(() => null),
		api<Program[]>(event, 'GET', '/api/publik/program').catch(() => [] as Program[])
	]);
	// Landing cukup memamerkan tiga program yang sedang berjalan; selebihnya di /program.
	return { hari, ringkasan, program: program.filter((p) => p.status === 'aktif').slice(0, 3) };
};
