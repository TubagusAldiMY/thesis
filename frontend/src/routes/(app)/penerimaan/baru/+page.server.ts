// Catat donasi langsung di masjid (tunai/transfer) — khusus Admin. Langsung berstatus
// diterima dan kuitansi terbit. Donasi dari halaman publik tidak lewat sini.
import { redirect } from '@sveltejs/kit';
import { parseAngka } from '$lib/format';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { Donasi, Program } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibAdmin(event.locals.user);
	// Program yang boleh dipilih petugas: aktif dan selesai (setoran tunai bisa datang
	// setelah batas waktu); draf belum boleh menerima donasi. ?program=<id> memilihnya lebih dulu.
	const program = (await api<Program[]>(event, 'GET', '/api/program')).filter((p) => p.status !== 'draf');
	return { program, programAwal: event.url.searchParams.get('program') ?? '' };
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
			tanggal: teks(fd, 'tanggal'),
			program_id: teks(fd, 'program_id'),
			anonim: fd.get('anonim') === 'on'
		};
		// Isian dipantulkan saat gagal, KECUALI No. HP (data pribadi tidak dipantulkan ulang).
		const nilai = {
			nama: body.nama,
			jenis: body.jenis,
			jumlah: jumlahTeks,
			metode: body.metode,
			keterangan: body.keterangan,
			tanggal: body.tanggal,
			program_id: body.program_id,
			anonim: body.anonim
		};
		const r = await aksi(() => api<Donasi>(event, 'POST', '/api/donasi', body), { nilai });
		if (r.gagal) return r.gagal;
		redirect(303, `/penerimaan/${r.ok.id}?tercatat=1`);
	}
};
