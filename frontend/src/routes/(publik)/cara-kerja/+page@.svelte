<!--
  Cara kerja ARAS: penjelasan lengkap untuk awam (pengunjung publik dan pengurus baru), bergaya landing masjid.
  Nama berkas "+page@.svelte" membuat halaman ini memakai layout root, bukan layout (publik) yang membatasi
  lebar 680 px, sehingga hero dan kisi lebar bisa dipakai.

  Isinya STATIS (bukan data database): satu contoh lima calon sintetis dihitung langkah demi langkah.
  Semua angka dihitung dari $lib/aras-contoh (dijaga aras-contoh.test.ts), jadi tidak ada angka ketikan
  tangan yang bisa menyimpang dari rumusnya. Perhitungan resmi tetap di backend Go.
  Gerak hanya transform/opacity dan mati bila pengguna memilih "kurangi gerakan".
-->
<script lang="ts">
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import NavPil from '$lib/components/NavPil.svelte';
	import { masjid } from '$lib/masjid';
	import { angka, desimal, rupiah } from '$lib/format';
	import { alternatifContoh, hitungContoh, kriteriaContoh, kuota } from '$lib/aras-contoh';

	const hasil = hitungContoh(kriteriaContoh, alternatifContoh);
	const [ideal, ...calon] = hasil.baris;
	const a1 = calon[0];

	// Contoh dana untuk kuota. Sama dengan Tabel 3.11 di skripsi.
	const dana = 3_000_000;
	const nominal = 1_000_000;
	const n = kuota(dana, nominal);

	const f4 = (x: number) => desimal(x, 4);
	// Kolom C1 setelah dibalik berukuran sangat kecil (0,0000020), jadi ditampilkan dalam satuan 10⁻⁶.
	const mikro = (x: number) => desimal(x * 1e6, 3);
	const bobotMaks = Math.max(...kriteriaContoh.map((k) => k.bobot));

	// Satu warna per kriteria untuk batang bertumpuk "S" (hijau bertingkat + dua emas).
	const warna = ['#17473a', '#3f7563', '#79a592', '#d9a033', '#f0cf86'];

	const navigasi = [
		{ href: '#gagasan', label: 'Gagasan' },
		{ href: '#alur', label: 'Alur' },
		{ href: '#kriteria', label: 'Kriteria' },
		{ href: '#hitung', label: 'Contoh hitung' },
		{ href: '#hasil', label: 'Hasil' },
		{ href: '#tanya', label: 'Tanya jawab' }
	];

	const ringkas = [
		['Nilai yang sama untuk semua', 'Setiap calon dinilai dengan kriteria yang sama: penghasilan, tanggungan, tempat tinggal, kesehatan, dan pekerjaan.'],
		['Satu angka 0 sampai 1', 'ARAS menggabungkan semua nilai menjadi satu angka (K). Makin dekat ke 1, makin layak calon itu dibantu.'],
		['Manusia yang memutuskan', 'Angka itu hanya usulan urutan. Usulan baru sah setelah disetujui Ketua DKM.']
	];

	const alur = [
		['Buat periode penyaluran', 'Admin', 'Memilih jenis dana (zakat atau infak), mengisi dana yang tersedia dan nominal tiap penerima. Aplikasi langsung menghitung berapa orang yang bisa dibantu (kuota).'],
		['Pilih calon penerima', 'Admin', 'Memilih siapa saja yang masuk daftar calon. Pada periode zakat hanya golongan asnaf yang diizinkan yang bisa dipilih, supaya dana zakat tidak salah sasaran.'],
		['Isi penilaian', 'Admin', 'Mengisi nilai setiap calon pada setiap kriteria. Aplikasi menolak nilai di luar rentang dan menampilkan arti skala agar semua petugas memakai ukuran yang sama.'],
		['Hitung dengan ARAS', 'Sistem', 'Aplikasi menghitung peringkat dan menyimpannya sebagai satu versi (riwayat tidak hilang). Calon teratas sebanyak kuota ditandai sebagai usulan penerima.'],
		['Persetujuan', 'Ketua DKM', 'Menyetujui atau menolak usulan. Penolakan wajib disertai catatan; Admin lalu memperbaiki penilaian dan menghitung ulang. Hasil lama tetap tersimpan.'],
		['Penyaluran dan laporan', 'Admin', 'Setelah disetujui, penyaluran dicatat per penerima (total tidak boleh melebihi dana), periode ditutup, lalu laporan dicetak.']
	];

	const skala = [
		[1, 'Layak, milik sendiri', 'Sehat', 'Tetap'],
		[2, 'Layak, sewa / menumpang', 'Sakit ringan', 'Serabutan teratur'],
		[3, 'Kurang layak', 'Sakit kronis / lansia', 'Serabutan tidak teratur'],
		[4, 'Tidak layak / tanpa tempat tinggal', 'Disabilitas berat / sakit parah', 'Tidak bekerja']
	];

	const alasan = [
		['A2', 'Peringkat 1', 'Penghasilannya paling kecil, rumahnya tidak layak, dan tidak bekerja. Hampir semua kriteria mengarah ke "sangat membutuhkan".'],
		['A3', 'Peringkat terakhir', 'Penghasilannya paling besar, sehat, dan rumahnya layak. Wajar bila paling akhir.'],
		['A5', 'Di bawah A1 dan A4', 'Tanggungannya paling banyak, tetapi penghasilannya relatif tinggi dan rumahnya layak. Satu kriteria yang tinggi tidak otomatis membuat calon naik.']
	];

	const aturan = [
		['Disaring sebelum dihitung', 'Pada periode zakat hanya calon dari golongan asnaf yang diizinkan yang ikut dihitung. Pada periode infak, semua mustahik aktif dapat dipilih.'],
		['Hasil hanya usulan', 'Daftar penerima baru sah setelah disetujui Ketua DKM. Bila ditolak, alasan wajib dicatat dan penilaian dapat diperbaiki lalu dihitung ulang.'],
		['Bila dua calon nilainya sama', 'Dianggap sama bila K-nya sama sampai empat desimal. Aplikasi lalu membandingkan nilai asli mulai dari kriteria berbobot terbesar (cost: lebih kecil menang, benefit: lebih besar menang). Bila masih sama, urutan ditentukan tetap sehingga hasilnya tidak berubah-ubah.'],
		['Riwayat tidak hilang', 'Tiap kali dihitung, hasilnya disimpan sebagai versi baru lengkap dengan nama dan bobot kriteria saat itu, sehingga hasil lama tetap bisa ditelusuri walau bobot kemudian diubah.']
	];

	const tanya = [
		['Apakah aplikasi yang memutuskan siapa penerimanya?', 'Tidak. Aplikasi hanya menghitung dan mengusulkan urutan. Keputusan akhir ada pada pengurus: Ketua DKM menyetujui atau menolak, dan penolakan wajib disertai catatan.'],
		['Siapa yang menentukan kriteria dan bobot?', 'Pengurus masjid. Petugas dapat mengubah nama, tipe, bobot, dan arti skala, menambah kriteria baru, atau menonaktifkannya (tidak ada yang dihapus agar riwayat tetap utuh). Jumlah bobot semua kriteria aktif harus tepat 100%, dan aplikasi menolak jika tidak pas.'],
		['Mengapa calon yang sangat membutuhkan bisa tidak di peringkat pertama?', 'Karena yang dinilai adalah gabungan semua kriteria, bukan satu kriteria saja. Seperti A5 pada contoh: tanggungannya paling banyak, tetapi kriteria lainnya tidak terlalu berat, sehingga gabungannya lebih rendah daripada A2.'],
		['Apakah data calon penerima tampil di halaman ini?', 'Tidak. Seluruh nama, angka, dan hasil di halaman ini adalah contoh rekaan. Data mustahik yang sebenarnya hanya dapat dilihat pengurus yang sudah masuk, dan NIK serta nomor HP disimpan terenkripsi.'],
		['Apakah angka contoh ini sama dengan yang dipakai masjid?', 'Tidak harus. Bobot, arti skala, dan jumlah dana di halaman ini hanya untuk memperlihatkan cara hitungnya. Angka yang dipakai masjid ditetapkan pengurus.']
	];

	// Stepper contoh hitung: satu langkah tampil pada satu waktu.
	const langkah = ['Calon ideal', 'Balik cost', 'Normalisasi', 'Bobot', 'Jumlah S', 'Skor K'];
	let aktif = $state(0);
	const geser = (arah: number) => (aktif = Math.min(langkah.length - 1, Math.max(0, aktif + arah)));
	// Panah kiri/kanan memindah langkah saat fokus ada di deretan tab (pola tab WAI-ARIA).
	function tombol(e: KeyboardEvent) {
		if (e.key === 'ArrowRight') geser(1);
		else if (e.key === 'ArrowLeft') geser(-1);
		else return;
		e.preventDefault();
		document.getElementById(`tab-${aktif}`)?.focus();
	}
