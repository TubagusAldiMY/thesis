<!-- Tab Kandidat: daftar bercentang + "n dari m dipilih · kuota N" + Kosongkan pilihan. Hanya bisa diubah saat draft. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { labelAsnaf } from '$lib/format';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const draft = $derived(data.detail.periode.status === 'draft');
	let mengirim = $state(false);

	let pilih = $state<string[]>(untrack(() => [...data.terpilih]));
	$effect.pre(() => {
		pilih = [...data.terpilih]; // sinkron ulang setelah disimpan
	});
	let cari = $state('');
	const tampil = $derived(data.layak.filter((m) => m.nama.toLowerCase().includes(cari.trim().toLowerCase())));
</script>

<svelte:head><title>Kandidat · {data.detail.periode.nama}</title></svelte:head>

<form
	method="POST"
	class="kartu tumpuk"
	use:enhance={() => {
		mengirim = true;
		return async ({ update }) => {
			await update({ reset: false });
			mengirim = false;
		};
	}}
>
	<div class="kartu-kepala">
		<div>
			<h2>Pilih kandidat</h2>
			<p class="redup">
				{data.detail.periode.jenis === 'zakat'
					? 'Hanya mustahik aktif dari golongan yang diizinkan periode ini.'
					: 'Infak: semua mustahik aktif bisa dipilih.'}
			</p>
		</div>
		<span class="badge {pilih.length >= data.detail.kuota ? 'hijau' : 'kuning'}">
			{pilih.length} dari {data.layak.length} dipilih · kuota {data.detail.kuota}
		</span>
	</div>

	{#if form?.tersimpan}<p class="kotak-sukses" role="status">Daftar kandidat tersimpan.</p>{/if}
	{#if form?.error}<p class="kotak-galat" role="alert">{form.fields?.kandidat ?? form.error}</p>{/if}
	{#if !draft}<p class="kotak-info">Periode tidak lagi berstatus draft; kandidat terkunci.</p>{/if}
	{#if data.tidakLayak.length > 0}
		<p class="kotak-galat" role="alert">
			Kandidat berikut sudah nonaktif atau golongannya tidak diizinkan, dan akan dibuang saat disimpan:
			{data.tidakLayak.map((m) => m.nama).join(', ')}.
		</p>
	{/if}

	{#if data.layak.length === 0}
		<p class="redup kosong">Belum ada mustahik aktif yang memenuhi syarat. <a href="/mustahik/baru">Tambah mustahik</a>.</p>
	{:else}
		<div class="alat">
			<label class="sr-only" for="cari">Cari nama</label>
			<input id="cari" type="search" placeholder="Saring nama…" bind:value={cari} />
			{#if draft}
				<button type="button" class="tombol kecil" onclick={() => (pilih = [...new Set([...pilih, ...tampil.map((m) => m.id)])])}>Pilih semua yang tampil</button>
				<button type="button" class="tombol kecil" onclick={() => (pilih = [])}>Kosongkan pilihan</button>
			{/if}
		</div>

		<ul class="daftar">
			{#each tampil as m (m.id)}
				<li>
					<label>
						<input type="checkbox" name="mustahik_id" value={m.id} bind:group={pilih} disabled={!draft} />
						<span class="nama"><strong>{m.nama}</strong><small class="redup">{m.alamat}</small></span>
						<span class="badge">{labelAsnaf[m.asnaf]}</span>
					</label>
				</li>
			{/each}
		</ul>
		{#if draft}
			<!-- Pilihan yang sedang tersaring tetap ikut terkirim. -->
			{#each pilih.filter((id) => !tampil.some((m) => m.id === id)) as id (id)}
				<input type="hidden" name="mustahik_id" value={id} />
			{/each}
		{/if}
	{/if}

	{#if draft}
		<div class="baris-aksi ujung">
			<button class="tombol" type="submit" disabled={mengirim}>Simpan</button>
			<button class="tombol utama" type="submit" name="lanjut" value="1" disabled={mengirim || pilih.length === 0}>Simpan & isi penilaian →</button>
		</div>
	{/if}
</form>

<style>
	.alat {
		display: flex;
		gap: 0.6rem;
		flex-wrap: wrap;
	}
	.alat input {
		flex: 1;
		min-width: 200px;
	}
	.daftar {
		list-style: none;
		padding: 0;
		margin: 0;
		border: 1px solid var(--garis);
		border-radius: 12px;
	}
	.daftar li + li {
		border-top: 1px solid var(--garis);
	}
	.daftar label {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		padding: 0.75rem 1rem;
		cursor: pointer;
	}
	.daftar input {
		width: 20px;
		height: 20px;
		accent-color: var(--hijau);
		flex: none;
	}
	.nama {
		flex: 1;
		display: grid;
		min-width: 0;
	}
	.nama small {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.kosong {
		text-align: center;
		padding: 1.5rem 0;
	}
</style>
