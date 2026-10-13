// Formulir tambah (/mustahik/baru) dan ubah (/mustahik/<id>) mustahik.
// NIK/No. HP asli tidak pernah dikirim ke browser; saat mengubah, kolom
// kosong berarti "tidak diubah" (aturan di Go service).
import { redirect } from '@sveltejs/kit';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { Mustahik } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibAdmin(event.locals.user);
	const baru = event.params.id === 'baru';
	return {
		baru,
		mustahik: baru ? null : await api<Mustahik>(event, 'GET', `/api/mustahik/${encodeURIComponent(event.params.id)}`)
	};
};

export const actions: Actions = {
	default: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const body = {
			nama: teks(fd, 'nama'),
			nik: teks(fd, 'nik'),
			no_hp: teks(fd, 'no_hp'),
			alamat: teks(fd, 'alamat'),
			asnaf: teks(fd, 'asnaf'),
			aktif: fd.get('aktif') === 'on'
		};
		const baru = event.params.id === 'baru';
		// Isian dikembalikan saat gagal, KECUALI NIK/HP (data pribadi tidak dipantulkan ulang).
		const nilai = { nama: body.nama, alamat: body.alamat, asnaf: body.asnaf, aktif: body.aktif };
		const r = await aksi(
			() =>
				baru
					? api(event, 'POST', '/api/mustahik', body)
					: api(event, 'PUT', `/api/mustahik/${encodeURIComponent(event.params.id)}`, body),
			{ nilai }
		);
		if (r.gagal) return r.gagal;
		redirect(303, '/mustahik?tersimpan=1');
	}
};
