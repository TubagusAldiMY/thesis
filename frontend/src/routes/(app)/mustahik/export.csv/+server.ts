// GET /mustahik/export.csv: meneruskan CSV dari Go ke browser (browser tidak
// pernah bicara langsung ke Go). Filter q/asnaf ikut diteruskan supaya ekspor
// mengikuti apa yang sedang ditampilkan di layar. Hak akses tetap dicek Go.
import { error } from '@sveltejs/kit';
import { mentah, wajibAdmin } from '$lib/server/api';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async (event) => {
	wajibAdmin(event.locals.user);
	const p = new URLSearchParams({ format: 'csv' });
	const q = event.url.searchParams.get('q');
	const asnaf = event.url.searchParams.get('asnaf');
	if (q) p.set('q', q);
	if (asnaf) p.set('asnaf', asnaf);
	const res = await mentah(event, 'GET', '/api/mustahik?' + p);
	if (!res.ok) error(res.status === 403 ? 403 : 500, 'Data mustahik tidak dapat diunduh');
	return new Response(res.body, {
		headers: {
			'Content-Type': 'text/csv; charset=utf-8',
			'Content-Disposition': res.headers.get('Content-Disposition') ?? 'attachment; filename="mustahik.csv"',
			'Cache-Control': 'no-store'
		}
	});
};
