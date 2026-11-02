<!--
  Landing publik masjid. Mengikuti pola situs masjid pada umumnya: navigasi pil
  melayang dengan tombol Donasi menonjol, hero hijau tua beraksen emas, jadwal
  salat tepat di bawah hero, lalu donasi, penyaluran, dan kontak. Angka transparansi
  = agregat database tanpa data pribadi. Gerak hanya transform/opacity dan mati
  bila pengguna memilih "kurangi gerakan".
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import JadwalSalat from '$lib/components/JadwalSalat.svelte';
	import KartuKegiatan from '$lib/components/KartuKegiatan.svelte';
	import KartuProgram from '$lib/components/KartuProgram.svelte';
	import { angka, rupiah } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const r = $derived(data.ringkasan);
	// Panel angka hanya bila datanya termuat dan sudah ada yang dicatat (nol semua
	// terlihat seperti aplikasi yang belum dipakai); selain itu panel memakai teks umum.
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

<div class="halaman">
	<header class="atas">
		<div class="pil">
			<a class="merek" href="/">
				<IkonMasjid ukuran={30} />
				<strong>{masjid.nama}</strong>
			</a>
			<nav class="menu" aria-label="Navigasi utama">
				<a href="#jadwal">Jadwal salat</a>
				<a href="#donasi">Donasi</a>
				<a href="/program">Program</a>
				<a href="/kegiatan">Kegiatan</a>
				<a href="#penyaluran">Penyaluran</a>
			</nav>
			<div class="aksi-pil">
				<a class="masuk" href="/login">Masuk pengurus</a>
				<a class="tombol emas" href="/donasi">Donasi</a>
			</div>
		</div>
	</header>

	<section class="hero" class:foto={!!masjid.foto} style:--foto={masjid.foto ? `url(${JSON.stringify(masjid.foto)})` : undefined}>
		<div class="hero-pola" aria-hidden="true">
			<svg class="pola" width="100%" height="100%">
				<defs>
					<pattern id="bintang" width="80" height="80" patternUnits="userSpaceOnUse">
						<g fill="none" stroke="currentColor" stroke-width="1">
							<rect x="20" y="20" width="40" height="40" />
							<rect x="20" y="20" width="40" height="40" transform="rotate(45 40 40)" />
							<circle cx="40" cy="40" r="6" />
						</g>
					</pattern>
				</defs>
				<rect width="100%" height="100%" fill="url(#bintang)" />
			</svg>
		</div>

		<div class="lebar pahlawan">
			<div class="kiri">
				<p class="alis" style="--i: 0"><span class="garis-emas" aria-hidden="true"></span>Zakat · Infak · Sedekah</p>
				<h1 style="--i: 1">Zakat dan infak Anda, <span class="emas-teks">tersalurkan dengan adil dan tercatat.</span></h1>
				<p class="lead" style="--i: 2">
					Selamat datang di {masjid.nama}. Dana yang terkumpul disalurkan kepada yang berhak dan dapat dipertanggungjawabkan.
				</p>
				<div class="baris-aksi" style="--i: 3">
					<a class="tombol emas besar" href="/donasi">Donasi sekarang</a>
					<a class="tombol hantu besar" href="#penyaluran">Bagaimana dana disalurkan</a>
				</div>
			</div>

			<aside class="panel" id="transparansi" aria-label="Transparansi dana" style="--i: 2">
				{#if tampilTransparansi && r}
					<p class="panel-label">Transparansi dana</p>
					<p class="panel-angka">{rupiah(r.total_diterima)}</p>
					<p class="panel-ket">Dana terkumpul · dari {angka(r.jumlah_donasi)} donasi</p>

					<div class="salur">
						<div class="salur-baris"><span>Sudah disalurkan</span><strong>{rupiah(r.total_tersalur)}</strong></div>
						<div
							class="jalur"
							role="progressbar"
							aria-label="Bagian dana terkumpul yang sudah disalurkan"
							aria-valuemin="0"
							aria-valuemax="100"
							aria-valuenow={persenTersalur}
						>
							<span class="isi" style:--p={persenTersalur / 100}></span>
						</div>
						<p class="panel-ket">{persenTersalur}% dari dana terkumpul</p>
					</div>

					<dl class="rincian">
						<div><dt>Zakat</dt><dd>{rupiah(r.zakat_diterima)}</dd></div>
						<div><dt>Infak & sedekah</dt><dd>{rupiah(r.infak_diterima)}</dd></div>
						<div><dt>Penerima manfaat</dt><dd>{angka(r.penerima_manfaat)}</dd></div>
					</dl>
					<p class="panel-catatan">Dihitung dari donasi dan penyaluran yang tercatat di aplikasi masjid ini.</p>
				{:else}
					<p class="panel-label">Zakat · Infak · Sedekah</p>
					<p class="panel-judul">Setiap donasi dicatat dan mendapat kuitansi yang bisa dicetak.</p>
				{/if}
			</aside>
		</div>
	</section>

	<div class="lebar jadwal-wadah" id="jadwal">
		<JadwalSalat hari={data.hari} lintang={masjid.lintang} bujur={masjid.bujur} />
	</div>

	<main>
		{#if data.pengumuman.length > 0}
			<section class="lebar pengumuman-landing" aria-label="Pengumuman">
				{#each data.pengumuman as p (p.id)}
					<div class="pengumuman" role="note"><strong>{p.judul}</strong><span>{p.isi}</span></div>
				{/each}
			</section>
		{/if}

		<section class="lebar bagian dua-kolom" id="donasi">
			<div class="kolom-teks">
				<h2>Tunaikan zakat dan infak</h2>
				<p class="redup">Pilih jenis yang ingin Anda tunaikan. Setelah donasi diterima, kuitansi dapat dilihat dan dicetak.</p>
				<div class="baris-aksi">
					<a class="tombol utama" href="/donasi">Tunaikan sekarang</a>
				</div>
			</div>
			<ol class="ragam">
				{#each ragam as x, i (x.nama)}
					<li>
						<span class="no" aria-hidden="true">{String(i + 1).padStart(2, '0')}</span>
						<div>
							<h3>{x.nama}</h3>
							<p class="redup">{x.isi}</p>
						</div>
					</li>
				{/each}
			</ol>
		</section>

		{#if data.program.length > 0}
			<section class="lebar bagian program" id="program">
				<div class="kepala-program">
					<div>
						<h2>Program donasi</h2>
						<p class="redup">Dukung program kebaikan masjid yang sedang berjalan. Perkembangannya dilaporkan pengurus di setiap halaman program.</p>
					</div>
					<a class="tombol" href="/program">Lihat semua program →</a>
				</div>
				<div class="grid-program">
					{#each data.program as p (p.id)}<KartuProgram program={p} />{/each}
				</div>
			</section>
		{/if}

		{#if data.kegiatan.length > 0 || data.jadwal.length > 0}
			<section class="lebar bagian program" id="kegiatan">
				<div class="kepala-program">
					<div>
						<h2>Kegiatan masjid</h2>
						<p class="redup">Kegiatan keagamaan dan jadwal kajian di {masjid.nama}. Semua jamaah dipersilakan hadir.</p>
					</div>
					<a class="tombol" href="/kegiatan">Lihat semua kegiatan →</a>
				</div>
				{#if data.kegiatan.length > 0}
					<div class="grid-program">
						{#each data.kegiatan as k (k.id)}<KartuKegiatan kegiatan={k} />{/each}
					</div>
				{/if}
				{#if data.jadwal.length > 0}
					<div class="jadwal-landing">
						<h3>Jadwal rutin</h3>
						<ul>
							{#each data.jadwal as j (j.id)}
								<li>
									<span class="hari-jam"><strong>{j.hari_label}</strong> <span class="angka">{j.jam.replace(':', '.')}</span></span>
									<span>{j.judul}{#if j.pemateri} <span class="redup">· {j.pemateri}</span>{/if}</span>
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			</section>
		{/if}

		<section class="lebar bagian" id="penyaluran">
			<div class="kepala-bagian">
				<h2>Penyaluran yang adil dan terukur</h2>
				<p class="redup">
					Agar dana sampai kepada yang paling membutuhkan, penentuan penerima dibantu sistem pendukung keputusan dengan metode ARAS.
				</p>
			</div>
			<div class="dua-kolom balik">
				<ol class="garis-waktu">
					{#each alur as [judul, isi], i (judul)}
						<li>
							<span class="titik" aria-hidden="true">{i + 1}</span>
							<h3>{judul}</h3>
							<p class="redup">{isi}</p>
						</li>
					{/each}
				</ol>
				<div class="kriteria">
					<h3>Contoh kriteria penilaian</h3>
					<p class="redup">Pengurus dapat menyesuaikan kriteria dan bobotnya.</p>
					<ul>
						{#each kriteria as k (k)}<li>{k}</li>{/each}
					</ul>
					<a class="tombol" href="/cara-kerja">Pelajari cara kerja ARAS</a>
				</div>
			</div>
		</section>

	</main>

	<footer class="kaki" id="kontak">
		<div class="lebar kaki-isi">
			<div class="kaki-merek">
				<IkonMasjid ukuran={44} />
				<div>
					<p class="kaki-nama">{masjid.nama}</p>
					<p>Data pribadi (NIK, No. HP) dienkripsi dan tidak pernah ditampilkan utuh.</p>
				</div>
			</div>
			{#if adaKontak}
				<address class="kaki-kontak">
					<p class="kaki-judul">Kontak</p>
					{#if masjid.alamat}<p>{masjid.alamat}</p>{/if}
					{#if masjid.kontak}<p>{masjid.kontak}</p>{/if}
				</address>
			{/if}
			<p class="kaki-tautan"><a href="/donasi">Donasi</a> · <a href="/cara-kerja">Cara kerja ARAS</a> · <a href="/login">Masuk pengurus</a></p>
		</div>
	</footer>
</div>

<style>
	.halaman {
		--emas: #d9a033;
		--emas-terang: #f0cf86;
		--hijau-pekat: #0c2e25;
	}
	.lebar {
		max-width: 1120px;
		margin: 0 auto;
		padding: 0 1rem;
	}

	/* ---- Navigasi pil melayang (kaca gelap) ---- */
	.atas {
		position: sticky;
		top: 0.75rem;
		z-index: 20; /* di atas isi saat menggulir */
		margin: 0.75rem auto -4.5rem; /* menumpuk di atas hero */
		max-width: 1120px;
		padding: 0 1rem;
	}
	.pil {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		min-height: 3.75rem;
		padding: 0.5rem 0.6rem 0.5rem 1.1rem;
		border-radius: 999px;
		background: rgb(12 46 37 / 0.72);
		backdrop-filter: blur(14px);
		border: 1px solid rgb(255 255 255 / 0.14);
		box-shadow:
			inset 0 1px 0 rgb(255 255 255 / 0.12),
			0 18px 36px -20px rgb(12 46 37 / 0.7);
		color: #fff;
	}
	.merek {
		display: flex;
		align-items: center;
		gap: 0.65rem;
		color: #fff;
		text-decoration: none;
		min-width: 0;
	}
	.merek strong {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.menu {
		display: flex;
		gap: 1.5rem;
	}
	.menu a {
		color: rgb(255 255 255 / 0.8);
		text-decoration: none;
		font-size: 0.95rem;
		transition: color 0.2s;
	}
	.menu a:hover {
		color: var(--emas-terang);
	}
	.aksi-pil {
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	.masuk {
		color: rgb(255 255 255 / 0.8);
		font-size: 0.9rem;
		text-decoration: none;
		white-space: nowrap;
	}
	.masuk:hover {
		color: #fff;
		text-decoration: underline;
	}

	/* ---- Tombol beraksen emas ---- */
	.emas {
		background: var(--emas);
		border-color: transparent;
		color: #2b1d00;
		font-weight: 600;
	}
	.emas:hover {
		background: var(--emas-terang);
	}
	.hantu {
		background: transparent;
		border-color: rgb(255 255 255 / 0.35);
		color: #fff;
	}
	.hantu:hover {
		background: rgb(255 255 255 / 0.1);
	}
	.besar {
		padding: 0.8rem 1.5rem;
	}
	.tombol {
		transition:
			transform 0.2s cubic-bezier(0.16, 1, 0.3, 1),
			background-color 0.2s,
			border-color 0.2s;
	}
	.tombol:active {
		transform: translateY(1px) scale(0.98); /* terasa ditekan */
	}

	/* ---- Hero hijau tua ---- */
	.hero {
		position: relative;
		overflow: hidden;
		color: #fff;
		background: linear-gradient(165deg, var(--hijau-pekat) 0%, var(--hijau-tua) 55%, #0f3328 100%);
		padding: 9rem 0 8rem;
	}
	.hero.foto {
		background:
			linear-gradient(rgb(12 46 37 / 0.84), rgb(15 51 40 / 0.93)),
			var(--foto) center / cover;
	}
	.hero-pola {
		position: absolute;
		inset: 0;
		overflow: hidden;
		color: rgb(255 255 255 / 0.06);
		pointer-events: none;
	}
	.pola {
		position: absolute;
		inset: -80px; /* selisih satu ubin agar pergeseran mulus */
		width: calc(100% + 160px);
		height: calc(100% + 160px);
		animation: geser 70s linear infinite;
	}
	.pahlawan {
		position: relative;
		display: grid;
		grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
		gap: 3.5rem;
		align-items: center;
	}
	.kiri {
		display: grid;
		gap: 1.25rem;
		justify-items: start;
	}
	.alis {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--emas-terang);
	}
	.garis-emas {
		width: 1.75rem;
		height: 2px;
		background: var(--emas);
	}
	.kiri h1 {
		font-size: clamp(1.9rem, 4vw, 2.9rem);
		line-height: 1.12;
		letter-spacing: -0.025em;
		max-width: 20ch;
		color: #fff;
	}
	.emas-teks {
		color: var(--emas-terang);
	}
	.lead {
		font-size: 1.1rem;
		max-width: 54ch;
		color: rgb(255 255 255 / 0.78);
		line-height: 1.65;
	}

	/* ---- Panel transparansi (kaca) ---- */
	.panel {
		display: grid;
		gap: 1.1rem;
		padding: 2rem;
		border-radius: 28px;
		background: rgb(255 255 255 / 0.07);
		backdrop-filter: blur(14px);
		border: 1px solid rgb(255 255 255 / 0.16);
		box-shadow:
			inset 0 1px 0 rgb(255 255 255 / 0.12),
			0 32px 56px -32px rgb(0 0 0 / 0.5);
		translate: 0 1rem; /* sedikit turun: komposisi tidak simetris */
	}
	.panel-label {
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--emas-terang);
	}
	.panel-angka {
		font-size: clamp(2rem, 4.2vw, 2.7rem);
		font-weight: 700;
		letter-spacing: -0.02em;
		line-height: 1.05;
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.panel-ket {
		font-size: 0.9rem;
		color: rgb(255 255 255 / 0.7);
	}
	.panel-judul {
		font-size: 1.35rem;
		font-weight: 600;
		line-height: 1.35;
	}
	.salur {
		display: grid;
		gap: 0.55rem;
	}
	.salur-baris {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		align-items: baseline;
	}
	.salur-baris strong {
		font-variant-numeric: tabular-nums;
	}
	.jalur {
		height: 8px;
		border-radius: 99px;
		background: rgb(255 255 255 / 0.14);
		overflow: hidden;
	}
	.isi {
		display: block;
		height: 100%;
		border-radius: 99px;
		background: var(--emas);
		transform-origin: left;
		transform: scaleX(var(--p));
		animation: tumbuh 1.2s cubic-bezier(0.16, 1, 0.3, 1) 0.5s both;
	}
	.rincian {
		margin: 0;
		display: grid;
	}
	.rincian > div {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		padding: 0.7rem 0;
		border-top: 1px solid rgb(255 255 255 / 0.14);
	}
	.rincian dt {
		color: rgb(255 255 255 / 0.7);
	}
	.rincian dd {
		margin: 0;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}
	.panel-catatan {
		font-size: 0.8rem;
		color: rgb(255 255 255 / 0.6);
	}

	/* ---- Jadwal salat menumpuk di bawah hero ---- */
	.jadwal-wadah {
		position: relative;
		z-index: 5;
		margin-top: -3.5rem;
		scroll-margin-top: 5rem;
	}

	/* ---- Bagian bawah ---- */
	.bagian {
		padding-top: 4rem;
		padding-bottom: 4rem;
		scroll-margin-top: 5rem; /* tidak tertutup navigasi melayang */
	}
	.bagian + .bagian {
		border-top: 1px solid var(--garis);
	}
	.program {
		display: grid;
		gap: 1.5rem;
	}
	.pengumuman-landing {
		display: grid;
		gap: 0.6rem;
		padding-top: 1.5rem;
	}
	.pengumuman {
		display: grid;
		gap: 0.2rem;
		padding: 0.9rem 1.1rem;
		border-radius: 14px;
		border: 1px solid var(--kuning-garis);
		background: var(--kuning);
		color: var(--kuning-teks);
	}
	.jadwal-landing {
		display: grid;
		gap: 0.6rem;
		padding: 1.1rem 1.25rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 16px;
	}
	.jadwal-landing ul {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 0.6rem 1.5rem;
	}
	.jadwal-landing li {
		display: grid;
		grid-template-columns: 8.5rem 1fr;
		gap: 0.5rem;
		font-size: 0.92rem;
	}
	.hari-jam strong {
		color: var(--hijau);
	}
	.kepala-program {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 1rem;
		flex-wrap: wrap;
	}
	.kepala-program p {
		max-width: 60ch;
		margin-top: 0.5rem;
	}
	.grid-program {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 1.25rem;
	}
	.dua-kolom {
		display: grid;
		grid-template-columns: minmax(0, 5fr) minmax(0, 7fr);
		gap: 3.5rem;
		align-items: start;
	}
	.dua-kolom.balik {
		grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
	}
	.kolom-teks {
		display: grid;
		gap: 1rem;
		justify-items: start;
		position: sticky;
		top: 6rem;
	}
	.kolom-teks p,
	.kepala-bagian p {
		max-width: 52ch;
		line-height: 1.65;
	}
	.kolom-teks h2::after,
	.kepala-bagian h2::after {
		content: '';
		display: block;
		width: 2.5rem;
		height: 3px;
		margin-top: 0.7rem;
		border-radius: 2px;
		background: var(--emas); /* garis emas di bawah judul, seperti situs rujukan */
	}
	.kepala-bagian {
		display: grid;
		gap: 0.75rem;
		margin-bottom: 2.5rem;
	}

	/* Ragam donasi: daftar bernomor, dipisah garis (bukan kartu) */
	.ragam {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.ragam li {
		display: grid;
		grid-template-columns: 3rem minmax(0, 1fr);
		gap: 1rem;
		padding: 1.4rem 0;
		border-top: 1px solid var(--garis);
		transition: transform 0.35s cubic-bezier(0.16, 1, 0.3, 1);
	}
	.ragam li:first-child {
		border-top: 0;
		padding-top: 0;
	}
	.ragam li:hover {
		transform: translateX(6px);
	}
	.ragam h3 {
		margin-bottom: 0.25rem;
	}
	.no {
		font-variant-numeric: tabular-nums;
		font-weight: 600;
		color: var(--hijau);
	}

	/* Penyaluran: garis waktu vertikal */
	.garis-waktu {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
	}
	.garis-waktu li {
		position: relative;
		padding: 0 0 2rem 3.5rem;
	}
	.garis-waktu li:last-child {
		padding-bottom: 0;
	}
	.garis-waktu li:not(:last-child)::before {
		content: '';
		position: absolute;
		left: 1.05rem;
		top: 2.4rem;
		bottom: 0.4rem;
		width: 1px;
		background: var(--garis);
	}
	.titik {
		position: absolute;
		left: 0;
		top: -0.1rem;
		width: 2.1rem;
		height: 2.1rem;
		display: grid;
		place-items: center;
		border-radius: 50%;
		background: var(--hijau-muda);
		color: var(--hijau-tua);
		font-weight: 600;
		font-variant-numeric: tabular-nums;
	}
	.garis-waktu h3 {
		margin-bottom: 0.25rem;
	}
	.kriteria {
		display: grid;
		gap: 1rem;
		justify-items: start;
	}
	.kriteria ul {
		list-style: none;
		padding: 0;
		margin: 0;
		width: 100%;
	}
	.kriteria li {
		padding: 0.7rem 0;
		border-top: 1px solid var(--garis);
	}
	.kriteria li:last-child {
		border-bottom: 1px solid var(--garis);
	}
	/* ---- Footer hijau tua penuh lebar (identitas, kontak, tautan) ---- */
	.kaki {
		background: var(--hijau-pekat);
		color: rgb(255 255 255 / 0.75);
		font-size: 0.9rem;
		scroll-margin-top: 5rem;
	}
	.kaki-isi {
		display: grid;
		grid-template-columns: minmax(0, 5fr) minmax(0, 4fr) auto;
		gap: 2.5rem;
		align-items: start;
		padding-top: 2.5rem;
		padding-bottom: 2.5rem;
	}
	.kaki-kontak {
		display: grid;
		gap: 0.35rem;
		font-style: normal; /* <address> bawaannya miring */
		line-height: 1.55;
	}
	.kaki-judul {
		color: var(--emas-terang);
		font-weight: 600;
		font-size: 0.8rem;
		letter-spacing: 0.1em;
		text-transform: uppercase;
	}
	.kaki p {
		margin: 0;
	}
	.kaki-merek {
		display: flex;
		align-items: center;
		gap: 1rem;
	}
	.kaki-nama {
		color: #fff;
		font-weight: 600;
		font-size: 1.05rem;
		margin-bottom: 0.25rem;
	}
	.kaki a {
		color: var(--emas-terang);
	}

	/* ---- Gerak: hanya transform/opacity ---- */
	.kiri > *,
	.panel {
		animation: naik 0.8s cubic-bezier(0.16, 1, 0.3, 1) both;
		animation-delay: calc(var(--i, 0) * 80ms);
	}
	@keyframes naik {
		from {
			opacity: 0;
			transform: translateY(16px);
		}
	}
	@keyframes tumbuh {
		from {
			transform: scaleX(0);
		}
	}
	@keyframes geser {
		to {
			transform: translate(80px, 80px);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.kiri > *,
		.panel,
		.isi,
		.pola {
			animation: none;
		}
		.ragam li,
		.tombol {
			transition: none;
		}
		.ragam li:hover {
			transform: none;
		}
	}

	/* ---- Layar sempit: semua kolom menjadi satu ---- */
	@media (max-width: 860px) {
		.menu,
		.masuk {
			display: none; /* menu panjang tidak muat; tombol Donasi tetap ada */
		}
		.hero {
			padding: 7rem 0 6rem;
		}
		.pahlawan {
			grid-template-columns: minmax(0, 1fr);
			gap: 2.5rem;
		}
		.panel {
			translate: none;
			padding: 1.5rem;
		}
		.dua-kolom,
		.dua-kolom.balik {
			grid-template-columns: minmax(0, 1fr);
			gap: 2rem;
		}
		.kolom-teks {
			position: static;
		}
		.bagian {
			padding-top: 3rem;
			padding-bottom: 3rem;
		}
		.kaki-isi {
			grid-template-columns: minmax(0, 1fr);
			gap: 1.5rem;
		}
	}
</style>
