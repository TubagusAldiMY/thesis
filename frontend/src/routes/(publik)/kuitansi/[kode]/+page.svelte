<!--
  Kuitansi penerimaan zakat/infak — satu template untuk layar dan cetak/PDF
  (window.print(); @media print menyembunyikan tombol dan bingkai). Muncul
  hanya untuk donasi berstatus diterima/dibatalkan; selain itu hanya pesan status.
  Donasi dibatalkan tetap tampil dengan stempel "DIBATALKAN" (nomor tidak hilang).
-->
<script lang="ts">
	import { page } from '$app/state';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { labelJenisDonasi, namaMetode, rupiah, tanggal, terbilang } from '$lib/format';
	import { kopMasjid, masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const p = $derived(data.p);
	const sah = $derived(p.status === 'diterima' || p.status === 'dibatalkan');
	const batal = $derived(p.status === 'dibatalkan');
	const judulJenis = $derived(p.jenis === 'infak' ? 'INFAK' : 'ZAKAT');

	const metode = $derived(namaMetode(p.metode));

	async function bagikan() {
		// HP: Web Share API bila tersedia; selain itu salin tautan.
		const url = page.url.href;
		if (navigator.share) await navigator.share({ title: `Kuitansi ${p.nomor_kuitansi}`, url }).catch(() => undefined);
		else await navigator.clipboard?.writeText(url);
	}
</script>

<svelte:head><title>Kuitansi {p.nomor_kuitansi || ''} · {masjid.nama}</title></svelte:head>

{#if !sah}
	<div class="kartu tumpuk">
		<h1>Kuitansi belum tersedia</h1>
		<p class="redup">Kuitansi terbit setelah pembayaran dikonfirmasi. Status donasi ini: <strong>{p.status}</strong>.</p>
		<div class="baris-aksi">
			<a class="tombol utama" href="/donasi/selesai?kode={p.kode}">Lihat status donasi</a>
		</div>
	</div>
{:else}
	<div class="baris-aksi ujung tanpa-cetak aksi">
		<button class="tombol" type="button" onclick={bagikan}>Bagikan</button>
		<button class="tombol utama" type="button" onclick={() => window.print()}>Cetak / Simpan PDF</button>
	</div>

	<article class="kartu kuitansi" class:batal>
		<header class="kop">
			<div class="identitas">
				<IkonMasjid ukuran={56} />
				<div>
					<h1>{masjid.nama}</h1>
					{#if kopMasjid}<p>{kopMasjid}</p>{/if}
				</div>
			</div>
			<div class="kanan">
				<strong>KUITANSI {judulJenis}</strong>
				<p>No. {p.nomor_kuitansi}</p>
			</div>
		</header>

		{#if batal}<p class="stempel" role="status">DIBATALKAN</p>{/if}

		<dl class="isi">
			<dt>Telah terima dari</dt>
			<dd>{p.nama}</dd>
			<dt>Banyaknya uang</dt>
			<dd class="angka besar">{rupiah(p.jumlah)}</dd>
			<dt>Terbilang</dt>
			<dd><em>{terbilang(p.jumlah)}</em></dd>
			<dt>Untuk pembayaran</dt>
			<dd>{labelJenisDonasi[p.jenis]}</dd>
			<dt>Metode</dt>
			<dd>{metode}</dd>
			<dt>Tanggal</dt>
			<dd>{p.dibayar_pada ? tanggal(p.dibayar_pada) : '-'}</dd>
		</dl>

		<footer class="ttd">
			<p>Penerima,<br />Pengurus {masjid.nama}</p>
			<p class="redup catatan">
				Kuitansi ini diterbitkan oleh sistem {masjid.nama}{batal ? ' dan telah dibatalkan oleh pengurus' : ''}.
			</p>
		</footer>
	</article>
{/if}

<style>
	.kuitansi {
		position: relative;
		padding: 2rem;
		border: 2px solid var(--hijau-tua);
	}
	.kop {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		padding-bottom: 1rem;
		border-bottom: 2px solid var(--hijau-tua);
		margin-bottom: 1.25rem;
	}
	.identitas {
		display: flex;
		align-items: center;
		gap: 0.9rem;
	}
	.kop h1 {
		margin: 0;
		font-size: 1.25rem;
	}
	.kop p {
		margin: 0.15rem 0 0;
	}
	.kanan {
		text-align: right;
		/* Kop beralamat panjang menyempitkan kolom ini sampai "No." dan nomornya
		   terpisah di dua baris; kolom nomor tidak boleh menyusut. */
		flex-shrink: 0;
	}
	.kanan p {
		white-space: nowrap;
	}
	.aksi {
		margin-bottom: 0.75rem; /* tombol tidak menempel ke tepi kartu */
	}
	.isi {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 0.6rem 1.5rem;
		margin: 0;
	}
	.isi dt {
		color: var(--teks-redup);
	}
	.isi dd {
		margin: 0;
	}
	.besar {
		font-size: 1.4rem;
		font-weight: 700;
	}
	.ttd {
		margin-top: 2rem;
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 1rem;
	}
	.ttd p {
		margin: 0;
	}
	.catatan {
		max-width: 32ch;
		font-size: 0.8rem;
	}
	.stempel {
		position: absolute;
		top: 40%;
		left: 50%;
		transform: translate(-50%, -50%) rotate(-18deg);
		margin: 0;
		padding: 0.2em 0.8em;
		border: 4px solid var(--merah);
		border-radius: 6px;
		color: var(--merah);
		font-size: 2.4rem;
		font-weight: 800;
		letter-spacing: 0.08em;
		opacity: 0.55;
		pointer-events: none;
	}
	.batal .isi {
		text-decoration: line-through;
		text-decoration-color: var(--merah);
	}
	@media (max-width: 520px) {
		.kuitansi {
			padding: 1.25rem;
		}
		.kop {
			flex-direction: column;
		}
		.kanan {
			text-align: left;
		}
		.isi {
			grid-template-columns: 1fr;
			gap: 0.1rem;
		}
		.isi dd {
			margin-bottom: 0.6rem;
		}
	}
	@media print {
		.kuitansi {
			border-width: 2px;
			box-shadow: none;
		}
	}
</style>
