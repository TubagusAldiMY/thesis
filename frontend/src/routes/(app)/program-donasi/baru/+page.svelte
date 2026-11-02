<!-- Formulir program donasi baru (Admin). -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import FormProgram from '$lib/components/FormProgram.svelte';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
	let mengirim = $state(false);
	const nilai = $derived(form?.nilai ?? { judul: '', ringkasan: '', cerita: '', target: '', batas_waktu: '', status: 'draf' });
	const galat = (f: string) => (form?.fields as Record<string, string> | undefined)?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Program baru · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/program-donasi">← Program donasi</a>
	<h1>Program baru</h1>
	<p class="redup">Simpan sebagai draf dulu bila belum siap tampil. Gambar sampul diunggah setelah program tersimpan.</p>

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
		<FormProgram {nilai} {galat} baru />
		<div class="baris-aksi ujung">
			<a class="tombol" href="/program-donasi">Batal</a>
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : 'Simpan program'}</button>
		</div>
	</form>
</div>

<style>
	.sempit {
		max-width: 760px;
	}
</style>
