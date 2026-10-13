<!-- Daftar mustahik (FR-03): cari nama, chip golongan dinamis, paging. NIK/HP hanya tersamar. -->
<script lang="ts">
	import { labelAsnaf } from '$lib/format';
	import type { Asnaf } from '$lib/types';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const h = $derived(data.hasil);
	const totalAktif = $derived(Object.values(h.asnaf_counts).reduce((a, b) => a + (b ?? 0), 0));
	const jumlahHalaman = $derived(Math.max(1, Math.ceil(h.total / data.perHalaman)));

	/** tautan membuat URL dengan filter saat ini + perubahan. */
	function tautan(ubah: Record<string, string | number>) {
		const p = new URLSearchParams({ q: data.q, asnaf: data.asnaf, page: String(data.page) });
		for (const [k, v] of Object.entries(ubah)) p.set(k, String(v));
		for (const k of [...p.keys()]) if (!p.get(k) || (k === 'page' && p.get(k) === '1')) p.delete(k);
		const s = p.toString();
		return '/mustahik' + (s ? '?' + s : '');
	}
</script>

<svelte:head><title>Mustahik · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Mustahik</h1>
			<p class="redup">{totalAktif} mustahik aktif. Mustahik tidak dihapus; nonaktifkan bila tidak lagi diusulkan.</p>
		</div>
		<a class="tombol utama tambah" href="/mustahik/baru">+ Tambah mustahik</a>
	</div>

	{#if data.tersimpan}<p class="kotak-sukses" role="status">Data mustahik tersimpan.</p>{/if}

	<section class="kartu tumpuk">
		<form method="GET" class="cari" role="search">
			<label class="sr-only" for="q">Cari nama</label>
			<input id="q" name="q" type="search" placeholder="Cari nama mustahik…" value={data.q} maxlength="100" />
			{#if data.asnaf}<input type="hidden" name="asnaf" value={data.asnaf} />{/if}
			<button class="tombol" type="submit">Cari</button>
		</form>

		<nav class="chips" aria-label="Filter golongan">
			<a class="chip" href={tautan({ asnaf: '', page: 1 })} aria-current={data.asnaf === ''}>Semua</a>
			{#each Object.entries(h.asnaf_counts) as [a, n] (a)}
				<a class="chip" href={tautan({ asnaf: a, page: 1 })} aria-current={data.asnaf === a}>{labelAsnaf[a as Asnaf]} {n}</a>
			{/each}
		</nav>

		{#if h.items.length === 0}
			<p class="redup kosong">{data.q || data.asnaf ? 'Tidak ada mustahik yang cocok dengan filter.' : 'Belum ada mustahik. Tambahkan calon penerima pertama.'}</p>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead>
						<tr><th>Nama</th><th class="lebar-saja">NIK</th><th>Golongan</th><th class="lebar-saja">Alamat</th><th>Status</th><th class="kanan">Aksi</th></tr>
					</thead>
					<tbody>
						{#each h.items as m (m.id)}
							<tr>
								<td><strong>{m.nama}</strong></td>
								<td class="angka redup lebar-saja nowrap">{m.nik_tersamar}</td>
								<td>{labelAsnaf[m.asnaf]}</td>
								<td class="alamat lebar-saja">{m.alamat}</td>
								<td><span class="badge {m.aktif ? 'hijau' : ''}">{m.aktif ? 'Aktif' : 'Nonaktif'}</span></td>
								<td class="kanan"><a href="/mustahik/{m.id}">Ubah</a></td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="paging">
				<span class="redup">Halaman {data.page} dari {jumlahHalaman} · {h.total} data</span>
				<div class="baris-aksi">
					{#if data.page > 1}<a class="tombol kecil" href={tautan({ page: data.page - 1 })}>← Sebelumnya</a>{/if}
					{#if data.page < jumlahHalaman}<a class="tombol kecil" href={tautan({ page: data.page + 1 })}>Berikutnya →</a>{/if}
				</div>
			</div>
		{/if}
	</section>
</div>

<style>
	.cari {
		display: flex;
		gap: 0.75rem;
	}
	.alamat {
		max-width: 28ch;
		color: var(--teks-redup);
	}
	.nowrap {
		white-space: nowrap;
	}
	.kosong {
		padding: 1.5rem 0;
		text-align: center;
	}
	.paging {
		display: flex;
		justify-content: space-between;
		align-items: center;
		flex-wrap: wrap;
		gap: 0.75rem;
	}
	@media (max-width: 640px) {
		/* HP: NIK & alamat disembunyikan; tetap terlihat di formulir Ubah */
		.lebar-saja {
			display: none;
		}
	}
	@media (max-width: 900px) {
		/* Android: tombol tambah mengambang di atas navigasi bawah */
		.tambah {
			position: fixed;
			right: 1rem;
			bottom: 5rem;
			z-index: 11;
			border-radius: 999px;
			box-shadow: 0 8px 20px rgba(31, 90, 69, 0.35);
		}
	}
</style>
