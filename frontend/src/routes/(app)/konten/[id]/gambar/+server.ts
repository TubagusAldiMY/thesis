// Gambar sampul untuk pengurus (termasuk kegiatan draf), diteruskan dari Go
// dengan sesi pengurus. Tanpa sesi → 401 dari Go.
import { gambar } from '$lib/server/api';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = (event) => gambar(event, '/api/kegiatan/' + encodeURIComponent(event.params.id) + '/gambar');
