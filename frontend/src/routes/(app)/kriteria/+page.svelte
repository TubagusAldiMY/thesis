<!--
  Kriteria & bobot (FR-04/05). Bobot diketik dengan koma (0,30), kelipatan
  0,05; indikator total dihitung dalam persen bulat supaya tidak tertipu galat
  pecahan (0,15 + 0,15 ≠ 0,3 di floating point). Ketua hanya melihat.
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { desimal, parseDesimal } from '$lib/format';
	import type { Kriteria } from '$lib/types';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const admin = $derived(data.user.role === 'admin');
	let mengirim = $state(false);

	type Baris = { tipe: 'benefit' | 'cost'; bobot: string };
	const dariServer = (items: Kriteria[]) =>
		Object.fromEntries(items.map((k) => [k.id, { tipe: k.tipe, bobot: desimal(k.bobot, 2) }])) as Record<string, Baris>;
	// Nilai awal langsung (agar SSR sudah berisi), lalu diisi ulang setiap data
	// dari server berubah (mis. setelah disimpan).
	let isian = $state<Record<string, Baris>>(untrack(() => dariServer(data.kriteria.items)));
	$effect.pre(() => {
		isian = dariServer(data.kriteria.items);
	});

	const persen = (s: string) => Math.round(parseDesimal(s) * 100);
	const salah = (s: string) => {
		const p = persen(s);
		return Number.isNaN(p) || p < 5 || p > 100 || p % 5 !== 0;
	};
	const total = $derived(
		data.kriteria.items.filter((k) => k.aktif).reduce((n, k) => n + (persen(isian[k.id]?.bobot ?? '') || 0), 0)
	);
	const bukanKelipatan = $derived(Object.values(isian).some((b) => salah(b.bobot)));
	const siap = $derived(total === 100 && !bukanKelipatan);
</script>

<svelte:head><title>Kriteria & bobot · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Kriteria & bobot</h1>
		<p class="redup">
			Bobot menentukan seberapa besar pengaruh tiap kriteria pada ranking ARAS. Total bobot kriteria aktif harus tepat 1,00.
		</p>
	</div>

	{#if form?.tersimpan}<p class="kotak-sukses" role="status">Bobot kriteria tersimpan. Perhitungan berikutnya memakai bobot baru.</p>{/if}
	{#if form?.error}
		<p class="kotak-galat" role="alert">{form.fields?.bobot ?? form.fields?.tipe ?? form.error}</p>
	{/if}

	<form
		method="POST"
		class="kartu tumpuk"
		use:enhance={() => {
			mengirim = true;
			return async ({ update }) => {
				await update({ reset: false });
				mengirim = false;
			};
		}}
	>
		<div class="tabel-wadah">
			<table>
				<thead>
					<tr><th>Kode</th><th>Kriteria</th><th>Tipe</th><th>Bobot</th><th>Skala nilai</th></tr>
				</thead>
				<tbody>
					{#each data.kriteria.items as k (k.id)}
						<tr>
							<td><strong>{k.kode}</strong><input type="hidden" name="id" value={k.id} /></td>
							<td>{k.nama}</td>
							<td class="tipe">
								{#if admin && isian[k.id]}
									<label class="sr-only" for="tipe-{k.id}">Tipe {k.kode}</label>
									<select id="tipe-{k.id}" name="tipe.{k.id}" bind:value={isian[k.id].tipe}>
										<option value="benefit">benefit</option>
										<option value="cost">cost</option>
									</select>
								{:else}
									<span class="badge {k.tipe === 'cost' ? 'kuning' : 'hijau'}">{k.tipe}</span>
								{/if}
							</td>
							<td class="bobot">
								{#if admin && isian[k.id]}
									<label class="sr-only" for="bobot-{k.id}">Bobot {k.kode}</label>
									<input
										id="bobot-{k.id}"
										name="bobot.{k.id}"
										type="text"
										inputmode="decimal"
										class="angka"
										bind:value={isian[k.id].bobot}
										aria-invalid={salah(isian[k.id].bobot)}
									/>
								{:else}
									<span class="angka">{desimal(k.bobot, 2)}</span>
								{/if}
							</td>
							<td class="redup kecil">
								{#if k.skala}
									{Object.entries(k.skala)
										.map(([n, ket]) => `${n} ${ket}`)
										.join(' · ')}
								{:else}
									Nilai asli ({k.kode === 'C1' ? 'rupiah; isi 1 bila tidak berpenghasilan' : 'angka'})
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<div class="kaki">
			<p class="total {siap ? 'pas' : 'tidak'}" role="status">
				Total bobot: <strong class="angka">{desimal(total / 100, 2)}</strong>
				({total}%) —
				{#if bukanKelipatan}
					bobot harus kelipatan 0,05 (0,05–1,00)
				{:else if total === 100}
					siap disimpan
				{:else if total < 100}
					kurang {desimal((100 - total) / 100, 2)}
				{:else}
					lebih {desimal((total - 100) / 100, 2)}
				{/if}
			</p>
			{#if admin}
				<div class="baris-aksi">
					<button class="tombol" type="button" onclick={() => (isian = dariServer(data.kriteria.items))}>Kembalikan</button>
					<button class="tombol utama" type="submit" disabled={!siap || mengirim}>{mengirim ? 'Menyimpan…' : 'Simpan'}</button>
				</div>
			{:else}
				<p class="redup">Hanya Admin yang dapat mengubah bobot.</p>
			{/if}
		</div>
	</form>
</div>

<style>
	/* select global lebarnya 100% sel; tanpa batas bawah kolom ini menyusut
	   sampai "benefit"/"cost" terpotong karena kolom skala minta 18rem. */
	.tipe select {
		min-width: 8.5rem;
	}
	.bobot input {
		width: 6.5rem;
		text-align: right;
	}
	.kecil {
		font-size: 0.88rem;
		min-width: 18rem;
	}
	.kaki {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
	}
	.total {
		padding: 0.6rem 1rem;
		border-radius: 10px;
	}
	.pas {
		background: var(--hijau-muda);
		color: var(--hijau);
	}
	.tidak {
		background: var(--kuning);
		color: var(--kuning-teks);
	}
</style>
