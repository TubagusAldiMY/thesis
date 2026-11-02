<!-- Tab di halaman pengurus Konten: Kegiatan | Jadwal rutin | Pengumuman. -->
<script lang="ts">
	import { page } from '$app/state';

	const tab = [
		{ href: '/konten', label: 'Kegiatan' },
		{ href: '/konten/jadwal', label: 'Jadwal rutin' },
		{ href: '/konten/pengumuman', label: 'Pengumuman' }
	];
	// Kegiatan aktif juga untuk /konten/baru dan /konten/<id>; dua tab lain persis cocok.
	const aktif = (href: string) =>
		href === '/konten' ? !page.url.pathname.startsWith('/konten/jadwal') && !page.url.pathname.startsWith('/konten/pengumuman') : page.url.pathname === href;
</script>

<nav class="tab" aria-label="Jenis konten">
	{#each tab as t (t.href)}
		<a href={t.href} aria-current={aktif(t.href) ? 'page' : undefined}>{t.label}</a>
	{/each}
</nav>

<style>
	.tab {
		display: flex;
		gap: 0.25rem;
		border-bottom: 1px solid var(--garis);
		overflow-x: auto;
	}
	.tab a {
		padding: 0.7rem 1.1rem;
		text-decoration: none;
		color: var(--teks-redup);
		font-weight: 600;
		border-bottom: 3px solid transparent;
		white-space: nowrap;
	}
	.tab a[aria-current='page'] {
		color: var(--hijau);
		border-bottom-color: var(--hijau);
	}
	.tab a:focus-visible {
		outline: 2px solid var(--hijau-aktif);
		outline-offset: -2px;
	}
</style>
