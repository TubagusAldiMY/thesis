<!-- Kelola program donasi: daftar semua program (aktif, draf, selesai) dengan kemajuan dana. Hanya Admin yang membuat. -->
<script lang="ts">
	import { labelStatusProgram, rupiah, sisaWaktu, warnaStatusProgram } from '$lib/format';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const admin = $derived(data.user.role === 'admin');
</script>

<svelte:head><title>Program donasi · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Program Donasi</h1>
			<p class="redup">Penggalangan infak bertarget (mis. renovasi, santunan). Program aktif tampil di halaman publik <a href="/program">/program</a>.</p>
		</div>
		{#if admin}<a class="tombol utama" href="/program-donasi/baru">+ Program baru</a>{/if}
	</div>

	{#if data.program.length === 0}
		<p class="kartu redup">Belum ada program donasi.{admin ? ' Buat program pertama dengan tombol "Program baru".' : ''}</p>
	{:else}
		<div class="kartu tabel-wadah">
			<table>
				<thead>
					<tr><th>Program</th><th>Status</th><th class="kanan">Terkumpul</th><th class="lebar-saja">Kemajuan</th><th class="lebar-saja">Waktu</th></tr>
				</thead>
				<tbody>
					{#each data.program as p (p.id)}
						<tr>
							<td><a href="/program-donasi/{p.id}"><strong>{p.judul}</strong></a><small class="redup blok">{p.donatur} donatur</small></td>
							<td><span class="badge {warnaStatusProgram[p.status]}">{labelStatusProgram[p.status]}</span></td>
							<td class="kanan angka nowrap">{rupiah(p.terkumpul)}<small class="redup blok">dari {rupiah(p.target)}</small></td>
							<td class="lebar-saja">
								<div class="bar" aria-label="Kemajuan {p.persen}%"><span style:width="{p.persen}%"></span></div>
								<small class="redup">{p.persen}%</small>
							</td>
							<td class="lebar-saja nowrap">{p.status === 'draf' ? '-' : sisaWaktu(p)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style>
	.blok {
		display: block;
	}
	.nowrap {
		white-space: nowrap;
	}
	.bar {
		width: 140px;
		height: 8px;
		border-radius: 999px;
		background: var(--hijau-muda);
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--hijau);
	}
	@media (max-width: 700px) {
		.lebar-saja {
			display: none;
		}
	}
</style>
