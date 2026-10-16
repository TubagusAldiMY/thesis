<!--
  Landing publik masjid: sambutan, ragam zakat/infak + ajakan donasi, angka
  transparansi (agregat dari database, tanpa data pribadi), cara penyaluran
  (ARAS secara ringkas, rinciannya di /cara-kerja), dan kontak masjid.
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { angka, rupiah } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const r = $derived(data.ringkasan);
	// Bagian transparansi hanya tampil bila angkanya termuat dan sudah ada yang dicatat
	// (halaman depan penuh angka nol terlihat seperti aplikasi yang belum dipakai).
	const tampilTransparansi = $derived(!!r && (r.jumlah_donasi > 0 || r.total_tersalur > 0));
	const persenTersalur = $derived(r && r.total_diterima > 0 ? Math.min(100, Math.round((r.total_tersalur / r.total_diterima) * 100)) : 0);
	const adaKontak = $derived(!!(masjid.alamat || masjid.kontak));

	const ragam = [
		{ nama: 'Zakat fitrah', isi: 'Zakat wajib bagi setiap muslim pada bulan Ramadan, ditunaikan sebelum salat Idulfitri.' },
		{ nama: 'Zakat mal', isi: 'Zakat atas harta yang telah mencapai nisab dan dimiliki selama satu tahun (haul).' },
		{ nama: 'Zakat profesi', isi: 'Zakat atas penghasilan dari pekerjaan atau profesi yang telah mencapai nisab.' },
		{ nama: 'Infak & sedekah', isi: 'Sumbangan sukarela untuk kemaslahatan umat, berapa pun jumlahnya.' }
	];
	const alur = [
		['Pengurus mendata', 'Calon penerima (mustahik) dicatat pengurus; data pribadi seperti NIK dan No. HP disimpan terenkripsi.'],
		['Dinilai secara adil', 'Setiap calon dinilai dengan kriteria yang sama dan diranking otomatis memakai metode ARAS.'],
		['Disetujui dan dicatat', 'Ketua DKM menyetujui usulan penerima, lalu setiap penyaluran dicatat sampai laporan siap cetak.']
	];
	const kriteria = ['Penghasilan per bulan', 'Jumlah tanggungan', 'Kondisi tempat tinggal', 'Kondisi kesehatan / disabilitas', 'Status pekerjaan'];
</script>

<svelte:head>
	<title>{masjid.nama} · Zakat, Infak & Sedekah</title>
	<meta name="description" content="Tunaikan zakat, infak, dan sedekah di {masjid.nama}. Dana disalurkan kepada yang berhak dan dapat dipertanggungjawabkan." />
</svelte:head>

