<!--
  Detail kegiatan (publik): sampul, kategori, waktu & lokasi, isi, dan bagikan.
  Isi adalah teks biasa (Svelte meng-escape-nya); paragraf dipisah baris kosong.
-->
<script lang="ts">
	import { page } from '$app/state';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { paragraf, tanggalAcara } from '$lib/kegiatan';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const k = $derived(data.kegiatan);

	const tautan = $derived(`${page.url.origin}/kegiatan/${k.slug}`);
	const tautanWA = $derived('https://wa.me/?text=' + encodeURIComponent(`${k.judul} — ${tautan}`));
	let tersalin = $state(false);
	async function salinTautan() {
		try {
			await navigator.clipboard.writeText(tautan);
			tersalin = true;
			setTimeout(() => (tersalin = false), 2000);
		} catch {
			// clipboard ditolak browser: tombol WhatsApp tetap tersedia
		}
	}
</script>

<svelte:head>
	<title>{k.judul} · {masjid.nama}</title>
	<meta name="description" content={k.ringkasan} />
</svelte:head>

<a class="tautan-kembali" href="/kegiatan">← Semua kegiatan</a>

<article class="halaman">
	<div class="sampul">
		{#if k.ada_gambar}
			<img src="/kegiatan/{k.slug}/gambar?v={k.gambar_versi}" alt="Gambar kegiatan {k.judul}" />
		{:else}
			<span class="tanpa-gambar" aria-hidden="true"><IkonMasjid ukuran={72} /></span>
		{/if}
	</div>

	<header class="kepala">
		<span class="kategori">{k.kategori_label}</span>
		<h1>{k.judul}</h1>
		{#if k.tanggal_acara || k.lokasi}
			<dl class="info">
				{#if k.tanggal_acara}
					<div><dt>Waktu</dt><dd>{tanggalAcara(k.tanggal_acara, k.jam_acara)}</dd></div>
				{/if}
				{#if k.lokasi}
					<div><dt>Lokasi</dt><dd>{k.lokasi}</dd></div>
				{/if}
			</dl>
		{/if}
		<p class="ringkasan">{k.ringkasan}</p>
	</header>

	<div class="isi">
		{#each paragraf(k.isi) as x, i (i)}<p>{x}</p>{/each}
	</div>

	<div class="bagikan">
		<a class="tombol" href={tautanWA} target="_blank" rel="noopener noreferrer">Bagikan ke WhatsApp</a>
		<button class="tombol" type="button" onclick={salinTautan}>{tersalin ? 'Tautan disalin' : 'Salin tautan'}</button>
	</div>
</article>

<style>
	.halaman {
		display: grid;
		gap: 1.25rem;
		max-width: 780px;
		margin: 1rem auto 0;
	}
	.sampul {
		position: relative;
		aspect-ratio: 16 / 9;
		border-radius: 20px;
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
	.kepala {
		display: grid;
		gap: 0.75rem;
		justify-items: start;
	}
	.kategori {
		padding: 0.2rem 0.75rem;
		border-radius: 999px;
		background: var(--kuning);
		color: var(--kuning-teks);
		font-size: 0.8rem;
		font-weight: 700;
	}
	.info {
		display: grid;
		gap: 0.4rem;
		margin: 0;
		padding: 0.9rem 1.1rem;
		background: var(--hijau-muda);
		border-radius: 14px;
		width: 100%;
	}
	.info div {
		display: grid;
		grid-template-columns: 5rem 1fr;
		gap: 0.5rem;
	}
	.info dt {
		color: var(--teks-redup);
		font-size: 0.85rem;
	}
	.info dd {
		margin: 0;
		font-weight: 600;
	}
	.ringkasan {
		color: var(--teks-redup);
		font-size: 1.05rem;
	}
	.isi {
		display: grid;
		gap: 0.9rem;
		line-height: 1.75;
	}
	.bagikan {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		padding-top: 0.5rem;
		border-top: 1px solid var(--garis);
	}
</style>
