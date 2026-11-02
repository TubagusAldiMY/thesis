// Formulir donasi publik (tanpa login): muzaki mengisi, lalu dialihkan ke halaman
// bayar payment gateway lewat /donasi/lanjut (perantara di domain sendiri, lihat
// catatan di sana). Validasi sebenarnya ada di Go; di sini hanya merakit isian.
import { fail, redirect } from '@sveltejs/kit';
import { parseAngka } from '$lib/format';
import { aksi, api, teks } from '$lib/server/api';
import type { DonasiPublik, ProgramDetail } from '$lib/types';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async (event) => {
	// Go memberi tahu apakah donasi web menyala, supaya muzaki tidak mengisi form
	// panjang lalu baru mendapat galat.
	const { aktif } = await api<{ aktif: boolean }>(event, 'GET', '/api/publik/donasi-aktif');
	// ?program=<slug>: donasi diarahkan ke program tertentu (dari tombol "Donasi
	// sekarang"). Slug asing → halaman 404 dari Go, bukan formulir donasi umum.
	const slug = event.url.searchParams.get('program');
	const program = slug ? await api<ProgramDetail>(event, 'GET', '/api/publik/program/' + encodeURIComponent(slug)) : null;
	return { aktif, program };
};

const nilaiKosong = { nama: '', jenis: '', jumlah: '', keterangan: '', anonim: false };

export const actions: Actions = {
	default: async (event) => {
		const fd = await event.request.formData();
		// Kolom jebakan: disembunyikan dari manusia lewat CSS, tetapi bot yang
		// mengisi semua kolom akan mengisinya juga.
		if (teks(fd, 'website') !== '') {
			return fail(400, { error: 'Permintaan ditolak', fields: {} as Record<string, string>, nilai: nilaiKosong });
		}

		const jumlahTeks = teks(fd, 'jumlah');
		const jumlah = parseAngka(jumlahTeks);
		const body = {
			nama: teks(fd, 'nama'),
			no_hp: teks(fd, 'no_hp'),
			jenis: teks(fd, 'jenis'),
			jumlah: Number.isFinite(jumlah) ? jumlah : 0, // 0 → Go menolak dengan pesan "Jumlah harus antara …"
			keterangan: teks(fd, 'keterangan'),
			program: teks(fd, 'program'),
			anonim: fd.get('anonim') === 'on'
		};
		// Isian dipantulkan saat gagal, KECUALI No. HP (data pribadi tidak dipantulkan ulang).
		const nilai = { nama: body.nama, jenis: body.jenis, jumlah: jumlahTeks, keterangan: body.keterangan, anonim: body.anonim };

		const r = await aksi(() => api<DonasiPublik>(event, 'POST', '/api/publik/donasi', body), { nilai });
		if (r.gagal) return r.gagal;
		redirect(303, '/donasi/lanjut?kode=' + r.ok.kode);
	}
};
