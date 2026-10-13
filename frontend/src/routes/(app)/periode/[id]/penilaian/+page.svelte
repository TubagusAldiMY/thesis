<!--
  Tab Penilaian. Lebar ≥ 1100px: tabel kandidat × kriteria. HP/tablet: kartu
  per kandidat (CSS yang sama, tanpa duplikasi input). Kriteria berskala
  (C3–C5) memakai tombol segmen 1–4. Sel kosong bergaris merah; "Hitung ARAS"
  aktif setelah semua sel terisi (docs/04 bagian 4).
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { angka, desimal } from '$lib/format';
	import type { PenilaianMatriks } from '$lib/types';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const m = $derived(data.matriks);
	const status = $derived(data.detail.periode.status);
	const bisaUbah = $derived(status === 'draft');
	const bisaHitung = $derived(status === 'draft' || status === 'dihitung');
	let mengirim = $state(false);

	/** tampilkan nilai dari server dalam format Indonesia (800000 → "800.000", 2.5 → "2,5"). */
	const keTeks = (v: number, berskala: boolean) =>
		berskala ? String(v) : Number.isInteger(v) ? angka(v) : String(v).replace('.', ',');

	const dariServer = (mx: PenilaianMatriks) =>
		Object.fromEntries(
			mx.baris.map((b) => [
				b.mustahik.id,
				Object.fromEntries(
					mx.kriteria.map((k) => [k.kode, b.nilai[k.kode] !== undefined ? keTeks(b.nilai[k.kode], k.skala !== null) : ''])
				)
			])
		) as Record<string, Record<string, string>>;

	let isian = $state(untrack(() => dariServer(data.matriks)));
	$effect.pre(() => {
		isian = dariServer(data.matriks);
	});

	const barisLengkap = (mid: string) => m.kriteria.every((k) => (isian[mid]?.[k.kode] ?? '').trim() !== '');
	const lengkap = $derived(m.baris.filter((b) => barisLengkap(b.mustahik.id)).length);
	const semuaLengkap = $derived(m.total > 0 && lengkap === m.total);
	const galatSel = (mid: string, kode: string) => form?.fields?.[`nilai.${mid}.${kode}`] ?? '';
	const galatUmum = $derived(
		form?.fields ? (form.fields.penilaian ?? form.fields.bobot ?? form.fields.kandidat ?? '') : (form?.error ?? '')
	);
	const skalaKriteria = $derived(m.kriteria.filter((k) => k.skala));
</script>

<svelte:head><title>Penilaian · {data.detail.periode.nama}</title></svelte:head>

