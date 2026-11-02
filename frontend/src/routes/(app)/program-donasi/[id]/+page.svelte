<!--
  Kelola satu program donasi. Atas: ringkasan dana + tautan ke halaman publik,
  daftar donasi program, dan catat donasi tunai. Bawah: gambar sampul, ubah
  isian, dan kabar terbaru (Admin). Ketua melihat ringkasan dan kabar saja.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '@sveltejs/kit';
	import FormProgram from '$lib/components/FormProgram.svelte';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { labelStatusProgram, rupiah, sisaWaktu, tanggalPanjang, tanggalSaja, warnaStatusProgram } from '$lib/format';
	import type { NilaiProgram } from '$lib/program';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const p = $derived(data.program);
	const admin = $derived(data.user.role === 'admin');
	let mengirim = $state('');

	/** Gabungan bentuk hasil semua action halaman ini. */
	type HasilForm = {
		bagian?: string;
		sukses?: string;
		error?: string;
		fields?: Record<string, string>;
		nilai?: NilaiProgram;
		nilaiKabar?: { judul: string; isi: string };
	};
	const hasil = $derived(form as HasilForm | null);
	// Pesan dan galat hanya untuk bagian formulir yang baru dikirim.
	const untuk = (bagian: string) => (hasil?.bagian === bagian ? hasil : null);
	const galatDari = (bagian: string) => (f: string) => untuk(bagian)?.fields?.[f] ?? '';
	const nilaiUbah = $derived<NilaiProgram>(
		untuk('ubah')?.nilai ?? {
			judul: p.judul,
			ringkasan: p.ringkasan,
			cerita: p.cerita,
			target: String(p.target),
			batas_waktu: p.batas_waktu,
			status: p.status
		}
	);
	const nilaiKabar = $derived(untuk('kabar')?.nilaiKabar ?? { judul: '', isi: '' });

	/** enhance yang menandai bagian mana sedang dikirim; unggah & kabar dikosongkan setelah berhasil. */
	const kirim =
		(bagian: string): SubmitFunction =>
		() => {
			mengirim = bagian;
			return async ({ result, update }) => {
				await update({ reset: bagian !== 'ubah' && result.type === 'success' });
				mengirim = '';
			};
		};
</script>

