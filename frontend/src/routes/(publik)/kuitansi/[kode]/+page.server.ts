// Kuitansi donasi untuk muzaki (tanpa login). Aksesnya hanya lewat kode rahasia
// 128 bit di tautan; kode yang salah format diperlakukan sama dengan yang tidak ada (404).
// Kode ini tidak boleh masuk log (lihat pathLog di hooks.server.ts).
import { error } from '@sveltejs/kit';
import { api } from '$lib/server/api';
import type { DonasiPublik } from '$lib/types';
import type { PageServerLoad } from './$types';

const reKode = /^[0-9a-f]{32}$/;

export const load: PageServerLoad = async (event) => {
	if (!reKode.test(event.params.kode)) error(404, 'Kuitansi tidak ditemukan');
	return { p: await api<DonasiPublik>(event, 'GET', `/api/publik/donasi/${event.params.kode}`) };
};
