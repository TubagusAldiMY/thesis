// Kelola satu kegiatan (Admin & Ketua): ubah isian/status, unggah gambar sampul, hapus.
// Setiap hasil action membawa `bagian` supaya halaman tahu formulir mana yang menampilkan pesan.
import { fail, redirect } from '@sveltejs/kit';
import { isianKegiatan } from '$lib/kegiatan';
import { aksi, api, kirimBerkas } from '$lib/server/api';
import type { Kegiatan, KategoriKegiatan } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

/** Sama dengan service.GambarMaks di Go; dicek di sini juga supaya berkas besar tidak dikirim percuma. */
const GAMBAR_MAKS = 2 * 1024 * 1024;

const jalur = (id: string) => '/api/kegiatan/' + encodeURIComponent(id);

export const load: PageServerLoad = async (event) => ({
	kegiatan: await api<Kegiatan>(event, 'GET', jalur(event.params.id)),
	kategori: await api<KategoriKegiatan[]>(event, 'GET', '/api/kegiatan/kategori'),
	pesan: event.url.searchParams.get('dibuat') ? 'Kegiatan tersimpan. Tambahkan gambar sampul agar halaman publik lebih menarik.' : ''
});

export const actions: Actions = {
	ubah: async (event) => {
		const nilai = isianKegiatan(await event.request.formData());
		const r = await aksi(() => api<Kegiatan>(event, 'PUT', jalur(event.params.id), nilai), { bagian: 'ubah', nilai });
		if (r.gagal) return r.gagal;
		return { bagian: 'ubah', sukses: 'Perubahan tersimpan.' };
	},

	gambar: async (event) => {
		const berkas = (await event.request.formData()).get('gambar');
		if (!(berkas instanceof File) || berkas.size === 0) {
			return fail(422, { bagian: 'gambar', error: 'Periksa kembali isian', fields: { gambar: 'Pilih berkas gambar terlebih dahulu' } });
		}
		if (berkas.size > GAMBAR_MAKS) {
			return fail(422, { bagian: 'gambar', error: 'Periksa kembali isian', fields: { gambar: 'Ukuran gambar maksimal 2 MB' } });
		}
		const isi = await berkas.arrayBuffer();
		const r = await aksi(() => kirimBerkas(event, jalur(event.params.id) + '/gambar', isi), { bagian: 'gambar' });
		if (r.gagal) return r.gagal;
		return { bagian: 'gambar', sukses: 'Gambar sampul diperbarui.' };
	},

	hapus: async (event) => {
		await api(event, 'DELETE', jalur(event.params.id));
		redirect(303, '/konten?dihapus=1');
	}
};
