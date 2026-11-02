// Halaman satu kegiatan (publik). Slug tidak dikenal / kegiatan draf → 404 dari Go.
import { api } from '$lib/server/api';
import type { Kegiatan } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	kegiatan: await api<Kegiatan>(event, 'GET', '/api/publik/kegiatan/' + encodeURIComponent(event.params.slug))
});
