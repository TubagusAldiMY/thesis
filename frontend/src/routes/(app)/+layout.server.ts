// Penjaga semua halaman di grup (app): wajib masuk. Belum masuk → /login
// dengan alamat tujuan dibawa supaya setelah login kembali ke halaman ini.
import { redirect, type RequestEvent } from '@sveltejs/kit';
import { mentah } from '$lib/server/api';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async (event) => {
	if (!event.locals.user) {
		redirect(303, '/login?next=' + encodeURIComponent(event.url.pathname + event.url.search));
	}
	return { user: event.locals.user, donasiBaru: await donasiBaru(event) };
};

/**
 * Penanda notifikasi di menu Donasi: banyaknya donasi web yang sudah diterima tetapi
 * belum dilihat pengurus. Hanya hiasan, jadi galat apa pun (API mati, dsb.) menjadi 0
 * dan tidak boleh menjatuhkan halaman yang sedang dibuka.
 */
async function donasiBaru(event: RequestEvent): Promise<number> {
	try {
		const res = await mentah(event, 'GET', '/api/donasi/ringkas');
		if (!res.ok) return 0;
		return ((await res.json()) as { baru?: number }).baru ?? 0;
	} catch {
		return 0;
	}
}
