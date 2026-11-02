<!-- Formulir kegiatan baru. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import FormKegiatan from '$lib/components/FormKegiatan.svelte';
	import type { IsianKegiatan } from '$lib/kegiatan';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);
	const kosong: IsianKegiatan = { judul: '', kategori: '', ringkasan: '', isi: '', tanggal_acara: '', jam_acara: '', lokasi: '', status: 'draf' };
	const nilai = $derived((form?.nilai as IsianKegiatan | undefined) ?? kosong);
	const galat = (f: string) => (form?.fields as Record<string, string> | undefined)?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Kegiatan baru · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/konten">← Konten kegiatan</a>
	<h1>Kegiatan baru</h1>
	<p class="redup">Simpan sebagai draf dulu bila belum siap tampil. Gambar sampul diunggah setelah kegiatan tersimpan.</p>

	<form
		method="POST"
		class="kartu tumpuk"
		novalidate
		use:enhance={() => {
			mengirim = true;
			return async ({ update }) => {
				await update({ reset: false });
				mengirim = false;
			};
		}}
	>
		{#if galatUmum}<p class="kotak-galat" role="alert">{galatUmum}</p>{/if}
		<FormKegiatan {nilai} {galat} kategori={data.kategori} baru />
		<div class="baris-aksi ujung">
			<a class="tombol" href="/konten">Batal</a>
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : 'Simpan kegiatan'}</button>
		</div>
	</form>
</div>

<style>
	.sempit {
		max-width: 800px;
	}
</style>
