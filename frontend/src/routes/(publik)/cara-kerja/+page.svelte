<!--
  Cara kerja ARAS: penjelasan lengkap untuk awam (pengunjung publik dan pengurus baru).
  Isinya STATIS (bukan data database): satu contoh lima calon sintetis dihitung langkah demi langkah.
  Semua angka tabel dihitung dari $lib/aras-contoh (dijaga aras-contoh.test.ts), jadi tidak ada angka
  ketikan tangan yang bisa menyimpang dari rumusnya. Perhitungan resmi tetap di backend Go.
-->
<script lang="ts">
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

	const skala = [
		['1', 'Layak, milik sendiri', 'Sehat', 'Tetap'],
		['2', 'Layak, sewa / menumpang', 'Sakit ringan', 'Serabutan teratur'],
		['3', 'Kurang layak', 'Sakit kronis / lansia', 'Serabutan tidak teratur'],
		['4', 'Tidak layak / tanpa tempat tinggal', 'Disabilitas berat / sakit parah', 'Tidak bekerja']
	];

	const alur = [
		['Buat periode penyaluran', 'Petugas (Admin) memilih jenis dana (zakat atau infak), lalu mengisi dana yang tersedia dan nominal untuk tiap penerima. Aplikasi langsung menghitung berapa orang yang bisa dibantu (kuota).'],
		['Pilih calon penerima', 'Petugas memilih siapa saja yang masuk daftar calon. Pada periode zakat, hanya golongan asnaf yang diizinkan yang bisa dipilih, supaya dana zakat tidak salah sasaran.'],
		['Isi penilaian', 'Petugas mengisi nilai setiap calon pada setiap kriteria. Aplikasi menolak nilai yang di luar rentang dan menampilkan keterangan skala agar semua petugas memakai arti yang sama.'],
		['Hitung dengan ARAS', 'Aplikasi menghitung peringkat. Hasilnya disimpan sebagai satu versi (riwayat tidak hilang), dan sejumlah calon teratas ditandai sebagai usulan penerima.'],
		['Persetujuan Ketua DKM', 'Ketua DKM menyetujui atau menolak usulan. Penolakan wajib disertai catatan. Petugas dapat membuka kembali periode, memperbaiki penilaian, lalu menghitung ulang; hasil lama tetap tersimpan.'],
		['Penyaluran dan laporan', 'Setelah disetujui, petugas mencatat penyaluran per penerima (totalnya tidak boleh melebihi dana), menutup periode, lalu mencetak laporan.']
	];

	const alasan = [
		['A2', 'peringkat 1', 'penghasilannya paling kecil, rumahnya tidak layak, dan tidak bekerja. Hampir semua kriteria mengarah ke "sangat membutuhkan".'],
		['A3', 'peringkat terakhir', 'penghasilannya paling besar, sehat, dan rumahnya layak. Wajar bila paling akhir.'],
		['A5', 'di bawah A1 dan A4', 'tanggungannya paling banyak, tetapi penghasilannya relatif tinggi dan rumahnya layak. Satu kriteria yang tinggi tidak otomatis membuat calon naik ke atas.']
	];
</script>

