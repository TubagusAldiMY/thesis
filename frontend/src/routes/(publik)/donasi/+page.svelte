<!--
  Donasi zakat & infak (publik, tanpa login). Muzaki mengisi nama, jenis, dan
  jumlah; pembayaran (QRIS / virtual account / e-wallet) dilakukan di halaman
  payment gateway pada langkah berikutnya. Kuitansi terbit otomatis setelah
  pembayaran dikonfirmasi gateway.

  Tampilan: jenis donasi berupa pil (radio asli, jadi tetap jalan tanpa JS) dan
  jumlah punya tombol nominal cepat yang hanya mengisi kolom jumlah. Kontrak
  ke server tidak berubah: nama, jenis, jumlah, no_hp, keterangan, website.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { angka, labelJenisDonasi, parseAngka, semuaJenisDonasi } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);

	// Nominal cepat; hanya jalan pintas mengisi kolom jumlah (batas sebenarnya dijaga Go).
	const nominalCepat = [50_000, 100_000, 250_000, 500_000, 1_000_000];

	// Nilai awal dari isian yang dipantulkan saat gagal (No. HP sengaja tidak dipantulkan).
	// svelte-ignore state_referenced_locally
	let jumlah = $state(form?.nilai?.jumlah ?? '');
	// svelte-ignore state_referenced_locally
	let jenis = $state(form?.nilai?.jenis || 'zakat_fitrah');
	const nilai = $derived({ nama: form?.nilai?.nama ?? '', keterangan: form?.nilai?.keterangan ?? '' });

	const galat = (f: string) => form?.fields?.[f] ?? '';
	// Pesan umum (mis. 429 terlalu banyak permintaan, 502 gateway bermasalah) = error tanpa pesan per kolom.
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
	const jumlahAngka = $derived(parseAngka(jumlah));
</script>

<svelte:head><title>Donasi zakat & infak · {masjid.nama}</title></svelte:head>

