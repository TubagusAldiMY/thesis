// Landing publik: memuat angka transparansi (agregat tanpa data pribadi) dari Go.
// Bila API gagal, landing tetap tampil tanpa bagian itu: halaman depan masjid
// tidak boleh ikut rusak hanya karena angka tidak termuat.
import { api } from '$lib/server/api';
import type { RingkasanPublik } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	try {
		return { ringkasan: await api<RingkasanPublik>(event, 'GET', '/api/publik/ringkasan') };
	} catch {
		return { ringkasan: null };
	}
};
