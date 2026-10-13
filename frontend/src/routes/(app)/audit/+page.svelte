<!-- Audit log (FR-15/20): chip kategori, tabel hanya-baca, nama pelaku, paginasi. -->
<script lang="ts">
	import { labelAksi, waktu } from '$lib/format';
	import type { Audit } from '$lib/types';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const h = $derived(data.hasil);
	const jumlahHalaman = $derived(Math.max(1, Math.ceil(h.total / h.per_page)));
	const kategori = [
		['', 'Semua'],
		['login', 'Login'],
		['data', 'Data & penilaian'],
		['hitung', 'Hitung'],
		['keputusan', 'Keputusan'],
		['penyaluran', 'Penyaluran']
	];
	const warna: Record<string, string> = {
		LOGIN_GAGAL: 'merah',
		TOLAK: 'merah',
		SETUJU: 'hijau',
		HITUNG: 'kuning',
		SALUR: 'hijau',
		TUTUP: 'hijau'
	};

	const tautan = (k: string, page = 1) => {
		const p = new URLSearchParams();
		if (k) p.set('kategori', k);
		if (page > 1) p.set('page', String(page));
		const s = p.toString();
		return '/audit' + (s ? '?' + s : '');
	};

	/** keterangan meringkas detail (angka/jenis/versi) tanpa ID panjang. */
	function keterangan(a: Audit): string {
		const d = a.detail ?? {};
		const bagian: string[] = [];
		if (d.versi !== undefined) bagian.push(`run v${d.versi}`);
		if (d.kandidat !== undefined) bagian.push(`${d.kandidat} kandidat`);
		if (d.jumlah !== undefined) bagian.push(`${d.jumlah} kandidat`);
		if (d.kuota !== undefined) bagian.push(`kuota ${d.kuota}`);
		if (d.nominal !== undefined) bagian.push(`Rp ${Number(d.nominal).toLocaleString('id-ID')}`);
		if (d.jenis !== undefined) bagian.push(String(d.jenis));
		if (d.asnaf !== undefined) bagian.push(String(d.asnaf));
		if (d.aktif === false) bagian.push('dinonaktifkan');
		if (a.action === 'LOGIN_GAGAL') bagian.push(`IP ${a.ip}`);
		return bagian.join(' · ') || '–';
	}
</script>

<svelte:head><title>Audit log · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Audit log</h1>
		<p class="redup">Catatan perubahan data penting: siapa, kapan, dan apa. Hanya-baca.</p>
	</div>

	<section class="kartu tumpuk">
		<nav class="chips" aria-label="Filter kategori">
			{#each kategori as [k, label] (k)}
				<a class="chip" href={tautan(k)} aria-current={data.kategori === k}>{label}</a>
			{/each}
		</nav>

		{#if h.items.length === 0}
			<p class="redup kosong">Belum ada catatan.</p>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead><tr><th>Waktu</th><th>Pengguna</th><th>Aksi</th><th>Objek</th><th>Keterangan</th></tr></thead>
					<tbody>
						{#each h.items as a (a.id)}
							<tr>
								<td class="nowrap">{waktu(a.created_at)}</td>
								<td>{a.actor_nama || '(tidak dikenal)'}</td>
								<td><span class="badge {warna[a.action] ?? ''}">{labelAksi[a.action] ?? a.action}</span></td>
								<td class="redup">{a.resource}</td>
								<td class="redup">{keterangan(a)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="paging">
				<span class="redup">Halaman {h.page} dari {jumlahHalaman} · {h.total} catatan</span>
				<div class="baris-aksi">
					{#if h.page > 1}<a class="tombol kecil" href={tautan(data.kategori, h.page - 1)}>← Sebelumnya</a>{/if}
					{#if h.page < jumlahHalaman}<a class="tombol kecil" href={tautan(data.kategori, h.page + 1)}>Berikutnya →</a>{/if}
				</div>
			</div>
		{/if}
		<p class="redup kecil">NIK, No. HP, dan password tidak pernah dicatat di audit log.</p>
	</section>
</div>

<style>
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
	.kecil {
		font-size: 0.88rem;
	}
</style>
