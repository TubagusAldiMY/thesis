<!-- Daftar periode + chip filter status (Semua/Draft/Dihitung/Disetujui/Ditolak/Selesai). -->
<script lang="ts">
	import Status from '$lib/components/Status.svelte';
	import { angka, labelStatus, tanggal } from '$lib/format';
	import type { StatusPeriode } from '$lib/types';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const admin = $derived(data.user.role === 'admin');
	const filter: ('' | StatusPeriode)[] = ['', 'draft', 'dihitung', 'disetujui', 'ditolak', 'selesai'];
</script>

<svelte:head><title>Periode · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Periode penyaluran</h1>
			<p class="redup">Satu periode = satu putaran penyaluran zakat atau infak. <a href="/laporan">Lihat laporan periode selesai →</a></p>
		</div>
		{#if admin}<a class="tombol utama" href="/periode/baru">+ Buat periode</a>{/if}
	</div>

	<section class="kartu tumpuk">
		<nav class="chips" aria-label="Filter status">
			{#each filter as s (s)}
				<a class="chip" href={s ? `/periode?status=${s}` : '/periode'} aria-current={data.status === s}>{s ? labelStatus[s] : 'Semua'}</a>
			{/each}
		</nav>

		{#if data.items.length === 0}
			<p class="redup kosong">Belum ada periode{data.status ? ` berstatus ${labelStatus[data.status as StatusPeriode].toLowerCase()}` : ''}.</p>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead>
						<tr>
							<th>Nama periode</th><th>Jenis</th><th class="kanan">Dana (Rp)</th><th class="kanan">Per penerima</th>
							<th class="kanan">Kuota</th><th>Dibuat</th><th>Status</th><th class="kanan">Aksi</th>
						</tr>
					</thead>
					<tbody>
						{#each data.items as p (p.id)}
							<tr>
								<td><strong>{p.nama}</strong></td>
								<td>{p.jenis === 'zakat' ? 'Zakat' : 'Infak'}</td>
								<td class="kanan angka">{angka(p.dana_tersedia)}</td>
								<td class="kanan angka">{angka(p.nominal_per_penerima)}</td>
								<td class="kanan">{p.kuota}</td>
								<td>{tanggal(p.created_at)}</td>
								<td><Status status={p.status} /></td>
								<td class="kanan aksi">
									<a href="/periode/{p.id}">Buka</a>
									{#if p.status === 'selesai'} · <a href="/periode/{p.id}/laporan">Laporan</a>{/if}
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
	.kosong {
		padding: 1.5rem 0;
		text-align: center;
	}
	.aksi {
		white-space: nowrap;
	}
</style>
