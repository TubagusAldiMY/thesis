// Landing publik: memuat angka transparansi (agregat tanpa data pribadi) dari Go.
// Bila API gagal, landing tetap tampil tanpa bagian itu: halaman depan masjid
// tidak boleh ikut rusak hanya karena angka tidak termuat.
import { api } from '$lib/server/api';
import type { JadwalRutin, KegiatanDaftarPublik, Pengumuman, Program, RingkasanPublik } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	// Tanggal hari ini menurut kalender WIB (zona tetap +7), dasar jadwal salat.
	const w = new Date(Date.now() + 7 * 3600_000);
	const hari = { tahun: w.getUTCFullYear(), bulan: w.getUTCMonth() + 1, tanggal: w.getUTCDate() };
	// Permintaan paralel; kegagalan salah satu tidak menggagalkan landing.
	const [ringkasan, program, kegiatan, jadwal, pengumuman] = await Promise.all([
		api<RingkasanPublik>(event, 'GET', '/api/publik/ringkasan').catch(() => null),
		api<Program[]>(event, 'GET', '/api/publik/program').catch(() => [] as Program[]),
		api<KegiatanDaftarPublik>(event, 'GET', '/api/publik/kegiatan').catch(() => null),
		api<JadwalRutin[]>(event, 'GET', '/api/publik/jadwal-rutin').catch(() => [] as JadwalRutin[]),
		api<Pengumuman[]>(event, 'GET', '/api/publik/pengumuman').catch(() => [] as Pengumuman[])
	]);
	// Landing cukup memamerkan tiga program yang sedang berjalan dan tiga kegiatan
	// (yang akan datang lebih dulu, lalu yang terbaru); selebihnya di /program dan /kegiatan.
	const kegiatanLanding = kegiatan ? [...kegiatan.mendatang, ...kegiatan.terdahulu].slice(0, 3) : [];
	return {
		hari,
		ringkasan,
		program: program.filter((p) => p.status === 'aktif').slice(0, 3),
		kegiatan: kegiatanLanding,
		jadwal,
		pengumuman
	};
};
