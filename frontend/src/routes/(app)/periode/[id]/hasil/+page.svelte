<!--
  Tab Hasil (FR-10/11/12): tabel ranking S, K, status USULAN/CADANGAN, ringkasan
  run, "Lihat langkah hitung" (matriks, normalisasi, terbobot, S), dan panel
  keputusan yang hanya aktif untuk Ketua saat status dihitung.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { angka, desimal, rupiah, tanggal } from '$lib/format';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const h = $derived(data.hasil);
	const p = $derived(data.detail.periode);
	const ketua = $derived(data.user.role === 'ketua');
	const admin = $derived(data.user.role === 'admin');
	let mengirim = $state(false);

	// Kode alternatif A1..Am mengikuti urutan baris langkah hitung (= urutan di tab Penilaian).
	const kode = $derived(new Map((h?.langkah?.alternatif ?? []).map((id, i) => [id, `A${i + 1}`])));
	const usulan = $derived(h?.items.filter((x) => x.status === 'usulan') ?? []);
	const cadangan = $derived(h?.items.filter((x) => x.status === 'cadangan') ?? []);
	const lamaDilihat = $derived(Boolean(h && !h.run.is_current));
	const namaDari = (id: string) => h?.items.find((x) => x.mustahik.id === id)?.mustahik.nama ?? id;
	/** baris tabel langkah: 0 = A0 (ideal), 1..m = alternatif. */
	const labelBaris = (i: number) => (i === 0 ? 'A0 (ideal)' : `A${i} · ${namaDari(h?.langkah?.alternatif[i - 1] ?? '')}`);
	// Go menyimpan kolom cost di matriks x SUDAH dibalik (1/x). Untuk langkah 1
	// ditampilkan kembali nilai aslinya (1 ÷ nilai tersimpan) supaya cocok dengan input.
	const cost = $derived(new Set((h?.run.bobot ?? []).filter((b) => b.tipe === 'cost').map((b) => b.kode)));
	/** sel matriks mentah tampil seperti nilai aslinya (800.000 / 3); matriks lain 4 desimal. */
	function sel(v: number, mentah: boolean, kodeKolom: string): string {
		if (!mentah) return desimal(v);
		const asli = cost.has(kodeKolom) && v !== 0 ? Math.round(1 / v) : v;
		return Number.isInteger(asli) ? angka(asli) : desimal(asli);
	}

	const tabelLangkah = $derived(
		h?.langkah
			? [
					{ judul: '1. Matriks keputusan x (baris 0 = alternatif ideal A0; kolom cost tampil nilai asli, dibalik 1/x di langkah 2)', isi: h.langkah.matriks, mentah: true, s: false },
					{ judul: '2. Matriks ternormalisasi r (cost dibalik 1/x, lalu dibagi Σ kolom)', isi: h.langkah.normalisasi, mentah: false, s: false },
					{ judul: '3. Matriks terbobot d = w × r, dan S = Σ d', isi: h.langkah.terbobot, mentah: false, s: true }
				]
			: []
	);
</script>

<svelte:head><title>Hasil ARAS · {p.nama}</title></svelte:head>

