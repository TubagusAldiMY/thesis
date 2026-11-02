// Setup pertama kali: membuat akun Ketua pertama. Hanya hidup selama belum ada akun
// sama sekali; selebihnya dialihkan ke /login. Go juga menolak (404) bila sudah ada
// akun, jadi pengalihan ini hanya kenyamanan. Kode setup dicetak di log server.
import { fail, redirect } from '@sveltejs/kit';
import { aksi, api, teks } from '$lib/server/api';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const { aktif } = await api<{ aktif: boolean }>(event, 'GET', '/api/publik/setup');
	if (!aktif) redirect(303, '/login');
	return {};
};

export const actions: Actions = {
	default: async (event) => {
		const fd = await event.request.formData();
		const password = typeof fd.get('password') === 'string' ? (fd.get('password') as string) : '';
		const ulangi = typeof fd.get('ulangi') === 'string' ? (fd.get('ulangi') as string) : '';
		// Kode dan password tidak dipantulkan ulang ke formulir; hanya nama dan username.
		const nilai = { nama: teks(fd, 'nama'), username: teks(fd, 'username') };
		if (password !== ulangi) {
			return fail(422, { error: 'Periksa kembali isian', fields: { ulangi: 'Ulangi password tidak sama dengan password' } as Record<string, string>, nilai });
		}
		const r = await aksi(() => api(event, 'POST', '/api/publik/setup', { ...nilai, token: teks(fd, 'token'), password }), { nilai });
		if (r.gagal) return r.gagal;
		redirect(303, '/login?setup=1');
	}
};
