<!--
  Perantara menuju halaman bayar gateway. Navigasi otomatis lewat meta refresh
  (tidak butuh JavaScript, tidak terkena batas CSP form-action), dengan tombol
  sebagai cadangan bila peramban tidak mengalihkan.
-->
<script lang="ts">
	import { labelJenisDonasi, rupiah } from '$lib/format';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const p = $derived(data.p);
</script>

<svelte:head>
	<title>Mengalihkan ke pembayaran</title>
	<meta http-equiv="refresh" content="1;url={p.bayar_url}" />
</svelte:head>

<div class="kartu tumpuk">
	<h1>Satu langkah lagi</h1>
	<p>
		Anda akan dibawa ke halaman pembayaran untuk <strong>{labelJenisDonasi[p.jenis]}</strong> sebesar
		<strong>{rupiah(p.jumlah)}</strong> atas nama <strong>{p.nama}</strong>.
	</p>
	<div class="baris-aksi">
		<a class="tombol utama" href={p.bayar_url}>Lanjutkan ke pembayaran</a>
	</div>
	<p class="redup">
		Setelah membayar, Anda akan kembali ke sini untuk melihat kuitansi. Bila sempat tertutup, status donasi bisa dibuka lagi di
		<a href="/donasi/selesai?kode={p.kode}">tautan ini</a>; simpan tautannya.
	</p>
</div>
