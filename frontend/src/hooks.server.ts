// Hook server: berjalan untuk SETIAP permintaan ke SvelteKit.
//  1. request_id: terima dari klien bila aman, kalau tidak buat baru;
//  2. sesi: cookie `sesi` → GET /api/auth/me → event.locals.user;
//  3. header keamanan (OWASP A05) + X-Request-Id di respons;
//  4. log akses & error berbentuk JSON ke stdout (aturan logging #1–#5).

import { dev } from '$app/environment';
import type { Handle, HandleServerError } from '@sveltejs/kit';
import { mentah, NAMA_COOKIE } from '$lib/server/api';
import type { User } from '$lib/types';

const reRequestID = /^[A-Za-z0-9._-]{1,64}$/;

/**
 * Path untuk log. Kode di /kuitansi/<kode> adalah kunci rahasia tautan kuitansi
 * muzaki (siapa pun yang memegangnya bisa membukanya), jadi tidak boleh tercatat.
 * Query string memang tidak pernah dicatat (lihat komentar di handle).
 */
const pathLog = (path: string) => (path.startsWith('/kuitansi/') ? '/kuitansi/{kode}' : path);

function log(level: 'info' | 'error', msg: string, field: Record<string, unknown>) {
	console[level === 'error' ? 'error' : 'log'](
		JSON.stringify({ time: new Date().toISOString(), level, msg, service: 'frontend', ...field })
	);
}

export const handle: Handle = async ({ event, resolve }) => {
	const mulai = Date.now();
	const dariKlien = event.request.headers.get('x-request-id') ?? '';
	event.locals.requestId = reRequestID.test(dariKlien) ? dariKlien : crypto.randomUUID();
	event.locals.user = null;

	if (event.cookies.get(NAMA_COOKIE)) {
		const res = await mentah(event, 'GET', '/api/auth/me');
		if (res.ok) {
			event.locals.user = (await res.json()) as User;
		} else if (res.status === 401) {
			event.cookies.delete(NAMA_COOKIE, { path: '/' }); // sesi habis/dicabut
		}
	}

	const response = await resolve(event);
	const h = response.headers;
	h.set('X-Request-Id', event.locals.requestId);
	h.set('X-Content-Type-Options', 'nosniff');
	h.set('X-Frame-Options', 'DENY');
	h.set('Referrer-Policy', 'strict-origin-when-cross-origin');
	h.set('Permissions-Policy', 'geolocation=(), microphone=(), camera=()');
	if (!dev) h.set('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');

	// Query string tidak dicatat (bisa memuat kata kunci nama = data pribadi).
	log('info', 'http', {
		request_id: event.locals.requestId,
		method: event.request.method,
		path: pathLog(event.url.pathname),
		status: response.status,
		duration_ms: Date.now() - mulai,
		user_id: event.locals.user?.id ?? ''
	});
	return response;
};

export const handleError: HandleServerError = ({ error, event, status, message }) => {
	if (status !== 404) {
		log('error', 'kesalahan tak tertangani', {
			request_id: event.locals.requestId,
			path: pathLog(event.url.pathname),
			status,
			error: error instanceof Error ? error.message : String(error)
		});
	}
	return { message: status === 404 ? 'Halaman tidak ditemukan' : message, requestId: event.locals.requestId };
};
