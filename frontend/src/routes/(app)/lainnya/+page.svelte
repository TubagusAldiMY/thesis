<!-- Menu "Lainnya" (Android): Donasi, Kriteria & bobot, Audit log, Laporan, Profil & password, Tentang, Keluar. -->
<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const menu = $derived([
		{ href: '/penerimaan', label: 'Donasi', lencana: data.donasiBaru },
		{ href: '/program-donasi', label: 'Program donasi', lencana: 0 },
		{ href: '/kriteria', label: 'Kriteria & bobot', lencana: 0 },
		{ href: '/audit', label: 'Audit log', lencana: 0 },
		{ href: '/laporan', label: 'Laporan', lencana: 0 },
		{ href: '/profil', label: 'Profil & password', lencana: 0 },
		{ href: '/tentang', label: 'Tentang aplikasi', lencana: 0 }
	]);
</script>

<svelte:head><title>Lainnya · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Lainnya</h1>
		<p class="redup">{data.user.nama} · {data.user.role === 'admin' ? 'Admin' : 'Ketua DKM'}</p>
	</div>
	<nav class="kartu daftar" aria-label="Menu lainnya">
		{#each menu as m (m.href)}
			<a href={m.href}>
				<span>{m.label}{#if m.lencana}<span class="lencana" title="Donasi web baru">{m.lencana}</span>{/if}</span>
				<span aria-hidden="true">›</span>
			</a>
		{/each}
		<form method="POST" action="/logout"><button type="submit">Keluar</button></form>
	</nav>
</div>

<style>
	.daftar {
		padding: 0.25rem 0;
		display: grid;
	}
	.lencana {
		display: inline-block;
		margin-left: 0.5rem;
		min-width: 1.25rem;
		padding: 0 0.4rem;
		border-radius: 999px;
		background: var(--kuning);
		color: var(--kuning-teks);
		font-size: 0.75rem;
		font-weight: 700;
		line-height: 1.25rem;
		text-align: center;
	}
	.daftar a,
	.daftar button {
		display: flex;
		justify-content: space-between;
		padding: 1rem 1.25rem;
		text-decoration: none;
		color: var(--teks);
		border: none;
		border-bottom: 1px solid var(--garis);
		font: inherit;
		font-weight: 600;
		background: none;
		width: 100%;
		text-align: left;
		cursor: pointer;
	}
	.daftar button {
		color: var(--merah);
		border-bottom: none;
	}
</style>
