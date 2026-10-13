<!-- Formulir mustahik: nama, NIK, golongan, No. HP (opsional), alamat, kotak centang "Mustahik aktif". -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { labelAsnaf, semuaAsnaf } from '$lib/format';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);

	const m = $derived(data.mustahik);
	const nilai = $derived({
		nama: form?.nilai?.nama ?? m?.nama ?? '',
		alamat: form?.nilai?.alamat ?? m?.alamat ?? '',
		asnaf: form?.nilai?.asnaf ?? m?.asnaf ?? 'fakir',
		aktif: form?.nilai?.aktif ?? m?.aktif ?? true
	});
	const galat = (f: string) => form?.fields?.[f] ?? '';
	// Pesan umum (mis. 409 "sedang menjadi kandidat") = error tanpa pesan per kolom.
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>{data.baru ? 'Tambah' : 'Ubah'} mustahik · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/mustahik">← Daftar mustahik</a>
	<h1>{data.baru ? 'Tambah mustahik' : 'Ubah mustahik'}</h1>

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
			<label for="nama">Nama lengkap *</label>
			<input id="nama" name="nama" type="text" required maxlength="100" value={nilai.nama} aria-invalid={Boolean(galat('nama'))} aria-describedby={galat('nama') ? 'g-nama' : undefined} />
			{#if galat('nama')}<span id="g-nama" class="galat" role="alert">{galat('nama')}</span>{/if}
		</div>

		<div class="dua">
			<div class="field">
				<label for="nik">NIK {data.baru ? '*' : ''}</label>
				<input
					id="nik"
					name="nik"
					type="text"
					inputmode="numeric"
					autocomplete="off"
					maxlength="20"
					placeholder={data.baru ? '16 digit angka' : 'Kosongkan bila tidak diubah'}
					aria-invalid={Boolean(galat('nik'))}
					aria-describedby="h-nik{galat('nik') ? ' g-nik' : ''}"
				/>
				<small id="h-nik">{m ? `Tersimpan: ${m.nik_tersamar}` : 'Disimpan terenkripsi; hanya 4 digit terakhir yang ditampilkan.'}</small>
				{#if galat('nik')}<span id="g-nik" class="galat" role="alert">{galat('nik')}</span>{/if}
			</div>
			<div class="field">
				<label for="asnaf">Golongan (asnaf) *</label>
				<select id="asnaf" name="asnaf" value={nilai.asnaf} aria-invalid={Boolean(galat('asnaf'))}>
					{#each semuaAsnaf as a (a)}<option value={a}>{labelAsnaf[a]}</option>{/each}
				</select>
				{#if galat('asnaf')}<span class="galat" role="alert">{galat('asnaf')}</span>{/if}
			</div>
		</div>

		<div class="field">
			<label for="no_hp">No. HP (opsional)</label>
			<input
				id="no_hp"
				name="no_hp"
				type="text"
				inputmode="tel"
				autocomplete="off"
				maxlength="20"
				placeholder={m?.no_hp_tersamar ? 'Kosongkan bila tidak diubah' : 'mis. 081234567890'}
				aria-invalid={Boolean(galat('no_hp'))}
			/>
			{#if m?.no_hp_tersamar}<small>Tersimpan: {m.no_hp_tersamar}</small>{/if}
			{#if galat('no_hp')}<span class="galat" role="alert">{galat('no_hp')}</span>{/if}
		</div>

		<div class="field">
			<label for="alamat">Alamat *</label>
			<textarea id="alamat" name="alamat" required maxlength="500" aria-invalid={Boolean(galat('alamat'))}>{nilai.alamat}</textarea>
			{#if galat('alamat')}<span class="galat" role="alert">{galat('alamat')}</span>{/if}
		</div>

		<label class="centang">
			<input type="checkbox" name="aktif" checked={nilai.aktif} />
			<span>
				<strong>Mustahik aktif</strong>
				<small class="redup">Mustahik nonaktif tidak muncul sebagai kandidat baru; riwayat lamanya tetap tersimpan.</small>
			</span>
		</label>

		<div class="baris-aksi ujung">
			<a class="tombol" href="/mustahik">Batal</a>
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : 'Simpan'}</button>
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
		gap: 0.75rem;
		align-items: flex-start;
		cursor: pointer;
	}
	.centang input {
		width: 22px;
		height: 22px;
		margin-top: 2px;
		accent-color: var(--hijau);
	}
	.centang small {
		display: block;
	}
	@media (max-width: 640px) {
		.dua {
			grid-template-columns: 1fr;
		}
	}
</style>
