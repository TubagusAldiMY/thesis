import { dev } from '$app/environment';
import { fail, redirect } from '@sveltejs/kit';
import { api, mentah, NAMA_COOKIE, teks } from '$lib/server/api';
import type { ApiErrorBody } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

const DELAPAN_JAM = 8 * 60 * 60; // sama dengan service.DurasiSesi di Go

/** tujuanAman: hanya path internal ("/x"), bukan "//domain-lain" (open redirect). */
function tujuanAman(next: string | null): string {
	return next && next.startsWith('/') && !next.startsWith('//') ? next : '/dashboard';
}

export const load: PageServerLoad = async (event) => {
	const { locals, url } = event;
	if (locals.user) redirect(303, '/dashboard');
	// Instalasi baru (belum ada akun): tawarkan /setup. Gagal menghubungi Go = anggap tidak aktif,
	// halaman login tidak boleh ikut rusak.
	const setupAktif = await api<{ aktif: boolean }>(event, 'GET', '/api/publik/setup')
		.then((r) => r.aktif)
		.catch(() => false);
	return { sesiHabis: url.searchParams.get('sesi') === 'habis', setupSelesai: url.searchParams.get('setup') === '1', setupAktif };
};

export const actions: Actions = {
	default: async (event) => {
		const fd = await event.request.formData();
		const username = teks(fd, 'username');
		const password = typeof fd.get('password') === 'string' ? (fd.get('password') as string) : '';

		const res = await mentah(event, 'POST', '/api/auth/login', { username, password });
		if (!res.ok) {
			const b: ApiErrorBody = await res.json().catch(() => ({ error: 'Gagal masuk' }));
			return fail(res.status, {
				username,
				error: b.error,
				sisa: b.sisa_percobaan ?? null,
				retryAfter: b.retry_after ?? null
			});
		}

		// Go mengirim token lewat Set-Cookie; SvelteKit menyimpannya di cookie
		// miliknya sendiri (HttpOnly) — token tidak pernah terlihat JavaScript browser.
		const token = res.headers
			.getSetCookie()
			.map((c) => c.split(';')[0])
			.find((c) => c.startsWith(NAMA_COOKIE + '='))
			?.slice(NAMA_COOKIE.length + 1);
		if (!token) return fail(500, { username, error: 'Sesi tidak diterima dari server', sisa: null, retryAfter: null });

		event.cookies.set(NAMA_COOKIE, token, {
			path: '/',
			httpOnly: true,
			sameSite: 'strict',
			secure: !dev,
			maxAge: DELAPAN_JAM
		});
		redirect(303, tujuanAman(event.url.searchParams.get('next')));
	}
};
