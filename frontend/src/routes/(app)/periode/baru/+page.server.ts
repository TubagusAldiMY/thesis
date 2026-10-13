import { redirect } from '@sveltejs/kit';
import { parseAngka } from '$lib/format';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { Periode } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = ({ locals }) => {
	wajibAdmin(locals.user);
};

export const actions: Actions = {
	default: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const nilai = {
			nama: teks(fd, 'nama'),
			jenis: teks(fd, 'jenis'),
			dana: teks(fd, 'dana_tersedia'),
			nominal: teks(fd, 'nominal_per_penerima'),
			asnaf: fd.getAll('asnaf_diizinkan').filter((v): v is string => typeof v === 'string')
		};
		// Angka boleh diketik "3.000.000"; NaN → 0 supaya Go yang menolak dengan pesan per kolom.
		const angkaAtauNol = (s: string) => {
			const n = parseAngka(s);
			return Number.isNaN(n) ? 0 : n;
		};
		const r = await aksi(
			() =>
				api<Periode>(event, 'POST', '/api/periode', {
					nama: nilai.nama,
					jenis: nilai.jenis,
					dana_tersedia: angkaAtauNol(nilai.dana),
					nominal_per_penerima: angkaAtauNol(nilai.nominal),
					asnaf_diizinkan: nilai.jenis === 'zakat' ? nilai.asnaf : []
				}),
			{ nilai }
		);
		if (r.gagal) return r.gagal;
		redirect(303, `/periode/${r.ok.id}/kandidat`);
	}
};
