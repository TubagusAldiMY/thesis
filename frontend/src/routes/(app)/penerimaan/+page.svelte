<!--
  Daftar donasi: ringkasan penerimaan, penanda donasi web baru (notifikasi),
  filter status/jenis/nama, paging. Donasi web masuk otomatis saat pembayaran
  dikonfirmasi gateway; donasi tunai/transfer langsung dicatat Admin.
-->
<script lang="ts">
	import { labelJenisDonasi, labelStatusDonasi, rupiah, semuaJenisDonasi, tanggal, warnaStatusDonasi } from '$lib/format';
	import type { StatusDonasi } from '$lib/types';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const h = $derived(data.hasil);
	const admin = $derived(data.user.role === 'admin');
	const jumlahHalaman = $derived(Math.max(1, Math.ceil(h.total / data.perHalaman)));
	const statusFilter: StatusDonasi[] = ['diterima', 'menunggu', 'gagal', 'kedaluwarsa', 'dibatalkan'];

	/** tautan membuat URL dengan filter saat ini + perubahan. */
	function tautan(ubah: Record<string, string | number>) {
		const p = new URLSearchParams({ q: data.q, status: data.status, jenis: data.jenis, program: data.program?.id ?? '', page: String(data.page) });
		for (const [k, v] of Object.entries(ubah)) p.set(k, String(v));
		for (const k of [...p.keys()]) if (!p.get(k) || (k === 'page' && p.get(k) === '1')) p.delete(k);
		const s = p.toString();
		return '/penerimaan' + (s ? '?' + s : '');
	}
</script>

<svelte:head><title>Donasi · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Donasi</h1>
			<p class="redup">Penerimaan zakat dan infak: dari halaman publik (dibayar lewat gateway) dan yang dicatat petugas.</p>
		</div>
		{#if admin}<a class="tombol utama tambah" href="/penerimaan/baru">+ Catat donasi</a>{/if}
	</div>

	{#if data.tercatat}<p class="kotak-sukses" role="status">Donasi tercatat dan kuitansi sudah terbit.</p>{/if}

	{#if h.ringkas.baru > 0}
		<div class="kotak-info baris-baru" role="status">
			<span><strong>{h.ringkas.baru} donasi web baru</strong> sudah diterima dan belum dilihat.</span>
			<form method="POST" action="?/dibaca"><button class="tombol kecil" type="submit">Tandai semua sudah dibaca</button></form>
		</div>
	{/if}

	<section class="grid-kartu">
		<div class="kartu"><span class="label">Total diterima</span><strong class="besar angka">{rupiah(h.ringkas.total_diterima)}</strong></div>
		<div class="kartu"><span class="label">Jumlah donasi diterima</span><strong class="besar angka">{h.ringkas.jumlah_diterima}</strong></div>
		<div class="kartu">
			<span class="label">Halaman donasi publik</span>
			<a href="/donasi" class="tautan-publik">/donasi</a>
			<small class="redup">Bagikan alamat halaman ini ke jamaah.</small>
		</div>
	</section>

	<section class="kartu tumpuk">
		{#if data.program}
			<p class="kotak-info baris-baru" role="status">
				<span>Menampilkan donasi untuk program <strong>{data.program.judul}</strong>.</span>
				<a class="tombol kecil" href="/penerimaan">Hapus filter program</a>
			</p>
		{/if}
		<form method="GET" class="cari" role="search">
			{#if data.program}<input type="hidden" name="program" value={data.program.id} />{/if}
			<label class="sr-only" for="q">Cari nama donatur</label>
			<input id="q" name="q" type="search" placeholder="Cari nama donatur…" value={data.q} maxlength="100" />
			{#if data.status}<input type="hidden" name="status" value={data.status} />{/if}
			<label class="sr-only" for="jenis">Jenis</label>
			<select id="jenis" name="jenis" value={data.jenis}>
				<option value="">Semua jenis</option>
				{#each semuaJenisDonasi as j (j)}<option value={j}>{labelJenisDonasi[j]}</option>{/each}
			</select>
			<button class="tombol" type="submit">Terapkan</button>
		</form>

		<nav class="chips" aria-label="Filter status">
			<a class="chip" href={tautan({ status: '', page: 1 })} aria-current={data.status === ''}>Semua</a>
			{#each statusFilter as s (s)}
				<a class="chip" href={tautan({ status: s, page: 1 })} aria-current={data.status === s}>{labelStatusDonasi[s]}</a>
			{/each}
		</nav>

		{#if h.items.length === 0}
			<p class="redup kosong">{data.q || data.status || data.jenis ? 'Tidak ada donasi yang cocok dengan filter.' : 'Belum ada donasi.'}</p>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead>
						<tr>
							<th>Tanggal</th><th>Donatur</th><th>Jenis</th><th class="kanan">Jumlah</th><th class="lebar-saja">Sumber</th><th>Status</th><th class="kanan lebar-saja">Kuitansi</th>
						</tr>
					</thead>
					<tbody>
						{#each h.items as d (d.id)}
							<tr>
								<td class="nowrap">{tanggal(d.dibayar_pada ?? d.dibuat_pada)}</td>
								<td>
									<a class="nama" href="/penerimaan/{d.id}"><strong>{d.nama}</strong></a>
									{#if d.baru}<span class="badge kuning">Baru</span>{/if}
								</td>
								<td>
									{labelJenisDonasi[d.jenis]}
									{#if d.program_judul}<small class="program-kecil">{d.program_judul}</small>{/if}
								</td>
								<td class="kanan angka nowrap">{rupiah(d.jumlah)}</td>
								<td class="lebar-saja">{d.sumber === 'web' ? 'Web' : 'Petugas'}</td>
								<td><span class="badge {warnaStatusDonasi[d.status]}">{labelStatusDonasi[d.status]}</span></td>
								<td class="kanan lebar-saja nowrap"><a href="/penerimaan/{d.id}">{d.nomor_kuitansi || 'Detail'}</a></td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="paging">
				<span class="redup">Halaman {data.page} dari {jumlahHalaman} · {h.total} donasi</span>
				<div class="baris-aksi">
					{#if data.page > 1}<a class="tombol kecil" href={tautan({ page: data.page - 1 })}>← Sebelumnya</a>{/if}
					{#if data.page < jumlahHalaman}<a class="tombol kecil" href={tautan({ page: data.page + 1 })}>Berikutnya →</a>{/if}
				</div>
			</div>
		{/if}
	</section>
</div>

<style>
	.program-kecil {
		display: block;
		color: var(--teks-redup);
		font-size: 0.78rem;
	}
	.cari {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}
	.cari input[type='search'] {
		flex: 1;
		min-width: 12rem;
	}
	.cari select {
		width: auto; /* gaya global membuat select selebar kolom; di sini cukup selebar isinya */
		min-width: 10rem;
	}
	.baris-baru {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
	}
	.baris-baru form {
		margin: 0;
	}
	.besar {
		display: block;
		font-size: 1.4rem;
	}
	.tautan-publik {
		display: block;
		font-weight: 700;
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
	.nama {
		color: inherit;
		text-decoration: none;
	}
</style>
