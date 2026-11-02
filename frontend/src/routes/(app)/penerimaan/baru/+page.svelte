<!-- Formulir catat donasi langsung (tunai/transfer): nama, jenis, jumlah, metode, tanggal, No. HP opsional. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { hariIniISO, labelJenisDonasi, semuaJenisDonasi } from '$lib/format';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);

	const hariIni = hariIniISO();
	// Program dipilih → jenis dikunci ke infak (Go juga menolak zakat ke program).
	// svelte-ignore state_referenced_locally
	let programId = $state(form?.nilai?.program_id ?? data.programAwal);
	const nilai = $derived({
		nama: form?.nilai?.nama ?? '',
		anonim: form?.nilai?.anonim ?? false,
		jenis: form?.nilai?.jenis || 'infak',
		jumlah: form?.nilai?.jumlah ?? '',
		metode: form?.nilai?.metode || 'tunai',
		keterangan: form?.nilai?.keterangan ?? '',
		tanggal: form?.nilai?.tanggal || hariIni
	});
	const galat = (f: string) => form?.fields?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Catat donasi · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/penerimaan">← Daftar donasi</a>
	<h1>Catat donasi</h1>
	<p class="redup">Untuk penerimaan tunai atau transfer langsung di masjid. Kuitansi terbit setelah disimpan.</p>

	<form
		method="POST"
		class="kartu tumpuk"
		novalidate
		use:enhance={() => {
			mengirim = true;
			return async ({ update }) => {
				await update({ reset: false });
				mengirim = false;
			};
		}}
	>
		{#if galatUmum}<p class="kotak-galat" role="alert">{galatUmum}</p>{/if}

		<div class="field">
			<label for="nama">Nama donatur *</label>
			<input id="nama" name="nama" type="text" required maxlength="100" value={nilai.nama} aria-invalid={Boolean(galat('nama'))} />
			{#if galat('nama')}<span class="galat" role="alert">{galat('nama')}</span>{/if}
		</div>

		{#if data.program.length > 0}
			<div class="field">
				<label for="program_id">Untuk program (opsional)</label>
				<select id="program_id" name="program_id" bind:value={programId} aria-invalid={Boolean(galat('program'))}>
					<option value="">— Donasi umum —</option>
					{#each data.program as p (p.id)}<option value={p.id}>{p.judul}{p.status === 'selesai' ? ' (selesai)' : ''}</option>{/each}
				</select>
				<small>Pilih bila infak ini untuk program donasi tertentu; jumlahnya ikut menambah kemajuan program.</small>
				{#if galat('program')}<span class="galat" role="alert">{galat('program')}</span>{/if}
				{#if programId}
					<label class="centang">
						<input type="checkbox" name="anonim" checked={nilai.anonim} />
						<span>Donatur minta namanya tidak ditampilkan di daftar donatur publik</span>
					</label>
				{/if}
			</div>
		{/if}

		<div class="dua">
			<div class="field">
				<label for="jenis">Jenis *</label>
				<select id="jenis" name="jenis" value={programId ? 'infak' : nilai.jenis} disabled={Boolean(programId)} aria-invalid={Boolean(galat('jenis'))}>
					{#each semuaJenisDonasi as j (j)}<option value={j}>{labelJenisDonasi[j]}</option>{/each}
				</select>
				{#if programId}<input type="hidden" name="jenis" value="infak" />{/if}
				{#if galat('jenis')}<span class="galat" role="alert">{galat('jenis')}</span>{/if}
			</div>
			<div class="field">
				<label for="jumlah">Jumlah (Rp) *</label>
				<input id="jumlah" name="jumlah" type="text" inputmode="numeric" autocomplete="off" required placeholder="mis. 100.000" value={nilai.jumlah} aria-invalid={Boolean(galat('jumlah'))} />
				{#if galat('jumlah')}<span class="galat" role="alert">{galat('jumlah')}</span>{/if}
			</div>
		</div>

		<div class="dua">
			<div class="field">
				<label for="metode">Diterima lewat *</label>
				<select id="metode" name="metode" value={nilai.metode} aria-invalid={Boolean(galat('metode'))}>
					<option value="tunai">Tunai</option>
					<option value="transfer">Transfer bank</option>
				</select>
				{#if galat('metode')}<span class="galat" role="alert">{galat('metode')}</span>{/if}
			</div>
			<div class="field">
				<label for="tanggal">Tanggal diterima</label>
				<input id="tanggal" name="tanggal" type="date" max={hariIni} value={nilai.tanggal} aria-invalid={Boolean(galat('tanggal'))} />
				{#if galat('tanggal')}<span class="galat" role="alert">{galat('tanggal')}</span>{/if}
			</div>
		</div>

		<div class="field">
			<label for="no_hp">No. HP / WhatsApp (opsional)</label>
			<input id="no_hp" name="no_hp" type="text" inputmode="tel" autocomplete="off" maxlength="20" placeholder="mis. 081234567890" aria-invalid={Boolean(galat('no_hp'))} />
			<small>Untuk mengirim kuitansi lewat WhatsApp. Disimpan terenkripsi.</small>
			{#if galat('no_hp')}<span class="galat" role="alert">{galat('no_hp')}</span>{/if}
		</div>

		<div class="field">
			<label for="keterangan">Keterangan (opsional)</label>
			<input id="keterangan" name="keterangan" type="text" maxlength="200" value={nilai.keterangan} aria-invalid={Boolean(galat('keterangan'))} />
			{#if galat('keterangan')}<span class="galat" role="alert">{galat('keterangan')}</span>{/if}
		</div>

		<div class="baris-aksi ujung">
			<a class="tombol" href="/penerimaan">Batal</a>
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : 'Simpan & terbitkan kuitansi'}</button>
		</div>
	</form>
</div>

<style>
	.sempit {
		max-width: 760px;
	}
	.dua {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	.centang {
		display: flex;
		align-items: flex-start;
		gap: 0.5rem;
		margin-top: 0.5rem;
		font-weight: 400 !important;
		cursor: pointer;
	}
	@media (max-width: 560px) {
		.dua {
			grid-template-columns: 1fr;
		}
	}
</style>
