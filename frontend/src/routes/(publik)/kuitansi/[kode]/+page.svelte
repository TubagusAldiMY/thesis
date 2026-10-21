<!--
  Kuitansi penerimaan zakat/infak — satu template untuk layar dan cetak/PDF
  (window.print(); @media print menyembunyikan tombol dan bingkai, kertas A5 lanskap
  seperti PDF lampiran email di backend/internal/kuitansi). Muncul hanya untuk donasi
  berstatus diterima/dibatalkan; selain itu hanya pesan status.
  Donasi dibatalkan tetap tampil dengan stempel "DIBATALKAN" (nomor tidak hilang).
-->
<script lang="ts">
	import { page } from '$app/state';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { labelJenisDonasi, namaMetode, rupiah, tanggalPanjang, terbilang } from '$lib/format';
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

	<article class="kuitansi" class:batal>
		<div class="pita" aria-hidden="true"></div>
		<div class="badan">
			<header class="kop">
				<div class="identitas">
					<IkonMasjid ukuran={60} />
					<div>
						<h1>{masjid.nama}</h1>
						{#if kopMasjid}<p class="alamat">{kopMasjid}</p>{/if}
					</div>
				</div>
				<div class="judul">
					<strong class="nama-dokumen">KUITANSI {judulJenis}</strong>
					<span class="nomor">No. {p.nomor_kuitansi}</span>
				</div>
			</header>

			{#if batal}<p class="stempel" role="status">DIBATALKAN</p>{/if}

			<dl class="isi">
				<dt>Telah terima dari</dt>
				<dd class="donatur">{p.nama}</dd>
				<dt>Terbilang</dt>
				<dd><em>{terbilang(p.jumlah)}</em></dd>
				<dt>Untuk pembayaran</dt>
				<dd>{labelJenisDonasi[p.jenis]}</dd>
				<dt>Metode</dt>
				<dd>{metode}</dd>
				<dt>Tanggal</dt>
				<dd>{p.dibayar_pada ? tanggalPanjang(p.dibayar_pada) : '-'}</dd>
			</dl>

			<div class="penutup">
				<div class="jumlah">
					<span>Jumlah</span>
					<strong>{rupiah(p.jumlah)}</strong>
				</div>
				<div class="ttd">
					<p>Penerima,</p>
					<p class="nama-ttd">Pengurus {masjid.nama}</p>
				</div>
			</div>

			<p class="catatan">
				Kuitansi ini diterbitkan oleh sistem {masjid.nama}{batal ? ' dan telah dibatalkan oleh pengurus' : ''}.
			</p>
		</div>
	</article>
{/if}

<style>
	/* Kertas A5 lanskap: ukuran sama dengan PDF lampiran email. */
	@page {
		size: A5 landscape;
		margin: 8mm;
	}

	.aksi {
		margin-bottom: 0.75rem; /* tombol tidak menempel ke tepi kuitansi */
	}

	.kuitansi {
		position: relative;
		overflow: hidden;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 16px;
		box-shadow:
			0 1px 2px rgb(23 71 58 / 0.06),
			0 16px 36px -16px rgb(23 71 58 / 0.25);
	}
	.pita {
		height: 10px;
		background: linear-gradient(90deg, var(--hijau-tua), var(--hijau-aktif));
		print-color-adjust: exact;
		-webkit-print-color-adjust: exact;
	}
	.badan {
		padding: 1.5rem 2rem 1.25rem;
	}

	.kop {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1.25rem;
		padding-bottom: 1.1rem;
		border-bottom: 2px solid var(--hijau-tua);
		margin-bottom: 1.25rem;
	}
	.identitas {
		display: flex;
		align-items: center;
		gap: 0.9rem;
		min-width: 0;
	}
	.kop h1 {
		margin: 0;
		font-size: 1.2rem;
		color: var(--hijau-tua);
	}
	.alamat {
		margin: 0.2rem 0 0;
		font-size: 0.85rem;
		color: var(--teks-redup);
	}
	/* Alamat panjang menyempitkan kolom ini sampai nomor terpotong; kolom judul tidak boleh menyusut. */
	.judul {
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 0.45rem;
	}
	.nama-dokumen {
		font-size: 1.05rem;
		letter-spacing: 0.06em;
		color: var(--hijau-tua);
		white-space: nowrap;
	}
	.nomor {
		padding: 0.2rem 0.75rem;
		border: 1px solid var(--hijau-tua);
		border-radius: 999px;
		font-size: 0.85rem;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}

	/* Isian bergaris titik-titik seperti kuitansi kertas. */
	.isi {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 0.7rem 1.5rem;
		align-items: baseline;
		margin: 0;
	}
	.isi dt {
		color: var(--teks-redup);
		font-size: 0.9rem;
	}
	.isi dd {
		margin: 0;
		padding-bottom: 0.2rem;
		border-bottom: 1px dotted #b7b3a6;
		min-width: 0;
		overflow-wrap: anywhere;
	}
	.donatur {
		font-weight: 600;
	}

	.penutup {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 1.5rem;
		margin-top: 1.6rem;
	}
	.jumlah {
		display: flex;
		flex-direction: column;
		gap: 0.1rem;
		padding: 0.65rem 1.25rem 0.75rem;
		background: var(--hijau-muda);
		border: 1.5px solid var(--hijau-tua);
		border-radius: 12px;
		print-color-adjust: exact;
		-webkit-print-color-adjust: exact;
	}
	.jumlah span {
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--hijau);
	}
	.jumlah strong {
		font-size: 1.8rem;
		line-height: 1.1;
		font-weight: 800;
		font-variant-numeric: tabular-nums;
		color: var(--hijau-tua);
		white-space: nowrap;
	}
	.ttd {
		text-align: center;
		min-width: 12rem;
		font-size: 0.9rem;
	}
	.ttd p {
		margin: 0;
	}
	/* Ruang tanda tangan di atas garis, nama pengurus di bawahnya. */
	.nama-ttd {
		margin-top: 3.2rem !important;
		padding-top: 0.3rem;
		border-top: 1px solid var(--teks);
		font-weight: 600;
	}
	.catatan {
		margin: 1.1rem 0 0;
		font-size: 0.78rem;
		color: var(--teks-redup);
	}

	.stempel {
		position: absolute;
		top: 42%;
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
	.batal .isi,
	.batal .jumlah strong {
		text-decoration: line-through;
		text-decoration-color: var(--merah);
	}

	@media (max-width: 520px) {
		.badan {
			padding: 1.25rem 1.1rem 1rem;
		}
		.kop {
			flex-direction: column;
		}
		.judul {
			align-items: flex-start;
		}
		.isi {
			grid-template-columns: 1fr;
			gap: 0.1rem;
		}
		.isi dd {
			margin-bottom: 0.6rem;
		}
		.penutup {
			flex-direction: column;
			align-items: stretch;
		}
		.ttd {
			align-self: flex-end;
		}
	}

	@media print {
		.kuitansi {
			border: 1.5px solid var(--hijau-tua);
			border-radius: 10px;
			box-shadow: none;
			break-inside: avoid;
		}
		.badan {
			padding: 1rem 1.5rem 0.9rem;
		}
		.kop {
			padding-bottom: 0.8rem;
			margin-bottom: 0.9rem;
		}
		.isi {
			gap: 0.5rem 1.5rem;
		}
		.penutup {
			margin-top: 1rem;
		}
		.nama-ttd {
			margin-top: 2.4rem !important;
		}
		.catatan {
			margin-top: 0.8rem;
		}
	}
</style>