<svelte:head><title>{p.judul} · Program donasi</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/program-donasi">← Program donasi</a>

	<div class="kartu-kepala">
		<div>
			<h1>{p.judul}</h1>
			<p class="redup">{p.ringkasan}</p>
		</div>
		<span class="badge {warnaStatusProgram[p.status]}">{labelStatusProgram[p.status]}</span>
	</div>

	{#if data.pesan}<p class="kotak-sukses" role="status">{data.pesan}</p>{/if}

	<section class="kartu ringkas">
		<div>
			<span class="label">Terkumpul</span>
			<strong class="besar">{rupiah(p.terkumpul)}</strong>
			<small class="redup">dari target {rupiah(p.target)} ({p.persen}%)</small>
			<div class="bar"><span style:width="{p.persen}%"></span></div>
		</div>
		<dl>
			<div><dt>Donatur</dt><dd>{p.donatur}</dd></div>
			<div><dt>Batas waktu</dt><dd>{p.batas_waktu ? tanggalSaja(p.batas_waktu) : 'Tanpa batas'}</dd></div>
			<div><dt>Sisa</dt><dd>{p.status === 'draf' ? '-' : sisaWaktu(p)}</dd></div>
		</dl>
		<div class="baris-aksi">
			{#if p.status !== 'draf'}<a class="tombol" href="/program/{p.slug}" target="_blank" rel="noopener">Lihat halaman publik ↗</a>{/if}
			<a class="tombol" href="/penerimaan?program={p.id}">Donasi program ini</a>
			{#if admin && p.status !== 'draf'}<a class="tombol" href="/penerimaan/baru?program={p.id}">+ Catat donasi tunai</a>{/if}
		</div>
		{#if p.status === 'draf'}<p class="redup kecil">Program masih draf: belum tampil di halaman publik dan belum bisa menerima donasi.</p>{/if}
	</section>

	<section class="kartu tumpuk">
		<h2>Gambar sampul</h2>
		<div class="sampul">
			{#if p.ada_gambar}
				<img src="/program-donasi/{p.id}/gambar?v={p.gambar_versi}" alt="Gambar sampul {p.judul}" />
			{:else}
				<span class="tanpa-gambar" aria-hidden="true"><IkonMasjid ukuran={56} /></span>
			{/if}
		</div>
		{#if admin}
			<form method="POST" action="?/gambar" enctype="multipart/form-data" class="baris-unggah" use:enhance={kirim('gambar')}>
				{#if untuk('gambar')?.sukses}<p class="kotak-sukses" role="status">{untuk('gambar')?.sukses}</p>{/if}
				<div class="field">
					<label for="gambar">{p.ada_gambar ? 'Ganti gambar' : 'Unggah gambar'}</label>
					<input id="gambar" name="gambar" type="file" accept="image/jpeg,image/png,image/webp" aria-invalid={Boolean(galatDari('gambar')('gambar'))} />
					<small>JPG, PNG, atau WebP, maksimal 2 MB. Rasio 16:9 (mis. 1280×720) paling pas.</small>
					{#if galatDari('gambar')('gambar')}<span class="galat" role="alert">{galatDari('gambar')('gambar')}</span>{/if}
				</div>
				<button class="tombol utama" type="submit" disabled={mengirim === 'gambar'}>{mengirim === 'gambar' ? 'Mengunggah…' : 'Unggah'}</button>
			</form>
		{/if}
	</section>

	{#if admin}
		<form method="POST" action="?/ubah" class="kartu tumpuk" novalidate use:enhance={kirim('ubah')}>
			<h2>Ubah program</h2>
			{#if untuk('ubah')?.sukses}<p class="kotak-sukses" role="status">{untuk('ubah')?.sukses}</p>{/if}
			{#if untuk('ubah')?.error && Object.keys(untuk('ubah')?.fields ?? {}).length === 0}<p class="kotak-galat" role="alert">{untuk('ubah')?.error}</p>{/if}
			<FormProgram nilai={nilaiUbah} galat={galatDari('ubah')} />
			<div class="baris-aksi ujung">
				<button class="tombol utama" type="submit" disabled={mengirim === 'ubah'}>{mengirim === 'ubah' ? 'Menyimpan…' : 'Simpan perubahan'}</button>
			</div>
		</form>
	{/if}

	<section class="kartu tumpuk">
		<h2>Kabar terbaru</h2>
		<p class="redup">Laporkan perkembangan dan penyaluran dana. Kabar langsung tampil di halaman publik program.</p>
		{#if admin}
			<form method="POST" action="?/kabar" class="tumpuk kabar-baru" novalidate use:enhance={kirim('kabar')}>
				{#if untuk('kabar')?.sukses}<p class="kotak-sukses" role="status">{untuk('kabar')?.sukses}</p>{/if}
				<div class="field">
					<label for="judul-kabar">Judul kabar</label>
					<input id="judul-kabar" name="judul" type="text" maxlength="120" placeholder="mis. Material atap sudah dibeli" value={nilaiKabar.judul} aria-invalid={Boolean(galatDari('kabar')('judul'))} />
					{#if galatDari('kabar')('judul')}<span class="galat" role="alert">{galatDari('kabar')('judul')}</span>{/if}
				</div>
				<div class="field">
					<label for="isi-kabar">Isi kabar</label>
					<textarea id="isi-kabar" name="isi" rows="4" maxlength="5000" aria-invalid={Boolean(galatDari('kabar')('isi'))}>{nilaiKabar.isi}</textarea>
					{#if galatDari('kabar')('isi')}<span class="galat" role="alert">{galatDari('kabar')('isi')}</span>{/if}
				</div>
				<div class="baris-aksi ujung">
					<button class="tombol utama" type="submit" disabled={mengirim === 'kabar'}>{mengirim === 'kabar' ? 'Menyimpan…' : 'Tambah kabar'}</button>
				</div>
			</form>
		{/if}

		{#if p.kabar.length === 0}
			<p class="redup">Belum ada kabar.</p>
		{:else}
			<ul class="daftar-kabar">
				{#each p.kabar as k (k.id)}
					<li>
						<div>
							<small class="redup">{tanggalPanjang(k.dibuat_pada)}</small>
							<h3>{k.judul}</h3>
							<p class="isi-kabar">{k.isi}</p>
						</div>
						{#if admin}
							<form
								method="POST"
								action="?/hapusKabar"
								onsubmit={(e) => {
									if (!confirm('Hapus kabar ini dari halaman publik?')) e.preventDefault();
								}}
							>
								<input type="hidden" name="kabar" value={k.id} />
								<button class="tombol kecil bahaya" type="submit">Hapus</button>
							</form>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</section>
</div>

<style>
	.sempit {
		max-width: 860px;
	}
	.ringkas {
		display: grid;
		gap: 1rem;
	}
	.label {
		display: block;
		font-size: 0.85rem;
		color: var(--teks-redup);
	}
	.besar {
		display: block;
		font-size: 1.6rem;
		color: var(--hijau);
	}
	.bar {
		height: 10px;
		border-radius: 999px;
		background: var(--hijau-muda);
		overflow: hidden;
		margin-top: 0.5rem;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--hijau);
	}
	dl {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 1rem;
		margin: 0;
	}
	dt {
		font-size: 0.8rem;
		color: var(--teks-redup);
	}
	dd {
		margin: 0;
		font-weight: 700;
	}
	.kecil {
		font-size: 0.85rem;
	}
	.sampul {
		position: relative;
		aspect-ratio: 16 / 9;
		max-width: 520px;
		border-radius: 14px;
		overflow: hidden;
		background: linear-gradient(160deg, var(--hijau) 0%, var(--hijau-tua) 100%);
	}
	.sampul img {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.tanpa-gambar {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
		color: #fff;
		opacity: 0.85;
	}
	.baris-unggah {
		display: grid;
		gap: 0.75rem;
		justify-items: start;
	}
	textarea {
		resize: vertical;
	}
	.kabar-baru {
		padding-bottom: 1rem;
		border-bottom: 1px solid var(--garis);
	}
	.daftar-kabar {
		list-style: none;
		margin: 0;
		padding: 0;
		display: grid;
		gap: 1rem;
	}
	.daftar-kabar li {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid var(--garis);
	}
	.isi-kabar {
		white-space: pre-line;
		margin-top: 0.3rem;
	}
	@media (max-width: 560px) {
		dl {
			grid-template-columns: 1fr 1fr;
		}
	}
</style>
