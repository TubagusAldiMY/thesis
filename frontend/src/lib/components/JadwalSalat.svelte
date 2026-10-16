<!--
  Jadwal salat hari ini + penanda salat berikutnya. Waktu dihitung di aplikasi
  (lib/salat.ts), tanpa API luar. Server menentukan tanggal (WIB) supaya tampilan
  awal sama dengan hasil hidrasi; setelah terpasang, peramban memperbarui "berikutnya"
  tiap 30 detik. Tanda "berikutnya" memakai zona WIB tetap, bukan zona pengunjung.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import { jadwalSalat, jamMenit } from '$lib/salat';

	interface Hari {
		tahun: number;
		bulan: number;
		tanggal: number;
	}
	let { hari, lintang, bujur }: { hari: Hari; lintang: number; bujur: number } = $props();

	// null sampai terpasang di peramban: dipakai agar render server tidak menebak jam.
	let sekarang = $state<number | null>(null);
	let hariKlien = $state<Hari | null>(null);
	const tgl = $derived(hariKlien ?? hari);

	const j = $derived(jadwalSalat(tgl.tahun, tgl.bulan, tgl.tanggal, lintang, bujur));
	const waktu = $derived([
		{ nama: 'Subuh', m: j.subuh, salat: true },
		{ nama: 'Terbit', m: j.terbit, salat: false },
		{ nama: 'Dzuhur', m: j.dzuhur, salat: true },
		{ nama: 'Ashar', m: j.ashar, salat: true },
		{ nama: 'Maghrib', m: j.maghrib, salat: true },
		{ nama: 'Isya', m: j.isya, salat: true }
	]);

	// Salat berikutnya (Terbit bukan salat, dilewati). Lewat Isya → Subuh besok.
	const berikut = $derived.by(() => {
		if (sekarang === null) return null;
		const daftar = waktu.filter((x) => x.salat);
		const ada = daftar.find((x) => x.m > (sekarang as number));
		return ada
			? { nama: ada.nama, sisa: ada.m - (sekarang as number) }
			: { nama: daftar[0].nama, sisa: daftar[0].m + 1440 - (sekarang as number) };
	});

	function sisaTeks(menit: number): string {
		const jam = Math.floor(menit / 60);
		const mnt = menit % 60;
		return jam > 0 ? `${jam} jam ${mnt} menit` : `${mnt} menit`;
	}

	const labelTanggal = $derived(
		new Date(Date.UTC(tgl.tahun, tgl.bulan - 1, tgl.tanggal)).toLocaleDateString('id-ID', {
			weekday: 'long',
			day: 'numeric',
			month: 'long',
			year: 'numeric',
			timeZone: 'UTC'
		})
	);

	onMount(() => {
		const tik = () => {
			const w = new Date(Date.now() + 7 * 3600_000); // jam dinding WIB
			sekarang = w.getUTCHours() * 60 + w.getUTCMinutes();
			hariKlien = { tahun: w.getUTCFullYear(), bulan: w.getUTCMonth() + 1, tanggal: w.getUTCDate() };
		};
		tik();
		const id = setInterval(tik, 30_000);
		return () => clearInterval(id);
	});
</script>

<section class="jadwal" aria-label="Jadwal salat hari ini">
	<div class="kepala">
		<p class="judul">Jadwal salat</p>
		<p class="tanggal">{labelTanggal}</p>
		{#if berikut}
			<p class="berikut" aria-live="polite"><span class="denyut" aria-hidden="true"></span>{berikut.nama} dalam {sisaTeks(berikut.sisa)}</p>
		{/if}
	</div>
	<ul class="waktu">
		{#each waktu as x (x.nama)}
			<li class:aktif={berikut?.nama === x.nama} class:bukan-salat={!x.salat}>
				<span class="nama">{x.nama}</span>
				<strong class="jam">{jamMenit(x.m)}</strong>
			</li>
		{/each}
	</ul>
</section>

<style>
	.jadwal {
		display: grid;
		grid-template-columns: minmax(13rem, 1fr) minmax(0, 3fr);
		gap: 1.5rem;
		align-items: center;
		padding: 1.25rem 1.5rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 22px;
		box-shadow: 0 24px 40px -28px rgb(23 71 58 / 0.4); /* bayangan sewarna hijau */
	}
	.kepala {
		display: grid;
		gap: 0.2rem;
	}
	.kepala p {
		margin: 0;
	}
	.judul {
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--hijau);
	}
	.tanggal {
		font-weight: 600;
	}
	.berikut {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.9rem;
		color: var(--teks-redup);
	}
	.denyut {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: var(--kuning-garis);
		animation: denyut 2s ease-in-out infinite;
	}
	.waktu {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(6, minmax(0, 1fr));
		gap: 0.5rem;
	}
	.waktu li {
		display: grid;
		gap: 0.15rem;
		justify-items: center;
		padding: 0.7rem 0.25rem;
		border-radius: 14px;
		border-bottom: 2px solid transparent;
		transition:
			background-color 0.3s,
			transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
	}
	.nama {
		font-size: 0.8rem;
		color: var(--teks-redup);
	}
	.jam {
		font-size: 1.15rem;
		font-variant-numeric: tabular-nums;
		color: var(--hijau-tua);
	}
	.bukan-salat .jam {
		color: var(--teks-redup);
		font-weight: 500;
	}
	.waktu li.aktif {
		background: var(--hijau-muda);
		border-bottom-color: var(--kuning-garis);
		transform: translateY(-2px);
	}
	@keyframes denyut {
		0%,
		100% {
			opacity: 1;
			transform: scale(1);
		}
		50% {
			opacity: 0.35;
			transform: scale(0.7);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.denyut {
			animation: none;
		}
		.waktu li {
			transition: none;
		}
	}
	@media (max-width: 860px) {
		.jadwal {
			grid-template-columns: minmax(0, 1fr);
			gap: 1rem;
			padding: 1.1rem;
		}
		.waktu {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
	}
</style>
