// Daftar pengguna aplikasi — khusus Ketua (Yayasan/DKM). Go juga menolak Admin (403).
import { api, wajibKetua } from '$lib/server/api';
import type { Pengguna } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibKetua(event.locals.user);
	const sp = event.url.searchParams;
	return {
		pengguna: await api<Pengguna[]>(event, 'GET', '/api/pengguna'),
		pesan: sp.get('dibuat') ? 'Akun dibuat. Sampaikan password awal langsung kepada yang bersangkutan.' : ''
	};
};
