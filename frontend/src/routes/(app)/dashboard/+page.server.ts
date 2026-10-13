import { api } from '$lib/server/api';
import type { Dashboard } from '$lib/types';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => ({
	dash: await api<Dashboard>(event, 'GET', '/api/dashboard')
});
