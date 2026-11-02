<!-- Kelola konten kegiatan keagamaan (CMS): daftar kegiatan (draf dan terbit). Admin dan Ketua sama-sama boleh mengelola. -->
<script lang="ts">
	import TabKonten from '$lib/components/TabKonten.svelte';
	import { labelStatusKegiatan, tanggalAcara, warnaStatusKegiatan } from '$lib/kegiatan';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const items = $derived(data.daftar.items);
</script>

<svelte:head><title>Konten kegiatan · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Konten kegiatan</h1>
			<p class="redup">
				Kegiatan keagamaan Masjid Jami Al-Barokah (Maulid Nabi, Isra Mi'raj, kajian, dll.). Yang berstatus Terbit tampil di halaman publik <a href="/kegiatan">/kegiatan</a>.
			</p>
		</div>
		<a class="tombol utama" href="/konten/baru">+ Kegiatan baru</a>
	</div>

	<TabKonten />

	{#if data.pesan}<p class="kotak-sukses" role="status">{data.pesan}</p>{/if}

	{#if items.length === 0}
		<p class="kartu redup">Belum ada kegiatan. Buat yang pertama dengan tombol "Kegiatan baru".</p>
	{:else}
		<div class="kartu tabel-wadah">
			<table>
				<thead>
					<tr><th>Kegiatan</th><th class="lebar-saja">Kategori</th><th>Tanggal acara</th><th>Status</th></tr>
				</thead>
				<tbody>
					{#each items as k (k.id)}
						<tr>
							<td><a href="/konten/{k.id}"><strong>{k.judul}</strong></a></td>
							<td class="lebar-saja">{k.kategori_label}</td>
							<td>{k.tanggal_acara ? tanggalAcara(k.tanggal_acara, k.jam_acara) : '-'}</td>
							<td><span class="badge {warnaStatusKegiatan[k.status]}">{labelStatusKegiatan[k.status]}</span></td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style>
	@media (max-width: 700px) {
		.lebar-saja {
			display: none;
		}
	}
</style>
