<!-- Halaman bayar PALSU untuk development/demo. Lihat catatan di +page.server.ts. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { labelJenisDonasi, rupiah } from '$lib/format';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	let mengirim = $state(false);
	const p = $derived(data.p);
</script>

<svelte:head><title>Simulasi pembayaran</title></svelte:head>

<div class="kartu tumpuk">
	<p class="kotak-galat" role="note">
		<strong>SIMULASI.</strong> Ini bukan pembayaran sungguhan. Halaman ini hanya ada saat penyedia pembayaran tiruan menyala
		(development/demo).
	</p>
	<h1>Bayar {rupiah(p.jumlah)}</h1>
	<p>{labelJenisDonasi[p.jenis]} atas nama <strong>{p.nama}</strong></p>

	<form
		method="POST"
		use:enhance={() => {
			mengirim = true;
			return async ({ update }) => {
				await update();
				mengirim = false;
			};
		}}
	>
		<div class="baris-aksi">
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Memproses…' : 'Simulasikan bayar berhasil'}</button>
		</div>
	</form>
</div>
