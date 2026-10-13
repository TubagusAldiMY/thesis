<!-- Buat periode (FR-06) + kartu kuota & sisa dana yang dihitung langsung saat mengetik. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { angka, labelAsnaf, parseAngka, rupiah, semuaAsnaf } from '$lib/format';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
	let mengirim = $state(false);

	// Nilai awal dari isian yang ditolak server (bila ada); selanjutnya dikendalikan pengguna.
	const awal = untrack(() => form?.nilai);
	let jenis = $state(awal?.jenis ?? 'zakat');
	let dana = $state(awal?.dana ?? '');
	let nominal = $state(awal?.nominal ?? '');
	const terpilih: string[] = awal?.asnaf ?? ['fakir', 'miskin'];

	const nDana = $derived(parseAngka(dana));
	const nNominal = $derived(parseAngka(nominal));
	const kuota = $derived(nDana > 0 && nNominal > 0 ? Math.floor(nDana / nNominal) : 0);
	const sisa = $derived(kuota > 0 ? nDana - kuota * nNominal : 0);
	const galat = (f: string) => form?.fields?.[f] ?? '';

	/** rapikan menampilkan ribuan titik setelah kolom ditinggalkan (3000000 → 3.000.000). */
	const rapikan = (s: string) => (Number.isNaN(parseAngka(s)) ? s : angka(parseAngka(s)));
</script>

<svelte:head><title>Buat periode · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<a class="tautan-kembali" href="/periode">← Semua periode</a>
	<h1>Buat periode</h1>

	<div class="tata">
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
			<div class="field">
				<label for="nama">Nama periode *</label>
				<input id="nama" name="nama" type="text" placeholder="mis. Zakat Fitrah 1448 H" maxlength="100" value={form?.nilai?.nama ?? ''} aria-invalid={Boolean(galat('nama'))} />
				{#if galat('nama')}<span class="galat" role="alert">{galat('nama')}</span>{/if}
			</div>

			<fieldset class="field">
				<legend class="label">Jenis dana *</legend>
				<div class="pilihan">
					<label><input type="radio" name="jenis" value="zakat" bind:group={jenis} /> Zakat</label>
					<label><input type="radio" name="jenis" value="infak" bind:group={jenis} /> Infak</label>
				</div>
				{#if galat('jenis')}<span class="galat" role="alert">{galat('jenis')}</span>{/if}
			</fieldset>

			<div class="dua">
				<div class="field">
					<label for="dana">Dana tersedia (Rp) *</label>
					<input id="dana" name="dana_tersedia" type="text" inputmode="numeric" class="angka" placeholder="3.000.000" bind:value={dana} onblur={() => (dana = rapikan(dana))} aria-invalid={Boolean(galat('dana_tersedia'))} />
					{#if galat('dana_tersedia')}<span class="galat" role="alert">{galat('dana_tersedia')}</span>{/if}
				</div>
				<div class="field">
					<label for="nominal">Nominal per penerima (Rp) *</label>
					<input id="nominal" name="nominal_per_penerima" type="text" inputmode="numeric" class="angka" placeholder="1.000.000" bind:value={nominal} onblur={() => (nominal = rapikan(nominal))} aria-invalid={Boolean(galat('nominal_per_penerima'))} />
					{#if galat('nominal_per_penerima')}<span class="galat" role="alert">{galat('nominal_per_penerima')}</span>{/if}
				</div>
			</div>

			{#if jenis === 'zakat'}
				<fieldset class="field">
					<legend class="label">Golongan asnaf yang berhak *</legend>
					<div class="pilihan wrap">
						{#each semuaAsnaf as a (a)}
							<label class="chip-centang"><input type="checkbox" name="asnaf_diizinkan" value={a} checked={terpilih.includes(a)} /> {labelAsnaf[a]}</label>
						{/each}
					</div>
					<small class="redup">Hanya mustahik dari golongan ini yang bisa dipilih sebagai kandidat.</small>
					{#if galat('asnaf_diizinkan')}<span class="galat" role="alert">{galat('asnaf_diizinkan')}</span>{/if}
				</fieldset>
			{:else}
				<p class="kotak-info">Infak tidak dibatasi golongan: semua mustahik aktif bisa menjadi kandidat.</p>
			{/if}

			<div class="baris-aksi ujung">
				<a class="tombol" href="/periode">Batal</a>
				<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : 'Buat periode'}</button>
			</div>
		</form>

		<aside class="kartu ringkasan" aria-live="polite">
			<h2>Kuota penerima</h2>
			<p class="besar">{kuota} <span class="redup">penerima</span></p>
			<p class="redup">⌊dana ÷ nominal⌋ = ⌊{Number.isNaN(nDana) ? 0 : angka(nDana)} ÷ {Number.isNaN(nNominal) ? 0 : angka(nNominal)}⌋</p>
			<dl>
				<dt>Terpakai</dt>
				<dd class="angka">{rupiah(kuota * (Number.isNaN(nNominal) ? 0 : nNominal))}</dd>
				<dt>Sisa dana</dt>
				<dd class="angka">{rupiah(sisa)}</dd>
			</dl>
			<p class="redup kecil">Kandidat peringkat 1–{kuota || 'N'} menjadi usulan penerima; sisanya cadangan.</p>
		</aside>
	</div>
</div>

<style>
	.tata {
		display: grid;
		grid-template-columns: minmax(0, 2fr) minmax(260px, 1fr);
		gap: 1.5rem;
		align-items: start;
	}
	fieldset {
		border: none;
		padding: 0;
		margin: 0;
	}
	.pilihan {
		display: flex;
		gap: 1.25rem;
		padding-top: 0.25rem;
	}
	.pilihan.wrap {
		flex-wrap: wrap;
		gap: 0.6rem;
	}
	.pilihan input {
		accent-color: var(--hijau);
	}
	.chip-centang {
		border: 1px solid var(--garis);
		border-radius: 999px;
		padding: 0.35rem 0.85rem;
		cursor: pointer;
	}
	.dua {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	.ringkasan {
		display: grid;
		gap: 0.6rem;
		position: sticky;
		top: 1rem;
	}
	.besar {
		font-size: 2.4rem;
		font-weight: 700;
	}
	.besar span {
		font-size: 1rem;
		font-weight: 500;
	}
	dl {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.4rem 1rem;
		margin: 0.5rem 0;
	}
	dt {
		color: var(--teks-redup);
	}
	dd {
		margin: 0;
		text-align: right;
		font-weight: 600;
	}
	.kecil {
		font-size: 0.88rem;
	}
	@media (max-width: 900px) {
		.tata,
		.dua {
			grid-template-columns: 1fr;
		}
		.ringkasan {
			position: static;
		}
	}
</style>
