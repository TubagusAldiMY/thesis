// Penerima notifikasi pembayaran dari Midtrans. Go API hanya listen di 127.0.0.1
// (tidak publik), jadi alamat webhook yang didaftarkan di dashboard Midtrans
// adalah halaman ini: https://<domain-masjid>/midtrans/notifikasi.
//
// Rute ini hanya MENERUSKAN body mentah ke Go. Keasliannya tidak dicek di sini:
// Go memverifikasi tanda tangan SHA512 (order_id + status_code + gross_amount +
// ServerKey), memastikan jumlahnya cocok, dan menjaga idempotensi. Body JSON dari
// gateway tidak terkena pemeriksaan Origin SvelteKit (itu hanya untuk submit form).
import { teruskan } from '$lib/server/api';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async (event) => {
	const body = await event.request.text();
	const res = await teruskan(event, '/api/publik/midtrans/notifikasi', body);
	return new Response(await res.text(), {
		status: res.status,
		headers: { 'Content-Type': 'application/json; charset=utf-8' }
	});
};
