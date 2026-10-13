// Penjaga semua halaman di grup (app): wajib masuk. Belum masuk → /login
// dengan alamat tujuan dibawa supaya setelah login kembali ke halaman ini.
import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = ({ locals, url }) => {
	if (!locals.user) {
		redirect(303, '/login?next=' + encodeURIComponent(url.pathname + url.search));
	}
	return { user: locals.user };
};
