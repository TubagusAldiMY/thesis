import { fail } from '@sveltejs/kit';
import { parseDesimal } from '$lib/format';
import { parseSkala } from '$lib/kriteria';
import { aksi, api, teks, wajibAdmin } from '$lib/server/api';
import type { Kriteria } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

type DaftarKriteria = { items: Kriteria[]; total_persen: number };

export const load: PageServerLoad = async (event) => ({
	kriteria: await api<DaftarKriteria>(event, 'GET', '/api/kriteria')
});

export const actions: Actions = {
	// Semua baris dikirim sekaligus (yang sudah ada + yang baru): syarat Σ bobot = 100%,
	// jumlah kriteria aktif, dan nama unik berlaku atas satu paket utuh (FR-05).
	default: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const kunci = (nama: string) => fd.getAll(nama).filter((v): v is string => typeof v === 'string');

		// Satu baris isian; kunci = ID kriteria (yang ada) atau kunci sementara (yang baru).
		const baca = (k: string) => {
			const s = parseSkala(teks(fd, `skala.${k}`));
			return {
				nama: teks(fd, `nama.${k}`),
				tipe: teks(fd, `tipe.${k}`),
				bobot: parseDesimal(teks(fd, `bobot.${k}`)),
				skala: 'skala' in s ? s.skala : null,
				galatSkala: 'galat' in s ? s.galat : '',
				aktif: fd.get(`aktif.${k}`) === 'on'
			};
		};
		const items = kunci('id').map((id) => ({ id, ...baca(id) }));
		const baru = kunci('baru').map((k) => ({ k, ...baca(k) }));

		const salahSkala = [...items, ...baru].find((r) => r.galatSkala);
		if (salahSkala) {
			return fail(422, { error: 'Periksa kembali isian', fields: { skala: salahSkala.galatSkala } as Record<string, string> });
		}
		if ([...items, ...baru].some((r) => Number.isNaN(r.bobot))) {
			return fail(422, { error: 'Periksa kembali isian', fields: { bobot: 'Bobot harus angka, mis. 0,30' } as Record<string, string> });
		}

		const badan = {
			items: items.map(({ id, nama, tipe, bobot, skala, aktif }) => ({ id, nama, tipe, bobot, skala, aktif })),
			baru: baru.map(({ nama, tipe, bobot, skala }) => ({ nama, tipe, bobot, skala }))
		};
		const r = await aksi(() => api(event, 'PUT', '/api/kriteria', badan));
		if (r.gagal) return r.gagal;
		return { tersimpan: true };
	}
};
