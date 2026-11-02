// Daftar program donasi untuk pengunjung (tanpa login): program aktif dan yang
// sudah selesai. Draf tidak pernah dikirim Go ke sini.
import { api } from '$lib/server/api';
import type { Program } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const semua = await api<Program[]>(event, 'GET', '/api/publik/program');
	return {
		berjalan: semua.filter((p) => p.status === 'aktif'),
		selesai: semua.filter((p) => p.status === 'selesai')
	};
};
