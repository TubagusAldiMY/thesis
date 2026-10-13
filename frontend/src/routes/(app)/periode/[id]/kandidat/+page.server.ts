// Tab Kandidat (FR-07): centang mustahik aktif yang golongannya diizinkan periode.
import { redirect } from '@sveltejs/kit';
import { aksi, api, wajibAdmin } from '$lib/server/api';
import type { Mustahik, MustahikHalaman } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibAdmin(event.locals.user);
	const { detail } = await event.parent();
	const p = detail.periode;

	// Ambil semua mustahik aktif (API membatasi 100 per halaman).
	// ponytail: beberapa permintaan berurutan; cukup untuk ≤ 500 mustahik (NFR-06).
	const semua: Mustahik[] = [];
	for (let page = 1; ; page++) {
		const h = await api<MustahikHalaman>(event, 'GET', `/api/mustahik?aktif=true&per_page=100&page=${page}`);
		semua.push(...h.items);
		if (semua.length >= h.total || h.items.length === 0) break;
	}
	const diizinkan = (m: Mustahik) => p.jenis === 'infak' || (p.asnaf_diizinkan ?? []).includes(m.asnaf);
	const { items: kandidat } = await api<{ items: Mustahik[] }>(event, 'GET', `/api/periode/${p.id}/kandidat`);
	const layak = semua.filter(diizinkan);
	const idLayak = new Set(layak.map((m) => m.id));

	return {
		layak,
		terpilih: kandidat.map((m) => m.id),
		// Kandidat lama yang kini nonaktif/berganti golongan: harus dibuang sebelum dihitung.
		tidakLayak: kandidat.filter((m) => !idLayak.has(m.id) || !m.aktif)
	};
};

export const actions: Actions = {
	default: async (event) => {
		wajibAdmin(event.locals.user);
		const fd = await event.request.formData();
		const ids = fd.getAll('mustahik_id').filter((v): v is string => typeof v === 'string');
		const r = await aksi(() => api(event, 'PUT', `/api/periode/${encodeURIComponent(event.params.id)}/kandidat`, { mustahik_ids: ids }));
		if (r.gagal) return r.gagal;
		if (fd.get('lanjut') === '1') redirect(303, `/periode/${event.params.id}/penilaian`);
		return { tersimpan: true };
	}
};
