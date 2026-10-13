// POST /logout: cabut sesi di database (Go) lalu hapus cookie. Hanya POST
// (bukan GET) supaya tautan/gambar dari situs lain tidak bisa mengeluarkan
// pengguna; SvelteKit juga menolak POST form dari origin lain (CSRF).
import { redirect } from '@sveltejs/kit';
import { mentah, NAMA_COOKIE } from '$lib/server/api';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async (event) => {
	if (event.cookies.get(NAMA_COOKIE)) {
		await mentah(event, 'POST', '/api/auth/logout').catch(() => undefined);
	}
	event.cookies.delete(NAMA_COOKIE, { path: '/' });
	redirect(303, '/login');
};
