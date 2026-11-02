<!--
  Detail program donasi (publik). Desktop: kiri sampul, judul, dan tab Cerita /
  Kabar terbaru / Donatur; kanan kartu dana (kemajuan, tombol donasi, bagikan)
  yang menempel. Ponsel: kartu dana naik ke bawah judul supaya tombol donasi
  langsung terlihat. Teks cerita/kabar adalah teks biasa (Svelte meng-escape-nya);
  paragraf dipisah baris kosong.
-->
<script lang="ts">
	import { page } from '$app/state';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { rupiah, sisaWaktu, tanggalPanjang, tanggalSaja, waktu } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const p = $derived(data.program);

	type Tab = 'cerita' | 'kabar' | 'donatur';
	let tab = $state<Tab>('cerita');

	const paragraf = (s: string) => s.split(/\n\s*\n/).map((x) => x.trim()).filter(Boolean);
	const tautan = $derived(`${page.url.origin}/program/${p.slug}`);
	const tautanWA = $derived('https://wa.me/?text=' + encodeURIComponent(`${p.judul} — mari dukung bersama: ${tautan}`));
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
	<title>{p.judul} · {masjid.nama}</title>
	<meta name="description" content={p.ringkasan} />
</svelte:head>

<a class="tautan-kembali" href="/program">← Semua program</a>

