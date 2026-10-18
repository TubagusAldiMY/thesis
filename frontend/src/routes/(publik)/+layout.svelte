<!--
  Bingkai halaman publik (muzaki, tanpa login): donasi, status pembayaran, kuitansi.
  Sengaja terpisah dari shell aplikasi pengurus (sidebar/menu) — pengunjung tidak
  boleh melihat navigasi internal. Kop dan footer disembunyikan saat dicetak
  supaya kuitansi bersih (@media print lewat kelas .tanpa-cetak).
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { masjid } from '$lib/masjid';
	import { page } from '$app/state';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	// Hanya formulir donasi yang memakai tata letak dua kolom; halaman lain (kuitansi, status) tetap sempit.
	const lebar = $derived(page.route.id === '/(publik)/donasi');
</script>

<div class="bingkai">
	<header class="atas tanpa-cetak">
		<a class="merek" href="/">
			<IkonMasjid ukuran={32} />
			<strong>{masjid.nama}</strong>
		</a>
		<a class="tautan-login" href="/login">Masuk pengurus</a>
	</header>

	<main class="isi" class:lebar>
		{@render children()}
	</main>

	<footer class="bawah tanpa-cetak">
		<small>
			Data pribadi Anda hanya dipakai untuk pencatatan donasi dan kuitansi di {masjid.nama}. Nomor HP disimpan terenkripsi.
		</small>
	</footer>
</div>

<style>
	.bingkai {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}
	.atas {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--garis);
		background: var(--kartu);
	}
	.merek {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		text-decoration: none;
		color: var(--teks);
	}
	.tautan-login {
		font-size: 0.875rem;
		color: var(--teks-redup);
	}
	.isi {
		flex: 1;
		width: 100%;
		max-width: 680px;
		margin: 0 auto;
		padding: 1.5rem 1rem 3rem;
	}
	.isi.lebar {
		max-width: 1240px;
	}
	.bawah {
		padding: 1rem;
		text-align: center;
		color: var(--teks-redup);
	}
	@media print {
		.isi {
			max-width: none;
			padding: 0;
		}
	}
</style>
