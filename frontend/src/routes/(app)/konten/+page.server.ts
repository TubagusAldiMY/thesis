// Daftar kegiatan untuk pengurus (Admin & Ketua), termasuk draf.
import { api } from '$lib/server/api';
import type { KegiatanDaftarPengurus } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	daftar: await api<KegiatanDaftarPengurus>(event, 'GET', '/api/kegiatan'),
	pesan: event.url.searchParams.get('dihapus') ? 'Kegiatan dihapus.' : ''
});
