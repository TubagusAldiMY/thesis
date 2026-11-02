<!--
  Kartu satu program donasi (daftar /program dan landing): gambar sampul, judul,
  ringkasan, bar kemajuan, terkumpul dari target, dan sisa waktu. Tanpa gambar
  sampul dipakai bidang hijau berlogo masjid supaya kartu tetap seragam.
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { rupiah, sisaWaktu } from '$lib/format';
	import type { Program } from '$lib/types';

	let { program: p }: { program: Program } = $props();
</script>

<a class="kartu-program" href="/program/{p.slug}">
	<div class="sampul">
		{#if p.ada_gambar}
			<img src="/program/{p.slug}/gambar?v={p.gambar_versi}" alt="" loading="lazy" />
		{:else}
			<span class="tanpa-gambar" aria-hidden="true"><IkonMasjid ukuran={48} /></span>
		{/if}
		{#if p.status === 'selesai'}<span class="pita">Selesai</span>{/if}
	</div>
	<div class="isi">
		<h3>{p.judul}</h3>
		<p class="ringkasan">{p.ringkasan}</p>
		<div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={p.persen} aria-label="Kemajuan donasi">
			<span style:width="{p.persen}%"></span>
		</div>
		<p class="angka-utama"><strong>{rupiah(p.terkumpul)}</strong> <span class="redup">dari {rupiah(p.target)}</span></p>
		<p class="meta redup"><span>{p.donatur} donatur</span><span>{sisaWaktu(p)}</span></p>
	</div>
</a>

<style>
	.kartu-program {
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
	.kartu-program:hover {
		box-shadow: 0 6px 20px rgb(0 0 0 / 0.08);
		transform: translateY(-2px);
	}
	.kartu-program:focus-visible {
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
	.pita {
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
		gap: 0.6rem;
		padding: 1rem 1.1rem 1.15rem;
	}
	h3 {
		font-size: 1.05rem;
		line-height: 1.3;
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
	.bar {
		height: 8px;
		border-radius: 999px;
		background: var(--hijau-muda);
		overflow: hidden;
		margin-top: 0.2rem;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--hijau);
		border-radius: inherit;
	}
	.angka-utama {
		font-size: 0.95rem;
	}
	.meta {
		display: flex;
		justify-content: space-between;
		gap: 0.5rem;
		font-size: 0.82rem;
	}
</style>
