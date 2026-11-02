// Tambah pengguna — khusus Ketua. Password awal ditetapkan Ketua dan disampaikan
// langsung kepada yang bersangkutan; yang bersangkutan menggantinya lewat Profil.
import { fail, redirect } from '@sveltejs/kit';
import { aksi, api, teks, wajibKetua } from '$lib/server/api';
import type { Pengguna } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibKetua(event.locals.user);
	return {};
};

export const actions: Actions = {
	default: async (event) => {
		wajibKetua(event.locals.user);
		const fd = await event.request.formData();
		const password = typeof fd.get('password') === 'string' ? (fd.get('password') as string) : '';
		const ulangi = typeof fd.get('ulangi') === 'string' ? (fd.get('ulangi') as string) : '';
		// Password tidak dipantulkan ulang ke formulir; hanya username, nama, dan peran.
		const nilai = { username: teks(fd, 'username'), nama: teks(fd, 'nama'), role: teks(fd, 'role') };
		if (password !== ulangi) {
			return fail(422, { error: 'Periksa kembali isian', fields: { ulangi: 'Ulangi password tidak sama dengan password' } as Record<string, string>, nilai });
		}
		const r = await aksi(() => api<Pengguna>(event, 'POST', '/api/pengguna', { ...nilai, password }), { nilai });
		if (r.gagal) return r.gagal;
		redirect(303, '/pengguna?dibuat=1');
	}
};
