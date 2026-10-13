// Tab Penilaian (FR-08) + tombol Hitung ARAS (FR-09).
// Isian formulir bernama "nilai.<mustahikID>.<kodeKriteria>"; sel kosong dilewati
// (boleh disimpan sebagian). "Hitung" menyimpan dulu, lalu menghitung.
import { fail, redirect, type RequestEvent } from '@sveltejs/kit';
import { parseNilai } from '$lib/format';
import { aksi, api, wajibAdmin } from '$lib/server/api';
import type { PenilaianMatriks } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	wajibAdmin(event.locals.user);
	return {
		matriks: await api<PenilaianMatriks>(event, 'GET', `/api/periode/${encodeURIComponent(event.params.id)}/penilaian`)
	};
};

/** bacaNilai mengubah isian formulir menjadi paket penilaian per mustahik. */
function bacaNilai(fd: FormData) {
	const per: Record<string, Record<string, number>> = {};
	const salah: Record<string, string> = {};
	for (const [nama, v] of fd.entries()) {
		if (!nama.startsWith('nilai.') || typeof v !== 'string' || v.trim() === '') continue;
		const [, mid, kode] = nama.split('.');
		const n = parseNilai(v);
		if (!mid || !kode) continue;
		if (Number.isNaN(n)) {
			salah[nama] = `Nilai ${kode} harus angka (titik = ribuan, koma = desimal)`;
			continue;
		}
		(per[mid] ??= {})[kode] = n;
	}
	return { items: Object.entries(per).map(([mustahik_id, nilai]) => ({ mustahik_id, nilai })), salah };
}

async function simpan(event: RequestEvent, fd: FormData) {
	const { items, salah } = bacaNilai(fd);
	if (Object.keys(salah).length > 0) return { gagal: fail(422, { error: 'Periksa kembali isian', fields: salah }) };
	if (items.length === 0) return { ok: null };
	return aksi(() => api(event, 'PUT', `/api/periode/${encodeURIComponent(event.params.id as string)}/penilaian`, { items }));
}

export const actions: Actions = {
	simpan: async (event) => {
		wajibAdmin(event.locals.user);
		const r = await simpan(event, await event.request.formData());
		if (r.gagal) return r.gagal;
		return { tersimpan: true };
	},
	hitung: async (event) => {
		wajibAdmin(event.locals.user);
		const r = await simpan(event, await event.request.formData());
		if (r.gagal) return r.gagal;
		const h = await aksi(() => api(event, 'POST', `/api/periode/${encodeURIComponent(event.params.id)}/hitung`));
		if (h.gagal) return h.gagal;
		redirect(303, `/periode/${event.params.id}/hasil?dihitung=1`);
	}
};
