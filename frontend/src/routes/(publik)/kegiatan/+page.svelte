<!--
  Kegiatan keagamaan (publik): pengumuman, kegiatan mendatang, jadwal rutin
  mingguan, dan kegiatan terdahulu. Chip kategori menyaring daftar kegiatan.
-->
<script lang="ts">
	import KartuKegiatan from '$lib/components/KartuKegiatan.svelte';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const d = $derived(data.daftar);
	const kosong = $derived(d.mendatang.length === 0 && d.terdahulu.length === 0);
	/** Jadwal dikelompokkan per hari; Go sudah mengurutkan menurut hari lalu jam. */
	const perHari = $derived(
		data.jadwal.reduce<{ hari: string; item: typeof data.jadwal }[]>((acc, j) => {
			const g = acc.find((x) => x.hari === j.hari_label);
			if (g) g.item.push(j);
			else acc.push({ hari: j.hari_label, item: [j] });
			return acc;
		}, [])
	);
</script>

<svelte:head><title>Kegiatan · {masjid.nama}</title></svelte:head>

<div class="tumpuk">
	<header class="kepala">
		<h1>Kegiatan</h1>
		<p class="redup">Kegiatan keagamaan dan jadwal kajian di {masjid.nama}. Semua jamaah dan masyarakat sekitar dipersilakan hadir.</p>
	</header>

	{#if data.pengumuman.length > 0}
		<section class="tumpuk-kecil" aria-label="Pengumuman">
			{#each data.pengumuman as p (p.id)}
				<div class="pengumuman" role="note">
					<strong>{p.judul}</strong>
					<span>{p.isi}</span>
				</div>
			{/each}
		</section>
	{/if}

	<nav class="chips" aria-label="Filter kategori">
		<a class="chip" href="/kegiatan" aria-current={data.kategori === ''}>Semua</a>
		{#each d.kategori as k (k.kode)}
			<a class="chip" href="/kegiatan?kategori={k.kode}" aria-current={data.kategori === k.kode}>{k.label}</a>
		{/each}
	</nav>

	{#if kosong}
		<p class="kotak-info">
			{data.kategori ? 'Belum ada kegiatan pada kategori ini.' : 'Belum ada kegiatan yang diterbitkan.'}
			{#if data.kategori}<a href="/kegiatan">Lihat semua kegiatan</a>.{/if}
		</p>
	{/if}

	{#if d.mendatang.length > 0}
		<section class="tumpuk-kecil" aria-labelledby="judul-mendatang">
			<h2 id="judul-mendatang">Akan datang</h2>
			<div class="grid-kegiatan">
				{#each d.mendatang as k (k.id)}<KartuKegiatan kegiatan={k} />{/each}
			</div>
		</section>
	{/if}

	{#if perHari.length > 0}
		<section class="tumpuk-kecil" aria-labelledby="judul-jadwal">
			<h2 id="judul-jadwal">Jadwal rutin mingguan</h2>
			<div class="jadwal">
				{#each perHari as g (g.hari)}
					<div class="hari">
						<h3>{g.hari}</h3>
						<ul>
							{#each g.item as j (j.id)}
								<li>
									<span class="jam angka">{j.jam.replace(':', '.')}</span>
									<span>
										<strong>{j.judul}</strong>
										{#if j.pemateri}<small class="redup blok">{j.pemateri}</small>{/if}
										{#if j.lokasi}<small class="redup blok">{j.lokasi}</small>{/if}
										{#if j.keterangan}<small class="redup blok">{j.keterangan}</small>{/if}
									</span>
								</li>
							{/each}
						</ul>
					</div>
				{/each}
			</div>
		</section>
	{/if}

	{#if d.terdahulu.length > 0}
		<section class="tumpuk-kecil" aria-labelledby="judul-terdahulu">
			<h2 id="judul-terdahulu">Kegiatan sebelumnya</h2>
			<div class="grid-kegiatan">
				{#each d.terdahulu as k (k.id)}<KartuKegiatan kegiatan={k} />{/each}
			</div>
		</section>
	{/if}
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
	.pengumuman {
		display: grid;
		gap: 0.2rem;
		padding: 0.9rem 1.1rem;
		border-radius: 14px;
		border: 1px solid var(--kuning-garis);
		background: var(--kuning);
		color: var(--kuning-teks);
	}
	.grid-kegiatan {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 1.25rem;
	}
	.jadwal {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		gap: 1rem;
	}
	.hari {
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 14px;
		padding: 1rem 1.1rem;
		display: grid;
		gap: 0.6rem;
		align-content: start;
	}
	.hari h3 {
		color: var(--hijau);
	}
	.hari ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.7rem;
	}
	.hari li {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.75rem;
		align-items: start;
	}
	.jam {
		font-weight: 700;
		color: var(--hijau-tua);
		min-width: 3.2rem;
	}
	.blok {
		display: block;
	}
</style>