{#if !h}
	<section class="kartu kosong">
		<h2>Belum dihitung</h2>
		<p class="redup">Ranking muncul setelah Admin mengisi penilaian lengkap lalu menekan “Hitung ARAS”.</p>
		{#if admin}<a class="tombol utama" href="/periode/{p.id}/penilaian">Ke penilaian</a>{/if}
	</section>
{:else}
	{#if data.baruDihitung && !lamaDilihat}<p class="kotak-sukses" role="status">Perhitungan ARAS selesai (run v{h.run.versi}).</p>{/if}
	{#if lamaDilihat}
		<p class="kotak-info">Anda melihat ranking run lama (v{h.run.versi}). <a href="/periode/{p.id}/hasil">Lihat run terkini →</a></p>
	{/if}

	<div class="tata">
		<section class="kartu tumpuk">
			<div>
				<h2>Ranking derajat utilitas (K)</h2>
				<p class="redup kecil">
					Run v{h.run.versi} · {tanggal(h.run.created_at)} · Bobot
					{h.run.bobot.map((b) => `${b.kode} ${desimal(b.bobot, 2)}`).join(' · ')} · S₀ = {desimal(h.run.s0)}
				</p>
			</div>
			<div class="tabel-wadah">
				<table>
					<thead>
						<tr><th>#</th><th>Mustahik</th><th class="kanan lebar-saja">S</th><th>K</th><th class="lebar-saja">Status</th></tr>
					</thead>
					<tbody>
						{#each h.items as it (it.mustahik.id)}
							<tr class:usulan={it.status === 'usulan'}>
								<td>{it.ranking}</td>
								<td>
									<strong>{kode.get(it.mustahik.id) ?? ''} · {it.mustahik.nama}</strong>
									<span class="badge kapital hp-saja {it.status === 'usulan' ? 'hijau' : ''}">{it.status}</span>
								</td>
								<td class="kanan angka lebar-saja">{desimal(it.s)}</td>
								<td>
									<span class="k">
										<span class="bar" aria-hidden="true"><span class:abu={it.status !== 'usulan'} style:width="{Math.min(100, it.k * 100)}%"></span></span>
										<span class="angka">{desimal(it.k)}</span>
									</span>
								</td>
								<td class="lebar-saja"><span class="badge kapital {it.status === 'usulan' ? 'hijau' : ''}">{it.status}</span></td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>

			{#if h.langkah}
				<details class="langkah">
					<summary>Lihat langkah hitung (matriks, normalisasi, terbobot d = w·r)</summary>
					{#each tabelLangkah as t (t.judul)}
						<h3>{t.judul}</h3>
						<div class="tabel-wadah">
							<table class="mini">
								<thead>
									<tr>
										<th>Alternatif</th>
										{#each h.langkah.kriteria as k (k)}<th class="kanan">{k}{t.mentah && cost.has(k) ? ' (cost)' : ''}</th>{/each}
										{#if t.s}<th class="kanan">S</th>{/if}
									</tr>
								</thead>
								<tbody>
									{#each t.isi as baris, i (i)}
										<tr>
											<td>{labelBaris(i)}</td>
											{#each baris as v, j (j)}<td class="kanan angka">{sel(v, t.mentah, h.langkah.kriteria[j])}</td>{/each}
											{#if t.s}<td class="kanan angka"><strong>{desimal(baris.reduce((a, b) => a + b, 0))}</strong></td>{/if}
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/each}
					<p class="redup kecil">K = S ÷ S₀, lalu diurutkan menurun. Nilai seri dipecahkan oleh kriteria berbobot terbesar.</p>
				</details>
			{:else}
				<p class="redup kecil">Langkah hitung tidak tersimpan untuk run ini.</p>
			{/if}
		</section>

		<aside class="kartu tumpuk keputusan">
			<div>
				<h2>Keputusan Ketua DKM</h2>
				<p class="redup">
					{usulan.length} usulan penerima ({usulan.map((x) => kode.get(x.mustahik.id)).join(', ') || '–'}) · total
					{rupiah(usulan.length * p.nominal_per_penerima)}.
					{#if cadangan.length}Cadangan: {cadangan.map((x) => kode.get(x.mustahik.id)).join(', ')}.{/if}
				</p>
			</div>

			{#if form?.error && !form?.diputuskan}<p class="kotak-galat" role="alert">{form.fields?.catatan ?? form.error}</p>{/if}

			{#if p.status === 'dihitung' && !lamaDilihat}
				{#if ketua}
					<form
						method="POST"
						action="?/keputusan"
						use:enhance={() => {
							mengirim = true;
							return async ({ update }) => {
								await update();
								mengirim = false;
							};
						}}
					>
						<input type="hidden" name="keputusan" value="setuju" />
						<button class="tombol utama lebar" type="submit" disabled={mengirim}>Setujui usulan</button>
					</form>
					<details class="tolak">
						<summary class="tombol bahaya lebar">Tolak…</summary>
						<form
							method="POST"
							action="?/keputusan"
							class="tumpuk"
							use:enhance={() => {
								mengirim = true;
								return async ({ update }) => {
									await update({ reset: false });
									mengirim = false;
								};
							}}
						>
							<input type="hidden" name="keputusan" value="tolak" />
							<div class="field">
								<label for="catatan">Catatan penolakan (wajib)</label>
								<textarea id="catatan" name="catatan" required minlength="3" maxlength="1000" placeholder="mis. Data A3 perlu dicek ulang">{form?.catatan ?? ''}</textarea>
							</div>
							<button class="tombol bahaya" type="submit" disabled={mengirim}>Kirim penolakan</button>
						</form>
					</details>
				{:else}
					<p class="kotak-info">Menunggu keputusan Ketua DKM. Hanya Ketua yang dapat menyetujui atau menolak.</p>
					<a class="tombol" href="/periode/{p.id}/penilaian">Hitung ulang dari penilaian</a>
				{/if}
			{:else if p.status === 'disetujui' || p.status === 'selesai'}
				<p class="kotak-sukses" role="status">Usulan disetujui.{admin && p.status === 'disetujui' ? ' Lanjutkan pencatatan penyaluran.' : ''}</p>
				{#if admin && p.status === 'disetujui'}<a class="tombol utama lebar" href="/periode/{p.id}/penyaluran">Catat penyaluran</a>{/if}
			{:else if p.status === 'ditolak'}
				<p class="kotak-galat" role="status">Usulan ditolak. Lihat catatan di atas.</p>
			{/if}

			<a class="tombol lebar" href="/periode/{p.id}/laporan">Lihat laporan</a>
		</aside>
	</div>
{/if}

<style>
	.kosong {
		display: grid;
		gap: 0.75rem;
		justify-items: center;
		text-align: center;
		padding: 3rem 1rem;
	}
	.tata {
		display: grid;
		grid-template-columns: minmax(0, 2fr) minmax(280px, 1fr);
		gap: 1.5rem;
		align-items: start;
	}
	.kecil {
		font-size: 0.88rem;
	}
	tr.usulan {
		background: #fafcfa;
	}
	.k {
		display: inline-flex;
		align-items: center;
		gap: 0.75rem;
	}
	.bar {
		width: 64px;
		height: 8px;
		background: var(--abu);
		border-radius: 99px;
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--hijau);
	}
	.bar span.abu {
		background: #b9b4a6;
	}
	.langkah {
		border: 1px solid var(--garis);
		border-radius: 12px;
		padding: 0.9rem 1rem;
	}
	.langkah summary {
		cursor: pointer;
		font-weight: 600;
	}
	.langkah h3 {
		margin: 1.25rem 0 0.5rem;
		font-size: 0.98rem;
	}
	.mini td,
	.mini th {
		padding: 0.45rem 0.6rem;
		font-size: 0.88rem;
	}
	.keputusan {
		position: sticky;
		top: 1rem;
	}
	.lebar {
		width: 100%;
	}
	.tolak summary {
		list-style: none;
	}
	.tolak summary::-webkit-details-marker {
		display: none;
	}
	.tolak[open] summary {
		margin-bottom: 0.75rem;
	}
	.hp-saja {
		display: none;
	}
	@media (max-width: 640px) {
		/* HP: kolom S, grafik batang, dan kolom Status disembunyikan agar
		   K muat tanpa geser; status pindah ke bawah nama, S tetap ada di
		   "Lihat langkah hitung". */
		.lebar-saja,
		.bar {
			display: none;
		}
		.hp-saja {
			display: inline-block;
			margin-top: 0.3rem;
		}
	}
	@media (max-width: 1000px) {
		.tata {
			grid-template-columns: 1fr;
		}
		.keputusan {
			position: static;
		}
	}
</style>
