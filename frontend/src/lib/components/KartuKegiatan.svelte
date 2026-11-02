<!--
  Kartu satu kegiatan (daftar /kegiatan dan landing): gambar sampul, kategori,
  judul, tanggal & lokasi, ringkasan. Tanpa gambar dipakai bidang hijau berlogo
  masjid supaya kartu seragam.
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { tanggalAcara } from '$lib/kegiatan';
	import type { Kegiatan } from '$lib/types';

	let { kegiatan: k }: { kegiatan: Kegiatan } = $props();
</script>

<a class="kartu-kegiatan" href="/kegiatan/{k.slug}">
	<div class="sampul">
		{#if k.ada_gambar}
			<img src="/kegiatan/{k.slug}/gambar?v={k.gambar_versi}" alt="" loading="lazy" />
		{:else}
			<span class="tanpa-gambar" aria-hidden="true"><IkonMasjid ukuran={48} /></span>
		{/if}
		<span class="kategori">{k.kategori_label}</span>
	</div>
	<div class="isi">
		<h3>{k.judul}</h3>
		{#if k.tanggal_acara}<p class="waktu">{tanggalAcara(k.tanggal_acara, k.jam_acara)}</p>{/if}
		{#if k.lokasi}<p class="lokasi redup">{k.lokasi}</p>{/if}
		<p class="ringkasan">{k.ringkasan}</p>
	</div>
</a>

<style>
	.kartu-kegiatan {
		display: grid;
		grid-template-rows: auto 1fr;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 18px;
		overflow: hidden;
		text-decoration: none;
		color: var(--teks);
		transition: box-shadow 0.15s, transform 0.15s;
	}
	.kartu-kegiatan:hover {
		box-shadow: 0 6px 20px rgb(0 0 0 / 0.08);
		transform: translateY(-2px);
	}
	.kartu-kegiatan:focus-visible {
		outline: 2px solid var(--hijau-aktif);
		outline-offset: 2px;
	}
	.sampul {
		position: relative;
		aspect-ratio: 16 / 9;
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
	.kategori {
		position: absolute;
		top: 0.75rem;
		left: 0.75rem;
		padding: 0.2rem 0.65rem;
		border-radius: 999px;
		background: var(--kuning);
		color: var(--kuning-teks);
		font-size: 0.75rem;
		font-weight: 700;
	}
	.isi {
		display: grid;
		align-content: start;
		gap: 0.4rem;
		padding: 1rem 1.1rem 1.15rem;
	}
	h3 {
		font-size: 1.05rem;
		line-height: 1.3;
	}
	.waktu {
		color: var(--hijau);
		font-weight: 600;
		font-size: 0.88rem;
	}
	.lokasi {
		font-size: 0.85rem;
	}
	.ringkasan {
		color: var(--teks-redup);
		font-size: 0.9rem;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
