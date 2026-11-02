// Kelola satu akun — khusus Ketua: ubah nama & peran, reset password, nonaktifkan
// atau aktifkan kembali. Aturan (tidak boleh menyentuh akun sendiri, minimal satu
// Admin aktif) dijaga Go; di sini galatnya (409) hanya ditampilkan.
import { fail } from '@sveltejs/kit';
import { aksi, api, teks, wajibKetua } from '$lib/server/api';
import type { Pengguna } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

const jalur = (id: string) => '/api/pengguna/' + encodeURIComponent(id);

export const load: PageServerLoad = async (event) => {
	wajibKetua(event.locals.user);
	return { akun: await api<Pengguna>(event, 'GET', jalur(event.params.id)) };
};

export const actions: Actions = {
	ubah: async (event) => {
		wajibKetua(event.locals.user);
		const fd = await event.request.formData();
		const nilai = { nama: teks(fd, 'nama'), role: teks(fd, 'role') };
		const r = await aksi(() => api<Pengguna>(event, 'PUT', jalur(event.params.id), nilai), { bagian: 'ubah', nilai });
		if (r.gagal) return r.gagal;
		return { bagian: 'ubah', sukses: 'Perubahan tersimpan.' };
	},

	reset: async (event) => {
		wajibKetua(event.locals.user);
		const fd = await event.request.formData();
		const password = typeof fd.get('password') === 'string' ? (fd.get('password') as string) : '';
		const ulangi = typeof fd.get('ulangi') === 'string' ? (fd.get('ulangi') as string) : '';
		if (password !== ulangi) {
			return fail(422, { bagian: 'reset', error: 'Periksa kembali isian', fields: { ulangi: 'Ulangi password tidak sama dengan password' } as Record<string, string> });
		}
		const r = await aksi(() => api(event, 'POST', jalur(event.params.id) + '/reset-password', { password }), { bagian: 'reset' });
		if (r.gagal) return r.gagal;
		return { bagian: 'reset', sukses: 'Password diganti dan semua sesi akun ini dikeluarkan. Sampaikan password baru langsung kepada yang bersangkutan.' };
	},

	nonaktif: async (event) => {
		wajibKetua(event.locals.user);
		const r = await aksi(() => api(event, 'POST', jalur(event.params.id) + '/nonaktif'), { bagian: 'status' });
		if (r.gagal) return r.gagal;
		return { bagian: 'status', sukses: 'Akun dinonaktifkan dan semua sesinya dikeluarkan.' };
	},

	aktif: async (event) => {
		wajibKetua(event.locals.user);
		const r = await aksi(() => api(event, 'POST', jalur(event.params.id) + '/aktif'), { bagian: 'status' });
		if (r.gagal) return r.gagal;
		return { bagian: 'status', sukses: 'Akun diaktifkan kembali.' };
	}
};
