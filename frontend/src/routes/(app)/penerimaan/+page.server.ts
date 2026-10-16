// Daftar donasi (penerimaan zakat/infak) untuk pengurus: Admin dan Ketua boleh melihat;
// hanya Admin yang mencatat/membatalkan (dijaga Go dan halaman turunan).
import { redirect } from '@sveltejs/kit';
import { api } from '$lib/server/api';
import type { DonasiHalaman } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

const PER_HALAMAN = 20;
const STATUS = ['menunggu', 'diterima', 'gagal', 'kedaluwarsa', 'dibatalkan'];
const JENIS = ['zakat_fitrah', 'zakat_mal', 'zakat_profesi', 'infak'];

export const load: PageServerLoad = async (event) => {
	const sp = event.url.searchParams;
	// Nilai filter dari URL diperiksa dulu: nilai asing akan ditolak Go (422) dan menjadi halaman error.
	const status = STATUS.includes(sp.get('status') ?? '') ? (sp.get('status') as string) : '';
	const jenis = JENIS.includes(sp.get('jenis') ?? '') ? (sp.get('jenis') as string) : '';
	const q = (sp.get('q') ?? '').slice(0, 100);
	const page = Math.max(1, Number(sp.get('page')) || 1);
	const p = new URLSearchParams({ q, status, jenis, page: String(page), per_page: String(PER_HALAMAN) });
	return {
		q,
		status,
		jenis,
		page,
		perHalaman: PER_HALAMAN,
		hasil: await api<DonasiHalaman>(event, 'GET', '/api/donasi?' + p),
		tercatat: sp.get('tercatat') === '1'
	};
};

export const actions: Actions = {
	// Menghapus penanda "baru" (notifikasi) untuk semua donasi web yang sudah diterima.
	dibaca: async (event) => {
		await api(event, 'POST', '/api/donasi/dibaca');
		redirect(303, '/penerimaan');
	}
};
