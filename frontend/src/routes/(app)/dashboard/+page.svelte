<!-- Beranda (FR-18): 4 kartu ringkasan, stepper 5 langkah periode aktif, periode terbaru. -->
<script lang="ts">
	import Status from '$lib/components/Status.svelte';
	import { angka, labelAsnaf, rupiahRingkas } from '$lib/format';
	import type { Asnaf } from '$lib/types';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const d = $derived(data.dash);
	const admin = $derived(data.user.role === 'admin');

	const rincianAsnaf = $derived(
		Object.entries(d.asnaf_counts)
			.map(([a, n]) => `${labelAsnaf[a as Asnaf]} ${n}`)
			.join(' · ')
	);

	const langkah = ['Pilih kandidat', 'Isi penilaian', 'Hitung ARAS', 'Persetujuan Ketua', 'Penyaluran'];
	// Ke tab mana tombol utama mengarah, sesuai langkah aktif (indeks 1–5).
	const tujuan = ['kandidat', 'kandidat', 'penilaian', 'penilaian', 'hasil', 'penyaluran'];

	const keteranganPeriode = $derived.by(() => {
		const a = d.aktif;
		if (!a) return 'Tidak ada periode berjalan';
		if (a.periode.status === 'ditolak') return 'Usulan ditolak, perlu diperbaiki';
		return ['', 'Menunggu pemilihan kandidat', 'Penilaian belum lengkap', 'Siap dihitung', 'Menunggu persetujuan Ketua', 'Penyaluran berjalan'][a.langkah_aktif];
	});
</script>

<svelte:head><title>Beranda · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Assalamu'alaikum, {data.user.nama}</h1>
		<p class="redup">Ringkasan penyaluran dana masjid.</p>
	</div>

	<div class="grid-kartu">
		<div class="kartu ringkas">
			<p class="redup">Mustahik aktif</p>
			<p class="besar">{angka(d.mustahik_aktif)}</p>
			<p class="redup kecil">{rincianAsnaf || 'Belum ada data'}</p>
		</div>
		<div class="kartu ringkas">
			<p class="redup">Periode berjalan</p>
			<p class="besar">{d.periode_berjalan}</p>
			<p class="redup kecil">{keteranganPeriode}</p>
		</div>
		<div class="kartu ringkas">
			<p class="redup">Dana periode aktif</p>
			<p class="besar">{d.aktif ? rupiahRingkas(d.aktif.periode.dana_tersedia) : '–'}</p>
			<p class="redup kecil">
				{d.aktif ? `Kuota ${d.aktif.kuota} penerima × ${rupiahRingkas(d.aktif.periode.nominal_per_penerima)}` : 'Belum ada periode aktif'}
			</p>
		</div>
		<div class="kartu ringkas">
			<p class="redup">Total tersalur</p>
			<p class="besar">{rupiahRingkas(d.aktif?.total_tersalur ?? 0)}</p>
			<p class="redup kecil">{d.aktif?.total_tersalur ? 'Pada periode aktif' : 'Belum ada penyaluran tercatat'}</p>
		</div>
	</div>

	{#if d.aktif}
		{@const a = d.aktif}
		<section class="kartu">
			<div class="kartu-kepala">
				<div>
					<h2>{a.periode.nama}</h2>
					<p class="redup">
						Langkah berikutnya: {a.periode.status === 'ditolak' ? 'buka kembali & perbaiki penilaian' : langkah[a.langkah_aktif - 1].toLowerCase()}
					</p>
				</div>
				<a class="tombol utama" href="/periode/{a.periode.id}/{tujuan[a.langkah_aktif]}">
					{a.langkah_aktif >= 4 ? 'Lihat hasil ranking' : 'Lanjutkan'}
				</a>
			</div>
			<ol class="stepper">
				{#each langkah as l, i (l)}
					{@const no = i + 1}
					{@const keadaan = no < a.langkah_aktif ? 'selesai' : no === a.langkah_aktif ? 'sekarang' : 'nanti'}
					<li class={keadaan} aria-current={keadaan === 'sekarang' ? 'step' : undefined}>
						<small>{no} · {keadaan.toUpperCase()}</small>
						<strong>{l}</strong>
					</li>
				{/each}
			</ol>
		</section>
	{/if}

	<section class="kartu">
		<div class="kartu-kepala">
			<h2>Periode terbaru</h2>
			{#if admin}<a class="tombol" href="/periode/baru">+ Buat periode</a>{/if}
		</div>
		{#if d.terbaru.length === 0}
			<div class="kosong">
				<p><strong>Belum ada data</strong></p>
				<p class="redup">
					{admin ? 'Mulai dengan menambah mustahik, lalu buat periode penyaluran pertama.' : 'Admin belum membuat periode penyaluran.'}
				</p>
				{#if admin}<a class="tombol utama" href="/mustahik/baru">Tambah mustahik</a>{/if}
			</div>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead>
						<tr><th>Nama periode</th><th>Jenis</th><th class="kanan">Dana (Rp)</th><th>Status</th><th class="kanan">Aksi</th></tr>
					</thead>
					<tbody>
						{#each d.terbaru as p (p.id)}
							<tr>
								<td><strong>{p.nama}</strong></td>
								<td>{p.jenis === 'zakat' ? 'Zakat' : 'Infak'}</td>
								<td class="kanan angka">{angka(p.dana_tersedia)}</td>
								<td><Status status={p.status} /></td>
								<td class="kanan">
									{#if p.status === 'selesai'}
										<a href="/periode/{p.id}/laporan">Laporan</a>
									{:else}
										<a href="/periode/{p.id}">Buka</a>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</div>

<style>
	.ringkas {
		display: grid;
		gap: 0.35rem;
	}
	.besar {
		font-size: 2.3rem;
		font-weight: 700;
		line-height: 1.1;
	}
	.kecil {
		font-size: 0.9rem;
	}
	.stepper {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 0.75rem;
	}
	.stepper li {
		border-radius: 12px;
		padding: 0.9rem 1rem;
		display: grid;
		gap: 0.2rem;
		background: var(--abu);
		border: 2px solid transparent;
	}
	.stepper small {
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.03em;
	}
	.stepper .selesai {
		background: var(--hijau-muda);
		color: var(--hijau-tua);
	}
	.stepper .sekarang {
		background: var(--kuning);
		border-color: var(--kuning-garis);
		color: #5a3c00;
	}
	.kosong {
		display: grid;
		gap: 0.6rem;
		justify-items: center;
		text-align: center;
		padding: 2rem 1rem;
	}
	@media (max-width: 900px) {
		.stepper {
			grid-template-columns: 1fr;
		}
	}
</style>
