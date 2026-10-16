<!--
  Cara kerja ARAS: enam langkah perhitungan, contoh kriteria + bobot, dan contoh
  ranking. Semuanya penjelasan STATIS (bukan data database); dulunya sebagian ada
  di landing, dipindah ke sini supaya landing masjid tetap ringkas.
-->
<script lang="ts">
	import { masjid } from '$lib/masjid';

	const langkah = [
		['Matriks keputusan', 'Nilai tiap calon penerima pada 5 kriteria, ditambah alternatif ideal A₀ (nilai terbaik tiap kriteria).'],
		['Normalisasi', 'Kriteria benefit: r = x ÷ Σx. Kriteria cost dibalik dulu (1/x) lalu dinormalisasi.'],
		['Terbobot', 'd = w × r, dengan w = bobot kriteria (Σw = 1).'],
		['Fungsi optimalitas', 'S = Σ d pada satu baris; S₀ milik alternatif ideal.'],
		['Derajat utilitas', 'K = S ÷ S₀ (0–1). Semakin besar K, semakin layak menerima.'],
		['Ranking & kuota', 'Urut menurut K; N teratas = usulan penerima, N = ⌊dana ÷ nominal per penerima⌋.']
	];
	const kriteria = [
		{ kode: 'C1', nama: 'Penghasilan per bulan', tipe: 'cost', bobot: '0,30' },
		{ kode: 'C2', nama: 'Jumlah tanggungan', tipe: 'benefit', bobot: '0,20' },
		{ kode: 'C3', nama: 'Kondisi tempat tinggal', tipe: 'benefit', bobot: '0,20' },
		{ kode: 'C4', nama: 'Kondisi kesehatan / disabilitas', tipe: 'benefit', bobot: '0,15' },
		{ kode: 'C5', nama: 'Status pekerjaan', tipe: 'benefit', bobot: '0,15' }
	];
	const contoh = [
		{ alt: 'A2', k: 0.8939, ket: 'Penerima' },
		{ alt: 'A4', k: 0.7327, ket: 'Penerima' },
		{ alt: 'A1', k: 0.6888, ket: 'Penerima' },
		{ alt: 'A5', k: 0.5611, ket: 'Cadangan' },
		{ alt: 'A3', k: 0.4099, ket: 'Cadangan' }
	];
</script>

<svelte:head>
	<title>Cara kerja ARAS · {masjid.nama}</title>
	<meta name="description" content="Enam langkah metode ARAS (Additive Ratio Assessment) untuk meranking calon penerima zakat dan infak." />
</svelte:head>

<div class="tumpuk">
	<div>
		<a class="redup" href="/">← Beranda</a>
		<h1>Cara kerja ARAS</h1>
		<p class="redup">
			ARAS (Additive Ratio Assessment) meranking calon penerima dari nilai kelima kriteria mereka, dalam enam langkah.
		</p>
	</div>

	<ol class="langkah">
		{#each langkah as [judul, isi], i (judul)}
			<li class="kartu">
				<span class="badge hijau">Langkah {i + 1}</span>
				<h2>{judul}</h2>
				<p class="redup">{isi}</p>
			</li>
		{/each}
	</ol>

	<section class="kartu">
		<h2>Contoh lima kriteria dan bobot</h2>
		<div class="tabel-wadah">
			<table>
				<thead><tr><th>Kode</th><th>Kriteria</th><th>Tipe</th><th class="kanan">Bobot</th></tr></thead>
				<tbody>
					{#each kriteria as k (k.kode)}
						<tr>
							<td><strong>{k.kode}</strong></td>
							<td>{k.nama}</td>
							<td><span class="badge {k.tipe === 'cost' ? 'kuning' : 'hijau'}">{k.tipe}</span></td>
							<td class="kanan angka">{k.bobot}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="redup catatan">
			Cost = makin kecil makin layak (penghasilan). Benefit = makin besar makin layak. Bobot di atas hanya contoh; bobot yang dipakai
			ditetapkan pengurus masjid dan dapat disesuaikan.
		</p>
	</section>

	<section class="kartu">
		<h2>Contoh hasil ranking</h2>
		<p class="redup catatan">Contoh statis 5 calon penerima, dana Rp 3.000.000, Rp 1.000.000 per penerima (kuota 3).</p>
		<ol class="ranking">
			{#each contoh as c, i (c.alt)}
				<li>
					<span class="no">{i + 1}</span>
					<span class="alt">{c.alt}</span>
					<span class="bar" aria-hidden="true"><span style:width="{c.k * 100}%"></span></span>
					<span class="angka">{c.k.toFixed(4).replace('.', ',')}</span>
					<span class="badge kapital {c.ket === 'Penerima' ? 'hijau' : ''}">{c.ket}</span>
				</li>
			{/each}
		</ol>
	</section>
</div>

<style>
	.langkah {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: 1rem;
	}
	.langkah li {
		display: grid;
		gap: 0.5rem;
		align-content: start;
	}
	.langkah .badge {
		justify-self: start;
	}
	.langkah h2 {
		font-size: 1.1rem;
	}
	.catatan {
		font-size: 0.9rem;
		margin-top: 0.75rem;
	}
	.ranking {
		list-style: none;
		padding: 0;
		margin: 1rem 0 0;
		display: grid;
		gap: 0.6rem;
	}
	.ranking li {
		display: grid;
		grid-template-columns: 1.5rem 2.5rem 1fr auto auto;
		gap: 0.75rem;
		align-items: center;
	}
	.no {
		color: var(--teks-redup);
	}
	.alt {
		font-weight: 700;
	}
	.bar {
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
</style>
