// Halaman satu program untuk pengurus: ringkasan dana, gambar sampul, ubah
// isian, dan kabar terbaru. Ketua hanya melihat; semua action khusus Admin.
// Setiap hasil action membawa `bagian` supaya halaman tahu formulir mana yang
// sedang menampilkan pesan sukses/galat.
import { fail, redirect } from '@sveltejs/kit';
import { isianProgram } from '$lib/program';
import { aksi, api, kirimBerkas, teks, wajibAdmin } from '$lib/server/api';
import type { KabarProgram, Program, ProgramDetail } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

/** Sama dengan service.GambarMaks di Go; dicek di sini juga supaya berkas besar tidak dikirim percuma. */
const GAMBAR_MAKS = 2 * 1024 * 1024;

export const load: PageServerLoad = async (event) => {
	return {
		program: await api<ProgramDetail>(event, 'GET', '/api/program/' + encodeURIComponent(event.params.id)),
		pesan: event.url.searchParams.get('dibuat') ? 'Program tersimpan. Tambahkan gambar sampul agar halaman publik lebih menarik.' : ''
	};
};

export const actions: Actions = {
	ubah: async (event) => {
		wajibAdmin(event.locals.user);
		const { body, nilai } = isianProgram(await event.request.formData());
		const r = await aksi(() => api<Program>(event, 'PUT', '/api/program/' + encodeURIComponent(event.params.id), body), {
			nilai,
			bagian: 'ubah'
		});
		if (r.gagal) return r.gagal;
		return { bagian: 'ubah', sukses: 'Perubahan program tersimpan.' };
	},

	gambar: async (event) => {
		wajibAdmin(event.locals.user);
		const berkas = (await event.request.formData()).get('gambar');
		if (!(berkas instanceof File) || berkas.size === 0) {
			return fail(422, { bagian: 'gambar', error: 'Periksa kembali isian', fields: { gambar: 'Pilih berkas gambar terlebih dahulu' } });
		}
		if (berkas.size > GAMBAR_MAKS) {
			return fail(422, { bagian: 'gambar', error: 'Periksa kembali isian', fields: { gambar: 'Ukuran gambar maksimal 2 MB' } });
		}
		const isi = await berkas.arrayBuffer();
		const r = await aksi(() => kirimBerkas(event, '/api/program/' + encodeURIComponent(event.params.id) + '/gambar', isi), {
			bagian: 'gambar'
		});
		if (r.gagal) return r.gagal;
		return { bagian: 'gambar', sukses: 'Gambar sampul diperbarui.' };
	},

	kabar: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const body = { judul: teks(fd, 'judul'), isi: teks(fd, 'isi') };
		const r = await aksi(
			() => api<KabarProgram>(event, 'POST', '/api/program/' + encodeURIComponent(event.params.id) + '/kabar', body),
			{ bagian: 'kabar', nilaiKabar: body }
		);
		if (r.gagal) return r.gagal;
		return { bagian: 'kabar', sukses: 'Kabar terbaru ditambahkan.' };
	},

	hapusKabar: async (event) => {
		wajibAdmin(event.locals.user);
		const kabar = teks(await event.request.formData(), 'kabar');
		await api(event, 'DELETE', `/api/program/${encodeURIComponent(event.params.id)}/kabar/${encodeURIComponent(kabar)}`);
		redirect(303, `/program-donasi/${event.params.id}`);
	}
};
