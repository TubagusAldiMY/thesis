<!--
  Donasi zakat & infak (publik, tanpa login). Muzaki mengisi nama, jenis, dan
  jumlah; pembayaran (QRIS / virtual account / e-wallet) dilakukan di halaman
  payment gateway pada langkah berikutnya. Kuitansi terbit otomatis setelah
  pembayaran dikonfirmasi gateway.
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { labelJenisDonasi, semuaJenisDonasi } from '$lib/format';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);

	const nilai = $derived({
		nama: form?.nilai?.nama ?? '',
		jenis: form?.nilai?.jenis || 'zakat_fitrah',
		jumlah: form?.nilai?.jumlah ?? '',
		keterangan: form?.nilai?.keterangan ?? ''
	});
	const galat = (f: string) => form?.fields?.[f] ?? '';
	// Pesan umum (mis. 429 terlalu banyak permintaan, 502 gateway bermasalah) = error tanpa pesan per kolom.
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Donasi zakat & infak · {masjid.nama}</title></svelte:head>

<div class="tumpuk">
	<div>
		<h1>Donasi zakat & infak</h1>
		<p class="redup">
			Salurkan zakat dan infak Anda melalui {masjid.nama}. Setelah pembayaran berhasil, kuitansi dapat langsung dilihat dan dicetak.
		</p>
	</div>

	{#if !data.aktif}
		<p class="kotak-info" role="status">
			Donasi online belum diaktifkan oleh pengurus masjid. Silakan hubungi pengurus{masjid.kontak ? ` di ${masjid.kontak}` : ''} atau datang langsung ke masjid.
		</p>
	{:else}
		<form
			method="POST"
			class="kartu tumpuk"
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
				<label for="nama">Nama *</label>
				<input id="nama" name="nama" type="text" required maxlength="100" autocomplete="name" value={nilai.nama} aria-invalid={Boolean(galat('nama'))} />
				<small>Tertulis di kuitansi. Boleh memakai nama keluarga atau "Hamba Allah".</small>
				{#if galat('nama')}<span class="galat" role="alert">{galat('nama')}</span>{/if}
			</div>

			<div class="dua">
				<div class="field">
					<label for="jenis">Jenis *</label>
					<select id="jenis" name="jenis" value={nilai.jenis} aria-invalid={Boolean(galat('jenis'))}>
						{#each semuaJenisDonasi as j (j)}<option value={j}>{labelJenisDonasi[j]}</option>{/each}
					</select>
					{#if galat('jenis')}<span class="galat" role="alert">{galat('jenis')}</span>{/if}
				</div>
				<div class="field">
					<label for="jumlah">Jumlah (Rp) *</label>
					<input id="jumlah" name="jumlah" type="text" inputmode="numeric" autocomplete="off" required placeholder="mis. 100.000" value={nilai.jumlah} aria-invalid={Boolean(galat('jumlah'))} />
					{#if galat('jumlah')}<span class="galat" role="alert">{galat('jumlah')}</span>{/if}
				</div>
			</div>

			<div class="field">
				<label for="no_hp">No. HP / WhatsApp (opsional)</label>
				<input id="no_hp" name="no_hp" type="text" inputmode="tel" autocomplete="tel" maxlength="20" placeholder="mis. 081234567890" aria-invalid={Boolean(galat('no_hp'))} />
				<small>Hanya agar pengurus bisa mengirim konfirmasi. Disimpan terenkripsi.</small>
				{#if galat('no_hp')}<span class="galat" role="alert">{galat('no_hp')}</span>{/if}
			</div>

			<div class="field">
				<label for="keterangan">Pesan / keterangan (opsional)</label>
				<input id="keterangan" name="keterangan" type="text" maxlength="200" value={nilai.keterangan} aria-invalid={Boolean(galat('keterangan'))} />
				{#if galat('keterangan')}<span class="galat" role="alert">{galat('keterangan')}</span>{/if}
			</div>

			<!-- Kolom jebakan anti-bot: tidak terlihat dan tidak bisa difokus oleh manusia. -->
			<div class="jebakan" aria-hidden="true">
				<label>Website <input name="website" type="text" tabindex="-1" autocomplete="off" /></label>
			</div>

			<p class="redup kecil">
				Metode pembayaran (QRIS, virtual account, e-wallet) tersedia di halaman berikutnya. Pembayaran diproses oleh penyedia pembayaran;
				aplikasi ini tidak menyimpan data kartu atau rekening Anda. Jumlah donasi Rp 10.000 sampai Rp 100.000.000.
			</p>

			<div class="baris-aksi ujung">
				<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Memproses…' : 'Lanjut ke pembayaran'}</button>
			</div>
		</form>
	{/if}
</div>

<style>
	.dua {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	.kecil {
		font-size: 0.85rem;
	}
	.jebakan {
		position: absolute;
		left: -9999px;
		width: 1px;
		height: 1px;
		overflow: hidden;
	}
	@media (max-width: 560px) {
		.dua {
			grid-template-columns: 1fr;
		}
	}
</style>
