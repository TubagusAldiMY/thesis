// Profil & password (FR-21): password lama wajib, baru ≥ 12 karakter; Go
// mencabut sesi lain milik user ini, sesi yang dipakai sekarang tetap hidup.
import { fail } from '@sveltejs/kit';
import { aksi, api } from '$lib/server/api';
import type { Actions } from './$types';

export const actions: Actions = {
	default: async (event) => {
		const fd = await event.request.formData();
		const ambil = (n: string) => (typeof fd.get(n) === 'string' ? (fd.get(n) as string) : '');
		const lama = ambil('password_lama');
		const baru = ambil('password_baru');
		if (baru !== ambil('ulangi')) {
			return fail(422, { error: 'Periksa kembali isian', fields: { ulangi: 'Ulangi password tidak sama dengan password baru' } as Record<string, string> });
		}
		const r = await aksi(() => api(event, 'PUT', '/api/auth/password', { password_lama: lama, password_baru: baru }));
		if (r.gagal) return r.gagal;
		return { tersimpan: true };
	}
};
