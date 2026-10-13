<!--
  Shell aplikasi: sidebar hijau (desktop/tablet) atau bilah atas + navigasi
  bawah (HP). Menu yang bukan hak peran disembunyikan (FR-02); akses URL
  langsung tetap dijaga server (403).
-->
<script lang="ts">
	import { page } from '$app/state';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { masjid } from '$lib/masjid';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
	const admin = $derived(data.user.role === 'admin');

	const menu = $derived(
		[
			{ href: '/dashboard', label: 'Beranda', tampil: true },
			{ href: '/mustahik', label: 'Mustahik', tampil: admin },
			{ href: '/kriteria', label: 'Kriteria', tampil: true },
			{ href: '/periode', label: 'Periode', tampil: true },
			{ href: '/laporan', label: 'Laporan', tampil: true },
			{ href: '/audit', label: 'Audit log', tampil: true }
		].filter((m) => m.tampil)
	);
	// Android: Beranda · Mustahik · Periode · Lainnya (Ketua: tanpa Mustahik).
	const menuBawah = $derived(
		[
			{ href: '/dashboard', label: 'Beranda', tampil: true },
			{ href: '/mustahik', label: 'Mustahik', tampil: admin },
			{ href: '/periode', label: 'Periode', tampil: true },
			{ href: '/lainnya', label: 'Lainnya', tampil: true }
		].filter((m) => m.tampil)
	);

	const aktif = (href: string) => page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	const inisial = $derived(
		data.user.nama
			.split(/\s+/)
			.slice(0, 2)
			.map((s) => s[0]?.toUpperCase() ?? '')
			.join('')
	);
</script>

<div class="shell">
	<aside class="samping tanpa-cetak">
		<a class="merek" href="/dashboard">
			<IkonMasjid />
			<span>
				<strong>{masjid.nama}</strong>
				<small>SPK Zakat & Infak</small>
			</span>
		</a>
		<nav aria-label="Menu utama">
			{#each menu as m (m.href)}
				<a href={m.href} aria-current={aktif(m.href) ? 'page' : undefined}>{m.label}</a>
			{/each}
		</nav>
	</aside>

	<div class="utama">
		<header class="atas tanpa-cetak">
			<a class="merek-hp" href="/dashboard"><IkonMasjid ukuran={26} /> <strong>{masjid.nama}</strong></a>
			<details class="akun">
				<summary>
					<span class="inisial">{inisial}</span>
					<span class="nama-akun">{data.user.nama}</span>
					<span class="badge">{admin ? 'Admin' : 'Ketua DKM'}</span>
				</summary>
				<div class="menu-akun">
					<a href="/profil">Profil & password</a>
					<a href="/tentang">Tentang aplikasi</a>
					<form method="POST" action="/logout"><button type="submit">Keluar</button></form>
				</div>
			</details>
		</header>

		<main class="konten">
			{@render children()}
		</main>
	</div>

	<nav class="bawah tanpa-cetak" aria-label="Menu utama">
		{#each menuBawah as m (m.href)}
			<a href={m.href} aria-current={aktif(m.href) ? 'page' : undefined}>{m.label}</a>
		{/each}
	</nav>
</div>

<style>
	.shell {
		display: grid;
		grid-template-columns: 270px 1fr;
		min-height: 100vh;
	}
	.samping {
		background: var(--hijau);
		color: #fff;
		padding: 2rem 1.25rem;
		position: sticky;
		top: 0;
		height: 100vh;
		overflow-y: auto;
	}
	.merek {
		display: flex;
		gap: 0.75rem;
		align-items: center;
		color: #fff;
		text-decoration: none;
		margin: 0 0.5rem 2.25rem;
	}
	.merek strong {
		display: block;
		line-height: 1.2;
	}
	.merek small {
		opacity: 0.85;
	}
	.samping nav {
		display: grid;
		gap: 0.35rem;
	}
	.samping nav a {
		color: #fff;
		text-decoration: none;
		padding: 0.8rem 1rem;
		border-radius: 10px;
		opacity: 0.92;
	}
	.samping nav a:hover {
		background: rgba(255, 255, 255, 0.08);
	}
	.samping nav a[aria-current='page'] {
		background: var(--hijau-aktif);
		font-weight: 600;
		opacity: 1;
	}
	.utama {
		min-width: 0;
	}
	.atas {
		display: flex;
		justify-content: flex-end;
		align-items: center;
		padding: 1.25rem 2.5rem 0;
	}
	.merek-hp {
		display: none;
	}
	.akun {
		position: relative;
	}
	.akun summary {
		list-style: none;
		display: flex;
		align-items: center;
		gap: 0.6rem;
		background: #fff;
		border: 1px solid var(--garis);
		border-radius: 999px;
		padding: 0.35rem 0.9rem 0.35rem 0.4rem;
		cursor: pointer;
	}
	.akun summary::-webkit-details-marker {
		display: none;
	}
	.inisial {
		width: 34px;
		height: 34px;
		border-radius: 50%;
		background: var(--hijau-muda);
		color: var(--hijau);
		display: grid;
		place-items: center;
		font-weight: 700;
		font-size: 0.85rem;
	}
	.menu-akun {
		position: absolute;
		right: 0;
		top: calc(100% + 6px);
		background: #fff;
		border: 1px solid var(--garis);
		border-radius: 12px;
		padding: 0.4rem;
		min-width: 210px;
		display: grid;
		z-index: 20;
		box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
	}
	.menu-akun a,
	.menu-akun button {
		text-align: left;
		padding: 0.65rem 0.8rem;
		border-radius: 8px;
		text-decoration: none;
		color: var(--teks);
		background: none;
		border: none;
		font: inherit;
		cursor: pointer;
		width: 100%;
	}
	.menu-akun a:hover,
	.menu-akun button:hover {
		background: var(--krem);
	}
	.menu-akun button {
		color: var(--merah);
	}
	.konten {
		padding: 1.5rem 2.5rem 3rem;
		max-width: 1400px;
	}
	.bawah {
		display: none;
	}

	@media (max-width: 900px) {
		.shell {
			grid-template-columns: 1fr;
		}
		.samping {
			display: none;
		}
		.atas {
			justify-content: space-between;
			background: var(--hijau);
			color: #fff;
			padding: 0.75rem 1rem;
			position: sticky;
			top: 0;
			z-index: 10;
		}
		.merek-hp {
			display: flex;
			align-items: center;
			gap: 0.5rem;
			color: #fff;
			text-decoration: none;
			min-width: 0;
		}
		.merek-hp strong {
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}
		.nama-akun,
		.akun .badge {
			display: none;
		}
		.akun summary {
			padding: 0.2rem;
		}
		.konten {
			padding: 1.25rem 1rem 6rem;
		}
		.bawah {
			display: flex;
			position: fixed;
			bottom: 0;
			left: 0;
			right: 0;
			background: #fff;
			border-top: 1px solid var(--garis);
			z-index: 10;
			padding-bottom: env(safe-area-inset-bottom);
		}
		.bawah a {
			flex: 1;
			text-align: center;
			padding: 0.9rem 0.25rem;
			text-decoration: none;
			color: var(--teks-redup);
			font-weight: 600;
			font-size: 0.9rem;
		}
		.bawah a[aria-current='page'] {
			color: var(--hijau);
			box-shadow: inset 0 3px 0 var(--hijau);
		}
	}
	@media print {
		.shell {
			display: block;
		}
		.konten {
			padding: 0;
		}
	}
</style>