</script>

{#snippet matriks(baris: typeof hasil.baris, ambil: (b: (typeof hasil.baris)[number]) => number[], fmt: (x: number, j: number) => string, kepala: string[], bawah?: string[])}
	<div class="tabel-wadah">
		<table class="tabel-hitung">
			<thead>
				<tr>
					<th>Calon</th>
					{#each kepala as k (k)}<th class="kanan">{k}</th>{/each}
				</tr>
			</thead>
			<tbody>
				{#each baris as b (b.id)}
					<tr class:ideal={b.id === 'A0'}>
						<td><strong>{b.id === 'A0' ? 'A₀ (ideal)' : b.id}</strong></td>
						{#each ambil(b) as v, j (j)}<td class="kanan angka">{fmt(v, j)}</td>{/each}
					</tr>
				{/each}
				{#if bawah}
					<tr class="jumlah">
						<td><strong>Jumlah kolom (Σ)</strong></td>
						{#each bawah as v, j (j)}<td class="kanan angka"><strong>{v}</strong></td>{/each}
					</tr>
				{/if}
			</tbody>
		</table>
	</div>
{/snippet}

<svelte:head>
	<title>Cara kerja ARAS · {masjid.nama}</title>
	<meta name="description" content="Penjelasan lengkap dan contoh hitung metode ARAS (Additive Ratio Assessment) untuk meranking calon penerima zakat dan infak, dengan bahasa sehari-hari." />
</svelte:head>

<div class="halaman">
	<NavPil tautan={navigasi} />

	<section class="hero">
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
				<p class="alis" style="--i: 0"><span class="garis-emas" aria-hidden="true"></span>Metode ARAS</p>
				<h1 style="--i: 1">Bagaimana calon penerima diurutkan, <span class="emas-teks">adil dan bisa diperiksa.</span></h1>
				<p class="lead" style="--i: 2">
					Penjelasan dengan bahasa sehari-hari dan satu contoh lengkap: bagaimana aplikasi {masjid.nama} mengurutkan calon penerima zakat dan infak.
					Semua angka di sini adalah <strong>contoh rekaan</strong>, bukan data warga dan bukan keputusan pengurus.
				</p>
				<div class="baris-aksi" style="--i: 3">
					<a class="tombol emas besar" href="#hitung">Lihat contoh hitung</a>
					<a class="tombol hantu besar" href="/">Kembali ke beranda</a>
				</div>
			</div>

			<aside class="panel" aria-label="Contoh hasil peringkat" style="--i: 2">
				<p class="panel-label">Contoh hasil peringkat</p>
				<ol class="mini">
					{#each hasil.ranking as c, i (c.id)}
						<li>
							<span class="alt">{c.id}</span>
							<span class="jalur" aria-hidden="true"><span class="isi" style:--p={c.k}></span></span>
							<span class="angka nilai">{f4(c.k)}</span>
							<span class="cap" class:usul={i < n}>{i < n ? 'Usulan' : 'Cadangan'}</span>
						</li>
					{/each}
				</ol>
				<p class="panel-catatan">Lima calon rekaan, dana {rupiah(dana)}, {rupiah(nominal)} per penerima (kuota {n}).</p>
			</aside>
		</div>
	</section>

	<main>
		<div class="lebar ringkas-wadah">
			<ul class="ringkas">
				{#each ringkas as [judul, isi], i (judul)}
					<li class="kartu-mengambang">
						<span class="angka-besar" aria-hidden="true">{i + 1}</span>
						<h2>{judul}</h2>
						<p class="redup">{isi}</p>
					</li>
				{/each}
			</ul>
		</div>

		<section class="lebar bagian dua-kolom" id="gagasan">
			<div class="kolom-teks">
				<h2>Gagasan dasarnya: dibandingkan dengan "calon ideal"</h2>
				<p class="redup">
					Dana zakat dan infak terbatas, sedangkan calon banyak dan keadaannya berbeda-beda. Mana yang lebih perlu dibantu: orang berpenghasilan
					rendah tetapi tanggungannya sedikit, atau orang berpenghasilan sedang tetapi tanggungannya banyak? Kalau hanya mengandalkan kesan,
					jawabannya bisa berbeda antarpengurus dan sulit dijelaskan kepada jamaah dan donatur.
				</p>
				<p>
					Maka dibuatlah satu calon rekaan, <strong>A₀ (calon ideal)</strong>, yang pada tiap kriteria memakai nilai yang paling menggambarkan
					kebutuhan: penghasilan paling kecil, tanggungan paling banyak, rumah paling tidak layak, dan seterusnya. A₀ bukan orang sungguhan,
					hanya patokan. Setiap calon lalu diukur: <em>seberapa dekat keadaannya dengan A₀?</em>
				</p>
			</div>

			<figure class="timbang kartu-kaca">
				<figcaption>
					<strong>Panjang batang = nilai S</strong>
					<span class="redup">Tiap warna adalah satu kriteria. K = panjang batang calon ÷ panjang batang A₀.</span>
				</figcaption>
				<ul class="legenda" aria-label="Warna kriteria">
					{#each kriteriaContoh as k, j (k.kode)}
						<li><span class="kotak" style:background={warna[j]}></span>{k.kode} {k.nama}</li>
					{/each}
				</ul>
				<ol class="batang-baris">
					{#each [ideal, ...hasil.ranking] as b (b.id)}
						<li class:ideal={b.id === 'A0'}>
							<span class="alt">{b.id === 'A0' ? 'A₀' : b.id}</span>
							<span class="batang" role="img" aria-label="{b.id === 'A0' ? 'Calon ideal' : 'Calon ' + b.id}, S = {f4(b.s)}">
								<span class="isi-batang" style:width="{(b.s / ideal.s) * 100}%">
									{#each b.d as v, j (j)}
										<span class="segmen" style:width="{(v / b.s) * 100}%" style:background={warna[j]} title="{kriteriaContoh[j].kode}: {f4(v)}"></span>
									{/each}
								</span>
							</span>
							<span class="angka nilai-k">{b.id === 'A0' ? 'K = 1' : 'K = ' + f4(b.k)}</span>
						</li>
					{/each}
				</ol>
				<p class="redup catatan">Contoh: K untuk A2 = {f4(hasil.ranking[0].k)} berarti batangnya sekitar {Math.round(hasil.ranking[0].k * 100)}% dari batang A₀.</p>
			</figure>
		</section>

		<section class="lebar bagian" id="alur">
			<div class="kepala-bagian">
				<h2>Alur lengkap di aplikasi</h2>
				<p class="redup">Perhitungan ARAS hanya satu bagian dari proses. Dari awal sampai akhir, urutannya sebagai berikut.</p>
			</div>
			<ol class="alur">
				{#each alur as [judul, siapa, isi], i (judul)}
					<li class="kartu-alur">
						<div class="kepala-alur">
							<span class="nomor" aria-hidden="true">{i + 1}</span>
							<span class="peran" class:ketua={siapa === 'Ketua DKM'} class:sistem={siapa === 'Sistem'}>{siapa}</span>
						</div>
						<h3>{judul}</h3>
						<p class="redup">{isi}</p>
					</li>
				{/each}
			</ol>
		</section>

		<section class="lebar bagian" id="kriteria">
			<div class="kepala-bagian">
				<h2>Kriteria, bobot, dan skala</h2>
				<p class="redup">
					<strong>Kriteria</strong> adalah hal yang dinilai pada setiap calon. <strong>Bobot</strong> menunjukkan seberapa penting satu kriteria dibanding
					yang lain; jumlah seluruh bobot harus tepat 1,00 (100%). Berikut contoh yang dipakai di halaman ini.
				</p>
			</div>

			<ul class="kriteria-kisi">
				{#each kriteriaContoh as k (k.kode)}
					<li class="kartu-kriteria">
						<div class="atas-kriteria">
							<span class="kode">{k.kode}</span>
							<span class="badge {k.tipe === 'cost' ? 'kuning' : 'hijau'}">{k.tipe}</span>
						</div>
						<h3>{k.nama}</h3>
						<p class="redup kecil">Diisi sebagai: {k.satuan}</p>
						<div class="bobot">
							<span class="bobot-label">Bobot</span>
							<span class="bobot-jalur" aria-hidden="true"><span style:width="{(k.bobot / bobotMaks) * 100}%"></span></span>
							<span class="angka bobot-nilai">{desimal(k.bobot, 2)}</span>
						</div>
					</li>
				{/each}
			</ul>

			<div class="dua-kolom rata">
				<div class="penjelas">
					<h3>Apa itu benefit dan cost?</h3>
					<div class="arah">
						<div class="arah-kartu">
							<span class="panah naik-panah" aria-hidden="true">↑</span>
							<div>
								<strong>Benefit</strong>
								<p class="redup kecil">Makin <em>besar</em> nilainya, makin layak dibantu. Contoh: tanggungan 6 orang lebih membutuhkan daripada 2 orang.</p>
							</div>
						</div>
						<div class="arah-kartu">
							<span class="panah turun-panah" aria-hidden="true">↓</span>
							<div>
								<strong>Cost</strong>
								<p class="redup kecil">Makin <em>kecil</em> nilainya, makin layak dibantu. Contoh: penghasilan Rp500.000 lebih membutuhkan daripada Rp1.500.000.</p>
							</div>
						</div>
					</div>
				</div>
				<div class="penjelas">
					<h3>Skala 1 sampai 4</h3>
					<p class="redup kecil">Tiga kriteria dipilih sebagai tingkat. Angka 4 sengaja berarti <strong>paling membutuhkan</strong>, jadi ketiganya searah dan bertipe benefit.</p>
					<div class="tabel-wadah">
						<table class="skala">
							<thead><tr><th>Nilai</th><th>Tempat tinggal (C3)</th><th>Kesehatan (C4)</th><th>Pekerjaan (C5)</th></tr></thead>
							<tbody>
								{#each skala as [nilai, ...sel] (nilai)}
									<tr>
										<td><span class="tingkat t{nilai}">{nilai}</span></td>
										{#each sel as s (s)}<td>{s}</td>{/each}
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>
			<p class="redup catatan">Bobot dan arti skala hanya contoh. Yang dipakai sebenarnya ditetapkan pengurus lewat menu Kriteria, dan setiap perubahan tercatat di riwayat aktivitas.</p>
		</section>

		<section class="bagian-gelap" id="hitung">
			<div class="lebar">
				<div class="kepala-bagian terang">
					<h2>Contoh hitung, langkah demi langkah</h2>
					<p>
						Lima calon rekaan di bawah ini dihitung dalam enam langkah. Tampilan dibulatkan empat desimal, sedangkan perhitungan memakai
						angka penuh, jadi menjumlah ulang dengan kalkulator bisa selisih sekitar 0,0001.
					</p>
				</div>

				<div class="kartu-kaca data-awal">
					<h3>Data masukan (nilai yang diisi petugas)</h3>
					<div class="tabel-wadah">
						<table class="tabel-hitung gelap">
							<thead>
								<tr>
									<th>Calon</th>
									{#each kriteriaContoh as k (k.kode)}<th class="kanan">{k.kode}</th>{/each}
								</tr>
							</thead>
							<tbody>
								{#each alternatifContoh as a (a.id)}
									<tr>
										<td><strong>{a.id}</strong></td>
										{#each a.nilai as v, j (j)}<td class="kanan angka">{j === 0 ? rupiah(v) : angka(v)}</td>{/each}
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					<p class="catatan-gelap">Tidak ada calon yang unggul di semua kriteria, jadi perlu cara yang adil untuk menggabungkannya.</p>
				</div>

				<div class="stepper">
					<div class="tab-baris" role="tablist" aria-label="Enam langkah hitung" tabindex="-1" onkeydown={tombol}>
						{#each langkah as nama, i (nama)}
							<button
								id="tab-{i}"
								class="tab"
								class:aktif={aktif === i}
								class:lewat={i < aktif}
								role="tab"
								type="button"
								aria-selected={aktif === i}
								aria-controls="panel-langkah"
								tabindex={aktif === i ? 0 : -1}
								onclick={() => (aktif = i)}
							>
								<span class="tab-no">{i + 1}</span>
								<span class="tab-nama">{nama}</span>
							</button>
						{/each}
					</div>
					<div class="progres" aria-hidden="true"><span style:width="{((aktif + 1) / langkah.length) * 100}%"></span></div>

					<div class="panel-langkah" id="panel-langkah" role="tabpanel" aria-labelledby="tab-{aktif}">
						<p class="langkah-label">Langkah {aktif + 1} dari {langkah.length}</p>

						{#if aktif === 0}
							<h3>Tentukan calon ideal (A₀)</h3>
							<div class="sehari"><span class="chip">Sehari-hari</span><p>Ambil nilai terbaik pada tiap kriteria dari semua calon, lalu jadikan satu "calon ideal".</p></div>
							<p class="rumus">Benefit → nilai terbesar · Cost → nilai terkecil</p>
							<p class="contoh">
								<strong>Pada contoh:</strong> penghasilan terkecil {rupiah(hasil.a0[0])} (milik A2), tanggungan terbanyak {angka(hasil.a0[1])} (A5), dan nilai
								tertinggi untuk tempat tinggal, kesehatan, dan pekerjaan masing-masing {angka(hasil.a0[2])}, {angka(hasil.a0[3])}, dan {angka(hasil.a0[4])}.
							</p>
							<div class="tabel-wadah">
								<table class="tabel-hitung">
									<thead><tr><th>Calon</th>{#each kriteriaContoh as k (k.kode)}<th class="kanan">{k.kode}</th>{/each}</tr></thead>
									<tbody>
										<tr class="ideal">
											<td><strong>A₀ (ideal)</strong></td>
											{#each hasil.a0 as v, j (j)}<td class="kanan angka">{j === 0 ? rupiah(v) : angka(v)}</td>{/each}
										</tr>
									</tbody>
								</table>
							</div>
						{:else if aktif === 1}
							<h3>Samakan arah: balik kriteria cost</h3>
							<div class="sehari">
								<span class="chip">Sehari-hari</span>
								<p>Pada penghasilan, angka kecil justru bagus; pada kriteria lain, angka besar yang bagus. Supaya semuanya searah ("besar = lebih layak"), penghasilan dibalik menjadi 1 dibagi nilainya.</p>
							</div>
							<p class="rumus">x' = 1 ÷ x &nbsp;(hanya kriteria cost)</p>
							<p class="contoh">
								<strong>Pada contoh:</strong> penghasilan A1 Rp800.000 menjadi 1 ÷ 800.000 = 0,00000125. Penghasilan terkecil menjadi angka terbesar, dan
								penghasilan terbesar menjadi angka terkecil. Kolom C1 ditulis dalam satuan ×10⁻⁶ agar mudah dibaca.
							</p>
							{@render matriks(hasil.baris, (b) => b.x, (v, j) => (j === 0 ? mikro(v) : angka(v)), ['C1 (×10⁻⁶)', 'C2', 'C3', 'C4', 'C5'], [mikro(hasil.jumlah[0]), ...hasil.jumlah.slice(1).map(angka)])}
						{:else if aktif === 2}
							<h3>Samakan skala: normalisasi</h3>
							<div class="sehari">
								<span class="chip">Sehari-hari</span>
								<p>Rupiah, jumlah jiwa, dan skala 1–4 tidak bisa dijumlahkan begitu saja. Tiap angka dibagi jumlah kolomnya (baris "Jumlah kolom" di langkah 2), sehingga menjadi <em>porsi</em> antara 0 dan 1.</p>
							</div>
							<p class="rumus">r = x ÷ Σx</p>
							<p class="contoh"><strong>Pada contoh:</strong> tanggungan A1 adalah 5 dari jumlah kolom 26, jadi r = 5 ÷ 26 = {f4(a1.r[1])}.</p>
							{@render matriks(hasil.baris, (b) => b.r, (v) => f4(v), kriteriaContoh.map((k) => k.kode))}
						{:else if aktif === 3}
							<h3>Beri bobot</h3>
							<div class="sehari"><span class="chip">Sehari-hari</span><p>Kriteria yang lebih penting harus lebih berpengaruh. Setiap porsi dikalikan bobot kriterianya.</p></div>
							<p class="rumus">d = w × r</p>
							<p class="contoh"><strong>Pada contoh:</strong> porsi penghasilan A1 {f4(a1.r[0])} dikali bobot {desimal(kriteriaContoh[0].bobot, 2)} menjadi {f4(a1.d[0])}.</p>
							{@render matriks(hasil.baris, (b) => b.d, (v) => f4(v), kriteriaContoh.map((k) => `${k.kode} (×${desimal(k.bobot, 2)})`))}
						{:else if aktif === 4}
							<h3>Jumlahkan: nilai optimalitas (S)</h3>
							<div class="sehari"><span class="chip">Sehari-hari</span><p>Jumlahkan semua nilai berbobot milik satu calon. Hasilnya satu angka yang merangkum seluruh kriteria.</p></div>
							<p class="rumus">S = d₁ + d₂ + d₃ + d₄ + d₅</p>
							<p class="contoh">
								<strong>Pada contoh:</strong> S untuk A1 = {a1.d.map(f4).join(' + ')} = <strong>{f4(a1.s)}</strong>. Untuk calon ideal, S₀ = <strong>{f4(ideal.s)}</strong>.
							</p>
							<div class="tabel-wadah">
								<table class="tabel-hitung sempit">
									<thead><tr><th>Calon</th><th class="kanan">S</th></tr></thead>
									<tbody>
										{#each hasil.baris as b (b.id)}
											<tr class:ideal={b.id === 'A0'}>
												<td><strong>{b.id === 'A0' ? 'A₀ (ideal)' : b.id}</strong></td>
												<td class="kanan angka">{f4(b.s)}</td>
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
						{:else}
							<h3>Hitung derajat utilitas (K)</h3>
							<div class="sehari">
								<span class="chip">Sehari-hari</span>
								<p>Bandingkan S tiap calon dengan S₀ milik calon ideal. Hasilnya K: seberapa dekat calon itu dengan keadaan ideal. Calon ideal sendiri selalu bernilai 1.</p>
							</div>
							<p class="rumus">K = S ÷ S₀</p>
							<p class="contoh"><strong>Pada contoh:</strong> K untuk A1 = {f4(a1.s)} ÷ {f4(ideal.s)} = <strong>{f4(a1.k)}</strong>.</p>
							<div class="tabel-wadah">
								<table class="tabel-hitung sempit">
									<thead><tr><th>Calon</th><th class="kanan">S</th><th class="kanan">S₀</th><th class="kanan">K = S ÷ S₀</th></tr></thead>
									<tbody>
										{#each calon as b (b.id)}
											<tr>
												<td><strong>{b.id}</strong></td>
												<td class="kanan angka">{f4(b.s)}</td>
												<td class="kanan angka">{f4(ideal.s)}</td>
												<td class="kanan angka sorot"><strong>{f4(b.k)}</strong></td>
											</tr>
										{/each}
									</tbody>
								</table>
							</div>
						{/if}

						<div class="navigasi-langkah">
							<button class="tombol hantu" type="button" onclick={() => geser(-1)} disabled={aktif === 0}>← Sebelumnya</button>
							{#if aktif < langkah.length - 1}
								<button class="tombol emas" type="button" onclick={() => geser(1)}>Langkah berikutnya →</button>
							{:else}
								<a class="tombol emas" href="#hasil">Lihat hasil peringkat ↓</a>
							{/if}
						</div>
					</div>
				</div>
			</div>
		</section>

		<section class="lebar bagian dua-kolom" id="hasil">
			<div class="kolom-teks">
				<h2>Hasil: peringkat dan jumlah penerima</h2>
				<p class="redup">Calon diurutkan dari K terbesar. Jumlah penerima (kuota) ditentukan oleh dana, bukan oleh pengurus secara bebas.</p>
				<p class="rumus">Kuota = dana ÷ nominal per penerima, dibulatkan ke bawah</p>
				<p>
					<strong>Pada contoh:</strong> {rupiah(dana)} ÷ {rupiah(nominal)} = <strong>{n} orang</strong>. Tiga calon teratas menjadi <strong>usulan penerima</strong>,
					sisanya <strong>cadangan</strong> menurut urutan peringkat. Pembulatan ke bawah menjaga agar total yang dibagikan tidak pernah melebihi dana.
				</p>
			</div>
			<div class="hasil-kolom">
				<ol class="ranking kartu-kaca-terang">
					{#each hasil.ranking as c, i (c.id)}
						{#if i === n}
							<li class="batas" aria-label="Batas kuota"><span>Batas kuota: {n} penerima</span></li>
						{/if}
						<li class:usul={i < n}>
							<span class="no">{i + 1}</span>
							<span class="alt">{c.id}</span>
							<span class="bar" aria-hidden="true"><span style:width="{c.k * 100}%"></span></span>
							<span class="angka">{f4(c.k)}</span>
							<span class="badge kapital {i < n ? 'hijau' : ''}">{i < n ? 'Usulan' : 'Cadangan'}</span>
						</li>
					{/each}
				</ol>
				<h3>Apakah urutan ini masuk akal?</h3>
				<ul class="alasan">
					{#each alasan as [id, posisi, sebab] (id)}
						<li><span class="id-alasan">{id}</span><div><strong>{posisi}</strong><p class="redup">{sebab}</p></div></li>
					{/each}
				</ul>
				<p class="redup">Itulah gunanya menggabungkan semua kriteria: tidak ada satu kriteria yang menentukan sendirian, melainkan gabungannya sesuai bobot yang disepakati.</p>
			</div>
		</section>

		<section class="lebar bagian" id="aturan">
			<div class="kepala-bagian">
				<h2>Aturan tambahan yang perlu diketahui</h2>
			</div>
			<ul class="aturan-kisi">
				{#each aturan as [judul, isi], i (judul)}
					<li class="kartu-aturan">
						<span class="no-aturan" aria-hidden="true">{String(i + 1).padStart(2, '0')}</span>
						<h3>{judul}</h3>
						<p class="redup">{isi}</p>
					</li>
				{/each}
			</ul>
		</section>

		<section class="lebar bagian dua-kolom" id="tanya">
			<div class="kolom-teks">
				<h2>Pertanyaan yang sering muncul</h2>
				<p class="redup">Belum terjawab? Tanyakan langsung kepada pengurus masjid.</p>
			</div>
			<div class="akordion">
				{#each tanya as [t, j] (t)}
					<details>
						<summary>{t}</summary>
						<p class="redup">{j}</p>
					</details>
				{/each}
			</div>
		</section>

		<section class="lebar seruan">
			<div class="seruan-isi">
				<div>
					<h2>Siap melihat dari sisi pengurus?</h2>
					<p>Masuk untuk mencatat mustahik, menilai, menghitung, dan melaporkan penyaluran.</p>
				</div>
				<div class="baris-aksi">
					<a class="tombol emas besar" href="/login">Masuk pengurus</a>
					<a class="tombol hantu besar" href="/donasi">Salurkan donasi</a>
				</div>
			</div>
		</section>
	</main>

	<footer class="kaki">
		<div class="lebar kaki-isi">
			<div class="kaki-merek">
				<IkonMasjid ukuran={44} />
				<div>
					<p class="kaki-nama">{masjid.nama}</p>
					<p>Data pribadi (NIK, No. HP) dienkripsi dan tidak pernah ditampilkan utuh.</p>
				</div>
			</div>
			<p class="kaki-tautan"><a href="/">Beranda</a> · <a href="/donasi">Donasi</a> · <a href="/login">Masuk pengurus</a></p>
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
	.kecil {
		font-size: 0.92rem;
	}
	.catatan {
		font-size: 0.9rem;
		margin-top: 1rem;
	}

	/* ---- Tombol beraksen emas (dasar .tombol ada di app.css) ---- */
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
	.hantu:hover:not(:disabled) {
		background: rgb(255 255 255 / 0.1);
	}
	.hantu:disabled {
		opacity: 0.4;
		cursor: not-allowed;
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
		transform: translateY(1px) scale(0.98);
	}

	/* ---- Hero ---- */
	.hero {
		position: relative;
		overflow: hidden;
		color: #fff;
		background: linear-gradient(165deg, var(--hijau-pekat) 0%, var(--hijau-tua) 55%, #0f3328 100%);
		padding: 9rem 0 8rem;
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
		inset: -80px;
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
	.lead strong {
		color: #fff;
	}

	/* Panel kaca berisi pratinjau peringkat */
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
		translate: 0 1rem;
	}
	.panel-label {
		font-size: 0.8rem;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--emas-terang);
	}
	.panel-catatan {
		font-size: 0.8rem;
		color: rgb(255 255 255 / 0.6);
	}
	.mini {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.8rem;
	}
	.mini li {
		display: grid;
		grid-template-columns: 2rem 1fr auto 5.2rem;
		gap: 0.75rem;
		align-items: center;
	}
	.mini .alt {
		font-weight: 700;
	}
	.mini .nilai {
		font-size: 0.95rem;
		color: rgb(255 255 255 / 0.85);
	}
	.mini .jalur {
		height: 8px;
		border-radius: 99px;
		background: rgb(255 255 255 / 0.14);
		overflow: hidden;
	}
	.mini .isi {
		display: block;
		height: 100%;
		border-radius: 99px;
		background: var(--emas);
		transform-origin: left;
		transform: scaleX(var(--p));
		animation: tumbuh 1.2s cubic-bezier(0.16, 1, 0.3, 1) 0.5s both;
	}
	.cap {
		justify-self: end;
		font-size: 0.68rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		padding: 0.2rem 0.55rem;
		border-radius: 999px;
		background: rgb(255 255 255 / 0.12);
		color: rgb(255 255 255 / 0.75);
	}
	.cap.usul {
		background: var(--emas);
		color: #2b1d00;
	}

	/* ---- Tiga kartu ringkas menumpuk di bawah hero ---- */
	.ringkas-wadah {
		position: relative;
		z-index: 5;
		margin-top: -3.5rem;
	}
	.ringkas {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 1rem;
	}
	.kartu-mengambang {
		display: grid;
		gap: 0.5rem;
		align-content: start;
		padding: 1.5rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 20px;
		box-shadow: 0 24px 40px -28px rgb(12 46 37 / 0.5);
	}
	.kartu-mengambang h2 {
		font-size: 1.1rem;
	}
	.angka-besar {
		font-size: 2.2rem;
		font-weight: 700;
		line-height: 1;
		color: var(--emas);
	}

	/* ---- Bagian umum ---- */
	.bagian {
		padding-top: 4.5rem;
		padding-bottom: 1rem;
		scroll-margin-top: 5rem;
	}
	.dua-kolom {
		display: grid;
		grid-template-columns: minmax(0, 5fr) minmax(0, 7fr);
		gap: 3.5rem;
		align-items: start;
	}
	.dua-kolom.rata {
		grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
		gap: 2rem;
		margin-top: 2rem;
	}
	.kolom-teks {
		display: grid;
		gap: 1rem;
		position: sticky;
		top: 6rem;
	}
	.kolom-teks p {
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
		background: var(--emas);
	}
	.kepala-bagian {
		display: grid;
		gap: 0.75rem;
		margin-bottom: 2rem;
	}
	.kepala-bagian p {
		max-width: 62ch;
		line-height: 1.65;
	}
	.rumus {
		font-family: var(--mono);
		background: var(--hijau-muda);
		color: var(--hijau-tua);
		border-radius: 10px;
		padding: 0.55rem 0.8rem;
		font-size: 0.93rem;
		overflow-x: auto;
	}

	/* ---- Batang S bertumpuk ---- */
	.kartu-kaca {
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 24px;
		padding: 1.75rem;
		box-shadow: 0 24px 40px -32px rgb(12 46 37 / 0.45);
	}
	.timbang {
		margin: 0;
		display: grid;
		gap: 1rem;
	}
	.timbang figcaption {
		display: grid;
		gap: 0.2rem;
	}
	.legenda {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem 1rem;
		font-size: 0.82rem;
		color: var(--teks-redup);
	}
	.legenda li {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}
	.kotak {
		width: 0.8rem;
		height: 0.8rem;
		border-radius: 3px;
	}
	.batang-baris {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.65rem;
	}
	.batang-baris li {
		display: grid;
		grid-template-columns: 2rem 1fr 5.6rem;
		gap: 0.75rem;
		align-items: center;
	}
	.batang-baris li.ideal {
		padding-bottom: 0.65rem;
		border-bottom: 1px dashed var(--garis);
	}
	.alt {
		font-weight: 700;
	}
	.batang {
		display: block;
		height: 22px;
		background: var(--abu);
		border-radius: 8px;
		overflow: hidden;
	}
	.isi-batang {
		display: flex;
		height: 100%;
		transform-origin: left;
		animation: tumbuh 1s cubic-bezier(0.16, 1, 0.3, 1) both;
	}
	.segmen {
		display: block;
		height: 100%;
	}
	.nilai-k {
		font-size: 0.88rem;
		text-align: right;
		color: var(--teks-redup);
	}

	/* ---- Alur ---- */
	.alur {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 1rem;
	}
	.kartu-alur {
		display: grid;
		gap: 0.55rem;
		align-content: start;
		padding: 1.4rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 18px;
		transition:
			transform 0.35s cubic-bezier(0.16, 1, 0.3, 1),
			box-shadow 0.35s;
	}
	.kartu-alur:hover {
		transform: translateY(-4px);
		box-shadow: 0 22px 36px -26px rgb(12 46 37 / 0.55);
	}
	.kepala-alur {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	.nomor {
		width: 2.2rem;
		height: 2.2rem;
		display: grid;
		place-items: center;
		border-radius: 50%;
		background: var(--hijau);
		color: #fff;
		font-weight: 700;
	}
	.peran {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		padding: 0.22rem 0.65rem;
		border-radius: 999px;
		background: var(--hijau-muda);
		color: var(--hijau-tua);
	}
	.peran.ketua {
		background: var(--kuning);
		color: var(--kuning-teks);
	}
	.peran.sistem {
		background: var(--abu);
		color: #4a4945;
	}
	.kartu-alur h3 {
		font-size: 1.08rem;
	}
	.kartu-alur p {
		line-height: 1.6;
		font-size: 0.95rem;
	}

	/* ---- Kriteria ---- */
	.kriteria-kisi {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
		gap: 1rem;
	}
	.kartu-kriteria {
		display: grid;
		gap: 0.5rem;
		align-content: start;
		padding: 1.25rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 18px;
	}
	.kartu-kriteria h3 {
		font-size: 1rem;
	}
	.atas-kriteria {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	.kode {
		font-weight: 700;
		font-size: 1.05rem;
		color: var(--hijau);
	}
	.bobot {
		display: grid;
		grid-template-columns: auto 1fr auto;
		gap: 0.6rem;
		align-items: center;
		margin-top: 0.4rem;
	}
	.bobot-label {
		font-size: 0.8rem;
		color: var(--teks-redup);
	}
	.bobot-jalur {
		height: 8px;
		border-radius: 99px;
		background: var(--abu);
		overflow: hidden;
	}
	.bobot-jalur span {
		display: block;
		height: 100%;
		background: var(--emas);
		border-radius: 99px;
	}
	.bobot-nilai {
		font-weight: 600;
	}
	.penjelas {
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 20px;
		padding: 1.5rem;
		display: grid;
		gap: 0.9rem;
		align-content: start;
	}
	.arah {
		display: grid;
		gap: 0.8rem;
	}
	.arah-kartu {
		display: grid;
		grid-template-columns: 2.6rem 1fr;
		gap: 0.9rem;
		align-items: start;
	}
	.panah {
		width: 2.6rem;
		height: 2.6rem;
		display: grid;
		place-items: center;
		border-radius: 12px;
		font-size: 1.4rem;
		font-weight: 700;
	}
	.naik-panah {
		background: var(--hijau-muda);
		color: var(--hijau);
	}
	.turun-panah {
		background: var(--kuning);
		color: var(--kuning-teks);
	}
	.skala th,
	.skala td {
		padding: 0.55rem 0.6rem;
		font-size: 0.9rem;
	}
	.tingkat {
		display: inline-grid;
		place-items: center;
		width: 1.9rem;
		height: 1.9rem;
		border-radius: 8px;
		font-weight: 700;
	}
	.t1 {
		background: #e9f3ed;
		color: var(--hijau-tua);
	}
	.t2 {
		background: #b8d6c5;
		color: var(--hijau-tua);
	}
	.t3 {
		background: #5f9a82;
		color: #fff;
	}
	.t4 {
		background: var(--hijau-tua);
		color: #fff;
	}

	/* ---- Bagian gelap: contoh hitung ---- */
	.bagian-gelap {
		margin-top: 4.5rem;
		padding: 4.5rem 0;
		scroll-margin-top: 3rem;
		color: #fff;
		background: linear-gradient(165deg, var(--hijau-pekat) 0%, var(--hijau-tua) 60%, #0f3328 100%);
	}
	.kepala-bagian.terang p {
		color: rgb(255 255 255 / 0.78);
	}
	.kepala-bagian.terang h2 {
		color: #fff;
	}
	.data-awal {
		background: rgb(255 255 255 / 0.07);
		border-color: rgb(255 255 255 / 0.16);
		box-shadow: none;
		display: grid;
		gap: 0.9rem;
		margin-bottom: 1.5rem;
	}
	.data-awal h3 {
		color: #fff;
	}
	.catatan-gelap {
		font-size: 0.9rem;
		color: rgb(255 255 255 / 0.7);
	}

	/* Tabel hitung (dipakai di panel terang dan kartu gelap) */
	.tabel-hitung th,
	.tabel-hitung td {
		padding: 0.55rem 0.7rem;
		font-size: 0.93rem;
		white-space: nowrap; /* "Rp 800.000" tidak boleh terpecah; tabel digulir mendatar di HP */
	}
	.tabel-hitung tr.ideal td {
		background: var(--hijau-muda);
	}
	.tabel-hitung tr.jumlah td {
		border-top: 2px solid var(--garis);
	}
	.tabel-hitung td.sorot {
		background: #fdf3da;
		color: #5a3b00;
	}
	.tabel-hitung.sempit {
		max-width: 28rem;
	}
	.tabel-hitung.gelap th {
		color: rgb(255 255 255 / 0.7);
		border-bottom-color: rgb(255 255 255 / 0.2);
	}
	.tabel-hitung.gelap td {
		border-bottom-color: rgb(255 255 255 / 0.12);
		color: #fff;
	}

	/* Stepper */
	.stepper {
		display: grid;
		gap: 0.9rem;
	}
	.tab-baris {
		display: grid;
		grid-template-columns: repeat(6, minmax(0, 1fr));
		gap: 0.5rem;
	}
	.tab {
		display: grid;
		justify-items: center;
		gap: 0.35rem;
		padding: 0.7rem 0.4rem;
		border-radius: 14px;
		border: 1px solid rgb(255 255 255 / 0.16);
		background: rgb(255 255 255 / 0.06);
		color: rgb(255 255 255 / 0.8);
		font: inherit;
		cursor: pointer;
		transition:
			background-color 0.2s,
			border-color 0.2s,
			transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
	}
	.tab:hover {
		background: rgb(255 255 255 / 0.12);
	}
	.tab:focus-visible {
		outline: 2px solid var(--emas-terang);
		outline-offset: 2px;
	}
	.tab-no {
		width: 1.8rem;
		height: 1.8rem;
		display: grid;
		place-items: center;
		border-radius: 50%;
		background: rgb(255 255 255 / 0.14);
		font-weight: 700;
	}
	.tab-nama {
		font-size: 0.85rem;
		text-align: center;
		line-height: 1.2;
	}
	.tab.lewat .tab-no {
		background: rgb(240 207 134 / 0.3);
		color: var(--emas-terang);
	}
	.tab.aktif {
		background: var(--emas);
		border-color: var(--emas);
		color: #2b1d00;
		font-weight: 600;
	}
	.tab.aktif .tab-no {
		background: #2b1d00;
		color: var(--emas-terang);
	}
	.progres {
		height: 4px;
		border-radius: 99px;
		background: rgb(255 255 255 / 0.14);
		overflow: hidden;
	}
	.progres span {
		display: block;
		height: 100%;
		background: var(--emas);
		transition: width 0.4s cubic-bezier(0.16, 1, 0.3, 1);
	}
	.panel-langkah {
		display: grid;
		gap: 1rem;
		padding: 1.75rem;
		border-radius: 24px;
		background: var(--kartu);
		color: var(--teks);
		box-shadow: 0 32px 56px -32px rgb(0 0 0 / 0.55);
		animation: naik 0.5s cubic-bezier(0.16, 1, 0.3, 1) both;
	}
	.langkah-label {
		font-size: 0.78rem;
		font-weight: 700;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--kuning-teks);
	}
	.panel-langkah h3 {
		font-size: 1.35rem;
	}
	.sehari {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.8rem;
		align-items: start;
		padding: 0.9rem 1rem;
		border-radius: 14px;
		background: #fdf6e6;
		border-left: 4px solid var(--emas);
	}
	.sehari p {
		line-height: 1.6;
	}
	.chip {
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		padding: 0.25rem 0.6rem;
		border-radius: 999px;
		background: var(--emas);
		color: #2b1d00;
		white-space: nowrap;
		justify-self: start;
	}
	.contoh {
		line-height: 1.65;
	}
	.navigasi-langkah {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		flex-wrap: wrap;
		padding-top: 0.5rem;
	}
	.panel-langkah .hantu {
		color: var(--hijau-tua);
		border-color: var(--garis);
	}
	.panel-langkah .hantu:hover:not(:disabled) {
		background: var(--abu);
	}

	/* ---- Hasil ---- */
	.hasil-kolom {
		display: grid;
		gap: 1.1rem;
	}
	.kartu-kaca-terang {
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 24px;
		padding: 1.5rem;
		box-shadow: 0 24px 40px -32px rgb(12 46 37 / 0.45);
	}
	.ranking {
		list-style: none;
		margin: 0;
		display: grid;
		gap: 0.7rem;
	}
	.ranking li {
		display: grid;
		grid-template-columns: 1.5rem 2.5rem 1fr auto 6.5rem;
		gap: 0.75rem;
		align-items: center;
	}
	.ranking li.batas {
		display: block;
		text-align: center;
		position: relative;
		margin: 0.2rem 0;
		font-size: 0.78rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--kuning-teks);
	}
	.ranking li.batas::before {
		content: '';
		position: absolute;
		left: 0;
		right: 0;
		top: 50%;
		border-top: 2px dashed var(--kuning-garis);
	}
	.ranking li.batas span {
		position: relative;
		background: var(--kartu);
		padding: 0 0.75rem;
	}
	.ranking .badge {
		justify-self: start;
	}
	.no {
		color: var(--teks-redup);
	}
	.bar {
		height: 10px;
		background: var(--abu);
		border-radius: 99px;
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--hijau);
		border-radius: 99px;
	}
	.ranking li.usul .bar span {
		background: linear-gradient(90deg, var(--hijau-tua), var(--hijau-aktif));
	}
	.alasan {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 0.7rem;
	}
	.alasan li {
		display: grid;
		grid-template-columns: 2.6rem 1fr;
		gap: 0.9rem;
		align-items: start;
		padding: 0.9rem 1rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 14px;
	}
	.alasan p {
		font-size: 0.93rem;
		line-height: 1.55;
	}
	.id-alasan {
		width: 2.6rem;
		height: 2.6rem;
		display: grid;
		place-items: center;
		border-radius: 12px;
		background: var(--hijau-muda);
		color: var(--hijau-tua);
		font-weight: 700;
	}

	/* ---- Aturan ---- */
	.aturan-kisi {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 1rem;
	}
	.kartu-aturan {
		display: grid;
		gap: 0.5rem;
		padding: 1.5rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 18px;
		align-content: start;
	}
	.kartu-aturan p {
		line-height: 1.6;
	}
	.no-aturan {
		font-weight: 700;
		color: var(--emas);
		font-variant-numeric: tabular-nums;
	}

	/* ---- Tanya jawab ---- */
	.akordion {
		display: grid;
		gap: 0.7rem;
	}
	.akordion details {
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 16px;
		padding: 0 1.25rem;
		transition: border-color 0.2s;
	}
	.akordion details[open] {
		border-color: var(--hijau-aktif);
	}
	.akordion summary {
		cursor: pointer;
		font-weight: 600;
		padding: 1.05rem 0;
		list-style: none;
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		align-items: center;
	}
	.akordion summary::-webkit-details-marker {
		display: none;
	}
	.akordion summary::after {
		content: '+';
		font-size: 1.4rem;
		line-height: 1;
		color: var(--hijau);
		transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
	}
	.akordion details[open] summary::after {
		transform: rotate(45deg);
	}
	.akordion details p {
		padding-bottom: 1.1rem;
		line-height: 1.65;
	}

	/* ---- Seruan ---- */
	.seruan {
		padding-top: 4.5rem;
		padding-bottom: 4.5rem;
	}
	.seruan-isi {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1.5rem;
		flex-wrap: wrap;
		padding: 2.5rem;
		border-radius: 28px;
		color: #fff;
		background: linear-gradient(165deg, var(--hijau-pekat) 0%, var(--hijau-tua) 100%);
		box-shadow: 0 32px 56px -32px rgb(12 46 37 / 0.6);
	}
	.seruan-isi h2 {
		color: #fff;
	}
	.seruan-isi p {
		margin-top: 0.4rem;
		color: rgb(255 255 255 / 0.78);
	}

	/* ---- Footer ---- */
	.kaki {
		background: var(--hijau-pekat);
		color: rgb(255 255 255 / 0.75);
		font-size: 0.9rem;
	}
	.kaki-isi {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 1.5rem;
		flex-wrap: wrap;
		padding-top: 2.5rem;
		padding-bottom: 2.5rem;
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
		.panel-langkah,
		.isi-batang,
		.mini .isi,
		.pola {
			animation: none;
		}
		.kartu-alur,
		.tombol,
		.tab,
		.progres span,
		.akordion summary::after {
			transition: none;
		}
		.kartu-alur:hover {
			transform: none;
		}
	}

	/* ---- Layar sempit ---- */
	@media (max-width: 960px) {
		.alur {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}
	@media (max-width: 860px) {
		.hero {
			padding: 7rem 0 6rem;
		}
		.pahlawan,
		.dua-kolom,
		.dua-kolom.rata {
			grid-template-columns: minmax(0, 1fr);
			gap: 2rem;
		}
		.panel {
			translate: none;
			padding: 1.5rem;
		}
		.kolom-teks {
			position: static;
		}
		.ringkas,
		.aturan-kisi {
			grid-template-columns: minmax(0, 1fr);
		}
		.bagian {
			padding-top: 3.25rem;
		}
		.tab-baris {
			grid-template-columns: repeat(3, minmax(0, 1fr));
		}
		.seruan-isi {
			padding: 1.75rem;
		}
	}
	@media (max-width: 560px) {
		.alur {
			grid-template-columns: minmax(0, 1fr);
		}
		.sehari {
			grid-template-columns: minmax(0, 1fr);
		}
		.ranking li {
			grid-template-columns: 1.3rem 2.2rem 1fr auto;
		}
		.ranking li .badge {
			grid-column: 2 / -1;
		}
		.batang-baris li {
			grid-template-columns: 1.8rem 1fr 4.6rem;
		}
		.panel-langkah {
			padding: 1.25rem;
		}
	}
</style>
