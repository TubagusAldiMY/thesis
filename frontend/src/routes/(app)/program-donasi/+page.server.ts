// Daftar program donasi untuk pengurus (Admin & Ketua), termasuk draf.
import { api } from '$lib/server/api';
import type { Program } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	program: await api<Program[]>(event, 'GET', '/api/program')
});
