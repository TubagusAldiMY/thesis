// Catat donasi langsung di masjid (tunai/transfer) — khusus Admin. Langsung berstatus
// diterima dan kuitansi terbit. Donasi dari halaman publik tidak lewat sini.
import { redirect } from '@sveltejs/kit';
import { parseAngka } from '$lib/format';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { Donasi } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibAdmin(event.locals.user);
	return {};
};

export const actions: Actions = {
	default: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const jumlahTeks = teks(fd, 'jumlah');
		const jumlah = parseAngka(jumlahTeks);
		const body = {
			nama: teks(fd, 'nama'),
			no_hp: teks(fd, 'no_hp'),
			jenis: teks(fd, 'jenis'),
			jumlah: Number.isFinite(jumlah) ? jumlah : 0,
			metode: teks(fd, 'metode'),
			keterangan: teks(fd, 'keterangan'),
			tanggal: teks(fd, 'tanggal')
		};
		// Isian dipantulkan saat gagal, KECUALI No. HP (data pribadi tidak dipantulkan ulang).
		const nilai = {
			nama: body.nama,
			jenis: body.jenis,
			jumlah: jumlahTeks,
			metode: body.metode,
			keterangan: body.keterangan,
			tanggal: body.tanggal
		};
		const r = await aksi(() => api<Donasi>(event, 'POST', '/api/donasi', body), { nilai });
		if (r.gagal) return r.gagal;
		redirect(303, `/penerimaan/${r.ok.id}?tercatat=1`);
	}
};
