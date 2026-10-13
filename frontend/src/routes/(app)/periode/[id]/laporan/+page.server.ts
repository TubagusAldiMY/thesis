import { error } from '@sveltejs/kit';
import { apiOpsional } from '$lib/server/api';
import type { Laporan } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const laporan = await apiOpsional<Laporan>(event, `/api/periode/${encodeURIComponent(event.params.id)}/laporan`);
	if (!laporan) error(404, 'Laporan belum tersedia: periode ini belum pernah dihitung.');
	return { laporan };
};
