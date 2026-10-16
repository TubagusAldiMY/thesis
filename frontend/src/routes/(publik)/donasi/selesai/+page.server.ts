// Status donasi untuk muzaki. Menjadi tujuan pengalihan setelah muzaki selesai di
// halaman payment gateway. Parameter yang ditambahkan gateway (order_id,
// transaction_status, ...) SENGAJA diabaikan: tidak tepercaya. Status yang
// ditampilkan selalu dibaca dari Go, yang hanya berubah lewat notifikasi
// ber-tanda-tangan dari gateway.
import { redirect } from '@sveltejs/kit';
import { api } from '$lib/server/api';
import type { DonasiPublik } from '$lib/types';
import type { PageServerLoad } from './$types';

const reKode = /^[0-9a-f]{32}$/;

export const load: PageServerLoad = async (event) => {
	const kode = event.url.searchParams.get('kode') ?? '';
	if (!reKode.test(kode)) redirect(303, '/donasi');
	return { p: await api<DonasiPublik>(event, 'GET', `/api/publik/donasi/${kode}`) };
};
