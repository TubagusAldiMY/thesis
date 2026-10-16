<!--
  Kriteria & bobot (FR-04/05). Admin dapat mengubah nama, tipe, bobot, skala nilai,
  dan status aktif tiap kriteria, serta menambah kriteria baru. Semua baris disimpan
  sekaligus karena aturannya berlaku atas satu paket utuh: total bobot aktif 100%,
  2–10 kriteria aktif, nama unik. Bobot diketik dengan koma (0,30), kelipatan 0,05;
  total dihitung dalam persen bulat supaya tidak tertipu galat pecahan
  (0,15 + 0,15 ≠ 0,3 di floating point). Kriteria tidak dihapus, hanya dinonaktifkan,
  karena penilaian dan hasil lama menunjuk kriteria itu. Ketua hanya melihat.
-->
<script lang="ts">
	import { untrack } from 'svelte';
	import { enhance } from '$app/forms';
	import { desimal, parseDesimal } from '$lib/format';
	import { parseSkala, teksSkala } from '$lib/kriteria';
	import type { Kriteria } from '$lib/types';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const admin = $derived(data.user.role === 'admin');
	let mengirim = $state(false);

	const AKTIF_MIN = 2;
	const AKTIF_MAKS = 10;

	type Tipe = 'benefit' | 'cost';
	type Baris = { nama: string; tipe: Tipe; bobot: string; skala: string; aktif: boolean };
	const dariServer = (items: Kriteria[]) =>
		Object.fromEntries(
			items.map((k) => [k.id, { nama: k.nama, tipe: k.tipe, bobot: desimal(k.bobot, 2), skala: teksSkala(k.skala), aktif: k.aktif }])
		) as Record<string, Baris>;
	// Nilai awal langsung (agar SSR sudah berisi), lalu diisi ulang setiap data
	// dari server berubah (mis. setelah disimpan).
	let isian = $state<Record<string, Baris>>(untrack(() => dariServer(data.kriteria.items)));
	$effect.pre(() => {
		isian = dariServer(data.kriteria.items);
	});

	// Kriteria yang baru ditambahkan di layar (belum ada di database).
	type Baru = { kunci: string; nama: string; tipe: Tipe; bobot: string; skala: string };
	let baru = $state<Baru[]>([]);
	let urutBaru = 0;
	const tambah = () => baru.push({ kunci: `b${++urutBaru}`, nama: '', tipe: 'benefit', bobot: '0,05', skala: '' });
	const buang = (kunci: string) => (baru = baru.filter((b) => b.kunci !== kunci));
	const kembalikan = () => {
		isian = dariServer(data.kriteria.items);
		baru = [];
	};

	const persen = (s: string) => Math.round(parseDesimal(s) * 100);
	const bobotSalah = (s: string) => {
		const p = persen(s);
		return Number.isNaN(p) || p < 5 || p > 100 || p % 5 !== 0;
	};
	const namaSalah = (s: string) => s.trim().length < 3 || s.trim().length > 100;
	// Pesan bila skala tidak sah ('' = sah). Aturan akhir tetap dijaga Go.
	const pesanSkala = (s: string) => {
		const h = parseSkala(s);
		if ('galat' in h) return h.galat;
		if (h.skala && (Object.keys(h.skala).length < 2 || Object.keys(h.skala).length > 9)) return 'Skala butuh 2–9 pilihan';
		return '';
	};

	const aktifSekarang = $derived(data.kriteria.items.filter((k) => isian[k.id]?.aktif));
	const jumlahAktif = $derived(aktifSekarang.length + baru.length);
	const total = $derived(
		aktifSekarang.reduce((n, k) => n + (persen(isian[k.id].bobot) || 0), 0) + baru.reduce((n, b) => n + (persen(b.bobot) || 0), 0)
	);
	const semuaBaris = $derived([...Object.values(isian), ...baru]);
	const bukanKelipatan = $derived(semuaBaris.some((b) => bobotSalah(b.bobot)));
	const namaBermasalah = $derived(semuaBaris.some((b) => namaSalah(b.nama)));
	const skalaBermasalah = $derived(semuaBaris.map((b) => pesanSkala(b.skala)).find(Boolean) ?? '');
	const jumlahSalah = $derived(jumlahAktif < AKTIF_MIN || jumlahAktif > AKTIF_MAKS);
	const siap = $derived(total === 100 && !bukanKelipatan && !namaBermasalah && !skalaBermasalah && !jumlahSalah);

	const pesanGalat = $derived(form?.fields ? Object.values(form.fields).join(' · ') : (form?.error ?? ''));
