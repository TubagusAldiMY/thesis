<!--
  Cara kerja ARAS: 6 langkah perhitungan, penjelasan statis (bukan data database).
  Dulunya satu bagian di landing; dipindah ke halaman sendiri supaya landing singkat.
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
</style>
