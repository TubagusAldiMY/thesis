import { fail } from '@sveltejs/kit';
import { parseDesimal } from '$lib/format';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { Kriteria } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

type DaftarKriteria = { items: Kriteria[]; total_persen: number };

export const load: PageServerLoad = async (event) => ({
	kriteria: await api<DaftarKriteria>(event, 'GET', '/api/kriteria')
});

export const actions: Actions = {
	// Semua baris dikirim sekaligus: syarat Σ bobot = 100% berlaku atas satu paket utuh (FR-05).
	default: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const ids = fd.getAll('id').filter((v): v is string => typeof v === 'string');
		const items = ids.map((id) => ({
			id,
			tipe: teks(fd, `tipe.${id}`),
			bobot: parseDesimal(teks(fd, `bobot.${id}`))
		}));
		if (items.some((it) => Number.isNaN(it.bobot))) {
			return fail(422, { error: 'Periksa kembali isian', fields: { bobot: 'Bobot harus angka, mis. 0,30' } as Record<string, string> });
		}
		const r = await aksi(() => api(event, 'PUT', '/api/kriteria', { items }));
		if (r.gagal) return r.gagal;
		return { tersimpan: true };
	}
};
