// Daftar Laporan (FR-22) = periode berstatus selesai.
import { api } from '$lib/server/api';
import type { Periode } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	items: (await api<{ items: Periode[] }>(event, 'GET', '/api/periode?status=selesai')).items
});
