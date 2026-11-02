// Halaman satu program donasi (publik): cerita, kemajuan, kabar terbaru, dan
// donatur terbaru. Slug tidak dikenal / program draf → 404 dari Go.
import { api } from '$lib/server/api';
import type { ProgramDetail } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const program = await api<ProgramDetail>(event, 'GET', '/api/publik/program/' + encodeURIComponent(event.params.slug));
	return { program };
};