{#snippet matriks(baris: typeof hasil.baris, ambil: (b: (typeof hasil.baris)[number]) => number[], fmt: (x: number, j: number) => string, kepala: string[], bawah?: string[])}
	<div class="tabel-wadah">
		<table class="angka-tabel">
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

<div class="tumpuk">
	<div>
		<a class="redup" href="/">← Beranda</a>
		<h1>Cara kerja ARAS</h1>
		<p class="redup lead">
			Halaman ini menjelaskan, dengan bahasa sehari-hari dan satu contoh lengkap, bagaimana aplikasi mengurutkan calon penerima zakat dan
			infak di {masjid.nama}. Semua angka di sini adalah <strong>contoh rekaan</strong>: bukan data warga dan bukan keputusan pengurus.
		</p>
	</div>

	<section class="kartu ringkas">
		<h2>Ringkasnya</h2>
		<ul>
			<li>Setiap calon dinilai dengan <strong>kriteria yang sama</strong>: penghasilan, jumlah tanggungan, kondisi tempat tinggal, kesehatan, dan pekerjaan.</li>
			<li>
				ARAS menggabungkan semua nilai itu menjadi <strong>satu angka antara 0 dan 1</strong> (disebut derajat utilitas, atau <strong>K</strong>). Makin dekat ke 1, makin layak calon itu dibantu.
			</li>
			<li>
				Angka itu hanya <strong>usulan urutan</strong>. Yang menetapkan penerima tetap manusia: usulan baru sah setelah disetujui Ketua DKM.
			</li>
		</ul>
	</section>

	<section class="kartu">
		<h2>Kenapa perlu metode khusus?</h2>
		<p>
			Dana zakat dan infak terbatas, sedangkan calon penerima banyak dan keadaannya berbeda-beda. Mana yang lebih perlu dibantu: orang
			berpenghasilan rendah tetapi tanggungannya sedikit, atau orang berpenghasilan sedang tetapi tanggungannya banyak? Kalau hanya
			mengandalkan ingatan dan kesan, jawabannya bisa berbeda antarpengurus atau antarwaktu, dan sulit dijelaskan ketika ada yang bertanya
			"mengapa A menerima, sedangkan B tidak".
		</p>
		<p>
			Dengan ARAS, aturan mainnya tertulis dan sama untuk semua calon. Setiap langkah hitungnya juga dapat dibuka dan diperiksa ulang oleh
			pengurus, sehingga keputusan bisa dipertanggungjawabkan kepada jamaah dan donatur.
		</p>
	</section>

	<section class="kartu">
		<h2>Gagasan dasarnya: dibandingkan dengan "calon ideal"</h2>
		<p>
			Bayangkan kita membuat satu calon rekaan yang disebut <strong>A₀ (calon ideal)</strong>. Pada tiap kriteria, A₀ memakai nilai yang paling
			menggambarkan kebutuhan di antara semua calon: penghasilan paling kecil, tanggungan paling banyak, rumah paling tidak layak, dan
			seterusnya. A₀ bukan orang sungguhan; ia hanya patokan.
		</p>
		<p>
			Setelah itu, setiap calon sungguhan diukur: <em>seberapa dekat keadaannya dengan A₀?</em> Hasilnya adalah angka K. Kira-kira,
			K = 0,89 berarti keadaan calon itu sudah mencapai sekitar 89% dari "calon ideal". Calon dengan K terbesar diurutkan paling atas.
		</p>
	</section>

	<section class="kartu">
		<h2>Alur lengkap di aplikasi</h2>
		<p class="redup">Perhitungan ARAS hanya satu bagian dari proses. Dari awal sampai akhir, urutannya sebagai berikut.</p>
		<ol class="alur">
			{#each alur as [judul, isi], i (judul)}
				<li>
					<span class="nomor">{i + 1}</span>
					<div>
						<strong>{judul}</strong>
						<p class="redup">{isi}</p>
					</div>
				</li>
			{/each}
		</ol>
	</section>

	<section class="kartu">
		<h2>Kriteria dan bobot</h2>
		<p>
			<strong>Kriteria</strong> adalah hal-hal yang dinilai pada setiap calon. <strong>Bobot</strong> menunjukkan seberapa penting satu kriteria
			dibanding yang lain; jumlah seluruh bobot harus tepat 1,00 (100%). Tabel berikut adalah contoh yang dipakai di halaman ini.
		</p>
		<div class="tabel-wadah">
			<table>
				<thead><tr><th>Kode</th><th>Kriteria</th><th>Tipe</th><th class="kanan">Bobot</th><th>Cara mengisi</th></tr></thead>
				<tbody>
					{#each kriteriaContoh as k (k.kode)}
						<tr>
							<td><strong>{k.kode}</strong></td>
							<td>{k.nama}</td>
							<td><span class="badge {k.tipe === 'cost' ? 'kuning' : 'hijau'}">{k.tipe}</span></td>
							<td class="kanan angka">{desimal(k.bobot, 2)}</td>
							<td>{k.satuan}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		<h3>Apa itu benefit dan cost?</h3>
		<ul>
			<li><strong>Benefit</strong>: makin <em>besar</em> nilainya, makin layak dibantu. Contoh: tanggungan 6 orang lebih membutuhkan daripada 2 orang.</li>
			<li><strong>Cost</strong>: makin <em>kecil</em> nilainya, makin layak dibantu. Contoh: penghasilan Rp500.000 lebih membutuhkan daripada Rp1.500.000.</li>
		</ul>

		<h3>Skala 1 sampai 4</h3>
		<p>
			Tiga kriteria tidak berupa angka alami, jadi petugas memilih tingkat 1 sampai 4. Angka 4 sengaja berarti <strong>paling membutuhkan</strong>,
			sehingga ketiganya searah dan sama-sama bertipe benefit.
		</p>
		<div class="tabel-wadah">
			<table>
				<thead><tr><th>Nilai</th><th>Tempat tinggal (C3)</th><th>Kesehatan (C4)</th><th>Pekerjaan (C5)</th></tr></thead>
				<tbody>
					{#each skala as baris (baris[0])}
						<tr>
							<td><strong>{baris[0]}</strong></td>
							{#each baris.slice(1) as sel (sel)}<td>{sel}</td>{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		<p class="redup catatan">
			Bobot dan arti skala di atas hanya contoh. Yang dipakai sebenarnya ditetapkan pengurus masjid lewat menu Kriteria, dan setiap
			perubahan tercatat di riwayat aktivitas.
		</p>
	</section>

	<section class="kartu">
		<h2>Contoh: lima calon</h2>
		<p>
			Kita pakai lima calon rekaan (A1 sampai A5). Berikut nilai yang diisi petugas pada tiap kriteria. Perhatikan bahwa tidak ada calon yang
			unggul di semua kriteria, sehingga perlu cara yang adil untuk menggabungkannya.
		</p>
		<div class="tabel-wadah">
			<table class="angka-tabel">
				<thead>
					<tr><th>Calon</th>{#each kriteriaContoh as k (k.kode)}<th class="kanan">{k.kode}</th>{/each}</tr>
				</thead>
				<tbody>
					{#each alternatifContoh as a (a.id)}
						<tr>
							<td><strong>{a.id}</strong></td>
							{#each a.nilai as v, j (j)}
								<td class="kanan angka">{j === 0 ? rupiah(v) : angka(v)}</td>
							{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</section>

	<section class="langkah-wadah">
		<h2>Enam langkah hitung, satu per satu</h2>
		<p class="redup">
			Berikut apa yang dilakukan aplikasi, dengan angka dari contoh di atas. Tampilan dibulatkan sampai empat desimal, sedangkan
			perhitungan sebenarnya memakai angka penuh, jadi menjumlah ulang dengan kalkulator bisa selisih sekitar 0,0001.
		</p>

		<article class="kartu langkah">
			<span class="badge hijau">Langkah 1</span>
			<h3>Tentukan calon ideal (A₀)</h3>
			<p><strong>Dengan kata sehari-hari:</strong> ambil nilai terbaik pada tiap kriteria dari semua calon, lalu jadikan satu "calon ideal".</p>
			<p class="rumus">Benefit → nilai terbesar · Cost → nilai terkecil</p>
			<p>
				<strong>Pada contoh:</strong> penghasilan terkecil adalah {rupiah(hasil.a0[0])} (milik A2), tanggungan terbanyak {angka(hasil.a0[1])} (A5),
				dan nilai tertinggi untuk tempat tinggal, kesehatan, dan pekerjaan masing-masing {angka(hasil.a0[2])}, {angka(hasil.a0[3])}, dan
				{angka(hasil.a0[4])}.
			</p>
		</article>

		<article class="kartu langkah">
			<span class="badge hijau">Langkah 2</span>
			<h3>Samakan arah: balik kriteria cost</h3>
			<p>
				<strong>Dengan kata sehari-hari:</strong> pada penghasilan, angka kecil justru bagus; pada kriteria lain, angka besar yang bagus.
				Supaya semuanya searah ("besar = lebih layak"), nilai penghasilan dibalik menjadi 1 dibagi nilainya.
			</p>
			<p class="rumus">x' = 1 ÷ x &nbsp;(hanya kriteria cost)</p>
			<p>
				<strong>Pada contoh:</strong> penghasilan A1 Rp800.000 menjadi 1 ÷ 800.000 = 0,00000125. Penghasilan terkecil (Rp500.000) menjadi angka
				terbesar, dan penghasilan terbesar (Rp1.500.000) menjadi angka terkecil. Kolom C1 ditulis dalam satuan ×10⁻⁶ agar mudah dibaca.
			</p>
			{@render matriks(hasil.baris, (b) => b.x, (v, j) => (j === 0 ? mikro(v) : angka(v)), ['C1 (×10⁻⁶)', 'C2', 'C3', 'C4', 'C5'], [mikro(hasil.jumlah[0]), ...hasil.jumlah.slice(1).map(angka)])}
		</article>

		<article class="kartu langkah">
			<span class="badge hijau">Langkah 3</span>
			<h3>Samakan skala: normalisasi</h3>
			<p>
				<strong>Dengan kata sehari-hari:</strong> rupiah, jumlah jiwa, dan skala 1–4 tidak bisa dijumlahkan begitu saja. Maka tiap angka
				dibagi jumlah seluruh angka di kolomnya (baris "Jumlah kolom" pada langkah 2), sehingga berubah menjadi <em>porsi</em> (antara 0 dan
				1) dari total kolom itu.
			</p>
			<p class="rumus">r = x ÷ Σx</p>
			<p>
				<strong>Pada contoh:</strong> tanggungan A1 adalah 5 dari jumlah kolom 26, jadi r = 5 ÷ 26 = {f4(a1.r[1])}.
			</p>
			{@render matriks(hasil.baris, (b) => b.r, (v) => f4(v), kriteriaContoh.map((k) => k.kode))}
		</article>

		<article class="kartu langkah">
			<span class="badge hijau">Langkah 4</span>
			<h3>Beri bobot</h3>
			<p>
				<strong>Dengan kata sehari-hari:</strong> kriteria yang lebih penting harus lebih berpengaruh. Setiap porsi dikalikan bobot kriterianya.
			</p>
			<p class="rumus">d = w × r</p>
			<p>
				<strong>Pada contoh:</strong> porsi penghasilan A1 {f4(a1.r[0])} dikali bobot {desimal(kriteriaContoh[0].bobot, 2)} menjadi {f4(a1.d[0])}.
			</p>
			{@render matriks(hasil.baris, (b) => b.d, (v) => f4(v), kriteriaContoh.map((k) => `${k.kode} (×${desimal(k.bobot, 2)})`))}
		</article>

		<article class="kartu langkah">
			<span class="badge hijau">Langkah 5</span>
			<h3>Jumlahkan: nilai optimalitas (S)</h3>
			<p>
				<strong>Dengan kata sehari-hari:</strong> jumlahkan semua nilai berbobot milik satu calon. Hasilnya satu angka yang merangkum seluruh kriteria.
			</p>
			<p class="rumus">S = d₁ + d₂ + d₃ + d₄ + d₅</p>
			<p>
				<strong>Pada contoh:</strong> S untuk A1 = {a1.d.map(f4).join(' + ')} = <strong>{f4(a1.s)}</strong>. Untuk calon ideal, S₀ = <strong>{f4(ideal.s)}</strong>.
			</p>
			<div class="tabel-wadah">
				<table class="angka-tabel">
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
		</article>

		<article class="kartu langkah">
			<span class="badge hijau">Langkah 6</span>
			<h3>Hitung derajat utilitas (K)</h3>
			<p>
				<strong>Dengan kata sehari-hari:</strong> bandingkan S tiap calon dengan S₀ milik calon ideal. Hasilnya K, yaitu seberapa dekat calon itu
				dengan keadaan ideal. Calon ideal sendiri selalu bernilai 1.
			</p>
			<p class="rumus">K = S ÷ S₀</p>
			<p><strong>Pada contoh:</strong> K untuk A1 = {f4(a1.s)} ÷ {f4(ideal.s)} = <strong>{f4(a1.k)}</strong>.</p>
			<div class="tabel-wadah">
				<table class="angka-tabel">
					<thead><tr><th>Calon</th><th class="kanan">S</th><th class="kanan">S₀</th><th class="kanan">K = S ÷ S₀</th></tr></thead>
					<tbody>
						{#each calon as b (b.id)}
							<tr>
								<td><strong>{b.id}</strong></td>
								<td class="kanan angka">{f4(b.s)}</td>
								<td class="kanan angka">{f4(ideal.s)}</td>
								<td class="kanan angka"><strong>{f4(b.k)}</strong></td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</article>
	</section>

	<section class="kartu">
		<h2>Hasil: peringkat dan jumlah penerima</h2>
		<p>
			Calon diurutkan dari K terbesar. Jumlah penerima (kuota) ditentukan oleh dana, bukan oleh pengurus secara bebas:
		</p>
		<p class="rumus">Kuota = dana tersedia ÷ nominal per penerima, dibulatkan ke bawah</p>
		<p>
			<strong>Pada contoh:</strong> dana {rupiah(dana)} dan nominal {rupiah(nominal)} per penerima, jadi kuota = {n} orang. Tiga calon teratas
			menjadi <strong>usulan penerima</strong>; sisanya <strong>cadangan</strong> menurut urutan peringkat. Pembulatan ke bawah menjaga
			agar total yang dibagikan tidak pernah melebihi dana.
		</p>
		<ol class="ranking">
			{#each hasil.ranking as c, i (c.id)}
				<li>
					<span class="no">{i + 1}</span>
					<span class="alt">{c.id}</span>
					<span class="bar" aria-hidden="true"><span style:width="{c.k * 100}%"></span></span>
					<span class="angka">{f4(c.k)}</span>
					<span class="badge kapital {i < n ? 'hijau' : ''}">{i < n ? 'Usulan' : 'Cadangan'}</span>
				</li>
			{/each}
		</ol>

		<h3>Apakah urutan ini masuk akal?</h3>
		<ul>
			{#each alasan as [id, posisi, sebab] (id)}
				<li><strong>{id}</strong> ({posisi}): {sebab}</li>
			{/each}
		</ul>
		<p>
			Inilah gunanya menggabungkan semua kriteria: tidak ada satu kriteria yang menentukan sendirian. Yang menentukan adalah gabungan
			semuanya, sesuai bobot yang disepakati.
		</p>
	</section>

	<section class="kartu">
		<h2>Aturan tambahan yang perlu diketahui</h2>
		<ul>
			<li>
				<strong>Penyaringan sebelum dihitung.</strong> Pada periode zakat, hanya calon dari golongan asnaf yang diizinkan yang ikut dihitung. Pada
				periode infak, semua mustahik aktif dapat dipilih.
			</li>
			<li>
				<strong>Hasil hanya usulan.</strong> Daftar penerima baru sah setelah disetujui Ketua DKM. Jika ditolak, alasan wajib dicatat dan
				penilaian dapat diperbaiki lalu dihitung ulang.
			</li>
			<li>
				<strong>Bila dua calon nilainya sama.</strong> Dua calon dianggap sama bila K-nya sama sampai empat desimal. Aplikasi lalu melihat
				nilai asli mulai dari kriteria berbobot terbesar: untuk cost, nilai yang lebih kecil menang; untuk benefit, nilai yang lebih besar
				menang. Bila masih sama, urutan ditentukan secara tetap sehingga hasilnya tidak berubah-ubah.
			</li>
			<li>
				<strong>Riwayat tidak hilang.</strong> Setiap kali dihitung, hasilnya disimpan sebagai versi baru lengkap dengan nama dan bobot
				kriteria saat itu, sehingga hasil lama tetap bisa ditelusuri walaupun bobot kemudian diubah.
			</li>
		</ul>
	</section>

	<section class="kartu">
		<h2>Pertanyaan yang sering muncul</h2>
		<details>
			<summary>Apakah aplikasi yang memutuskan siapa penerimanya?</summary>
			<p>
				Tidak. Aplikasi hanya menghitung dan mengusulkan urutan. Keputusan akhir ada pada pengurus: Ketua DKM menyetujui atau menolak, dan
				penolakan wajib disertai catatan.
			</p>
		</details>
		<details>
			<summary>Siapa yang menentukan kriteria dan bobot?</summary>
			<p>
				Pengurus masjid. Petugas dapat mengubah nama, tipe, bobot, dan arti skala, menambah kriteria baru, atau menonaktifkannya (tidak
				ada yang dihapus agar riwayat tetap utuh). Jumlah bobot semua kriteria aktif harus tepat 100%, dan aplikasi menolak jika tidak
				pas.
			</p>
		</details>
		<details>
			<summary>Mengapa satu calon yang sangat membutuhkan bisa tidak di peringkat pertama?</summary>
			<p>
				Karena yang dinilai adalah gabungan semua kriteria, bukan satu kriteria saja. Seperti contoh A5 di atas: tanggungannya paling banyak,
				tetapi kriteria lainnya tidak terlalu berat, sehingga gabungannya lebih rendah daripada A2.
			</p>
		</details>
		<details>
			<summary>Apakah data calon penerima tampil di halaman ini?</summary>
			<p>
				Tidak. Seluruh nama, angka, dan hasil di halaman ini adalah contoh rekaan. Data mustahik yang sebenarnya hanya dapat dilihat
				pengurus yang sudah masuk, dan NIK serta nomor HP disimpan terenkripsi.
			</p>
		</details>
		<details>
			<summary>Apakah angka contoh ini sama dengan yang dipakai masjid?</summary>
			<p>
				Tidak harus. Bobot, arti skala, dan jumlah dana pada halaman ini hanya untuk memperlihatkan cara hitungnya. Angka yang dipakai
				masjid ditetapkan pengurus.
			</p>
		</details>
	</section>

	<p class="redup tutup">
		<a href="/">← Kembali ke beranda</a> · <a href="/donasi">Salurkan donasi</a>
	</p>
</div>

<style>
	.lead {
		margin-top: 0.5rem;
		line-height: 1.6;
	}
	.kartu > p + p,
	.kartu > ul + p,
	.kartu > p + .tabel-wadah,
	.kartu > .tabel-wadah + p,
	.kartu > .tabel-wadah + h3,
	.kartu > ul + h3,
	.kartu > p + ol,
	.kartu > h2 + p {
		margin-top: 0.75rem;
	}
	.kartu h2 {
		margin-bottom: 0;
	}
	.kartu h3 {
		margin: 1.25rem 0 0;
		font-size: 1.02rem;
	}
	.kartu p,
	.kartu li {
		line-height: 1.65;
	}
	.kartu ul {
		margin: 0.75rem 0 0;
		padding-left: 1.25rem;
		display: grid;
		gap: 0.4rem;
	}
	.ringkas {
		background: var(--hijau-muda);
		border-color: transparent;
	}
	.ringkas ul {
		margin-top: 0.5rem;
	}

	.alur {
		list-style: none;
		padding: 0;
		margin: 1rem 0 0;
		display: grid;
		gap: 0.9rem;
	}
	.alur li {
		display: grid;
		grid-template-columns: 2rem 1fr;
		gap: 0.75rem;
		align-items: start;
	}
	.alur p {
		margin-top: 0.15rem;
	}
	.nomor {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: 50%;
		background: var(--hijau);
		color: #fff;
		font-weight: 700;
	}

	.langkah-wadah {
		display: grid;
		gap: 1rem;
	}
	.langkah-wadah > * {
		min-width: 0;
	}
	.langkah {
		display: grid;
		gap: 0.7rem;
		align-content: start;
	}
	.langkah .badge {
		justify-self: start;
	}
	.langkah h3 {
		margin: 0;
		font-size: 1.1rem;
	}
	.rumus {
		font-family: var(--mono);
		background: var(--abu);
		border-radius: 8px;
		padding: 0.5rem 0.75rem;
		font-size: 0.95rem;
		overflow-x: auto;
	}
	.angka-tabel td,
	.angka-tabel th {
		padding: 0.5rem 0.6rem;
	}
	.angka-tabel tr.ideal td {
		background: var(--hijau-muda);
	}
	.angka-tabel tr.jumlah td {
		border-top: 2px solid var(--garis);
	}
	.catatan {
		font-size: 0.9rem;
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
		/* Kolom badge dibuat tetap supaya panjang batang sama untuk Usulan dan Cadangan. */
		grid-template-columns: 1.5rem 2.5rem 1fr auto 6.5rem;
		gap: 0.75rem;
		align-items: center;
	}
	.no {
		color: var(--teks-redup);
	}
	.alt {
		font-weight: 700;
	}
	.ranking .badge {
		justify-self: start;
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

	details {
		border-top: 1px solid var(--garis);
		padding: 0.75rem 0;
	}
	details:first-of-type {
		margin-top: 0.75rem;
	}
	summary {
		cursor: pointer;
		font-weight: 600;
	}
	details p {
		margin-top: 0.5rem;
	}
	.tutup {
		text-align: center;
	}
</style>
