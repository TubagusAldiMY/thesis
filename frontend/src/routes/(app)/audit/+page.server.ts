import { api } from '$lib/server/api';
import type { AuditHalaman } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const kategori = event.url.searchParams.get('kategori') ?? '';
	const page = Math.max(1, Number(event.url.searchParams.get('page')) || 1);
	const p = new URLSearchParams({ page: String(page) });
	if (kategori) p.set('kategori', kategori);
	return { kategori, hasil: await api<AuditHalaman>(event, 'GET', '/api/audit?' + p) };
};
