// Halaman perantara setelah form donasi dikirim. Mengapa tidak langsung dialihkan
// ke domain gateway dari form action? CSP aplikasi memakai `form-action 'self'`, dan
// peramban (Chrome) juga menerapkannya pada pengalihan setelah pengiriman form:
// pengalihan 303 langsung ke domain lain akan DIBLOKIR pada form tanpa JavaScript.
// Jadi form dialihkan ke halaman ini (domain sendiri), lalu halaman ini yang
// membawa muzaki ke gateway lewat navigasi biasa (meta refresh + tautan).
import { redirect } from '@sveltejs/kit';
import { api } from '$lib/server/api';
import type { DonasiPublik } from '$lib/types';
import type { PageServerLoad } from './$types';

const reKode = /^[0-9a-f]{32}$/;

export const load: PageServerLoad = async (event) => {
	const kode = event.url.searchParams.get('kode') ?? '';
	if (!reKode.test(kode)) redirect(303, '/donasi');
	const p = await api<DonasiPublik>(event, 'GET', `/api/publik/donasi/${kode}`);
	// Sudah dibayar/kedaluwarsa/gagal (atau tautan bayar hilang): tidak ada yang perlu dibayar lagi.
	if (p.status !== 'menunggu' || !p.bayar_url) redirect(303, '/donasi/selesai?kode=' + kode);
	return { p };
};
