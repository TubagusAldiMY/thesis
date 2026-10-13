// Data kepala periode (status, dana, kuota, riwayat run, catatan penolakan)
// dipakai bersama oleh semua tab di bawah /periode/[id].
import { api } from '$lib/server/api';
import type { PeriodeDetail } from '$lib/types';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async (event) => ({
	detail: await api<PeriodeDetail>(event, 'GET', `/api/periode/${encodeURIComponent(event.params.id)}`)
});
