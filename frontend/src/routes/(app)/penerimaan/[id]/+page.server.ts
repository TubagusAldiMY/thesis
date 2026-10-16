// Detail satu donasi: data, tautan kuitansi, kirim via WhatsApp (Admin), dan pembatalan (Admin).
// No. HP lengkap hanya dikirim Go untuk Admin; Ketua hanya menerima versi tersamar.
import { redirect } from '@sveltejs/kit';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { Donasi } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	donasi: await api<Donasi>(event, 'GET', `/api/donasi/${encodeURIComponent(event.params.id)}`),
	tercatat: event.url.searchParams.get('tercatat') === '1',
	dibatalkan: event.url.searchParams.get('dibatalkan') === '1'
});

export const actions: Actions = {
	batal: async (event) => {
		wajibAdmin(event.locals.user);
		const alasan = teks(await event.request.formData(), 'alasan');
		const id = encodeURIComponent(event.params.id);
		const r = await aksi(() => api(event, 'POST', `/api/donasi/${id}/batal`, { alasan }), { alasan });
		if (r.gagal) return r.gagal;
		redirect(303, `/penerimaan/${id}?dibatalkan=1`);
	}
};
