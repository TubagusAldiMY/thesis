import { api } from '$lib/server/api';
import type { Dashboard } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	// Tanggal hari ini menurut kalender WIB (zona tetap +7), dasar jadwal salat.
	const w = new Date(Date.now() + 7 * 3600_000);
	return {
		hari: { tahun: w.getUTCFullYear(), bulan: w.getUTCMonth() + 1, tanggal: w.getUTCDate() },
		dash: await api<Dashboard>(event, 'GET', '/api/dashboard')
	};
};
