<!-- Menu "Lainnya" (Android): Kriteria & bobot, Audit log, Laporan, Profil & password, Tentang, Keluar. -->
<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const menu = [
		{ href: '/kriteria', label: 'Kriteria & bobot' },
		{ href: '/audit', label: 'Audit log' },
		{ href: '/laporan', label: 'Laporan' },
		{ href: '/profil', label: 'Profil & password' },
		{ href: '/tentang', label: 'Tentang aplikasi' }
	];
</script>

<svelte:head><title>Lainnya · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Lainnya</h1>
		<p class="redup">{data.user.nama} · {data.user.role === 'admin' ? 'Admin' : 'Ketua DKM'}</p>
	</div>
	<nav class="kartu daftar" aria-label="Menu lainnya">
		{#each menu as m (m.href)}
			<a href={m.href}>{m.label} <span aria-hidden="true">›</span></a>
		{/each}
		<form method="POST" action="/logout"><button type="submit">Keluar</button></form>
	</nav>
</div>

<style>
	.daftar {
		padding: 0.25rem 0;
		display: grid;
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
