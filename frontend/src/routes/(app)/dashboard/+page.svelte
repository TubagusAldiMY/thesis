<!--
  Beranda: ringkasan dana masjid secara umum (penerimaan vs penyaluran, zakat dan
  infak dibukukan terpisah, grafik 12 bulan, donasi terbaru), lalu bagian
  penyaluran ARAS (stepper periode aktif + periode terbaru).
-->
<script lang="ts">
	import JadwalSalat from '$lib/components/JadwalSalat.svelte';
	import Status from '$lib/components/Status.svelte';
	import { angka, labelAsnaf, labelBulan, labelJenisDonasi, rupiah, rupiahRingkas, semuaJenisDonasi, tanggal } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { Asnaf } from '$lib/types';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const d = $derived(data.dash);
	const k = $derived(d.keuangan);
	const admin = $derived(data.user.role === 'admin');

	const rincianAsnaf = $derived(
		Object.entries(d.asnaf_counts)
			.map(([a, n]) => `${labelAsnaf[a as Asnaf]} ${n}`)
			.join(' · ')
	);

	// Grafik batang: tinggi relatif terhadap nilai terbesar (minimal 1 supaya tidak bagi nol).
	const maksBulan = $derived(Math.max(1, ...k.bulanan.flatMap((b) => [b.penerimaan, b.penyaluran])));
	const adaArus = $derived(k.bulanan.some((b) => b.penerimaan > 0 || b.penyaluran > 0));
	const persen = (n: number, maks: number) => `${Math.round((Math.max(0, n) / maks) * 100)}%`;

	const perJenis = $derived(semuaJenisDonasi.map((j) => ({ jenis: j, jumlah: k.per_jenis[j] ?? 0 })));
	const maksJenis = $derived(Math.max(1, ...perJenis.map((x) => x.jumlah)));

	const dana = $derived([
		{ nama: 'Dana zakat', ket: 'Fitrah, mal, dan profesi', v: k.zakat },
		{ nama: 'Dana infak', ket: 'Infak dan sedekah', v: k.infak }
	]);

	const langkah = ['Pilih kandidat', 'Isi penilaian', 'Hitung ARAS', 'Persetujuan Ketua', 'Penyaluran'];
	// Ke tab mana tombol utama mengarah, sesuai langkah aktif (indeks 1–5).
	const tujuan = ['kandidat', 'kandidat', 'penilaian', 'penilaian', 'hasil', 'penyaluran'];

	const keteranganPeriode = $derived.by(() => {
		const a = d.aktif;
		if (!a) return 'Tidak ada periode berjalan';
		if (a.periode.status === 'ditolak') return 'Usulan ditolak, perlu diperbaiki';
		return ['', 'Menunggu pemilihan kandidat', 'Penilaian belum lengkap', 'Siap dihitung', 'Menunggu persetujuan Ketua', 'Penyaluran berjalan'][a.langkah_aktif];
	});
</script>

<svelte:head><title>Beranda · SPK Zakat</title></svelte:head>

