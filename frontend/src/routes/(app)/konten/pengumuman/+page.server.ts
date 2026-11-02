// Pengumuman singkat dengan masa tayang, untuk pengurus: daftar + formulir tambah/ubah
// di satu halaman. ?ubah=<id> mengisi formulir dengan pengumuman itu.
import { redirect } from '@sveltejs/kit';
import { aksi, api, teks } from '$lib/server/api';
import type { Pengumuman } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const daftar = await api<Pengumuman[]>(event, 'GET', '/api/pengumuman');
	const ubahId = event.url.searchParams.get('ubah') ?? '';
	const sp = event.url.searchParams;
	return {
		daftar,
		sedangDiubah: daftar.find((p) => p.id === ubahId) ?? null,
		pesan: sp.get('tersimpan') ? 'Pengumuman tersimpan.' : sp.get('dihapus') ? 'Pengumuman dihapus.' : ''
	};
};

export const actions: Actions = {
	// Tanpa `id` di formulir = pengumuman baru; dengan `id` = ubah pengumuman itu.
	simpan: async (event) => {
		const fd = await event.request.formData();
		const id = teks(fd, 'id');
		const body = { judul: teks(fd, 'judul'), isi: teks(fd, 'isi'), tampil_sampai: teks(fd, 'tampil_sampai'), terbit: fd.get('terbit') === 'on' };
		const nilai = { ...body, id };
		const r = await aksi(
			() => (id ? api(event, 'PUT', '/api/pengumuman/' + encodeURIComponent(id), body) : api(event, 'POST', '/api/pengumuman', body)),
			{ nilai }
		);
		if (r.gagal) return r.gagal;
		redirect(303, '/konten/pengumuman?tersimpan=1');
	},

	hapus: async (event) => {
		const id = teks(await event.request.formData(), 'id');
		await api(event, 'DELETE', '/api/pengumuman/' + encodeURIComponent(id));
		redirect(303, '/konten/pengumuman?dihapus=1');
	}
};
