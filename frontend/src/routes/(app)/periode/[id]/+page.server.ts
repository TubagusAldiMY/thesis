// /periode/<id> tidak punya isi sendiri: arahkan ke tab yang paling relevan.
// Admin + draft → Kandidat; disetujui/selesai → Penyaluran; selain itu Hasil.
import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent, params }) => {
	const { detail, user } = await parent();
	const s = detail.periode.status;
	let tab = 'hasil';
	if (user.role === 'admin' && s === 'draft') tab = 'kandidat';
	else if (s === 'disetujui' || s === 'selesai') tab = 'penyaluran';
	redirect(307, `/periode/${params.id}/${tab}`);
};
