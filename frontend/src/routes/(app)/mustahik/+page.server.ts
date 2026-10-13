import { api, wajibAdmin } from '$lib/server/api';
import type { MustahikHalaman } from '$lib/types';
import type { PageServerLoad } from './$types';

const PER_HALAMAN = 20;

export const load: PageServerLoad = async (event) => {
	wajibAdmin(event.locals.user);
	const q = event.url.searchParams.get('q') ?? '';
	const asnaf = event.url.searchParams.get('asnaf') ?? '';
	const page = Math.max(1, Number(event.url.searchParams.get('page')) || 1);
	const p = new URLSearchParams({ q, asnaf, page: String(page), per_page: String(PER_HALAMAN) });
	return {
		q,
		asnaf,
		page,
		perHalaman: PER_HALAMAN,
		hasil: await api<MustahikHalaman>(event, 'GET', '/api/mustahik?' + p),
		tersimpan: event.url.searchParams.get('tersimpan') === '1'
	};
};
