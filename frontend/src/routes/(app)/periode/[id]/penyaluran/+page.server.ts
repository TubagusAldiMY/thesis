// Tab Penyaluran (FR-13): catat realisasi per penerima, lalu tutup periode.
import { redirect } from '@sveltejs/kit';
import { parseAngka } from '$lib/format';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { RekapPenyaluran } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	rekap: await api<RekapPenyaluran>(event, 'GET', `/api/periode/${encodeURIComponent(event.params.id)}/penyaluran`)
});

export const actions: Actions = {
	catat: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const mustahikID = teks(fd, 'mustahik_id');
		const nominal = parseAngka(teks(fd, 'nominal'));
		const r = await aksi(
			() =>
				api(event, 'POST', `/api/periode/${encodeURIComponent(event.params.id)}/penyaluran`, {
					mustahik_id: mustahikID,
					nominal: Number.isNaN(nominal) ? 0 : nominal,
					tanggal: teks(fd, 'tanggal')
				}),
			{ mustahikID }
		);
		if (r.gagal) return r.gagal;
		return { dicatat: mustahikID };
	},
	tutup: async (event) => {
		wajibAdmin(event.locals.user);
		const r = await aksi(() => api(event, 'POST', `/api/periode/${encodeURIComponent(event.params.id)}/tutup`));
		if (r.gagal) return r.gagal;
		redirect(303, `/periode/${event.params.id}/laporan`);
	}
};
