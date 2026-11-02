// Gambar sampul kegiatan untuk publik, diteruskan dari Go (draf → 404).
// Lewat SvelteKit supaya tetap satu origin (CSP img-src 'self') dan API Go tidak diekspos.
import { gambar } from '$lib/server/api';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = (event) => gambar(event, '/api/publik/kegiatan/' + encodeURIComponent(event.params.slug) + '/gambar');
