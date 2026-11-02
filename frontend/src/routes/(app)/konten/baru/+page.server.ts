// Buat kegiatan baru (Admin & Ketua). Setelah tersimpan diarahkan ke halaman
// kegiatan untuk mengunggah gambar sampul.
import { redirect } from '@sveltejs/kit';
import { isianKegiatan } from '$lib/kegiatan';
import { aksi, api } from '$lib/server/api';
import type { Kegiatan, KategoriKegiatan } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	kategori: await api<KategoriKegiatan[]>(event, 'GET', '/api/kegiatan/kategori')
});

export const actions: Actions = {
	default: async (event) => {
		const nilai = isianKegiatan(await event.request.formData());
		const r = await aksi(() => api<Kegiatan>(event, 'POST', '/api/kegiatan', nilai), { nilai });
		if (r.gagal) return r.gagal;
		redirect(303, `/konten/${r.ok.id}?dibuat=1`);
	}
};
