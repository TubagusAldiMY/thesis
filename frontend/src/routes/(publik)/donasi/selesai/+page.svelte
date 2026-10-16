<!--
  Status donasi: menunggu konfirmasi, diterima (tautan kuitansi), gagal,
  kedaluwarsa, atau dibatalkan. Selama menunggu, halaman memuat ulang dirinya
  tiap 5 detik lewat meta refresh (tanpa JavaScript) sampai gateway memberi kabar.
-->
<script lang="ts">
	import { labelJenisDonasi, labelStatusDonasi, rupiah, warnaStatusDonasi } from '$lib/format';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const p = $derived(data.p);
</script>

<svelte:head>
	<title>Status donasi</title>
	{#if p.status === 'menunggu'}<meta http-equiv="refresh" content="5" />{/if}
</svelte:head>

<div class="kartu tumpuk">
	<div>
		<span class="badge {warnaStatusDonasi[p.status]}">{labelStatusDonasi[p.status]}</span>
		<h1>
			{#if p.status === 'diterima'}Alhamdulillah, donasi Anda sudah kami terima
			{:else if p.status === 'menunggu'}Menunggu konfirmasi pembayaran
			{:else if p.status === 'dibatalkan'}Kuitansi dibatalkan
			{:else if p.status === 'kedaluwarsa'}Waktu pembayaran habis
			{:else}Pembayaran tidak berhasil{/if}
		</h1>
	</div>

	<dl class="ringkasan">
		<dt>Atas nama</dt><dd>{p.nama}</dd>
		<dt>Jenis</dt><dd>{labelJenisDonasi[p.jenis]}</dd>
		<dt>Jumlah</dt><dd class="angka">{rupiah(p.jumlah)}</dd>
		{#if p.nomor_kuitansi}<dt>No. kuitansi</dt><dd>{p.nomor_kuitansi}</dd>{/if}
	</dl>

	{#if p.status === 'diterima'}
		<p>Jazakumullahu khairan. Kuitansi dapat dilihat dan dicetak kapan saja lewat tautan di bawah; simpan tautan halaman ini.</p>
		<div class="baris-aksi">
			<a class="tombol utama" href="/kuitansi/{p.kode}">Lihat & cetak kuitansi</a>
		</div>
	{:else if p.status === 'menunggu'}
		<p class="redup">
			Halaman ini diperbarui otomatis. Bila Anda sudah membayar, konfirmasi dari penyedia pembayaran biasanya tiba dalam hitungan detik
			sampai beberapa menit.
		</p>
		{#if p.bayar_url}
			<div class="baris-aksi">
				<a class="tombol" href={p.bayar_url}>Belum membayar? Lanjutkan ke pembayaran</a>
			</div>
		{/if}
	{:else if p.status === 'dibatalkan'}
		<p class="redup">Kuitansi ini dibatalkan oleh pengurus masjid. Hubungi pengurus bila ada yang perlu dikonfirmasi.</p>
	{:else}
		<p class="redup">Tidak ada dana yang dipotong untuk donasi ini. Anda dapat mencoba berdonasi kembali.</p>
		<div class="baris-aksi">
			<a class="tombol utama" href="/donasi">Donasi lagi</a>
		</div>
	{/if}
</div>

<style>
	.ringkasan {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 0.35rem 1.25rem;
		margin: 0;
	}
	.ringkasan dt {
		color: var(--teks-redup);
	}
	.ringkasan dd {
		margin: 0;
		font-weight: 600;
	}
</style>
