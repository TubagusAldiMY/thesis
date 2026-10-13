// Tab Hasil: ranking ARAS + langkah hitung + panel keputusan Ketua (FR-10/12).
// Juga menampung aksi "Buka kembali" (dipanggil dari kartu ditolak di layout).
import { error, redirect } from '@sveltejs/kit';
import { aksi, api, apiOpsional, teks, wajibAdmin } from '$lib/server/api';
import type { HasilAras } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const versi = Number(event.url.searchParams.get('versi')) || 0;
	const hasil = await apiOpsional<HasilAras>(event, `/api/periode/${encodeURIComponent(event.params.id)}/hasil?versi=${versi}`);
	if (!hasil && versi > 0) error(404, `Run v${versi} tidak ditemukan`);
	return { hasil, versi, baruDihitung: event.url.searchParams.get('dihitung') === '1' };
};

export const actions: Actions = {
	keputusan: async (event) => {
		const fd = await event.request.formData();
		const body = { keputusan: teks(fd, 'keputusan'), catatan: teks(fd, 'catatan') };
		const r = await aksi(() => api(event, 'POST', `/api/periode/${encodeURIComponent(event.params.id)}/keputusan`, body), {
			catatan: body.catatan
		});
		if (r.gagal) return r.gagal;
		return { diputuskan: body.keputusan };
	},
	bukaKembali: async (event) => {
		wajibAdmin(event.locals.user);
		const r = await aksi(() => api(event, 'POST', `/api/periode/${encodeURIComponent(event.params.id)}/buka-kembali`));
		if (r.gagal) return r.gagal;
		redirect(303, `/periode/${event.params.id}/penilaian`);
	}
};