</script>

<svelte:head><title>Kriteria & bobot · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Kriteria & bobot</h1>
		<p class="redup">
			Kriteria menentukan apa yang dinilai pada tiap calon penerima, dan bobot menentukan pengaruhnya pada ranking ARAS. Total bobot
			kriteria aktif harus tepat 1,00.
		</p>
	</div>

	{#if form?.tersimpan}<p class="kotak-sukses" role="status">Kriteria tersimpan. Perhitungan berikutnya memakai kriteria baru.</p>{/if}
	{#if pesanGalat}<p class="kotak-galat" role="alert">{pesanGalat}</p>{/if}

	<form
		method="POST"
		class="kartu tumpuk"
		use:enhance={() => {
			mengirim = true;
			return async ({ result, update }) => {
				await update({ reset: false });
				if (result.type === 'success') baru = [];
				mengirim = false;
			};
		}}
	>
		<div class="tabel-wadah">
			<table>
				<thead>
					<tr><th>Kode</th><th>Kriteria</th><th>Tipe</th><th>Bobot</th><th>Skala nilai</th><th>Aktif</th></tr>
				</thead>
				<tbody>
					{#each data.kriteria.items as k (k.id)}
						{@const b = isian[k.id]}
						<tr class:nonaktif={b && !b.aktif}>
							<td><strong>{k.kode}</strong><input type="hidden" name="id" value={k.id} /></td>
							{#if admin && b}
								<td>
									<label class="sr-only" for="nama-{k.id}">Nama {k.kode}</label>
									<input id="nama-{k.id}" name="nama.{k.id}" type="text" class="nama" maxlength="100" bind:value={b.nama} aria-invalid={namaSalah(b.nama)} />
								</td>
								<td class="tipe">
									<label class="sr-only" for="tipe-{k.id}">Tipe {k.kode}</label>
									<select id="tipe-{k.id}" name="tipe.{k.id}" bind:value={b.tipe}>
										<option value="benefit">benefit</option>
										<option value="cost">cost</option>
									</select>
								</td>
								<td class="bobot">
									<label class="sr-only" for="bobot-{k.id}">Bobot {k.kode}</label>
									<input
										id="bobot-{k.id}"
										name="bobot.{k.id}"
										type="text"
										inputmode="decimal"
										class="angka"
										bind:value={b.bobot}
										aria-invalid={bobotSalah(b.bobot)}
									/>
								</td>
								<td>
									<label class="sr-only" for="skala-{k.id}">Skala nilai {k.kode}</label>
									<textarea
										id="skala-{k.id}"
										name="skala.{k.id}"
										rows="3"
										placeholder={'Kosong = angka asli\n1 = sehat\n2 = sakit ringan'}
										bind:value={b.skala}
										aria-invalid={!!pesanSkala(b.skala)}
									></textarea>
								</td>
								<td class="aktif">
									<label>
										<input type="checkbox" name="aktif.{k.id}" bind:checked={b.aktif} />
										<span class="sr-only">Kriteria {k.kode} aktif</span>
										<span aria-hidden="true">{b.aktif ? 'Aktif' : 'Nonaktif'}</span>
									</label>
								</td>
							{:else}
								<td>{k.nama}</td>
								<td><span class="badge {k.tipe === 'cost' ? 'kuning' : 'hijau'}">{k.tipe}</span></td>
								<td><span class="angka">{desimal(k.bobot, 2)}</span></td>
								<td class="redup kecil">
									{#if k.skala}
										{Object.entries(k.skala)
											.map(([n, ket]) => `${n} ${ket}`)
											.join(' · ')}
									{:else}
										Angka asli
									{/if}
								</td>
								<td>{k.aktif ? 'Aktif' : 'Nonaktif'}</td>
							{/if}
						</tr>
					{/each}

					{#each baru as b (b.kunci)}
						<tr class="baru">
							<td>
								<strong>Baru</strong>
								<input type="hidden" name="baru" value={b.kunci} />
							</td>
							<td>
								<label class="sr-only" for="nama-{b.kunci}">Nama kriteria baru</label>
								<input
									id="nama-{b.kunci}"
									name="nama.{b.kunci}"
									type="text"
									class="nama"
									maxlength="100"
									placeholder="mis. Jumlah anak sekolah"
									bind:value={b.nama}
									aria-invalid={namaSalah(b.nama)}
								/>
							</td>
							<td class="tipe">
								<label class="sr-only" for="tipe-{b.kunci}">Tipe kriteria baru</label>
								<select id="tipe-{b.kunci}" name="tipe.{b.kunci}" bind:value={b.tipe}>
									<option value="benefit">benefit</option>
									<option value="cost">cost</option>
								</select>
							</td>
							<td class="bobot">
								<label class="sr-only" for="bobot-{b.kunci}">Bobot kriteria baru</label>
								<input
									id="bobot-{b.kunci}"
									name="bobot.{b.kunci}"
									type="text"
									inputmode="decimal"
									class="angka"
									bind:value={b.bobot}
									aria-invalid={bobotSalah(b.bobot)}
								/>
							</td>
							<td>
								<label class="sr-only" for="skala-{b.kunci}">Skala nilai kriteria baru</label>
								<textarea
									id="skala-{b.kunci}"
									name="skala.{b.kunci}"
									rows="3"
									placeholder={'Kosong = angka asli\n1 = sehat\n2 = sakit ringan'}
									bind:value={b.skala}
									aria-invalid={!!pesanSkala(b.skala)}
								></textarea>
							</td>
							<td><button class="tombol" type="button" onclick={() => buang(b.kunci)}>Batalkan</button></td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if admin}
			<ul class="catatan redup kecil">
				<li><strong>Tipe:</strong> <em>benefit</em> = makin besar makin layak; <em>cost</em> = makin kecil makin layak (mis. penghasilan).</li>
				<li>
					<strong>Skala nilai:</strong> kosongkan bila nilainya angka asli (mis. rupiah). Isi satu pilihan per baris dengan bentuk
					<code>nilai = keterangan</code> bila penilai memilih dari daftar; nilai berupa angka 1–9.
				</li>
				<li>
					<strong>Nonaktif</strong> = tidak ikut penilaian dan ARAS, tetapi riwayatnya tetap. Kriteria tidak bisa dihapus. Kriteria
					baru atau yang diaktifkan harus dinilai pada semua kandidat periode yang sedang berjalan sebelum bisa dihitung;
					hasil dan laporan yang sudah ada tidak berubah.
				</li>
			</ul>
		{/if}

		<div class="kaki">
			<p class="total {siap ? 'pas' : 'tidak'}" role="status">
				Total bobot: <strong class="angka">{desimal(total / 100, 2)}</strong>
				({total}%) —
				{#if bukanKelipatan}
					bobot harus kelipatan 0,05 (0,05–1,00)
				{:else if namaBermasalah}
					nama kriteria 3–100 huruf
				{:else if skalaBermasalah}
					{skalaBermasalah}
				{:else if jumlahSalah}
					jumlah kriteria aktif harus {AKTIF_MIN}–{AKTIF_MAKS} (sekarang {jumlahAktif})
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
					<button class="tombol" type="button" onclick={tambah} disabled={jumlahAktif >= AKTIF_MAKS}>+ Tambah kriteria</button>
					<button class="tombol" type="button" onclick={kembalikan}>Kembalikan</button>
					<button class="tombol utama" type="submit" disabled={!siap || mengirim}>{mengirim ? 'Menyimpan…' : 'Simpan'}</button>
				</div>
			{:else}
				<p class="redup">Hanya Admin yang dapat mengubah kriteria.</p>
			{/if}
		</div>
	</form>
</div>

<style>
	/* select global lebarnya 100% sel; tanpa batas bawah kolom ini menyusut
	   sampai "benefit"/"cost" terpotong. */
	.tipe select {
		min-width: 8.5rem;
	}
	.nama {
		min-width: 14rem;
		width: 100%;
	}
	.bobot input {
		width: 6.5rem;
		text-align: right;
	}
	textarea {
		min-width: 16rem;
		width: 100%;
		resize: vertical;
		font: inherit;
		font-size: 0.9rem;
	}
	.aktif label {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		white-space: nowrap;
	}
	tr.nonaktif td {
		opacity: 0.6;
	}
	tr.nonaktif td.aktif {
		opacity: 1;
	}
	tr.baru {
		background: var(--hijau-muda);
	}
	.kecil {
		font-size: 0.88rem;
	}
	.catatan {
		margin: 0;
		padding-left: 1.2rem;
		display: grid;
		gap: 0.35rem;
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