<form
	method="POST"
	action="?/simpan"
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
	<div class="kartu-kepala">
		<div>
			<h2>Nilai tiap kandidat</h2>
			<p class="redup">Penghasilan dalam rupiah (isi 1 bila tidak berpenghasilan). C3–C5 memakai skala 1–4.</p>
		</div>
		<span class="badge {semuaLengkap ? 'hijau' : 'kuning'}">Lengkap: {lengkap}/{m.total} kandidat</span>
	</div>

	{#if form?.tersimpan}<p class="kotak-sukses" role="status">Penilaian tersimpan.</p>{/if}
	{#if galatUmum}<p class="kotak-galat" role="alert">{galatUmum}</p>{/if}
	{#if form?.fields && !galatUmum && Object.keys(form.fields).length > 0}
		<p class="kotak-galat" role="alert">Ada nilai yang tidak sah — sel bergaris merah menunjukkan letaknya.</p>
	{/if}
	{#if !bisaUbah}<p class="kotak-info">Periode berstatus {status}; penilaian terkunci (buka kembali bila ditolak).</p>{/if}

	{#if m.total === 0}
		<p class="redup kosong">Belum ada kandidat. <a href="/periode/{data.detail.periode.id}/kandidat">Pilih kandidat dulu</a>.</p>
	{:else}
		<div class="grid" style:--kolom={m.kriteria.length}>
			<div class="baris kepala-grid" aria-hidden="true">
				<span>Mustahik</span>
				{#each m.kriteria as k (k.id)}
					<span>{k.kode} {k.nama.split(' ')[0]}<small>{k.tipe} · {desimal(k.bobot, 2)}</small></span>
				{/each}
			</div>
			{#each m.baris as b, i (b.mustahik.id)}
				{@const mid = b.mustahik.id}
				<div class="baris" class:belum={!barisLengkap(mid)} role="group" aria-labelledby="nama-{mid}">
					<strong id="nama-{mid}" class="nama">A{i + 1} · {b.mustahik.nama}</strong>
					{#each m.kriteria as k (k.id)}
						{@const kosong = (isian[mid]?.[k.kode] ?? '').trim() === ''}
						{@const galat = galatSel(mid, k.kode)}
						<div class="sel">
							<span class="label-hp">{k.kode} · {k.nama}</span>
							{#if k.skala}
								<div class="segmen" role="radiogroup" aria-label="{k.kode} {b.mustahik.nama}" aria-invalid={kosong || Boolean(galat)}>
									{#each Object.entries(k.skala) as [n, ket] (n)}
										<label title={ket}>
											<input type="radio" name="nilai.{mid}.{k.kode}" value={n} bind:group={isian[mid][k.kode]} disabled={!bisaUbah} />
											<span>{n}</span>
										</label>
									{/each}
								</div>
							{:else}
								<input
									type="text"
									inputmode="decimal"
									class="angka"
									name="nilai.{mid}.{k.kode}"
									aria-label="{k.kode} {b.mustahik.nama}"
									bind:value={isian[mid][k.kode]}
									disabled={!bisaUbah}
									aria-invalid={kosong || Boolean(galat)}
								/>
							{/if}
							{#if galat}<small class="galat" role="alert">{galat}</small>{/if}
						</div>
					{/each}
				</div>
			{/each}
		</div>
	{/if}

	<div class="aksi" class:lengket={bisaUbah}>
		{#if bisaHitung}
			<p class={semuaLengkap ? 'redup' : 'galat'}>
				{semuaLengkap ? 'Semua nilai terisi. Siap dihitung.' : 'Sel bergaris merah masih kosong. Hitung ARAS aktif setelah semua nilai terisi.'}
			</p>
		{/if}
		<div class="baris-aksi">
			{#if bisaUbah}<button class="tombol" type="submit" disabled={mengirim}>Simpan</button>{/if}
			{#if bisaHitung}
				<button class="tombol utama" type="submit" formaction="?/hitung" disabled={mengirim || !semuaLengkap}>
					{mengirim ? 'Memproses…' : status === 'dihitung' ? 'Hitung ulang' : 'Hitung ARAS'}
				</button>
			{/if}
		</div>
	</div>
</form>

{#if skalaKriteria.length > 0}
	<div class="legenda">
		{#each skalaKriteria as k (k.id)}
			<div class="kartu">
				<h3>{k.kode} {k.nama}</h3>
				<ol>
					{#each Object.entries(k.skala ?? {}) as [n, ket] (n)}<li><strong>{n}</strong> {ket}</li>{/each}
				</ol>
			</div>
		{/each}
	</div>
{/if}

<style>
	.kosong {
		text-align: center;
		padding: 1.5rem 0;
	}
	.grid {
		display: grid;
	}
	.baris {
		display: grid;
		grid-template-columns: minmax(180px, 1.4fr) repeat(var(--kolom), minmax(110px, 1fr));
		gap: 0.75rem;
		align-items: center;
		padding: 0.75rem 0.5rem;
		border-bottom: 1px solid var(--garis);
	}
	.kepala-grid {
		color: var(--teks-redup);
		font-weight: 600;
		font-size: 0.92rem;
	}
	.kepala-grid small {
		display: block;
		font-weight: 400;
		font-size: 0.82rem;
	}
	.label-hp {
		display: none;
	}
	.sel {
		display: grid;
		gap: 0.25rem;
	}
	.sel input[type='text'] {
		text-align: right;
	}
	.segmen {
		display: inline-flex;
		border: 1px solid #d6d2c7;
		border-radius: 10px;
		overflow: hidden;
		width: max-content;
	}
	.segmen[aria-invalid='true'] {
		border-color: var(--merah);
	}
	.segmen label {
		position: relative;
		cursor: pointer;
	}
	.segmen input {
		position: absolute;
		opacity: 0;
		inset: 0;
	}
	.segmen span {
		display: grid;
		place-items: center;
		width: 34px;
		height: 42px;
		font-weight: 600;
		border-left: 1px solid var(--garis);
	}
	.segmen label:first-child span {
		border-left: none;
	}
	.segmen input:checked + span {
		background: var(--hijau);
		color: #fff;
	}
	.segmen input:focus-visible + span {
		outline: 3px solid #9cc5b2;
		outline-offset: -3px;
	}
	.segmen input:disabled + span {
		cursor: not-allowed;
		opacity: 0.7;
	}
	.aksi {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
	}
	.legenda {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
		gap: 1rem;
		margin-top: 1.5rem;
	}
	.legenda ol {
		list-style: none;
		padding: 0;
		margin: 0.6rem 0 0;
		display: grid;
		gap: 0.35rem;
		color: var(--teks-redup);
	}

	/* HP/tablet: tiap kandidat menjadi kartu (docs/04: grid tablet terpotong → kartu). */
	@media (max-width: 1100px) {
		.kepala-grid {
			display: none;
		}
		.baris {
			grid-template-columns: 1fr;
			border: 1px solid var(--garis);
			border-radius: 12px;
			padding: 1rem;
			margin-bottom: 0.75rem;
		}
		.baris.belum {
			border-color: #efc3bf;
		}
		.label-hp {
			display: block;
			font-size: 0.88rem;
			color: var(--teks-redup);
		}
		.sel input[type='text'] {
			text-align: left;
		}
		.segmen {
			display: flex;
			width: 100%;
		}
		.segmen label {
			flex: 1;
		}
		.segmen span {
			width: auto;
		}
		.aksi.lengket {
			position: sticky;
			bottom: 4.5rem;
			background: #fff;
			padding: 0.75rem 0;
		}
	}
</style>
