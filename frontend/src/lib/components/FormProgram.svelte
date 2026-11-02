<!--
  Kolom isian program donasi, dipakai halaman "Program baru" dan "Ubah program".
  Elemen <form> dan tombol simpan milik halaman pemanggil (action berbeda).
-->
<script lang="ts">
	import { hariIniISO, labelStatusProgram } from '$lib/format';
	import type { NilaiProgram } from '$lib/program';
	import type { StatusProgram } from '$lib/types';

	let { nilai, galat, baru = false }: { nilai: NilaiProgram; galat: (f: string) => string; baru?: boolean } = $props();
	const semuaStatus: StatusProgram[] = ['draf', 'aktif', 'selesai'];
	const keterangan: Record<StatusProgram, string> = {
		draf: 'Belum tampil di halaman publik',
		aktif: 'Tampil dan menerima donasi',
		selesai: 'Tetap tampil, tidak menerima donasi'
	};
</script>

<div class="field">
	<label for="judul">Judul program *</label>
	<input id="judul" name="judul" type="text" required maxlength="120" placeholder="mis. Renovasi Atap Masjid" value={nilai.judul} aria-invalid={Boolean(galat('judul'))} />
	{#if baru}<small>Alamat halaman publik dibuat dari judul ini dan tidak berubah lagi.</small>{/if}
	{#if galat('judul')}<span class="galat" role="alert">{galat('judul')}</span>{/if}
</div>

<div class="field">
	<label for="ringkasan">Ringkasan *</label>
	<input id="ringkasan" name="ringkasan" type="text" required maxlength="200" placeholder="Satu kalimat yang tampil di kartu program" value={nilai.ringkasan} aria-invalid={Boolean(galat('ringkasan'))} />
	{#if galat('ringkasan')}<span class="galat" role="alert">{galat('ringkasan')}</span>{/if}
</div>

<div class="field">
	<label for="cerita">Cerita program *</label>
	<textarea id="cerita" name="cerita" rows="9" required maxlength="10000" placeholder="Latar belakang, rencana penggunaan dana, dan siapa yang dibantu. Pisahkan paragraf dengan baris kosong." aria-invalid={Boolean(galat('cerita'))}>{nilai.cerita}</textarea>
	{#if galat('cerita')}<span class="galat" role="alert">{galat('cerita')}</span>{/if}
</div>

<div class="dua">
	<div class="field">
		<label for="target">Target dana (Rp) *</label>
		<input id="target" name="target" type="text" inputmode="numeric" autocomplete="off" required placeholder="mis. 25.000.000" value={nilai.target} aria-invalid={Boolean(galat('target'))} />
		{#if galat('target')}<span class="galat" role="alert">{galat('target')}</span>{/if}
	</div>
	<div class="field">
		<label for="batas_waktu">Batas waktu (opsional)</label>
		<input id="batas_waktu" name="batas_waktu" type="date" min={baru ? hariIniISO() : undefined} value={nilai.batas_waktu} aria-invalid={Boolean(galat('batas_waktu'))} />
		<small>Kosongkan bila program berjalan tanpa batas waktu.</small>
		{#if galat('batas_waktu')}<span class="galat" role="alert">{galat('batas_waktu')}</span>{/if}
	</div>
</div>

<fieldset class="field status">
	<legend>Status *</legend>
	{#each semuaStatus as s (s)}
		<label class="pilih">
			<input type="radio" name="status" value={s} checked={(nilai.status || 'draf') === s} />
			<span><strong>{labelStatusProgram[s]}</strong> <small class="redup">{keterangan[s]}</small></span>
		</label>
	{/each}
	{#if galat('status')}<span class="galat" role="alert">{galat('status')}</span>{/if}
</fieldset>

<style>
	textarea {
		resize: vertical;
	}
	.dua {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	.status {
		border: 0;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.4rem;
	}
	.status legend {
		font-weight: 600;
		margin-bottom: 0.4rem;
		padding: 0;
	}
	.pilih {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		font-weight: 400 !important;
		cursor: pointer;
	}
	.pilih input {
		accent-color: var(--hijau);
	}
	@media (max-width: 560px) {
		.dua {
			grid-template-columns: 1fr;
		}
	}
</style>
