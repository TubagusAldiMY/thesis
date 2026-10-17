<!--
  Navigasi pil melayang (kaca gelap) untuk halaman publik bergaya landing. Menumpuk di atas hero hijau tua
  halaman yang memakainya (margin bawah negatif), jadi halaman pemakai harus mulai dengan hero gelap.
  Tautan menu diberikan dari luar karena tiap halaman punya jangkar sendiri; menu disembunyikan di layar sempit
  (tombol Donasi tetap ada). Gaya mengikuti navigasi di landing (routes/+page.svelte).
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { masjid } from '$lib/masjid';

	let { tautan = [] }: { tautan?: { href: string; label: string }[] } = $props();
</script>

<header class="atas">
	<div class="pil">
		<a class="merek" href="/">
			<IkonMasjid ukuran={30} />
			<strong>{masjid.nama}</strong>
		</a>
		<nav class="menu" aria-label="Navigasi halaman">
			{#each tautan as t (t.href)}<a href={t.href}>{t.label}</a>{/each}
		</nav>
		<div class="aksi-pil">
			<a class="masuk" href="/login">Masuk pengurus</a>
			<a class="tombol emas" href="/donasi">Donasi</a>
		</div>
	</div>
</header>

<style>
	.atas {
		--emas: #d9a033;
		--emas-terang: #f0cf86;
		position: sticky;
		top: 0.75rem;
		z-index: 20; /* di atas isi saat menggulir */
		margin: 0.75rem auto -4.5rem; /* menumpuk di atas hero */
		max-width: 1120px;
		padding: 0 1rem;
	}
	.pil {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		min-height: 3.75rem;
		padding: 0.5rem 0.6rem 0.5rem 1.1rem;
		border-radius: 999px;
		background: rgb(12 46 37 / 0.72);
		backdrop-filter: blur(14px);
		border: 1px solid rgb(255 255 255 / 0.14);
		box-shadow:
			inset 0 1px 0 rgb(255 255 255 / 0.12),
			0 18px 36px -20px rgb(12 46 37 / 0.7);
		color: #fff;
	}
	.merek {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		color: #fff;
		text-decoration: none;
		min-width: 0;
	}
	.merek strong {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.menu {
		display: flex;
		gap: 1.5rem;
	}
	.menu a {
		color: rgb(255 255 255 / 0.8);
		text-decoration: none;
		font-size: 0.95rem;
		transition: color 0.2s;
	}
	.menu a:hover {
		color: var(--emas-terang);
	}
	.aksi-pil {
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	.masuk {
		color: rgb(255 255 255 / 0.8);
		font-size: 0.9rem;
		text-decoration: none;
		white-space: nowrap;
	}
	.masuk:hover {
		color: #fff;
		text-decoration: underline;
	}
	.emas {
		background: var(--emas);
		border-color: transparent;
		color: #2b1d00;
		font-weight: 600;
	}
	.emas:hover {
		background: var(--emas-terang);
	}
	@media (max-width: 860px) {
		.menu,
		.masuk {
			display: none; /* menu panjang tidak muat; tombol Donasi tetap ada */
		}
	}
</style>
