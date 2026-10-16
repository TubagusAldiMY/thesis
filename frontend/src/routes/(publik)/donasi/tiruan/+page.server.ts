// SIMULASI pembayaran — hanya untuk development/demo (PAYMENT_PROVIDER=tiruan di
// Go). Pada penyedia asli (Midtrans) endpoint Go yang dipanggil di sini tidak ada,
// sehingga halaman ini menjadi 404. Tombolnya memicu jalur notifikasi yang sama
// dengan gateway sungguhan, jadi seluruh alur sampai kuitansi bisa dicoba tanpa akun Midtrans.
import { redirect } from '@sveltejs/kit';
import { api } from '$lib/server/api';
import type { DonasiPublik } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

const reKode = /^[0-9a-f]{32}$/;

const kodeDari = (url: URL) => {
	const kode = url.searchParams.get('kode') ?? '';
	if (!reKode.test(kode)) redirect(303, '/donasi');
	return kode;
};

export const load: PageServerLoad = async (event) => {
	const kode = kodeDari(event.url);
	const p = await api<DonasiPublik>(event, 'GET', `/api/publik/donasi/${kode}`);
	if (p.status !== 'menunggu') redirect(303, '/donasi/selesai?kode=' + kode);
	return { p };
};

export const actions: Actions = {
	default: async (event) => {
		const kode = kodeDari(event.url);
		await api(event, 'POST', `/api/publik/tiruan/${kode}/bayar`);
		redirect(303, '/donasi/selesai?kode=' + kode);
	}
};
