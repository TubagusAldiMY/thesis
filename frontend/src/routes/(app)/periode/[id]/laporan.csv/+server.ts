// GET /periode/<id>/laporan.csv: meneruskan CSV dari Go ke browser (browser
// tidak pernah bicara langsung ke Go). Hak akses tetap dicek Go lewat cookie sesi.
import { error } from '@sveltejs/kit';
import { mentah } from '$lib/server/api';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async (event) => {
	if (!event.locals.user) error(401, 'Silakan masuk terlebih dahulu');
	const res = await mentah(event, 'GET', `/api/periode/${encodeURIComponent(event.params.id)}/laporan?format=csv`);
	if (!res.ok) error(res.status === 404 ? 404 : res.status === 403 ? 403 : 500, 'Laporan tidak dapat diunduh');
	return new Response(res.body, {
		headers: {
			'Content-Type': 'text/csv; charset=utf-8',
			'Content-Disposition': res.headers.get('Content-Disposition') ?? 'attachment; filename="laporan.csv"',
			'Cache-Control': 'no-store'
		}
	});
};
