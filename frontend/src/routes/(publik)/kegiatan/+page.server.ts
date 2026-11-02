// Halaman publik kegiatan: kegiatan mendatang dan terdahulu (bisa disaring per
// kategori lewat ?kategori=), jadwal rutin mingguan, dan pengumuman yang sedang
// tayang. Draf tidak pernah dikirim Go ke sini.
import { api } from '$lib/server/api';
import type { JadwalRutin, KegiatanDaftarPublik, Pengumuman } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	// Kode kategori hanya huruf kecil/angka/garis bawah; nilai asing ditolak Go (422).
	const kategori = (event.url.searchParams.get('kategori') ?? '').replace(/[^a-z_]/g, '').slice(0, 30);
	const q = kategori ? '?kategori=' + kategori : '';
	const [daftar, jadwal, pengumuman] = await Promise.all([
		api<KegiatanDaftarPublik>(event, 'GET', '/api/publik/kegiatan' + q),
		api<JadwalRutin[]>(event, 'GET', '/api/publik/jadwal-rutin'),
		api<Pengumuman[]>(event, 'GET', '/api/publik/pengumuman')
	]);
	return { kategori, daftar, jadwal, pengumuman };
};