<div class="tumpuk dash">
	<!-- Hero hijau tua beraksen emas, senada dengan landing publik. -->
	<section class="hero">
		<div class="hero-pola" aria-hidden="true">
			<svg class="pola" width="100%" height="100%">
				<defs>
					<pattern id="bintang-dash" width="80" height="80" patternUnits="userSpaceOnUse">
						<g fill="none" stroke="currentColor" stroke-width="1">
							<rect x="20" y="20" width="40" height="40" />
							<rect x="20" y="20" width="40" height="40" transform="rotate(45 40 40)" />
							<circle cx="40" cy="40" r="6" />
						</g>
					</pattern>
				</defs>
				<rect width="100%" height="100%" fill="url(#bintang-dash)" />
			</svg>
		</div>
		<div class="hero-isi">
			<div class="sapaan">
				<div>
					<p class="alis"><span class="garis-emas" aria-hidden="true"></span>Beranda pengurus</p>
					<h1>Assalamu'alaikum, {data.user.nama}</h1>
					<p class="lead">Ringkasan dana zakat dan infak {masjid.nama}.</p>
				</div>
				{#if admin}
					<div class="baris-aksi">
						<a class="tombol emas" href="/penerimaan/baru">+ Catat donasi</a>
						<a class="tombol hantu" href="/mustahik/baru">+ Mustahik</a>
						<a class="tombol hantu" href="/periode/baru">+ Periode</a>
					</div>
				{/if}
			</div>

			<div class="stat-kaca">
				<div class="stat">
					<p class="label">Diterima bulan ini</p>
					<p class="besar">{rupiahRingkas(k.diterima_bulan_ini)}</p>
					<p class="ket">{labelBulan(k.bulanan[k.bulanan.length - 1].bulan)}</p>
				</div>
				<div class="stat">
					<p class="label">Total penerimaan</p>
					<p class="besar">{rupiahRingkas(k.total_diterima)}</p>
					<p class="ket">{k.jumlah_donasi ? `Dari ${angka(k.jumlah_donasi)} donasi` : 'Belum ada donasi tercatat'}</p>
				</div>
				<div class="stat">
					<p class="label">Total penyaluran</p>
					<p class="besar">{rupiahRingkas(k.total_tersalur)}</p>
					<p class="ket">{k.total_tersalur ? 'Dari semua periode' : 'Belum ada penyaluran tercatat'}</p>
				</div>
				<div class="stat">
					<p class="label">Saldo tercatat</p>
					<p class="besar" class:minus-hero={k.saldo < 0}>{rupiahRingkas(k.saldo)}</p>
					<p class="ket">Penerimaan dikurangi penyaluran yang tercatat</p>
				</div>
			</div>
		</div>
	</section>

	<div class="jadwal-wadah">
		<JadwalSalat hari={data.hari} lintang={masjid.lintang} bujur={masjid.bujur} />
	</div>

	<section class="dua">
		{#each dana as x (x.nama)}
			<div class="kartu dana">
				<div>
					<h2>{x.nama}</h2>
					<p class="redup kecil">{x.ket}</p>
				</div>
				<dl>
					<div><dt>Diterima</dt><dd class="angka">{rupiah(x.v.diterima)}</dd></div>
					<div><dt>Tersalur</dt><dd class="angka">{rupiah(x.v.tersalur)}</dd></div>
					<div class="saldo"><dt>Saldo</dt><dd class="angka" class:minus={x.v.saldo < 0}>{rupiah(x.v.saldo)}</dd></div>
				</dl>
				<div
					class="bar"
					role="progressbar"
					aria-label="Bagian {x.nama.toLowerCase()} yang sudah tersalur"
					aria-valuemin="0"
					aria-valuemax="100"
					aria-valuenow={x.v.diterima ? Math.min(100, Math.round((x.v.tersalur / x.v.diterima) * 100)) : 0}
				>
					<span style:width={x.v.diterima ? persen(Math.min(x.v.tersalur, x.v.diterima), x.v.diterima) : '0%'}></span>
				</div>
			</div>
		{/each}
	</section>

	<section class="kartu">
		<div class="kartu-kepala">
			<div>
				<h2>Penerimaan dan penyaluran 12 bulan terakhir</h2>
				<p class="redup kecil">Per bulan kalender (WIB).</p>
			</div>
			<ul class="legenda" aria-hidden="true">
				<li><span class="titik masuk"></span>Penerimaan</li>
				<li><span class="titik keluar"></span>Penyaluran</li>
			</ul>
		</div>
		{#if !adaArus}
			<p class="redup kosong-kecil">Belum ada penerimaan atau penyaluran dalam 12 bulan terakhir.</p>
		{:else}
			<div class="grafik" role="img" aria-label="Grafik batang penerimaan dan penyaluran per bulan; angka lengkap ada di tabel di bawahnya">
				{#each k.bulanan as b (b.bulan)}
					<div class="kolom" title="{labelBulan(b.bulan)}: diterima {rupiah(b.penerimaan)}, tersalur {rupiah(b.penyaluran)}">
						<div class="pasangan">
							<span class="batang masuk" style:height={persen(b.penerimaan, maksBulan)}></span>
							<span class="batang keluar" style:height={persen(b.penyaluran, maksBulan)}></span>
						</div>
						<small>{labelBulan(b.bulan).split(' ')[0]}</small>
					</div>
				{/each}
			</div>
			<details class="angka-grafik">
				<summary>Lihat angka per bulan</summary>
				<div class="tabel-wadah">
					<table>
						<thead><tr><th>Bulan</th><th class="kanan">Diterima (Rp)</th><th class="kanan">Tersalur (Rp)</th></tr></thead>
						<tbody>
							{#each k.bulanan as b (b.bulan)}
								<tr><td>{labelBulan(b.bulan)}</td><td class="kanan angka">{angka(b.penerimaan)}</td><td class="kanan angka">{angka(b.penyaluran)}</td></tr>
							{/each}
						</tbody>
					</table>
				</div>
			</details>
		{/if}
	</section>

	<section class="dua">
		<div class="kartu">
			<div class="kartu-kepala">
				<h2>Penerimaan menurut jenis</h2>
			</div>
			<ul class="jenis">
				{#each perJenis as x (x.jenis)}
					<li>
						<span>{labelJenisDonasi[x.jenis]}</span>
						<span class="bar" aria-hidden="true"><span style:width={persen(x.jumlah, maksJenis)}></span></span>
						<span class="angka">{rupiah(x.jumlah)}</span>
					</li>
				{/each}
			</ul>
		</div>

		<div class="kartu">
			<div class="kartu-kepala">
				<h2>Donasi terbaru</h2>
				<a class="tombol" href="/penerimaan">Lihat semua</a>
			</div>
			{#if k.donasi_terbaru.length === 0}
				<div class="kosong">
					<p><strong>Belum ada donasi</strong></p>
					<p class="redup">{admin ? 'Catat donasi tunai atau transfer yang masuk.' : 'Belum ada donasi yang diterima.'}</p>
					{#if admin}<a class="tombol utama" href="/penerimaan/baru">Catat donasi</a>{/if}
				</div>
			{:else}
				<ul class="terbaru">
					{#each k.donasi_terbaru as x (x.id)}
						<li>
							<div>
								<a href="/penerimaan/{x.id}"><strong>{x.nama}</strong></a>
								<p class="redup kecil">
									{labelJenisDonasi[x.jenis]} · {x.sumber === 'web' ? 'Online' : 'Petugas'}{x.dibayar_pada ? ` · ${tanggal(x.dibayar_pada)}` : ''}
								</p>
							</div>
							<span class="angka">{rupiah(x.jumlah)}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</section>

	<div class="judul-bagian">
		<h2>Penyaluran (metode ARAS)</h2>
		<p class="redup">
			<strong>{angka(d.mustahik_aktif)}</strong> mustahik aktif{rincianAsnaf ? ` · ${rincianAsnaf}` : ''}
		</p>
	</div>

	{#if d.aktif}
		{@const a = d.aktif}
		<section class="kartu">
			<div class="kartu-kepala">
				<div>
					<h2>{a.periode.nama}</h2>
					<p class="redup">
						{keteranganPeriode} · Dana {rupiahRingkas(a.periode.dana_tersedia)}, kuota {a.kuota} penerima × {rupiahRingkas(a.periode.nominal_per_penerima)}
					</p>
					<p class="redup">
						Langkah berikutnya: {a.periode.status === 'ditolak' ? 'Buka kembali & perbaiki penilaian' : langkah[a.langkah_aktif - 1]}
					</p>
				</div>
				<a class="tombol utama" href="/periode/{a.periode.id}/{tujuan[a.langkah_aktif]}">
					{a.langkah_aktif >= 4 ? 'Lihat hasil ranking' : 'Lanjutkan'}
				</a>
			</div>
			<ol class="stepper">
				{#each langkah as l, i (l)}
					{@const no = i + 1}
					{@const keadaan = no < a.langkah_aktif ? 'selesai' : no === a.langkah_aktif ? 'sekarang' : 'nanti'}
					<li class={keadaan} aria-current={keadaan === 'sekarang' ? 'step' : undefined}>
						<small>{no} · {keadaan.toUpperCase()}</small>
						<strong>{l}</strong>
					</li>
				{/each}
			</ol>
		</section>
	{/if}

	<section class="kartu">
		<div class="kartu-kepala">
			<h2>Periode terbaru</h2>
			{#if admin}<a class="tombol" href="/periode/baru">+ Buat periode</a>{/if}
		</div>
		{#if d.terbaru.length === 0}
			<div class="kosong">
				<p><strong>Belum ada periode</strong></p>
				<p class="redup">
					{admin ? 'Tambah mustahik, lalu buat periode penyaluran pertama.' : 'Admin belum membuat periode penyaluran.'}
				</p>
				{#if admin}<a class="tombol utama" href="/mustahik/baru">Tambah mustahik</a>{/if}
			</div>
		{:else}
			<div class="tabel-wadah">
				<table>
					<thead>
						<tr><th>Nama periode</th><th>Jenis</th><th class="kanan">Dana (Rp)</th><th>Status</th><th class="kanan">Aksi</th></tr>
					</thead>
					<tbody>
						{#each d.terbaru as p (p.id)}
							<tr>
								<td><strong>{p.nama}</strong></td>
								<td>{p.jenis === 'zakat' ? 'Zakat' : 'Infak'}</td>
								<td class="kanan angka">{angka(p.dana_tersedia)}</td>
								<td><Status status={p.status} /></td>
								<td class="kanan">
									{#if p.status === 'selesai'}
										<a href="/periode/{p.id}/laporan">Laporan</a>
									{:else}
										<a href="/periode/{p.id}">Buka</a>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>
</div>

<style>
	/* ---- Tema senada landing: hijau tua + satu aksen emas ---- */
	.dash {
		--emas: var(--kuning-garis);
		--emas-terang: #f0cf86;
		--hijau-pekat: #0c2e25;
	}
	.hero {
		position: relative;
		overflow: hidden;
		border-radius: 28px;
		color: #fff;
		background: linear-gradient(165deg, var(--hijau-pekat) 0%, var(--hijau-tua) 55%, #0f3328 100%);
		padding: 2rem 2rem 5rem; /* ruang bawah untuk kartu jadwal yang menumpuk */
		box-shadow: 0 32px 56px -34px rgb(12 46 37 / 0.6); /* bayangan sewarna hijau */
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
	.hero-isi {
		position: relative;
		display: grid;
		gap: 1.75rem;
	}
	.sapaan {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 1.25rem;
		flex-wrap: wrap;
	}
	.alis {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		font-size: 0.78rem;
		font-weight: 600;
		letter-spacing: 0.14em;
		text-transform: uppercase;
		color: var(--emas-terang);
		margin: 0 0 0.5rem;
	}
	.garis-emas {
		width: 1.75rem;
		height: 2px;
		background: var(--emas);
	}
	.hero h1 {
		color: #fff;
		font-size: clamp(1.5rem, 3vw, 2.1rem);
		letter-spacing: -0.02em;
		line-height: 1.15;
	}
	.lead {
		margin: 0.35rem 0 0;
		color: rgb(255 255 255 / 0.78);
	}
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
	.tombol {
		transition:
			transform 0.2s cubic-bezier(0.16, 1, 0.3, 1),
			background-color 0.2s,
			border-color 0.2s;
	}
	.tombol:active {
		transform: translateY(1px) scale(0.98); /* terasa ditekan */
	}
	/* Empat angka utama dalam ubin kaca */
	.stat-kaca {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
		gap: 0.75rem;
	}
	.stat {
		display: grid;
		gap: 0.3rem;
		align-content: start;
		padding: 1rem 1.1rem;
		border-radius: 18px;
		background: rgb(255 255 255 / 0.07);
		backdrop-filter: blur(10px);
		border: 1px solid rgb(255 255 255 / 0.16);
		box-shadow: inset 0 1px 0 rgb(255 255 255 / 0.12);
	}
	.stat p {
		margin: 0;
	}
	.stat .label {
		font-size: 0.78rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--emas-terang);
	}
	.stat .besar {
		font-size: clamp(1.4rem, 2.6vw, 1.9rem);
		font-weight: 700;
		letter-spacing: -0.02em;
		line-height: 1.1;
		font-variant-numeric: tabular-nums;
		overflow-wrap: anywhere;
	}
	.stat .ket {
		font-size: 0.85rem;
		color: rgb(255 255 255 / 0.7);
	}
	.minus-hero {
		color: #ffb4ab; /* merah muda: tetap terbaca di latar hijau tua */
	}
	/* Kartu jadwal salat menumpuk di tepi bawah hero */
	.jadwal-wadah {
		position: relative;
		margin: -3.25rem 1.25rem 0;
	}
	/* Garis emas di bawah judul bagian, seperti landing */
	.kartu-kepala h2::after,
	.judul-bagian h2::after {
		content: '';
		display: block;
		width: 2.25rem;
		height: 3px;
		margin-top: 0.5rem;
		border-radius: 2px;
		background: var(--emas);
	}
	.dana .bar span {
		background: var(--emas); /* bagian yang sudah tersalur */
	}
	.hero-isi > * {
		animation: naik 0.7s cubic-bezier(0.16, 1, 0.3, 1) both;
	}
	.hero-isi > :nth-child(2) {
		animation-delay: 0.08s;
	}
	@keyframes naik {
		from {
			opacity: 0;
			transform: translateY(14px);
		}
	}
	@keyframes geser {
		to {
			transform: translate(80px, 80px);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.pola,
		.hero-isi > * {
			animation: none;
		}
		.tombol {
			transition: none;
		}
	}
	@media (max-width: 640px) {
		.hero {
			padding: 1.25rem 1rem 4.5rem;
			border-radius: 22px;
		}
		.stat-kaca {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
		.jadwal-wadah {
			margin: -3rem 0.4rem 0;
		}
	}
	.kecil {
		font-size: 0.9rem;
	}
	.minus {
		color: var(--merah);
	}
	.dua {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 340px), 1fr));
		gap: 1rem;
	}
	.dana {
		display: grid;
		gap: 0.9rem;
		align-content: start;
	}
	.dana dl {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 0.75rem;
		margin: 0;
	}
	.dana dt {
		color: var(--teks-redup);
		font-size: 0.85rem;
	}
	.dana dd {
		margin: 0;
		font-weight: 600;
	}
	.dana .saldo dd {
		color: var(--hijau);
	}
	.dana .saldo dd.minus {
		color: var(--merah);
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
	.legenda {
		display: flex;
		gap: 1rem;
		list-style: none;
		padding: 0;
		margin: 0;
		font-size: 0.9rem;
		color: var(--teks-redup);
	}
	.titik {
		display: inline-block;
		width: 10px;
		height: 10px;
		border-radius: 3px;
		margin-right: 0.35rem;
	}
	.masuk {
		background: var(--hijau);
	}
	.keluar {
		background: var(--kuning-garis);
	}
	.grafik {
		display: grid;
		grid-template-columns: repeat(12, minmax(0, 1fr));
		gap: 0.35rem;
		align-items: end;
	}
	.kolom {
		display: grid;
		gap: 0.35rem;
		justify-items: center;
	}
	.pasangan {
		display: flex;
		align-items: flex-end;
		gap: 2px;
		height: 150px;
		width: 100%;
		justify-content: center;
	}
	.batang {
		display: block;
		width: min(40%, 14px);
		min-height: 2px;
		border-radius: 3px 3px 0 0;
	}
	.kolom small {
		color: var(--teks-redup);
		font-size: 0.75rem;
	}
	.angka-grafik {
		margin-top: 1rem;
	}
	.angka-grafik summary {
		cursor: pointer;
		color: var(--hijau);
		font-weight: 600;
	}
	.jenis,
	.terbaru {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: 0.75rem;
	}
	.jenis li {
		display: grid;
		grid-template-columns: 7.5rem 1fr auto;
		gap: 0.75rem;
		align-items: center;
	}
	.terbaru li {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		align-items: baseline;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid var(--garis);
	}
	.terbaru li:last-child {
		border-bottom: 0;
		padding-bottom: 0;
	}
	.terbaru p {
		margin: 0;
	}
	.terbaru .angka,
	.dana dd {
		white-space: nowrap; /* "Rp 600.000" tidak boleh terpecah di HP */
	}
	.judul-bagian {
		margin-top: 0.5rem;
	}
	.kosong-kecil {
		padding: 1rem 0;
	}
	.stepper {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 0.75rem;
	}
	.stepper li {
		border-radius: 12px;
		padding: 0.9rem 1rem;
		display: grid;
		gap: 0.2rem;
		background: var(--abu);
		border: 2px solid transparent;
	}
	.stepper small {
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.03em;
	}
	.stepper .selesai {
		background: var(--hijau-muda);
		color: var(--hijau-tua);
	}
	.stepper .sekarang {
		background: var(--kuning);
		border-color: var(--kuning-garis);
		color: #5a3c00;
	}
	.kosong {
		display: grid;
		gap: 0.6rem;
		justify-items: center;
		text-align: center;
		padding: 2rem 1rem;
	}
	@media (max-width: 900px) {
		.stepper {
			grid-template-columns: 1fr;
		}
	}
	@media (max-width: 520px) {
		/* Tiga kolom terlalu sempit di HP: tumpuk, label kiri dan angka kanan. */
		.dana dl {
			grid-template-columns: 1fr;
			gap: 0.4rem;
		}
		.dana dl > div {
			display: flex;
			justify-content: space-between;
			gap: 1rem;
		}
		.jenis li {
			grid-template-columns: 1fr auto;
		}
		.jenis li .bar {
			grid-column: 1 / -1;
			order: 3;
		}
	}
</style>
