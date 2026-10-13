<!-- Tentang aplikasi (FR-22): versi + ringkasan metode ARAS, statis. -->
<script lang="ts">
	import { masjid } from '$lib/masjid';
</script>

<svelte:head><title>Tentang aplikasi · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<h1>Tentang aplikasi</h1>
	<section class="kartu tumpuk">
		<div>
			<h2>SPK Penyaluran Zakat & Infak</h2>
			<p class="redup">Versi 0.1.0 · {masjid.nama}</p>
		</div>
		<p>
			Aplikasi ini membantu pengurus masjid menentukan penerima dana zakat dan infak secara adil dan transparan.
			Calon penerima dinilai dengan lima kriteria berbobot, lalu diranking memakai metode
			<strong>ARAS (Additive Ratio Assessment)</strong>.
		</p>
		<ol class="langkah">
			<li>Susun matriks keputusan dan tambahkan alternatif ideal A₀ (nilai terbaik tiap kriteria).</li>
			<li>Normalisasi: benefit r = x ÷ Σx; cost dibalik 1/x lalu dinormalisasi.</li>
			<li>Matriks terbobot d = w × r.</li>
			<li>Fungsi optimalitas S = Σ d tiap alternatif; S₀ untuk A₀.</li>
			<li>Derajat utilitas K = S ÷ S₀; makin besar makin layak.</li>
			<li>N teratas menjadi usulan penerima, N = ⌊dana ÷ nominal per penerima⌋; sisanya cadangan.</li>
		</ol>
		<p class="redup">
			Keputusan akhir tetap di tangan Ketua DKM. Setiap perhitungan disimpan bersama bobot saat itu sehingga hasilnya
			bisa dijelaskan ulang kapan saja. NIK dan No. HP dienkripsi dan tidak pernah ditampilkan utuh.
		</p>
	</section>
</div>

<style>
	.sempit {
		max-width: 760px;
	}
	.langkah {
		display: grid;
		gap: 0.4rem;
		padding-left: 1.25rem;
		margin: 0;
	}
</style>
