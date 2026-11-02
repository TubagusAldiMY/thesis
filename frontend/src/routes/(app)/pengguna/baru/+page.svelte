<!-- Formulir tambah pengguna (Ketua): username, nama, peran, password awal. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import { labelRole } from '$lib/format';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
	let mengirim = $state(false);
	const nilai = $derived(form?.nilai ?? { username: '', nama: '', role: 'admin' });
	const galat = (f: string) => (form?.fields as Record<string, string> | undefined)?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Tambah pengguna · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/pengguna">← Pengguna</a>
	<h1>Tambah pengguna</h1>
	<p class="redup">Sampaikan password awal langsung kepada yang bersangkutan, lalu mintalah menggantinya di menu Profil & password.</p>

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
			<label for="nama">Nama lengkap *</label>
			<input id="nama" name="nama" type="text" required maxlength="100" value={nilai.nama} aria-invalid={Boolean(galat('nama'))} />
			{#if galat('nama')}<span class="galat" role="alert">{galat('nama')}</span>{/if}
		</div>

		<div class="dua">
			<div class="field">
				<label for="username">Username *</label>
				<input id="username" name="username" type="text" required maxlength="50" autocomplete="off" autocapitalize="none" placeholder="mis. petugas.amil" value={nilai.username} aria-invalid={Boolean(galat('username'))} />
				<small>3–50 karakter: huruf kecil, angka, titik, garis bawah, atau strip.</small>
				{#if galat('username')}<span class="galat" role="alert">{galat('username')}</span>{/if}
			</div>
			<div class="field">
				<label for="role">Peran *</label>
				<select id="role" name="role" value={nilai.role} aria-invalid={Boolean(galat('role'))}>
					<option value="admin">{labelRole.admin}: mengelola data</option>
					<option value="ketua">{labelRole.ketua}: menyetujui dan mengelola pengguna</option>
				</select>
				{#if galat('role')}<span class="galat" role="alert">{galat('role')}</span>{/if}
			</div>
		</div>

		<div class="dua">
			<div class="field">
				<label for="password">Password awal *</label>
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
			<a class="tombol" href="/pengguna">Batal</a>
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : 'Buat akun'}</button>
		</div>
	</form>
</div>

<style>
	.sempit {
		max-width: 760px;
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
