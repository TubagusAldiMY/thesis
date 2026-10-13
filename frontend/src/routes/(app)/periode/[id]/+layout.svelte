<!--
  Kepala halaman periode + tab. Admin: Kandidat · Penilaian · Hasil ·
  Penyaluran · Laporan. Ketua: Hasil · Penyaluran · Laporan (FR-02).
  Bila ditolak: kartu catatan Ketua + "Buka kembali" + riwayat run (FR-19).
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import Status from '$lib/components/Status.svelte';
	import { desimal, labelAsnaf, rupiah, waktu } from '$lib/format';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	const p = $derived(data.detail.periode);
	const admin = $derived(data.user.role === 'admin');
	const dasar = $derived(`/periode/${p.id}`);
	const sudahDihitung = $derived(data.detail.runs.length > 0);

	const tab = $derived(
		[
			{ href: `${dasar}/kandidat`, label: 'Kandidat', tampil: admin },
			{ href: `${dasar}/penilaian`, label: 'Penilaian', tampil: admin },
			{ href: `${dasar}/hasil`, label: 'Hasil', tampil: true },
			{ href: `${dasar}/penyaluran`, label: 'Penyaluran', tampil: true },
			{ href: `${dasar}/laporan`, label: 'Laporan', tampil: sudahDihitung }
		].filter((t) => t.tampil)
	);
</script>

<div class="tumpuk">
	<div class="tanpa-cetak kepala">
		<a class="tautan-kembali" href="/periode">← Semua periode</a>
		<div class="judul">
			<h1>{p.nama}</h1>
			<Status status={p.status} />
		</div>
		<p class="info">
			<span>Jenis: <strong>{p.jenis === 'zakat' ? 'Zakat' : 'Infak'}</strong></span>
			<span>Dana: <strong class="angka">{rupiah(p.dana_tersedia)}</strong></span>
			<span>Nominal/penerima: <strong class="angka">{rupiah(p.nominal_per_penerima)}</strong></span>
			<span>Kuota: <strong>{data.detail.kuota} penerima</strong></span>
			{#if p.jenis === 'zakat' && p.asnaf_diizinkan}
				<span>Asnaf: <strong>{p.asnaf_diizinkan.map((a) => labelAsnaf[a]).join(', ')}</strong></span>
			{/if}
		</p>
	</div>

	{#if p.status === 'ditolak'}
		<section class="kartu ditolak tanpa-cetak" aria-labelledby="judul-tolak">
			<div class="kartu-kepala">
				<div>
					<h2 id="judul-tolak">Usulan ditolak Ketua DKM</h2>
					<p class="catatan">“{data.detail.catatan_penolakan}”</p>
					{#if data.detail.keputusan[0]}
						<p class="redup kecil">{data.detail.keputusan[0].oleh_nama} · {waktu(data.detail.keputusan[0].created_at)} · run v{data.detail.keputusan[0].versi}</p>
					{/if}
				</div>
				{#if admin}
					<form method="POST" action="{dasar}/hasil?/bukaKembali" use:enhance>
						<button class="tombol utama" type="submit">Buka kembali</button>
					</form>
				{/if}
			</div>
			<p class="redup">
				{admin
					? 'Buka kembali untuk memperbaiki kandidat atau penilaian, lalu hitung ulang. Ranking run lama tetap tersimpan.'
					: 'Menunggu Admin memperbaiki dan menghitung ulang.'}
			</p>
			<details>
				<summary>Riwayat perhitungan ({data.detail.runs.length} run)</summary>
				<ul class="riwayat">
					{#each data.detail.runs as r (r.versi)}
						<li>
							<a href="{dasar}/hasil?versi={r.versi}">Run v{r.versi}</a>
							· {waktu(r.dibuat_pada)} · {r.jumlah_kandidat} kandidat · teratas {r.teratas} (K {desimal(r.teratas_k)})
							{#if r.is_current}<span class="badge">terkini</span>{/if}
						</li>
					{/each}
				</ul>
			</details>
		</section>
	{/if}

	<nav class="tab tanpa-cetak" aria-label="Bagian periode">
		{#each tab as t (t.href)}
			<a href={t.href} aria-current={page.url.pathname === t.href ? 'page' : undefined}>{t.label}</a>
		{/each}
	</nav>

	{@render children()}
</div>

<style>
	.kepala {
		display: grid;
		gap: 0.6rem;
	}
	.judul {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		flex-wrap: wrap;
	}
	.info {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem 2rem;
		color: var(--teks-redup);
	}
	.info strong {
		color: var(--teks);
	}
	.tab {
		display: flex;
		gap: 0.5rem;
		border-bottom: 1px solid var(--garis);
		overflow-x: auto;
		scrollbar-width: none;
	}
	.tab a {
		padding: 0.75rem 1.25rem;
		text-decoration: none;
		color: var(--teks-redup);
		font-weight: 600;
		white-space: nowrap;
		border-bottom: 3px solid transparent;
		margin-bottom: -1px;
	}
	.tab a[aria-current='page'] {
		color: var(--hijau);
		border-bottom-color: var(--hijau);
	}
	.ditolak {
		border-color: #efc3bf;
		background: #fffaf9;
		display: grid;
		gap: 0.75rem;
	}
	.ditolak .kartu-kepala {
		margin-bottom: 0;
	}
	.catatan {
		font-size: 1.1rem;
		margin: 0.4rem 0;
	}
	.kecil {
		font-size: 0.88rem;
	}
	.riwayat {
		margin: 0.5rem 0 0;
		padding-left: 1.25rem;
		display: grid;
		gap: 0.3rem;
	}
	summary {
		cursor: pointer;
		font-weight: 600;
	}
</style>
