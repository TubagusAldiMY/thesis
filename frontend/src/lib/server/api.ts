// Jembatan SvelteKit (server) → Go API. Browser TIDAK pernah memanggil Go
// langsung: token sesi hanya hidup di cookie HttpOnly milik SvelteKit, lalu
// diteruskan ke Go di sini. Paket $lib/server tidak bisa diimpor kode browser.

import { env } from '$env/dynamic/private';
import { error, fail, redirect, type RequestEvent } from '@sveltejs/kit';
import type { ApiErrorBody, User } from '$lib/types';

export const NAMA_COOKIE = 'sesi';

const baseURL = () => env.API_URL || 'http://127.0.0.1:8090';

/** Error yang pantas ditampilkan di formulir (400/409/422/429), bukan halaman error. */
export class ApiGagal extends Error {
	constructor(
		public status: number,
		public body: ApiErrorBody
	) {
		super(body.error);
	}
}

/**
 * mentah mengirim permintaan ke Go API dan mengembalikan Response apa adanya.
 * Meneruskan: cookie sesi, IP browser (X-Real-IP, untuk penguncian login &
 * audit), dan X-Request-Id (log SvelteKit & Go bisa dikaitkan).
 */
export async function mentah(event: RequestEvent, method: string, path: string, body?: unknown): Promise<Response> {
	const headers: Record<string, string> = {
		'X-Request-Id': event.locals.requestId,
		'X-Real-IP': event.getClientAddress()
	};
	const token = event.cookies.get(NAMA_COOKIE);
	if (token) headers.Cookie = `${NAMA_COOKIE}=${token}`;
	if (body !== undefined) headers['Content-Type'] = 'application/json';
	try {
		return await fetch(baseURL() + path, {
			method,
			headers,
			body: body === undefined ? undefined : JSON.stringify(body),
			signal: AbortSignal.timeout(15_000)
		});
	} catch {
		error(503, 'Server API tidak dapat dihubungi. Coba lagi beberapa saat.');
	}
}

/**
 * api = mentah + penerjemahan status:
 *  - 2xx → body JSON (204 → undefined)
 *  - 401 → cookie dihapus, alihkan ke /login (sesi habis)
 *  - 403/404 → halaman error
 *  - 400/409/422/429 → lempar ApiGagal (ditangkap `aksi` untuk formulir)
 *  - lainnya → halaman error 500
 */
export async function api<T>(event: RequestEvent, method: string, path: string, body?: unknown): Promise<T> {
	return baca<T>(event, await mentah(event, method, path, body));
}

/** apiOpsional = api, tetapi 404 menjadi null (mis. periode yang belum pernah dihitung). */
export async function apiOpsional<T>(event: RequestEvent, path: string): Promise<T | null> {
	const res = await mentah(event, 'GET', path);
	return res.status === 404 ? null : baca<T>(event, res);
}

async function baca<T>(event: RequestEvent, res: Response): Promise<T> {
	if (res.ok) {
		return (res.status === 204 ? undefined : await res.json()) as T;
	}
	const b: ApiErrorBody = await res.json().catch(() => ({ error: 'Kesalahan tidak dikenal' }));
	switch (res.status) {
		case 401:
			event.cookies.delete(NAMA_COOKIE, { path: '/' });
			redirect(303, '/login?sesi=habis&next=' + encodeURIComponent(event.url.pathname));
		case 403:
			error(403, b.error);
		case 404:
			error(404, 'Data tidak ditemukan');
		case 400:
		case 409:
		case 422:
		case 429:
			throw new ApiGagal(res.status, b);
		default:
			error(500, { message: 'Terjadi kesalahan pada server', requestId: b.request_id });
	}
}

/**
 * aksi membungkus isi form action. Hasilnya { ok } bila berhasil, atau
 * { gagal } = fail(status, {error, fields, ...ekstra}) bila Go menolak isian
 * (pesan per kolom ditampilkan halaman; `ekstra` mis. isian sebelumnya agar
 * formulir tidak kosong). redirect/error lain tetap dilempar.
 *
 *   const r = await aksi(() => api(...), { nilai });
 *   if (r.gagal) return r.gagal;
 */
export async function aksi<T, E extends Record<string, unknown> = Record<string, never>>(fn: () => Promise<T>, ekstra?: E) {
	try {
		return { ok: await fn() };
	} catch (e) {
		if (e instanceof ApiGagal) {
			return { gagal: fail(e.status, { error: e.body.error, fields: e.body.fields ?? {}, ...(ekstra as E) }) };
		}
		throw e;
	}
}

/** wajibAdmin: halaman khusus Admin dibalas 403 untuk Ketua (deny-by-default, FR-02). */
export function wajibAdmin(user: User | null): asserts user is User {
	if (user?.role !== 'admin') {
		// Judul "Akses tidak diizinkan" sudah ditulis +error.svelte; pesan cukup alasannya.
		error(403, 'Halaman ini hanya untuk Admin. Anda masuk sebagai Ketua DKM.');
	}
}

/** teks mengambil isian formulir sebagai string terpangkas ("" bila tidak ada). */
export function teks(fd: FormData, nama: string): string {
	const v = fd.get(nama);
	return typeof v === 'string' ? v.trim() : '';
}
