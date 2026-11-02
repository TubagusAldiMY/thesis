// Jadwal rutin mingguan (kajian dll.) untuk pengurus: daftar + formulir tambah/ubah
// di satu halaman. ?ubah=<id> mengisi formulir dengan jadwal itu.
import { redirect } from '@sveltejs/kit';
import { aksi, api, teks } from '$lib/server/api';
import type { JadwalRutin } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	const jadwal = await api<JadwalRutin[]>(event, 'GET', '/api/jadwal-rutin');
	const ubahId = event.url.searchParams.get('ubah') ?? '';
	const sp = event.url.searchParams;
	return {
		jadwal,
		sedangDiubah: jadwal.find((j) => j.id === ubahId) ?? null,
		pesan: sp.get('tersimpan') ? 'Jadwal tersimpan.' : sp.get('dihapus') ? 'Jadwal dihapus.' : ''
	};
};

export const actions: Actions = {
	// Tanpa `id` di formulir = jadwal baru; dengan `id` = ubah jadwal itu.
	simpan: async (event) => {
		const fd = await event.request.formData();
		const id = teks(fd, 'id');
		const hari = Number(teks(fd, 'hari'));
		const body = {
			judul: teks(fd, 'judul'),
			pemateri: teks(fd, 'pemateri'),
			hari: Number.isInteger(hari) ? hari : -1, // -1 → Go menolak dengan pesan "Pilih hari"
			jam: teks(fd, 'jam'),
			lokasi: teks(fd, 'lokasi'),
			keterangan: teks(fd, 'keterangan'),
			aktif: fd.get('aktif') === 'on'
		};
		const nilai = { ...body, id, hari: String(body.hari) };
		const r = await aksi(
			() => (id ? api(event, 'PUT', '/api/jadwal-rutin/' + encodeURIComponent(id), body) : api(event, 'POST', '/api/jadwal-rutin', body)),
			{ nilai }
		);
		if (r.gagal) return r.gagal;
		redirect(303, '/konten/jadwal?tersimpan=1');
	},

	hapus: async (event) => {
		const id = teks(await event.request.formData(), 'id');
		await api(event, 'DELETE', '/api/jadwal-rutin/' + encodeURIComponent(id));
		redirect(303, '/konten/jadwal?dihapus=1');
	}
};
