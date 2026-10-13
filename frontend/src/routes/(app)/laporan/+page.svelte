<!-- Daftar laporan: setiap baris membuka /periode/[id]/laporan. -->
<script lang="ts">
	import { angka, tanggal } from '$lib/format';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head><title>Laporan · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Laporan</h1>
		<p class="redup">Laporan tersedia untuk periode yang sudah ditutup (selesai).</p>
	</div>
	<section class="kartu">
		{#if data.items.length === 0}
			<p class="redup kosong">Belum ada periode selesai.</p>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead><tr><th>Periode</th><th>Jenis</th><th class="kanan">Dana (Rp)</th><th>Diperbarui</th><th class="kanan">Aksi</th></tr></thead>
					<tbody>
						{#each data.items as p (p.id)}
							<tr>
								<td><strong>{p.nama}</strong></td>
								<td>{p.jenis === 'zakat' ? 'Zakat' : 'Infak'}</td>
								<td class="kanan angka">{angka(p.dana_tersedia)}</td>
								<td>{tanggal(p.updated_at)}</td>
								<td class="kanan"><a href="/periode/{p.id}/laporan">Buka laporan</a></td>
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
</style>
