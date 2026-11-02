// Buat program donasi baru — khusus Admin. Setelah tersimpan diarahkan ke
// halaman program untuk mengunggah gambar sampul dan menulis kabar.
import { redirect } from '@sveltejs/kit';
import { isianProgram } from '$lib/program';
import { aksi, api, wajibAdmin } from '$lib/server/api';
import type { Program } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibAdmin(event.locals.user);
	return {};
};

export const actions: Actions = {
	default: async (event) => {
		wajibAdmin(event.locals.user);
		const { body, nilai } = isianProgram(await event.request.formData());
		const r = await aksi(() => api<Program>(event, 'POST', '/api/program', body), { nilai });
		if (r.gagal) return r.gagal;
		redirect(303, `/program-donasi/${r.ok.id}?dibuat=1`);
	}
};