<div class="halaman">
	<header class="kepala">
		<div class="sampul">
			{#if p.ada_gambar}
				<img src="/program/{p.slug}/gambar?v={p.gambar_versi}" alt="Gambar program {p.judul}" />
			{:else}
				<span class="tanpa-gambar" aria-hidden="true"><IkonMasjid ukuran={72} /></span>
			{/if}
		</div>
		<h1>{p.judul}</h1>
		<p class="ringkasan">{p.ringkasan}</p>
	</header>

	<aside class="samping">
		<div class="kartu-dana">
			<p class="terkumpul"><strong>{rupiah(p.terkumpul)}</strong></p>
			<p class="redup">terkumpul dari target {rupiah(p.target)}</p>
			<div class="bar" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={p.persen} aria-label="Kemajuan donasi">
				<span style:width="{p.persen}%"></span>
			</div>
			<dl class="statistik">
				<div><dt>Donatur</dt><dd>{p.donatur}</dd></div>
				<div><dt>Sisa waktu</dt><dd>{sisaWaktu(p)}</dd></div>
			</dl>
			{#if p.batas_waktu}<p class="redup kecil">Batas donasi: {tanggalSaja(p.batas_waktu)}</p>{/if}

			{#if p.terbuka}
				<a class="tombol utama besar" href="/donasi?program={p.slug}">Donasi sekarang</a>
			{:else}
				<p class="kotak-info">Program ini sudah tidak menerima donasi. Terima kasih kepada seluruh donatur.</p>
			{/if}

			<div class="bagikan">
				<a class="tombol" href={tautanWA} target="_blank" rel="noopener noreferrer">Bagikan ke WhatsApp</a>
				<button class="tombol" type="button" onclick={salinTautan}>{tersalin ? 'Tautan disalin' : 'Salin tautan'}</button>
			</div>
		</div>
		<p class="redup kecil catatan">Dana program dikelola pengurus {masjid.nama}. Setiap donasi mendapat kuitansi resmi bernomor.</p>
	</aside>

	<section class="isi">
		<div class="tab" role="tablist" aria-label="Isi program">
			<button type="button" role="tab" aria-selected={tab === 'cerita'} onclick={() => (tab = 'cerita')}>Cerita</button>
			<button type="button" role="tab" aria-selected={tab === 'kabar'} onclick={() => (tab = 'kabar')}>
				Kabar terbaru <span class="hitung">{p.kabar.length}</span>
			</button>
			<button type="button" role="tab" aria-selected={tab === 'donatur'} onclick={() => (tab = 'donatur')}>
				Donatur <span class="hitung">{p.donatur}</span>
			</button>
		</div>

		<div class="panel" role="tabpanel">
			{#if tab === 'cerita'}
				<div class="teks">
					{#each paragraf(p.cerita) as x, i (i)}<p>{x}</p>{/each}
				</div>
			{:else if tab === 'kabar'}
				{#if p.kabar.length === 0}
					<p class="redup">Belum ada kabar. Pengurus akan menuliskan perkembangan dan penyaluran dana di sini.</p>
				{:else}
					<ol class="linimasa">
						{#each p.kabar as k (k.id)}
							<li>
								<time datetime={k.dibuat_pada}>{tanggalPanjang(k.dibuat_pada)}</time>
								<h3>{k.judul}</h3>
								<div class="teks">{#each paragraf(k.isi) as x, i (i)}<p>{x}</p>{/each}</div>
							</li>
						{/each}
					</ol>
				{/if}
			{:else if p.donatur_terbaru.length === 0}
				<p class="redup">Belum ada donatur. Jadilah yang pertama mendukung program ini.</p>
			{:else}
				<ul class="donatur">
					{#each p.donatur_terbaru as d, i (i)}
						<li>
							<span class="inisial" aria-hidden="true">{d.nama.trim()[0]?.toUpperCase() ?? '?'}</span>
							<span class="siapa"><strong>{d.nama}</strong><small class="redup">{waktu(d.dibayar_pada)}</small></span>
							<span class="jumlah">{rupiah(d.jumlah)}</span>
						</li>
					{/each}
				</ul>
				{#if p.donatur > p.donatur_terbaru.length}
					<p class="redup kecil">Menampilkan {p.donatur_terbaru.length} donatur terbaru dari {p.donatur}.</p>
				{/if}
			{/if}
		</div>
	</section>
</div>

<style>
	.halaman {
		display: grid;
		grid-template-columns: minmax(0, 7fr) minmax(0, 4fr);
		grid-template-areas:
			'kepala samping'
			'isi samping';
		column-gap: 2rem;
		row-gap: 1rem;
		align-items: start;
		margin-top: 1rem;
	}
	.kepala {
		grid-area: kepala;
		display: grid;
		gap: 0.75rem;
	}
	.samping {
		grid-area: samping;
		position: sticky;
		top: 1.5rem;
		display: grid;
		gap: 0.75rem;
	}
	.isi {
		grid-area: isi;
		display: grid;
		gap: 1rem;
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
	.ringkasan {
		color: var(--teks-redup);
		font-size: 1.05rem;
	}
	.tab {
		display: flex;
		gap: 0.25rem;
		border-bottom: 1px solid var(--garis);
		overflow-x: auto;
	}
	.tab button {
		border: 0;
		background: none;
		font: inherit;
		font-weight: 600;
		color: var(--teks-redup);
		padding: 0.75rem 1rem;
		border-bottom: 3px solid transparent;
		cursor: pointer;
		white-space: nowrap;
	}
	.tab button[aria-selected='true'] {
		color: var(--hijau);
		border-bottom-color: var(--hijau);
	}
	.tab button:focus-visible {
		outline: 2px solid var(--hijau-aktif);
		outline-offset: -2px;
	}
	.hitung {
		display: inline-block;
		margin-left: 0.3rem;
		padding: 0 0.45rem;
		border-radius: 999px;
		background: var(--abu);
		font-size: 0.75rem;
	}
	.teks {
		display: grid;
		gap: 0.8rem;
		line-height: 1.7;
	}
	.linimasa {
		list-style: none;
		margin: 0;
		padding: 0 0 0 1.25rem;
		border-left: 2px solid var(--hijau-muda);
		display: grid;
		gap: 1.5rem;
	}
	.linimasa li {
		position: relative;
		display: grid;
		gap: 0.35rem;
	}
	.linimasa li::before {
		content: '';
		position: absolute;
		left: calc(-1.25rem - 7px);
		top: 0.3rem;
		width: 12px;
		height: 12px;
		border-radius: 50%;
		background: var(--hijau);
	}
	.linimasa time {
		font-size: 0.82rem;
		color: var(--teks-redup);
	}
	.donatur {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
	}
	.donatur li {
		display: grid;
		grid-template-columns: auto 1fr auto;
		gap: 0.85rem;
		align-items: center;
		padding: 0.75rem 0;
		border-bottom: 1px solid var(--garis);
	}
	.inisial {
		display: grid;
		place-items: center;
		width: 38px;
		height: 38px;
		border-radius: 50%;
		background: var(--hijau-muda);
		color: var(--hijau-tua);
		font-weight: 700;
	}
	.siapa {
		display: grid;
		min-width: 0;
	}
	.siapa strong {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.jumlah {
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}
	.kartu-dana {
		display: grid;
		gap: 0.6rem;
		padding: 1.4rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 20px;
		box-shadow: 0 1px 2px rgb(0 0 0 / 0.04);
	}
	.terkumpul {
		font-size: 1.6rem;
		color: var(--hijau);
		letter-spacing: -0.01em;
	}
	.bar {
		height: 10px;
		border-radius: 999px;
		background: var(--hijau-muda);
		overflow: hidden;
		margin: 0.3rem 0;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--hijau);
		border-radius: inherit;
	}
	.statistik {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
		margin: 0.25rem 0;
	}
	.statistik div {
		display: grid;
		gap: 0.1rem;
	}
	.statistik dt {
		font-size: 0.8rem;
		color: var(--teks-redup);
	}
	.statistik dd {
		margin: 0;
		font-weight: 700;
	}
	.besar {
		width: 100%;
		justify-content: center;
		padding: 0.95rem 1.25rem;
		font-size: 1.05rem;
		font-weight: 700;
		border-radius: 14px;
		margin-top: 0.4rem;
	}
	.bagikan {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem;
	}
	.bagikan .tombol {
		justify-content: center;
		font-size: 0.85rem;
	}
	.kecil {
		font-size: 0.82rem;
	}
	.catatan {
		text-align: center;
	}
	@media (max-width: 900px) {
		.halaman {
			grid-template-columns: 1fr;
			grid-template-areas: 'kepala' 'samping' 'isi';
		}
		.samping {
			position: static;
		}
	}
</style>
