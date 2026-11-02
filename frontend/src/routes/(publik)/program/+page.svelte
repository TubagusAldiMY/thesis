<!-- Daftar program donasi (publik): kartu bergambar dengan bar kemajuan, dikelompokkan berjalan/selesai. -->
<script lang="ts">
	import KartuProgram from '$lib/components/KartuProgram.svelte';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Program donasi · {masjid.nama}</title></svelte:head>

<div class="tumpuk">
	<header class="kepala">
		<h1>Program Donasi</h1>
		<p class="redup">
			Pilih program kebaikan yang ingin Anda dukung. Setiap infak tercatat dengan kuitansi resmi, dan perkembangannya dilaporkan pengurus di halaman program.
		</p>
	</header>

	{#if data.berjalan.length === 0 && data.selesai.length === 0}
		<p class="kotak-info">Belum ada program donasi. Anda tetap bisa berdonasi zakat dan infak umum di <a href="/donasi">halaman donasi</a>.</p>
	{/if}

	{#if data.berjalan.length > 0}
		<section class="tumpuk-kecil" aria-labelledby="judul-berjalan">
			<h2 id="judul-berjalan">Sedang berjalan</h2>
			<div class="grid-program">
				{#each data.berjalan as p (p.id)}<KartuProgram program={p} />{/each}
			</div>
		</section>
	{/if}

	{#if data.selesai.length > 0}
		<section class="tumpuk-kecil" aria-labelledby="judul-selesai">
			<h2 id="judul-selesai">Sudah selesai</h2>
			<div class="grid-program">
				{#each data.selesai as p (p.id)}<KartuProgram program={p} />{/each}
			</div>
		</section>
	{/if}

	<p class="redup kecil">Ingin menunaikan zakat? Zakat disalurkan kepada mustahik melalui <a href="/donasi">donasi umum</a>.</p>
</div>

<style>
	.kepala {
		display: grid;
		gap: 0.5rem;
		max-width: 720px;
	}
	.tumpuk-kecil {
		display: grid;
		gap: 1rem;
	}
	.grid-program {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 1.25rem;
	}
	.kecil {
		font-size: 0.875rem;
	}
</style>
