<!--
  Tab Penyaluran: tiga kartu (tercatat n/N, total tersalur, sisa dana); tiap
  penerima usulan punya tanggal + nominal (bawaan = nominal per penerima) +
  tombol Catat sekali; cadangan hanya keterangan. "Tutup periode" aktif setelah
  semua penerima tercatat (docs/04 bagian 4).
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { angka, hariIniISO, parseAngka, rupiah, tanggalSaja } from '$lib/format';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const r = $derived(data.rekap);
	const p = $derived(data.detail.periode);
	const bisaCatat = $derived(data.user.role === 'admin' && p.status === 'disetujui');
	const semuaTercatat = $derived(r.jumlah_penerima > 0 && r.tercatat === r.jumlah_penerima);
	let mengirim = $state('');
	let nominal = $state<Record<string, string>>({});
	const nilaiNominal = (id: string) => nominal[id] ?? angka(p.nominal_per_penerima);
</script>

<svelte:head><title>Penyaluran · {p.nama}</title></svelte:head>

<div class="tumpuk">
	<div class="grid-kartu">
		<div class="kartu ringkas">
			<p class="redup">Penerima tercatat</p>
			<p class="besar">{r.tercatat}/{r.jumlah_penerima}</p>
		</div>
		<div class="kartu ringkas">
			<p class="redup">Total tersalur</p>
			<p class="besar angka">{rupiah(r.total_tersalur)}</p>
		</div>
		<div class="kartu ringkas">
			<p class="redup">Sisa dana</p>
			<p class="besar angka">{rupiah(r.sisa_dana)}</p>
		</div>
	</div>

	{#if p.status === 'draft' || p.status === 'dihitung' || p.status === 'ditolak'}
		<p class="kotak-info">Penyaluran baru bisa dicatat setelah Ketua DKM menyetujui usulan penerima.</p>
	{/if}
	{#if form?.error}<p class="kotak-galat" role="alert">{form.fields?.nominal ?? form.fields?.tanggal ?? form.fields?.mustahik_id ?? form.fields?.penyaluran ?? form.error}</p>{/if}

	<section class="kartu tumpuk">
		<h2>Realisasi per penerima</h2>
		{#if r.baris.length === 0}
			<p class="redup">Belum ada hasil perhitungan.</p>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead>
						<tr><th>#</th><th>Penerima</th><th>Tanggal</th><th class="kanan">Nominal (Rp)</th><th class="kanan">Aksi</th></tr>
					</thead>
					<tbody>
						{#each r.baris as b (b.mustahik.id)}
							{@const id = b.mustahik.id}
							<tr class:cadangan={b.status === 'cadangan'}>
								<td>{b.ranking}</td>
								<td>
									<strong>{b.mustahik.nama}</strong>
									{#if b.status === 'cadangan'}<br /><small class="redup">Cadangan — tidak disalurkan</small>{/if}
								</td>
								{#if b.penyaluran}
									<td>{tanggalSaja(b.penyaluran.tanggal)}</td>
									<td class="kanan angka">{angka(b.penyaluran.nominal)}</td>
									<td class="kanan"><span class="badge hijau">Tercatat</span></td>
								{:else if b.status === 'usulan' && bisaCatat}
									{@const lebih = parseAngka(nilaiNominal(id)) > r.sisa_dana}
									<td>
										<input form="catat-{id}" type="date" name="tanggal" value={hariIniISO()} aria-label="Tanggal penyaluran {b.mustahik.nama}" required />
									</td>
									<td class="kanan">
										<input
											form="catat-{id}"
											type="text"
											inputmode="numeric"
											name="nominal"
											class="angka nominal"
											value={nilaiNominal(id)}
											oninput={(e) => (nominal[id] = e.currentTarget.value)}
											aria-label="Nominal untuk {b.mustahik.nama}"
											aria-invalid={lebih}
										/>
										{#if lebih}<small class="galat">Melebihi sisa dana</small>{/if}
									</td>
									<td class="kanan">
										<form
											id="catat-{id}"
											method="POST"
											action="?/catat"
											use:enhance={() => {
												mengirim = id;
												return async ({ update }) => {
													await update();
													mengirim = '';
												};
											}}
										>
											<input type="hidden" name="mustahik_id" value={id} />
											<button class="tombol utama kecil" type="submit" disabled={lebih || mengirim !== ''}>{mengirim === id ? 'Mencatat…' : 'Catat'}</button>
										</form>
									</td>
								{:else}
									<td class="redup">–</td>
									<td class="kanan redup">–</td>
									<td class="kanan">{#if b.status === 'usulan'}<span class="badge">Belum</span>{/if}</td>
								{/if}
							</tr>
						{/each}
					</tbody>
					<tfoot>
						<tr><td colspan="3"><strong>Total</strong></td><td class="kanan angka"><strong>{angka(r.total_tersalur)}</strong></td><td></td></tr>
					</tfoot>
				</table>
			</div>
		{/if}

		{#if bisaCatat}
			<div class="kaki">
				<p class="redup">{semuaTercatat ? 'Semua penerima sudah tercatat.' : `Masih ${r.jumlah_penerima - r.tercatat} penerima belum dicatat.`}</p>
				<form method="POST" action="?/tutup" use:enhance>
					<button class="tombol utama" type="submit" disabled={!semuaTercatat}>Tutup periode</button>
				</form>
			</div>
		{/if}
	</section>
</div>

<style>
	.ringkas {
		display: grid;
		gap: 0.3rem;
	}
	.besar {
		font-size: 1.9rem;
		font-weight: 700;
	}
	tr.cadangan td {
		color: var(--teks-redup);
	}
	input[type='date'] {
		min-width: 9.5rem;
	}
	.nominal {
		width: 9rem;
		text-align: right;
	}
	tfoot td {
		border-top: 2px solid var(--garis);
	}
	.kaki {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
		flex-wrap: wrap;
	}
</style>
