import { api } from '$lib/server/api';
import type { Periode } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const status = event.url.searchParams.get('status') ?? '';
	const q = status ? '?status=' + encodeURIComponent(status) : '';
	const { items } = await api<{ items: Periode[] }>(event, 'GET', '/api/periode' + q);
	return { status, items };
};
