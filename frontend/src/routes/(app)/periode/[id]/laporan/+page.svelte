<!--
  Laporan periode (FR-14) — satu template untuk layar, cetak/PDF, dan CSV:
  kop masjid, ringkasan, A. kriteria & bobot (potret saat dihitung), B. ranking,
  C. realisasi + total, kolom tanda tangan Admin & Ketua. Cetak lewat
  window.print(); @media print menyembunyikan navigasi & tombol (NFR-09).
-->
<script lang="ts">
	import { angka, desimal, rupiah, tanggal, tanggalSaja } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const l = $derived(data.laporan);
	const p = $derived(l.periode);
	const penerima = $derived(l.ranking.filter((r) => r.status === 'usulan').length);

	async function bagikan() {
		// Android: Web Share API bila tersedia; selain itu salin tautan.
		const url = location.href;
		if (navigator.share) await navigator.share({ title: `Laporan ${p.nama}`, url }).catch(() => undefined);
		else await navigator.clipboard?.writeText(url);
	}
</script>

<svelte:head><title>Laporan · {p.nama}</title></svelte:head>

<div class="baris-aksi ujung tanpa-cetak">
	<a class="tombol" href="/periode/{p.id}/laporan.csv" download>Unduh CSV</a>
	<button class="tombol" type="button" onclick={bagikan}>Bagikan</button>
	<button class="tombol utama" type="button" onclick={() => window.print()}>Cetak / Simpan PDF</button>
</div>

<article class="kartu laporan">
	<header class="kop">
		<div>
			<h1>{masjid.nama}</h1>
			<p>{masjid.alamat} · {masjid.kontak}</p>
		</div>
		<div class="kanan">
			<strong>LAPORAN PENYALURAN {p.jenis === 'zakat' ? 'ZAKAT' : 'INFAK'}</strong>
			<p>Dicetak {tanggal(l.dibuat_pada)}</p>
		</div>
	</header>

	<section>
		<h2>{p.nama}</h2>
		<dl class="ringkasan">
			<dt>Dana tersedia</dt><dd class="angka">{rupiah(p.dana_tersedia)}</dd>
			<dt>Nominal per penerima</dt><dd class="angka">{rupiah(p.nominal_per_penerima)}</dd>
			<dt>Kuota</dt><dd>{l.kuota} penerima</dd>
			<dt>Penerima disetujui</dt><dd>{penerima} orang</dd>
			<dt>Total tersalur</dt><dd class="angka">{rupiah(l.total_tersalur)}</dd>
			<dt>Metode</dt><dd>ARAS, run v{l.run.versi} ({tanggal(l.run.created_at)}), S₀ = {desimal(l.run.s0)}</dd>
		</dl>
	</section>

	<section>
		<h3>A. Kriteria & bobot</h3>
		<table>
			<thead><tr><th>Kode</th><th>Kriteria</th><th>Tipe</th><th class="kanan">Bobot</th></tr></thead>
			<tbody>
				{#each l.kriteria as k (k.kode)}
					<tr><td>{k.kode}</td><td>{k.nama}</td><td>{k.tipe}</td><td class="kanan angka">{desimal(k.bobot, 2)}</td></tr>
				{/each}
			</tbody>
		</table>
	</section>

	<section>
		<h3>B. Hasil ranking ARAS</h3>
		<table>
			<thead><tr><th>Peringkat</th><th>Nama</th><th class="kanan">S</th><th class="kanan">K</th><th>Keterangan</th></tr></thead>
			<tbody>
				{#each l.ranking as r (r.mustahik.id)}
					<tr>
						<td>{r.ranking}</td>
						<td>{r.mustahik.nama}</td>
						<td class="kanan angka">{desimal(r.s)}</td>
						<td class="kanan angka">{desimal(r.k)}</td>
						<td>{r.status === 'usulan' ? 'Penerima' : 'Cadangan'}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

	<section>
		<h3>C. Realisasi penyaluran</h3>
		{#if l.penyaluran.length === 0}
			<p class="redup">Belum ada penyaluran tercatat.</p>
		{:else}
			<table>
				<thead><tr><th>No</th><th>Nama</th><th>Tanggal</th><th class="kanan">Nominal (Rp)</th></tr></thead>
				<tbody>
					{#each l.penyaluran as b, i (b.mustahik.id)}
						<tr>
							<td>{i + 1}</td>
							<td>{b.mustahik.nama}</td>
							<td>{b.penyaluran ? tanggalSaja(b.penyaluran.tanggal) : ''}</td>
							<td class="kanan angka">{angka(b.penyaluran?.nominal ?? 0)}</td>
						</tr>
					{/each}
				</tbody>
				<tfoot>
					<tr><td colspan="3"><strong>Total</strong></td><td class="kanan angka"><strong>{angka(l.total_tersalur)}</strong></td></tr>
				</tfoot>
			</table>
		{/if}
	</section>

	<footer class="ttd">
		<div>
			<p>Dibuat oleh,<br />Admin / Petugas Amil</p>
			<p class="nama">{l.admin_nama || '[NAMA ADMIN]'}</p>
		</div>
		<div>
			<p>Disetujui oleh,<br />Ketua DKM{l.disetujui_pada ? ` · ${tanggal(l.disetujui_pada)}` : ''}</p>
			<p class="nama">{l.ketua_nama || '[NAMA KETUA DKM]'}</p>
		</div>
	</footer>
</article>

<style>
	.laporan {
		display: grid;
		gap: 1.75rem;
		max-width: 900px;
		margin-top: 1rem;
	}
	/* Di HP tabel laporan digeser per bagian, bukan seluruh halaman. */
	.laporan > section {
		min-width: 0;
		overflow-x: auto;
	}
	.kop {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		border-bottom: 3px double var(--teks);
		padding-bottom: 1rem;
		flex-wrap: wrap;
	}
	.kop h1 {
		font-size: 1.5rem;
	}
	.kop p {
		color: var(--teks-redup);
		font-size: 0.92rem;
	}
	h3 {
		margin-bottom: 0.6rem;
	}
	.ringkasan {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 0.35rem 1.5rem;
		margin: 0.75rem 0 0;
	}
	.ringkasan dt {
		color: var(--teks-redup);
	}
	.ringkasan dd {
		margin: 0;
		font-weight: 600;
	}
	th,
	td {
		padding: 0.5rem 0.6rem;
		border: 1px solid var(--garis);
	}
	tfoot td {
		background: var(--krem);
	}
	.ttd {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 2rem;
		margin-top: 1rem;
		text-align: center;
	}
	.ttd .nama {
		margin-top: 4.5rem;
		font-weight: 700;
		text-decoration: underline;
	}
	@media print {
		.laporan {
			max-width: none;
			margin: 0;
		}
		th,
		td {
			border-color: #999;
		}
		section,
		.ttd {
			break-inside: avoid;
		}
	}
</style>