<header class="atas">
	<div class="lebar baris">
		<a class="merek" href="/">
			<IkonMasjid ukuran={34} />
			<strong>{masjid.nama}</strong>
		</a>
		<nav class="menu" aria-label="Navigasi utama">
			<a href="#donasi">Donasi</a>
			{#if tampilTransparansi}<a href="#transparansi">Transparansi</a>{/if}
			<a href="#penyaluran">Penyaluran</a>
			{#if adaKontak}<a href="#kontak">Kontak</a>{/if}
		</nav>
		<a class="tombol utama masuk" href="/login">Masuk pengurus</a>
	</div>
</header>

<main>
	<section class="lebar pahlawan">
		<p class="label-kecil">Assalamu'alaikum warahmatullahi wabarakatuh</p>
		<h1>Selamat datang di {masjid.nama}</h1>
		<p class="redup lead">
			Tunaikan zakat, infak, dan sedekah Anda dengan mudah. Dana yang terkumpul disalurkan kepada yang berhak, tercatat rapi,
			dan dapat dipertanggungjawabkan.
		</p>
		<div class="baris-aksi">
			<a class="tombol utama" href="/donasi">Donasi sekarang</a>
			<a class="tombol" href={tampilTransparansi ? '#transparansi' : '#penyaluran'}>
				{tampilTransparansi ? 'Lihat transparansi dana' : 'Bagaimana dana disalurkan'}
			</a>
		</div>
	</section>

	<section class="lebar bagian" id="donasi">
		<h2>Tunaikan zakat dan infak</h2>
		<p class="redup pengantar">Pilih jenis yang ingin Anda tunaikan. Setelah donasi diterima, kuitansi dapat dilihat dan dicetak.</p>
		<ul class="ragam">
			{#each ragam as x (x.nama)}
				<li class="kartu">
					<h3>{x.nama}</h3>
					<p class="redup">{x.isi}</p>
				</li>
			{/each}
		</ul>
		<div class="baris-aksi">
			<a class="tombol utama" href="/donasi">Tunaikan sekarang</a>
		</div>
	</section>

	{#if tampilTransparansi && r}
		<section class="lebar bagian" id="transparansi">
			<h2>Transparansi dana</h2>
			<p class="redup pengantar">Angka berikut dihitung langsung dari donasi dan penyaluran yang tercatat di aplikasi masjid ini.</p>
			<div class="statistik">
				<div class="kartu stat">
					<p class="redup">Dana terkumpul</p>
					<p class="besar">{rupiah(r.total_diterima)}</p>
					<p class="redup kecil">Dari {angka(r.jumlah_donasi)} donasi</p>
				</div>
				<div class="kartu stat">
					<p class="redup">Sudah disalurkan</p>
					<p class="besar">{rupiah(r.total_tersalur)}</p>
					<p class="redup kecil">{persenTersalur}% dari dana terkumpul</p>
				</div>
				<div class="kartu stat">
					<p class="redup">Penerima manfaat</p>
					<p class="besar">{angka(r.penerima_manfaat)}</p>
					<p class="redup kecil">Orang yang telah menerima penyaluran</p>
				</div>
			</div>
			<p class="redup rincian">Zakat {rupiah(r.zakat_diterima)} · Infak & sedekah {rupiah(r.infak_diterima)}</p>
		</section>
	{/if}

	<section class="lebar bagian" id="penyaluran">
		<h2>Penyaluran yang adil dan terukur</h2>
		<p class="redup pengantar">
			Agar dana sampai kepada yang paling membutuhkan, penentuan penerima dibantu sistem pendukung keputusan dengan metode ARAS.
		</p>
		<ol class="alur">
			{#each alur as [judul, isi], i (judul)}
				<li class="kartu">
					<span class="badge hijau">Langkah {i + 1}</span>
					<h3>{judul}</h3>
					<p class="redup">{isi}</p>
				</li>
			{/each}
		</ol>
		<p class="redup kriteria-info">Kriteria penilaian:</p>
		<ul class="daftar-kriteria">
			{#each kriteria as k (k)}<li>{k}</li>{/each}
		</ul>
		<div class="baris-aksi">
			<a class="tombol" href="/cara-kerja">Pelajari cara kerja ARAS</a>
		</div>
	</section>

	{#if adaKontak}
		<section class="lebar bagian" id="kontak">
			<h2>Kontak {masjid.nama}</h2>
			<div class="kartu kontak">
				{#if masjid.alamat}<p><strong>Alamat</strong><br />{masjid.alamat}</p>{/if}
				{#if masjid.kontak}<p><strong>Kontak</strong><br />{masjid.kontak}</p>{/if}
			</div>
		</section>
	{/if}
</main>

<footer class="lebar kaki redup">
	<p>{masjid.nama} · Data pribadi (NIK, No. HP) dienkripsi dan tidak pernah ditampilkan utuh.</p>
	<p><a href="/login">Masuk pengurus</a> · <a href="/cara-kerja">Cara kerja ARAS</a></p>
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
		flex-wrap: wrap;
	}
	.merek {
		display: flex;
		gap: 0.75rem;
		align-items: center;
		color: inherit;
		text-decoration: none;
	}
	.menu {
		display: flex;
		gap: 1.25rem;
		flex-wrap: wrap;
	}
	.menu a {
		color: #fff;
		opacity: 0.9;
		text-decoration: none;
	}
	.menu a:hover {
		opacity: 1;
		text-decoration: underline;
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
	}
	.pahlawan h1 {
		font-size: clamp(2rem, 5vw, 3.2rem);
		max-width: 20ch;
	}
	.lead {
		font-size: 1.1rem;
		max-width: 60ch;
	}
	.label-kecil {
		color: var(--hijau);
		font-weight: 600;
	}
	.bagian {
		display: grid;
		gap: 1rem;
		margin-bottom: 3rem;
		scroll-margin-top: 1rem;
	}
	.pengantar {
		max-width: 65ch;
	}
	.ragam,
	.alur {
		list-style: none;
		padding: 0;
		margin: 0.5rem 0;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr));
		gap: 1rem;
	}
	.ragam li,
	.alur li {
		display: grid;
		gap: 0.5rem;
		align-content: start;
	}
	.alur .badge {
		justify-self: start;
	}
	.statistik {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
		gap: 1rem;
	}
	.stat {
		display: grid;
		gap: 0.35rem;
	}
	.besar {
		font-size: clamp(1.5rem, 3.5vw, 2rem);
		font-weight: 700;
		line-height: 1.15;
		overflow-wrap: anywhere;
	}
	.kecil {
		font-size: 0.9rem;
	}
	.rincian {
		font-size: 0.95rem;
	}
	.kriteria-info {
		margin: 0.5rem 0 0;
	}
	.daftar-kriteria {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
	.daftar-kriteria li {
		background: var(--hijau-muda);
		color: var(--hijau-tua);
		border-radius: 99px;
		padding: 0.3rem 0.85rem;
		font-size: 0.9rem;
	}
	.kontak {
		display: grid;
		gap: 1rem;
	}
	.kontak p {
		margin: 0;
	}
	.kaki {
		padding-top: 2rem;
		padding-bottom: 2rem;
		font-size: 0.9rem;
		display: grid;
		gap: 0.25rem;
	}
	.kaki p {
		margin: 0;
	}
	@media (max-width: 640px) {
		.menu {
			order: 3;
			width: 100%;
		}
	}
</style>
