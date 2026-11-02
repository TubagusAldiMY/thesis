<!--
  Setup pertama kali (publik, tanpa login): membuat akun Ketua pertama. Hanya
  tampil selama belum ada akun. Butuh kode setup yang dicetak di log server
  (bukti bahwa yang membuka halaman ini memegang akses ke server).
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
	let mengirim = $state(false);
	const nilai = $derived(form?.nilai ?? { nama: '', username: '' });
	const galat = (f: string) => (form?.fields as Record<string, string> | undefined)?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Setup awal · {masjid.nama}</title></svelte:head>

<div class="tumpuk sempit">
	<header>
		<h1>Setup awal</h1>
		<p class="redup">
			Aplikasi ini belum punya akun. Buat akun <strong>Ketua</strong> pertama di sini; setelah itu Ketua membuat akun Admin dan pengurus lain lewat menu Pengguna.
		</p>
	</header>

	<details class="kotak-info">
		<summary>Di mana kode setup?</summary>
		<p>
			Kode dicetak di log server saat aplikasi pertama kali berjalan. Dengan Docker:
			<code>docker compose logs api | grep kode_setup</code>. Halaman ini hanya bisa dipakai sekali, dan menutup sendiri setelah akun pertama dibuat.
		</p>
	</details>

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
			<label for="token">Kode setup *</label>
			<input id="token" name="token" type="text" required maxlength="64" autocomplete="off" autocapitalize="none" spellcheck="false" aria-invalid={Boolean(galat('token'))} />
			{#if galat('token')}<span class="galat" role="alert">{galat('token')}</span>{/if}
		</div>

		<div class="field">
			<label for="nama">Nama lengkap Ketua *</label>
			<input id="nama" name="nama" type="text" required maxlength="100" autocomplete="name" value={nilai.nama} aria-invalid={Boolean(galat('nama'))} />
			{#if galat('nama')}<span class="galat" role="alert">{galat('nama')}</span>{/if}
		</div>

		<div class="field">
			<label for="username">Username *</label>
			<input id="username" name="username" type="text" required maxlength="50" autocomplete="username" autocapitalize="none" placeholder="mis. ketua" value={nilai.username} aria-invalid={Boolean(galat('username'))} />
			<small>3–50 karakter: huruf kecil, angka, titik, garis bawah, atau strip.</small>
			{#if galat('username')}<span class="galat" role="alert">{galat('username')}</span>{/if}
		</div>

		<div class="dua">
			<div class="field">
				<label for="password">Password *</label>
				<input id="password" name="password" type="password" autocomplete="new-password" required minlength="12" maxlength="128" aria-invalid={Boolean(galat('password'))} />
				<small>Minimal 12 karakter.</small>
				{#if galat('password')}<span class="galat" role="alert">{galat('password')}</span>{/if}
			</div>
			<div class="field">
				<label for="ulangi">Ulangi password *</label>
				<input id="ulangi" name="ulangi" type="password" autocomplete="new-password" required maxlength="128" aria-invalid={Boolean(galat('ulangi'))} />
				{#if galat('ulangi')}<span class="galat" role="alert">{galat('ulangi')}</span>{/if}
			</div>
		</div>

		<div class="baris-aksi ujung">
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Membuat…' : 'Buat akun Ketua'}</button>
		</div>
	</form>
</div>

<style>
	.sempit {
		max-width: 640px;
		margin: 1rem auto 0;
	}
	header {
		display: grid;
		gap: 0.5rem;
	}
	details summary {
		cursor: pointer;
		font-weight: 600;
	}
	details p {
		margin-top: 0.5rem;
		line-height: 1.6;
	}
	code {
		font-family: var(--mono);
		font-size: 0.85em;
		background: var(--abu);
		padding: 0.1rem 0.35rem;
		border-radius: 6px;
		word-break: break-all;
	}
	.dua {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	@media (max-width: 560px) {
		.dua {
			grid-template-columns: 1fr;
		}
	}
</style>
