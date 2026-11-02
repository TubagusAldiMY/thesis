<!--
  Kelola satu kegiatan: ringkasan + tautan ke halaman publik, gambar sampul,
  ubah isian, dan hapus. Admin dan Ketua sama-sama boleh.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '@sveltejs/kit';
	import FormKegiatan from '$lib/components/FormKegiatan.svelte';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { labelStatusKegiatan, tanggalAcara, warnaStatusKegiatan } from '$lib/kegiatan';
	import type { IsianKegiatan } from '$lib/kegiatan';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const k = $derived(data.kegiatan);
	let mengirim = $state('');

	/** Gabungan bentuk hasil semua action halaman ini. */
	type Hasil = { bagian?: string; sukses?: string; error?: string; fields?: Record<string, string>; nilai?: IsianKegiatan };
	const hasil = $derived(form as Hasil | null);
	// Pesan dan galat hanya untuk bagian formulir yang baru dikirim.
	const untuk = (bagian: string) => (hasil?.bagian === bagian ? hasil : null);
	const galatDari = (bagian: string) => (f: string) => untuk(bagian)?.fields?.[f] ?? '';
	const nilaiUbah = $derived<IsianKegiatan>(
		untuk('ubah')?.nilai ?? {
			judul: k.judul,
			kategori: k.kategori,
			ringkasan: k.ringkasan,
			isi: k.isi,
			tanggal_acara: k.tanggal_acara,
			jam_acara: k.jam_acara,
			lokasi: k.lokasi,
			status: k.status
		}
	);

	/** enhance yang menandai bagian mana sedang dikirim; formulir unggah dikosongkan setelah berhasil. */
	const kirim =
		(bagian: string): SubmitFunction =>
		() => {
			mengirim = bagian;
			return async ({ result, update }) => {
				await update({ reset: bagian === 'gambar' && result.type === 'success' });
				mengirim = '';
			};
		};
</script>

<svelte:head><title>{k.judul} · Konten kegiatan</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/konten">← Konten kegiatan</a>

	<div class="kartu-kepala">
		<div>
			<h1>{k.judul}</h1>
			<p class="redup">{k.kategori_label}{k.tanggal_acara ? ' · ' + tanggalAcara(k.tanggal_acara, k.jam_acara) : ''}{k.lokasi ? ' · ' + k.lokasi : ''}</p>
		</div>
		<span class="badge {warnaStatusKegiatan[k.status]}">{labelStatusKegiatan[k.status]}</span>
	</div>

	{#if data.pesan}<p class="kotak-sukses" role="status">{data.pesan}</p>{/if}

	{#if k.status === 'terbit'}
		<p><a class="tombol" href="/kegiatan/{k.slug}" target="_blank" rel="noopener">Lihat halaman publik ↗</a></p>
	{:else}
		<p class="redup kecil">Kegiatan masih draf: belum tampil di halaman publik.</p>
	{/if}

	<section class="kartu tumpuk">
		<h2>Gambar sampul</h2>
		<div class="sampul">
			{#if k.ada_gambar}
				<img src="/konten/{k.id}/gambar?v={k.gambar_versi}" alt="Gambar sampul {k.judul}" />
			{:else}
				<span class="tanpa-gambar" aria-hidden="true"><IkonMasjid ukuran={56} /></span>
			{/if}
		</div>
		<form method="POST" action="?/gambar" enctype="multipart/form-data" class="baris-unggah" use:enhance={kirim('gambar')}>
			{#if untuk('gambar')?.sukses}<p class="kotak-sukses" role="status">{untuk('gambar')?.sukses}</p>{/if}
			<div class="field">
				<label for="gambar">{k.ada_gambar ? 'Ganti gambar' : 'Unggah gambar'}</label>
				<input id="gambar" name="gambar" type="file" accept="image/jpeg,image/png,image/webp" aria-invalid={Boolean(galatDari('gambar')('gambar'))} />
				<small>JPG, PNG, atau WebP, maksimal 2 MB. Rasio 16:9 (mis. 1280×720) paling pas.</small>
				{#if galatDari('gambar')('gambar')}<span class="galat" role="alert">{galatDari('gambar')('gambar')}</span>{/if}
			</div>
			<button class="tombol utama" type="submit" disabled={mengirim === 'gambar'}>{mengirim === 'gambar' ? 'Mengunggah…' : 'Unggah'}</button>
		</form>
	</section>

	<form method="POST" action="?/ubah" class="kartu tumpuk" novalidate use:enhance={kirim('ubah')}>
		<h2>Ubah kegiatan</h2>
		{#if untuk('ubah')?.sukses}<p class="kotak-sukses" role="status">{untuk('ubah')?.sukses}</p>{/if}
		{#if untuk('ubah')?.error && Object.keys(untuk('ubah')?.fields ?? {}).length === 0}<p class="kotak-galat" role="alert">{untuk('ubah')?.error}</p>{/if}
		<FormKegiatan nilai={nilaiUbah} galat={galatDari('ubah')} kategori={data.kategori} />
		<div class="baris-aksi ujung">
			<button class="tombol utama" type="submit" disabled={mengirim === 'ubah'}>{mengirim === 'ubah' ? 'Menyimpan…' : 'Simpan perubahan'}</button>
		</div>
	</form>

	<section class="kartu tumpuk">
		<h2>Hapus kegiatan</h2>
		<p class="redup">Kegiatan hilang dari halaman publik dan daftar ini. Bila hanya ingin menyembunyikan sementara, ubah statusnya menjadi Draf.</p>
		<form
			method="POST"
			action="?/hapus"
			onsubmit={(e) => {
				if (!confirm(`Hapus kegiatan "${k.judul}"?`)) e.preventDefault();
			}}
		>
			<button class="tombol bahaya" type="submit">Hapus kegiatan</button>
		</form>
	</section>
</div>

<style>
	.sempit {
		max-width: 860px;
	}
	.kecil {
		font-size: 0.85rem;
	}
	.sampul {
		position: relative;
		aspect-ratio: 16 / 9;
		max-width: 520px;
		border-radius: 14px;
		overflow: hidden;
		background: linear-gradient(160deg, var(--hijau) 0%, var(--hijau-tua) 100%);
	}
	.sampul img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.tanpa-gambar {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
		color: #fff;
		opacity: 0.85;
	}
	.baris-unggah {
		display: grid;
		gap: 0.75rem;
		justify-items: start;
	}
</style>
