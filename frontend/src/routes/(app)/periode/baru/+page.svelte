<!-- Buat periode (FR-06) + kartu kuota & sisa dana yang dihitung langsung saat mengetik. -->
<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { angka, labelAsnaf, parseAngka, rupiah, semuaAsnaf } from '$lib/format';
	import { hitungPembagian, type CaraBagi } from '$lib/periode';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
	let mengirim = $state(false);

	// Nilai awal dari isian yang ditolak server (bila ada); selanjutnya dikendalikan pengguna.
	const awal = untrack(() => form?.nilai);
	// Nama diikat ke state (bukan value={form?...}): atribut value yang digabung ke efek templat yang sama dengan kartu
	// kuota ikut dijalankan ulang saat dana/nominal berubah dan mengosongkan nama yang sudah diketik.
	let nama = $state(awal?.nama ?? '');
	let jenis = $state(awal?.jenis ?? 'zakat');
	let dana = $state(awal?.dana ?? '');
	// Dua cara membagi dana, pilih salah satu: nominal per penerima, atau jumlah penerima (nominal dihitung).
	let cara = $state<CaraBagi>(awal?.cara === 'jumlah' ? 'jumlah' : 'nominal');
	let nominal = $state(awal?.nominal ?? '');
	let jumlah = $state(awal?.jumlah ?? '');
	const terpilih: string[] = awal?.asnaf ?? ['fakir', 'miskin'];

	// NaN (ketikan belum angka) dianggap 0; hasilnya sama dengan yang dihitung server.
	const aman = (n: number) => (Number.isNaN(n) ? 0 : n);
	const nDana = $derived(aman(parseAngka(dana)));
	const pembagian = $derived(hitungPembagian(cara, nDana, aman(parseAngka(cara === 'nominal' ? nominal : jumlah))));
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
				<input id="nama" name="nama" type="text" placeholder="mis. Zakat Fitrah 1448 H" maxlength="100" bind:value={nama} aria-invalid={Boolean(galat('nama'))} />
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

			<div class="field">
				<label for="dana">Dana tersedia (Rp) *</label>
				<input id="dana" name="dana_tersedia" type="text" inputmode="numeric" class="angka" placeholder="3.000.000" bind:value={dana} onblur={() => (dana = rapikan(dana))} aria-invalid={Boolean(galat('dana_tersedia'))} />
				{#if galat('dana_tersedia')}<span class="galat" role="alert">{galat('dana_tersedia')}</span>{/if}
			</div>

			<fieldset class="field">
				<legend class="label">Cara membagi dana *</legend>
				<div class="cara">
					<label class="cara-pilihan" class:aktif={cara === 'nominal'}>
						<input type="radio" name="cara" value="nominal" bind:group={cara} />
						<span class="cara-judul">Nominal per penerima</span>
						<span class="cara-ket">Anda tentukan jumlah uang tiap orang; jumlah penerima dihitung dari dana.</span>
					</label>
					<label class="cara-pilihan" class:aktif={cara === 'jumlah'}>
						<input type="radio" name="cara" value="jumlah" bind:group={cara} />
						<span class="cara-judul">Dibagi jumlah penerima</span>
						<span class="cara-ket">Anda tentukan jumlah orang; nominal tiap orang dihitung dari dana.</span>
					</label>
				</div>
			</fieldset>

			{#if cara === 'nominal'}
				<div class="field">
					<label for="nominal">Nominal per penerima (Rp) *</label>
					<input id="nominal" name="nominal_per_penerima" type="text" inputmode="numeric" class="angka" placeholder="1.000.000" bind:value={nominal} onblur={() => (nominal = rapikan(nominal))} aria-invalid={Boolean(galat('nominal_per_penerima'))} />
					{#if galat('nominal_per_penerima')}<span class="galat" role="alert">{galat('nominal_per_penerima')}</span>{/if}
				</div>
			{:else}
				<div class="field">
					<label for="jumlah">Jumlah penerima (orang) *</label>
					<input id="jumlah" name="jumlah_penerima" type="text" inputmode="numeric" class="angka" placeholder="3" bind:value={jumlah} aria-invalid={Boolean(galat('jumlah_penerima') || pembagian.peringatan)} />
					{#if galat('jumlah_penerima')}<span class="galat" role="alert">{galat('jumlah_penerima')}</span>{:else if pembagian.peringatan}<span class="galat" role="alert">{pembagian.peringatan}</span>{/if}
				</div>
			{/if}

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
			<p class="besar">{pembagian.kuota} <span class="redup">penerima</span></p>
			{#if cara === 'nominal'}
				<p class="redup">⌊dana ÷ nominal⌋ = ⌊{angka(nDana)} ÷ {angka(pembagian.nominal)}⌋</p>
			{:else}
				<p class="redup">Nominal tiap penerima = ⌊dana ÷ jumlah⌋</p>
				<p class="nominal-hasil angka">{pembagian.kuota > 0 ? rupiah(pembagian.nominal) : '–'}</p>
			{/if}
			<dl>
				<dt>Terpakai</dt>
				<dd class="angka">{rupiah(pembagian.terpakai)}</dd>
				<dt>Sisa dana</dt>
				<dd class="angka">{rupiah(pembagian.sisa)}</dd>
			</dl>
			<p class="redup kecil">Kandidat peringkat 1–{pembagian.kuota || 'N'} menjadi usulan penerima; sisanya cadangan.</p>
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
	/* Pilihan cara membagi dana: dua kartu yang bisa diklik; kartu terpilih ditandai garis hijau. */
	.cara {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
		padding-top: 0.25rem;
	}
	.cara-pilihan {
		display: grid;
		grid-template-columns: auto 1fr;
		column-gap: 0.6rem;
		row-gap: 0.2rem;
		align-items: start;
		padding: 0.85rem 1rem;
		border: 1px solid var(--garis);
		border-radius: var(--radius);
		cursor: pointer;
		transition:
			border-color 0.2s,
			background-color 0.2s;
	}
	.cara-pilihan input {
		accent-color: var(--hijau);
		margin-top: 0.25rem;
		grid-row: span 2;
	}
	.cara-pilihan.aktif {
		border-color: var(--hijau);
		background: var(--hijau-muda);
	}
	.cara-judul {
		font-weight: 600;
	}
	.cara-ket {
		grid-column: 2;
		font-size: 0.88rem;
		color: var(--teks-redup);
		line-height: 1.4;
	}
	.nominal-hasil {
		font-size: 1.4rem;
		font-weight: 700;
		color: var(--hijau);
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
		.cara {
			grid-template-columns: 1fr;
		}
		.ringkasan {
			position: static;
		}
	}
</style>
