<!--
  Perantara menuju pembayaran. Dengan Client Key + JavaScript: Snap popup Midtrans
  terbuka di atas halaman ini. Tanpa itu (kunci kosong, snap.js gagal dimuat, atau
  JavaScript mati): muzaki dibawa ke halaman bayar Midtrans lewat navigasi biasa
  (tidak terkena batas CSP form-action), dengan tombol sebagai cadangan.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { labelJenisDonasi, rupiah } from '$lib/format';
	import { infoSnap } from '$lib/snap';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const p = $derived(data.p);
	const bayarUrl = $derived(p.bayar_url ?? '');
	// null = tanpa popup (Client Key kosong / tautan bukan tautan Snap yang dikenali).
	const snap = $derived(data.kunciKlien ? infoSnap(p.bayar_url) : null);
	let popupDitutup = $state(false);

	interface SnapJs {
		pay(token: string, cb: { onSuccess(): void; onPending(): void; onError(): void; onClose(): void }): void;
	}

	// Hasil pembayaran TIDAK dipercaya dari callback ini: semuanya hanya membawa muzaki
	// ke halaman status, yang menunggu kabar resmi (webhook) dari Midtrans.
	function bukaPopup() {
		const s = (window as unknown as { snap?: SnapJs }).snap;
		if (!snap || !s) return;
		popupDitutup = false;
		const keStatus = () => goto('/donasi/selesai?kode=' + p.kode);
		s.pay(snap.token, {
			onSuccess: keStatus,
			onPending: keStatus,
			onError: keStatus,
			onClose: () => (popupDitutup = true)
		});
	}

	onMount(() => {
		if (!snap) {
			window.location.assign(bayarUrl);
			return;
		}
		const el = document.createElement('script');
		el.src = snap.skrip;
		el.dataset.clientKey = data.kunciKlien;
		el.onload = bukaPopup;
		el.onerror = () => window.location.assign(bayarUrl); // CSP/jaringan memblokir snap.js
		document.head.appendChild(el);
		return () => el.remove();
	});
</script>

<svelte:head>
	<title>Pembayaran donasi</title>
	<noscript><meta http-equiv="refresh" content="1;url={p.bayar_url}" /></noscript>
</svelte:head>

<div class="kartu tumpuk">
	<h1>Satu langkah lagi</h1>
	<p>
		Pembayaran untuk <strong>{labelJenisDonasi[p.jenis]}</strong> sebesar
		<strong>{rupiah(p.jumlah)}</strong> atas nama <strong>{p.nama}</strong>.
	</p>
	<div class="baris-aksi">
		{#if snap && popupDitutup}
			<button type="button" class="tombol utama" onclick={bukaPopup}>Buka pembayaran lagi</button>
			<a class="tombol" href={bayarUrl}>Bayar di halaman Midtrans</a>
		{:else}
			<a class="tombol utama" href={bayarUrl}>Lanjutkan ke pembayaran</a>
		{/if}
	</div>
	<p class="redup">
		Setelah membayar, Anda akan kembali ke sini untuk melihat kuitansi. Bila sempat tertutup, status donasi bisa dibuka lagi di
		<a href="/donasi/selesai?kode={p.kode}">tautan ini</a>; simpan tautannya.
	</p>
</div>
