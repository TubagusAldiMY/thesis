<!-- Halaman error: 403 "Akses tidak diizinkan", 404 "Halaman tidak ditemukan", lainnya umum. -->
<script lang="ts">
	import { page } from '$app/state';

	const judul = $derived(
		page.status === 403
			? 'Akses tidak diizinkan'
			: page.status === 404
				? 'Halaman tidak ditemukan'
				: page.status === 503
					? 'Layanan belum tersedia'
					: 'Terjadi kesalahan'
	);
</script>

<svelte:head><title>{judul} · SPK Zakat</title></svelte:head>

<main class="wadah">
	<div class="kartu isi">
		<p class="kode angka">{page.status}</p>
		<h1>{judul}</h1>
		<p class="redup">{page.error?.message}</p>
		{#if page.error?.requestId}
			<p class="redup kecil">Kode permintaan: <span class="angka">{page.error.requestId}</span></p>
		{/if}
		<div class="baris-aksi">
			<a class="tombol utama" href="/dashboard">Kembali ke beranda</a>
		</div>
	</div>
</main>

<style>
	.wadah {
		min-height: 100vh;
		display: grid;
		place-items: center;
		padding: 1rem;
	}
	.isi {
		max-width: 480px;
		display: grid;
		gap: 0.75rem;
		text-align: center;
		justify-items: center;
	}
	.kode {
		font-size: 3rem;
		font-weight: 700;
		color: var(--hijau);
	}
	.kecil {
		font-size: 0.85rem;
	}
</style>
