<!--
  Landing publik (FR-16): penjelasan, 5 kriteria + bobot, rumus ARAS, dan
  contoh ranking STATIS dari docs/02 (bukan data database).
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { masjid } from '$lib/masjid';

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
	<title>SPK Penyaluran Zakat & Infak · {masjid.nama}</title>
	<meta name="description" content="Sistem pendukung keputusan penyaluran dana zakat dan infak masjid dengan metode ARAS." />
</svelte:head>

<header class="atas">
	<div class="lebar baris">
		<div class="merek">
			<IkonMasjid ukuran={30} />
			<div>
				<strong>{masjid.nama}</strong>
				<small>SPK Zakat & Infak</small>
			</div>
		</div>
		<a class="tombol utama" href="/login">Masuk</a>
	</div>
</header>

<main>
	<section class="lebar pahlawan">
		<p class="label-kecil">Metode ARAS · Additive Ratio Assessment</p>
		<h1>Penyaluran zakat & infak yang adil, terukur, dan bisa dipertanggungjawabkan.</h1>
		<p class="redup lead">
			Pengurus mencatat calon penerima (mustahik), menilai lima kriteria, lalu sistem meranking otomatis.
			Ketua DKM menyetujui usulan, dan setiap penyaluran tercatat sampai laporan siap cetak.
		</p>
		<div class="baris-aksi">
			<a class="tombol utama" href="/login">Masuk ke aplikasi</a>
			<a class="tombol" href="/donasi">Donasi zakat & infak</a>
			<a class="tombol" href="#cara-kerja">Lihat cara kerja</a>
		</div>
	</section>

	<section class="lebar dua">
		<div class="kartu">
			<h2>Lima kriteria penilaian</h2>
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
			<p class="redup catatan">Cost = makin kecil makin layak (penghasilan). Benefit = makin besar makin layak.</p>
		</div>

		<div class="kartu">
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
		</div>
	</section>

	<section class="lebar" id="cara-kerja">
		<h2 class="judul-bagian">Cara kerja ARAS</h2>
		<ol class="langkah">
			{#each langkah as [judul, isi], i (judul)}
				<li class="kartu">
					<span class="badge hijau">Langkah {i + 1}</span>
					<h3>{judul}</h3>
					<p class="redup">{isi}</p>
				</li>
			{/each}
		</ol>
	</section>
</main>

<footer class="lebar kaki redup">
	{masjid.nama} · Data pribadi (NIK, No. HP) dienkripsi dan tidak pernah ditampilkan utuh.
</footer>

<style>
	.lebar {
		max-width: 1120px;
		margin: 0 auto;
		padding: 0 1rem;
	}
	.atas {
		background: var(--hijau);
		color: #fff;
		padding: 0.9rem 0;
	}
	.baris {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1rem;
	}
	.merek {
		display: flex;
		gap: 0.75rem;
		align-items: center;
	}
	.merek small {
		display: block;
		opacity: 0.8;
	}
	.atas .tombol.utama {
		background: #fff;
		color: var(--hijau);
	}
	.pahlawan {
		padding-top: 4rem;
		padding-bottom: 3rem;
		display: grid;
		gap: 1.25rem;
		max-width: 1120px;
	}
	.pahlawan h1 {
		font-size: clamp(2rem, 5vw, 3.2rem);
		max-width: 22ch;
	}
	.lead {
		font-size: 1.1rem;
		max-width: 60ch;
	}
	.label-kecil {
		color: var(--hijau);
		font-weight: 600;
	}
	.dua {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 320px), 1fr));
		gap: 1.5rem;
		margin-bottom: 3rem;
	}
	.dua h2,
	.judul-bagian {
		margin-bottom: 1rem;
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
	.langkah {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
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
	.kaki {
		padding-top: 2rem;
		padding-bottom: 2rem;
		font-size: 0.9rem;
	}
</style>
