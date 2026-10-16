// Identitas masjid untuk sidebar, landing, dan kop laporan. Diisi lewat env
// (frontend/.env), bukan ditulis di kode, supaya aplikasi bisa dipakai masjid lain.
import { env } from '$env/dynamic/public';

export const masjid = {
	nama: env.PUBLIC_MASJID_NAMA || 'Masjid Jami Al-Barokah',
	// Kosong bila belum diisi: kop laporan tidak boleh mencetak placeholder.
	alamat: env.PUBLIC_MASJID_ALAMAT || '',
	kontak: env.PUBLIC_MASJID_KONTAK || '',
	// Alamat gambar logo (berkas di frontend/static/). Bawaan = lambang Masjid Jami' Al-Barokah
	// yang disertakan; masjid lain cukup mengganti berkas itu atau mengisi alamat lain.
	// Versi lengkap (dengan tulisan) ada di /logo-masjid-lengkap.jpeg untuk ukuran besar.
	logo: env.PUBLIC_MASJID_LOGO || '/logo-masjid.jpg',
	// Foto masjid untuk latar hero landing, mis. "/foto-masjid.jpg" (berkas di frontend/static/).
	// Kosong = latar hijau berpola.
	foto: env.PUBLIC_MASJID_FOTO || '',
	// Titik masjid untuk jadwal salat. Bawaan: Pamulang, Tangerang Selatan; isi koordinat
	// masjid yang sebenarnya (selisih 1° bujur ≈ 4 menit). Lintang selatan bernilai negatif.
	lintang: Number(env.PUBLIC_MASJID_LINTANG) || -6.3436,
	bujur: Number(env.PUBLIC_MASJID_BUJUR) || 106.7365
};

/** Baris kedua kop: "alamat · kontak", melewati bagian yang kosong. */
export const kopMasjid = [masjid.alamat, masjid.kontak].filter(Boolean).join(' · ');
