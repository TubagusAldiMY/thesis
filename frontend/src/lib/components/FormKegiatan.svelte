<!--
  Kolom isian kegiatan, dipakai halaman "Kegiatan baru" dan "Ubah kegiatan".
  Elemen <form> dan tombol simpan milik halaman pemanggil (action berbeda).
-->
<script lang="ts">
	import { labelStatusKegiatan } from '$lib/kegiatan';
	import type { IsianKegiatan } from '$lib/kegiatan';
	import type { KategoriKegiatan, StatusKegiatan } from '$lib/types';

	let { nilai, galat, kategori, baru = false }: { nilai: IsianKegiatan; galat: (f: string) => string; kategori: KategoriKegiatan[]; baru?: boolean } = $props();
	const semuaStatus: StatusKegiatan[] = ['draf', 'terbit'];
	const keterangan: Record<StatusKegiatan, string> = {
		draf: 'Hanya pengurus yang melihat',
		terbit: 'Tampil di halaman publik'
	};
</script>

<div class="field">
	<label for="judul">Judul *</label>
	<input id="judul" name="judul" type="text" required maxlength="120" placeholder="mis. Peringatan Maulid Nabi Muhammad SAW 1448 H" value={nilai.judul} aria-invalid={Boolean(galat('judul'))} />
	{#if baru}<small>Alamat halaman publik dibuat dari judul ini dan tidak berubah lagi.</small>{/if}
	{#if galat('judul')}<span class="galat" role="alert">{galat('judul')}</span>{/if}
</div>

<div class="dua">
	<div class="field">
		<label for="kategori">Kategori *</label>
		<select id="kategori" name="kategori" value={nilai.kategori || 'kajian'} aria-invalid={Boolean(galat('kategori'))}>
			{#each kategori as k (k.kode)}<option value={k.kode}>{k.label}</option>{/each}
		</select>
		{#if galat('kategori')}<span class="galat" role="alert">{galat('kategori')}</span>{/if}
	</div>
	<div class="field">
		<label for="lokasi">Lokasi (opsional)</label>
		<input id="lokasi" name="lokasi" type="text" maxlength="120" placeholder="mis. Aula Masjid Al-Barokah" value={nilai.lokasi} aria-invalid={Boolean(galat('lokasi'))} />
		{#if galat('lokasi')}<span class="galat" role="alert">{galat('lokasi')}</span>{/if}
	</div>
</div>

<div class="dua">
	<div class="field">
		<label for="tanggal_acara">Tanggal acara (opsional)</label>
		<input id="tanggal_acara" name="tanggal_acara" type="date" value={nilai.tanggal_acara} aria-invalid={Boolean(galat('tanggal_acara'))} />
		<small>Kosongkan untuk artikel biasa tanpa tanggal acara.</small>
		{#if galat('tanggal_acara')}<span class="galat" role="alert">{galat('tanggal_acara')}</span>{/if}
	</div>
	<div class="field">
		<label for="jam_acara">Jam mulai (opsional, WIB)</label>
		<input id="jam_acara" name="jam_acara" type="time" value={nilai.jam_acara} aria-invalid={Boolean(galat('jam_acara'))} />
		{#if galat('jam_acara')}<span class="galat" role="alert">{galat('jam_acara')}</span>{/if}
	</div>
</div>

<div class="field">
	<label for="ringkasan">Ringkasan *</label>
	<input id="ringkasan" name="ringkasan" type="text" required maxlength="200" placeholder="Satu-dua kalimat yang tampil di kartu kegiatan" value={nilai.ringkasan} aria-invalid={Boolean(galat('ringkasan'))} />
	{#if galat('ringkasan')}<span class="galat" role="alert">{galat('ringkasan')}</span>{/if}
</div>

<div class="field">
	<label for="isi">Isi *</label>
	<textarea id="isi" name="isi" rows="10" required maxlength="10000" placeholder="Uraian acara: rangkaian kegiatan, pembicara, dan hal yang perlu dibawa. Pisahkan paragraf dengan baris kosong." aria-invalid={Boolean(galat('isi'))}>{nilai.isi}</textarea>
	{#if galat('isi')}<span class="galat" role="alert">{galat('isi')}</span>{/if}
</div>

<fieldset class="field status">
	<legend>Status *</legend>
	{#each semuaStatus as s (s)}
		<label class="pilih">
			<input type="radio" name="status" value={s} checked={(nilai.status || 'draf') === s} />
			<span><strong>{labelStatusKegiatan[s]}</strong> <small class="redup">{keterangan[s]}</small></span>
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
