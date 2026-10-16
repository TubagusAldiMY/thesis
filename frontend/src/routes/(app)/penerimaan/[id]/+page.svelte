<!--
  Detail donasi: data donatur, status, tautan kuitansi (untuk dicetak atau dikirim
  lewat WhatsApp), dan pembatalan kuitansi (Admin). Pembatalan tidak menghapus:
  nomor kuitansi tetap tercatat dan kuitansi tampil "DIBATALKAN".
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import { labelJenisDonasi, labelStatusDonasi, namaMetode, rupiah, tanggal, tautanWA, warnaStatusDonasi } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let membatalkan = $state(false);
	let tersalin = $state(false);

	const d = $derived(data.donasi);
	const admin = $derived(data.user.role === 'admin');
	const adaKuitansi = $derived(d.status === 'diterima' || d.status === 'dibatalkan');
	const urlKuitansi = $derived(`${page.url.origin}/kuitansi/${d.kode_publik}`);
	const pesanWA = $derived(
		`Assalamu'alaikum ${d.nama}. Jazakumullahu khairan, ${labelJenisDonasi[d.jenis]} Anda sebesar ${rupiah(d.jumlah)} telah kami terima di ${masjid.nama}. Kuitansi: ${urlKuitansi}`
	);
	// Hanya Admin menerima No. HP lengkap; Ketua tidak punya nomor untuk dihubungi.
	const tautanKirim = $derived(admin && d.no_hp && d.status === 'diterima' ? tautanWA(d.no_hp, pesanWA) : null);
	const galatBatal = $derived(form?.fields?.alasan || (form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : ''));

	async function salin() {
		await navigator.clipboard?.writeText(urlKuitansi);
		tersalin = true;
		setTimeout(() => (tersalin = false), 2000);
	}
</script>

<svelte:head><title>Donasi {d.nomor_kuitansi || d.nama} · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/penerimaan">← Daftar donasi</a>

	<div class="kartu-kepala">
		<div>
			<h1>{d.nama}</h1>
			<p class="redup">{labelJenisDonasi[d.jenis]} · {d.sumber === 'web' ? 'dari halaman publik' : 'dicatat petugas'}</p>
		</div>
		<span class="badge {warnaStatusDonasi[d.status]}">{labelStatusDonasi[d.status]}</span>
	</div>

	{#if data.tercatat}<p class="kotak-sukses" role="status">Donasi tercatat dan kuitansi sudah terbit.</p>{/if}
	{#if data.dibatalkan}<p class="kotak-sukses" role="status">Kuitansi dibatalkan.</p>{/if}

	<section class="kartu tumpuk">
		<dl class="rincian">
			<dt>Jumlah</dt><dd class="angka besar">{rupiah(d.jumlah)}</dd>
			<dt>No. kuitansi</dt><dd>{d.nomor_kuitansi || '-'}</dd>
			<dt>Diterima</dt><dd>{d.dibayar_pada ? tanggal(d.dibayar_pada) : '-'}</dd>
			<dt>Dibuat</dt><dd>{tanggal(d.dibuat_pada)}</dd>
			<dt>Metode</dt><dd>{namaMetode(d.metode)}</dd>
			<dt>No. HP</dt><dd>{d.no_hp || d.no_hp_tersamar || '-'}</dd>
			<dt>Keterangan</dt><dd>{d.keterangan || '-'}</dd>
			{#if d.status === 'dibatalkan'}<dt>Alasan batal</dt><dd>{d.alasan_batal}</dd>{/if}
		</dl>
	</section>

	{#if adaKuitansi}
		<section class="kartu tumpuk">
			<h2>Kuitansi</h2>
			<div class="baris-aksi">
				<a class="tombol utama" href="/kuitansi/{d.kode_publik}">Lihat & cetak kuitansi</a>
				<button class="tombol" type="button" onclick={salin}>{tersalin ? 'Tautan tersalin ✓' : 'Salin tautan'}</button>
				{#if tautanKirim}
					<a class="tombol" href={tautanKirim} target="_blank" rel="noopener noreferrer">Kirim via WhatsApp</a>
				{/if}
			</div>
			{#if admin && d.status === 'diterima' && !tautanKirim}
				<p class="redup">
					{d.no_hp ? 'No. HP tidak bisa dipakai untuk WhatsApp.' : 'Tidak ada No. HP tersimpan, jadi tombol WhatsApp tidak tersedia.'}
					Anda tetap bisa menyalin tautan kuitansi.
				</p>
			{/if}
		</section>
	{/if}

	{#if admin && d.status === 'diterima'}
		<section class="kartu tumpuk">
			<details>
				<summary>Batalkan kuitansi…</summary>
				<form
					method="POST"
					action="?/batal"
					class="tumpuk"
					use:enhance={() => {
						membatalkan = true;
						return async ({ update }) => {
							await update({ reset: false });
							membatalkan = false;
						};
					}}
				>
					<p class="redup">Gunakan bila salah catat. Kuitansi tidak dihapus: nomornya tetap tercatat dan tampil "DIBATALKAN".</p>
					<div class="field">
						<label for="alasan">Alasan pembatalan *</label>
						<input id="alasan" name="alasan" type="text" required maxlength="200" value={form?.alasan ?? ''} aria-invalid={Boolean(galatBatal)} />
						{#if galatBatal}<span class="galat" role="alert">{galatBatal}</span>{/if}
					</div>
					<div class="baris-aksi ujung">
						<button class="tombol bahaya" type="submit" disabled={membatalkan}>{membatalkan ? 'Membatalkan…' : 'Batalkan kuitansi'}</button>
					</div>
				</form>
			</details>
		</section>
	{/if}
</div>

<style>
	.sempit {
		max-width: 760px;
	}
	.rincian {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 0.5rem 1.5rem;
		margin: 0;
	}
	.rincian dt {
		color: var(--teks-redup);
	}
	.rincian dd {
		margin: 0;
		word-break: break-word;
	}
	.besar {
		font-size: 1.3rem;
		font-weight: 700;
	}
	summary {
		cursor: pointer;
		font-weight: 600;
	}
	details[open] summary {
		margin-bottom: 1rem;
	}
</style>
