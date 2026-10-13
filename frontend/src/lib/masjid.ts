// Identitas masjid untuk sidebar, landing, dan kop laporan. Diisi lewat env
// (frontend/.env), bukan ditulis di kode, supaya aplikasi bisa dipakai masjid lain.
import { env } from '$env/dynamic/public';

export const masjid = {
	nama: env.PUBLIC_MASJID_NAMA || 'Masjid Jami Al-Barokah',
	alamat: env.PUBLIC_MASJID_ALAMAT || '[ALAMAT MASJID]',
	kontak: env.PUBLIC_MASJID_KONTAK || '[KONTAK]'
};