<div class="halaman">
	<header class="kepala">
		<span class="lencana">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				<path d="M12 3l8 3v6c0 4.5-3.2 8-8 9-4.8-1-8-4.5-8-9V6z" /><path d="M8.5 12l2.5 2.5L15.5 9" />
			</svg>
			Dikelola pengurus masjid
		</span>
		<h1>Donasi Zakat & Infak</h1>
		<p class="redup">
			Salurkan zakat dan infak Anda melalui {masjid.nama}. Setelah pembayaran berhasil, kuitansi digital dapat langsung dilihat dan dicetak.
		</p>
		<ul class="jaminan">
			<li>
				<span class="ikon hijau" aria-hidden="true">
					<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="11" width="14" height="9" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /></svg>
				</span>
				Data terlindungi
			</li>
			<li>
				<span class="ikon kuning" aria-hidden="true">
					<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 3h8l4 4v14H7z" /><path d="M10 12h6M10 16h6" /></svg>
				</span>
				Kuitansi otomatis
			</li>
		</ul>
	</header>

	{#if !data.aktif}
		<p class="kotak-info" role="status">
			Donasi online belum diaktifkan oleh pengurus masjid. Silakan hubungi pengurus{masjid.kontak ? ` di ${masjid.kontak}` : ''} atau datang langsung ke masjid.
		</p>
	{:else}
		<form
			method="POST"
			class="kartu-form"
			novalidate
			use:enhance={() => {
				mengirim = true;
				return async ({ update }) => {
					await update({ reset: false });
					mengirim = false;
				};
			}}
		>
			{#if galatUmum}<p class="kotak-galat" role="alert">{galatUmum}</p>{/if}

			<div class="field">
				<label for="nama">Nama lengkap <span class="wajib">*</span></label>
				<input id="nama" name="nama" type="text" required maxlength="100" autocomplete="name" placeholder="mis. Hamba Allah / nama pribadi" value={nilai.nama} aria-invalid={Boolean(galat('nama'))} />
				<small>Tertulis di kuitansi. Boleh memakai nama keluarga atau "Hamba Allah".</small>
				{#if galat('nama')}<span class="galat" role="alert">{galat('nama')}</span>{/if}
			</div>

			<fieldset class="field">
				<legend>Jenis donasi <span class="wajib">*</span></legend>
				<div class="pil-baris">
					{#each semuaJenisDonasi as j (j)}
						<label class="pil">
							<input type="radio" name="jenis" value={j} bind:group={jenis} />
							<span>{labelJenisDonasi[j]}</span>
						</label>
					{/each}
				</div>
				{#if galat('jenis')}<span class="galat" role="alert">{galat('jenis')}</span>{/if}
			</fieldset>

			<fieldset class="field">
				<legend>Jumlah donasi (Rp) <span class="wajib">*</span></legend>
				<div class="nominal-baris">
					{#each nominalCepat as n (n)}
						<button type="button" class="nominal" class:aktif={jumlahAngka === n} aria-pressed={jumlahAngka === n} onclick={() => (jumlah = angka(n))}>
							Rp {angka(n)}
						</button>
					{/each}
				</div>
				<div class="rupiah" class:salah={Boolean(galat('jumlah'))}>
					<span aria-hidden="true">Rp</span>
					<input id="jumlah" name="jumlah" type="text" inputmode="numeric" autocomplete="off" required placeholder="mis. 100.000" aria-label="Jumlah donasi dalam rupiah" bind:value={jumlah} aria-invalid={Boolean(galat('jumlah'))} />
				</div>
				<small>Jumlah donasi Rp 10.000 sampai Rp 100.000.000.</small>
				{#if galat('jumlah')}<span class="galat" role="alert">{galat('jumlah')}</span>{/if}
			</fieldset>

			<div class="field">
				<label for="no_hp">No. HP / WhatsApp <span class="opsi">(opsional)</span></label>
				<input id="no_hp" name="no_hp" type="text" inputmode="tel" autocomplete="tel" maxlength="20" placeholder="mis. 081234567890" aria-invalid={Boolean(galat('no_hp'))} />
				<small>Hanya agar pengurus bisa mengirim konfirmasi. Disimpan terenkripsi.</small>
				{#if galat('no_hp')}<span class="galat" role="alert">{galat('no_hp')}</span>{/if}
			</div>

			<div class="field">
				<label for="keterangan">Pesan / doa <span class="opsi">(opsional)</span></label>
				<textarea id="keterangan" name="keterangan" rows="3" maxlength="200" placeholder="Tuliskan doa atau hajat Anda (mis. semoga berkah untuk keluarga)" aria-invalid={Boolean(galat('keterangan'))}>{nilai.keterangan}</textarea>
				{#if galat('keterangan')}<span class="galat" role="alert">{galat('keterangan')}</span>{/if}
			</div>

			<!-- Kolom jebakan anti-bot: tidak terlihat dan tidak bisa difokus oleh manusia. -->
			<div class="jebakan" aria-hidden="true">
				<label>Website <input name="website" type="text" tabindex="-1" autocomplete="off" /></label>
			</div>

			<div class="metode">
				<div class="metode-judul"><strong>Metode pembayaran</strong></div>
				<div class="metode-daftar">
					<span>QRIS</span><span>Virtual account</span><span>E-wallet</span>
				</div>
				<p class="redup kecil">
					Pilihan lengkap tersedia di halaman berikutnya. Pembayaran diproses penyedia pembayaran; aplikasi ini tidak menyimpan data kartu atau rekening Anda.
				</p>
			</div>

			<div class="aksi">
				<button class="tombol utama besar" type="submit" disabled={mengirim}>
					{mengirim ? 'Memproses…' : 'Lanjut ke pembayaran'}
					{#if !mengirim}<span aria-hidden="true">→</span>{/if}
				</button>
				<p class="catatan">Data pribadi hanya dipakai untuk pencatatan donasi dan kuitansi resmi.</p>
			</div>
		</form>
	{/if}
</div>

<style>
	.halaman {
		display: grid;
		gap: 1.25rem;
	}
	.kepala {
		display: grid;
		gap: 0.75rem;
	}
	.kepala h1 {
		margin: 0;
		font-size: clamp(1.75rem, 6vw, 2.25rem);
		letter-spacing: -0.02em;
		line-height: 1.15;
	}
	.kepala p {
		margin: 0;
		line-height: 1.55;
	}
	.lencana {
		justify-self: start;
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		padding: 0.3rem 0.75rem;
		border-radius: 999px;
		background: var(--hijau-muda);
		color: var(--hijau-tua);
		font-size: 0.8rem;
		font-weight: 600;
	}
	.jaminan {
		list-style: none;
		margin: 0.25rem 0 0;
		padding: 0;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.6rem;
	}
	.jaminan li {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.55rem 0.75rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 12px;
		font-size: 0.875rem;
		font-weight: 500;
	}
	.ikon {
		display: inline-grid;
		place-items: center;
		width: 30px;
		height: 30px;
		border-radius: 9px;
		flex: none;
	}
	.ikon.hijau {
		background: var(--hijau-muda);
		color: var(--hijau);
	}
	.ikon.kuning {
		background: var(--kuning);
		color: var(--kuning-teks);
	}

	.kartu-form {
		display: grid;
		gap: 1.25rem;
		padding: 1.25rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 20px;
		box-shadow: 0 1px 2px rgb(0 0 0 / 0.04);
	}
	fieldset.field {
		border: 0;
		margin: 0;
		padding: 0;
		min-width: 0;
	}
	legend,
	.field label {
		font-weight: 600;
		margin-bottom: 0.4rem;
		padding: 0;
	}
	.wajib {
		color: var(--merah);
	}
	.opsi {
		color: var(--teks-redup);
		font-weight: 400;
	}
	.kartu-form :global(input[type='text']),
	.kartu-form textarea {
		width: 100%;
		padding: 0.8rem 0.95rem;
		border: 1px solid var(--garis);
		border-radius: 12px;
		background: var(--krem);
		font: inherit;
	}
	.kartu-form textarea {
		resize: vertical;
	}
	.kartu-form :global(input[type='text']:focus),
	.kartu-form textarea:focus {
		outline: 2px solid var(--hijau-aktif);
		outline-offset: 1px;
		background: var(--kartu);
	}

	.pil-baris,
	.nominal-baris {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-bottom: 0.6rem;
	}
	.pil input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}
	.pil span,
	.nominal {
		display: inline-block;
		padding: 0.55rem 1rem;
		border: 1px solid var(--garis);
		border-radius: 999px;
		background: var(--krem);
		font: inherit;
		font-weight: 500;
		cursor: pointer;
		color: var(--teks);
		transition: background 0.15s, border-color 0.15s, color 0.15s;
	}
	.nominal {
		border-radius: 12px;
		flex: 1 1 28%;
		padding: 0.55rem 0.4rem;
		white-space: nowrap;
		text-align: center;
	}
	.pil input:checked + span {
		background: var(--hijau);
		border-color: var(--hijau);
		color: #fff;
	}
	.pil input:focus-visible + span,
	.nominal:focus-visible {
		outline: 2px solid var(--hijau-aktif);
		outline-offset: 2px;
	}
	.nominal.aktif {
		background: var(--hijau-muda);
		border-color: var(--hijau);
		color: var(--hijau-tua);
		font-weight: 600;
	}

	.rupiah {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0 0.95rem;
		border: 1px solid var(--garis);
		border-radius: 12px;
		background: var(--krem);
		margin-bottom: 0.4rem;
	}
	.rupiah:focus-within {
		outline: 2px solid var(--hijau-aktif);
		outline-offset: 1px;
		background: var(--kartu);
	}
	.rupiah.salah {
		border-color: var(--merah);
	}
	.rupiah span {
		color: var(--teks-redup);
		font-weight: 600;
	}
	.rupiah :global(input[type='text']) {
		border: 0;
		background: transparent;
		padding: 0.8rem 0;
		font-size: 1.15rem;
		font-weight: 600;
		outline: none !important;
	}

	.metode {
		display: grid;
		gap: 0.6rem;
		padding: 0.9rem 1rem;
		border: 1px solid var(--hijau-muda);
		background: color-mix(in srgb, var(--hijau-muda) 55%, #fff);
		border-radius: 14px;
	}
	.metode-judul {
		font-size: 0.8rem;
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}
	.metode-daftar {
		display: flex;
		flex-wrap: wrap;
		gap: 0.4rem;
	}
	.metode-daftar span {
		padding: 0.2rem 0.65rem;
		background: var(--kartu);
		border: 1px solid var(--garis);
		border-radius: 8px;
		font-size: 0.8rem;
		font-weight: 600;
	}
	.metode p {
		margin: 0;
	}
	.kecil {
		font-size: 0.8rem;
	}

	/* Tombol aksi menempel di bawah layar saat form panjang (ponsel). */
	.aksi {
		position: sticky;
		bottom: 0;
		display: grid;
		gap: 0.5rem;
		margin: 0 -1.25rem -1.25rem;
		padding: 0.9rem 1.25rem 1.1rem;
		background: linear-gradient(to bottom, transparent, var(--kartu) 30%);
		border-radius: 0 0 20px 20px;
	}
	.besar {
		width: 100%;
		justify-content: center;
		padding: 0.95rem 1.25rem;
		font-size: 1.05rem;
		font-weight: 700;
		border-radius: 14px;
	}
	.catatan {
		margin: 0;
		text-align: center;
		font-size: 0.78rem;
		color: var(--teks-redup);
	}
	.jebakan {
		position: absolute;
		left: -9999px;
		width: 1px;
		height: 1px;
		overflow: hidden;
	}
	@media (max-width: 420px) {
		.kartu-form {
			padding: 1rem;
		}
		.aksi {
			margin: 0 -1rem -1rem;
			padding: 0.9rem 1rem 1rem;
		}
	}
</style>
