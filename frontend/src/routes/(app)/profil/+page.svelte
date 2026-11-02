<!-- Profil & password: tampil nama + peran; form ganti password (lama, baru, ulangi). -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);
	const galat = (f: string) => form?.fields?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Profil & password · SPK Zakat</title></svelte:head>

<div class="tumpuk sempit">
	<h1>Profil & password</h1>

	<section class="kartu">
		<dl>
			<dt>Nama</dt><dd>{data.user.nama}</dd>
			<dt>Username</dt><dd>{data.user.username}</dd>
			<dt>Peran</dt><dd>{data.user.role === 'admin' ? 'Admin / Petugas Amil' : 'Ketua DKM'}</dd>
		</dl>
		<p class="redup kecil">Nama dan peran diatur oleh Ketua lewat menu Pengguna.</p>
	</section>

	<form
		method="POST"
		class="kartu tumpuk"
		use:enhance={() => {
			mengirim = true;
			return async ({ update }) => {
				await update();
				mengirim = false;
			};
		}}
	>
		<h2>Ganti password</h2>
		{#if form?.tersimpan}
			<p class="kotak-sukses" role="status">Password diganti. Sesi di perangkat lain sudah dikeluarkan.</p>
		{/if}
		{#if galatUmum}<p class="kotak-galat" role="alert">{galatUmum}</p>{/if}

		<div class="field">
			<label for="lama">Password lama</label>
			<input id="lama" name="password_lama" type="password" autocomplete="current-password" required maxlength="128" aria-invalid={Boolean(galat('password_lama'))} />
			{#if galat('password_lama')}<span class="galat" role="alert">{galat('password_lama')}</span>{/if}
		</div>
		<div class="field">
			<label for="baru">Password baru</label>
			<input id="baru" name="password_baru" type="password" autocomplete="new-password" required minlength="12" maxlength="128" aria-describedby="h-baru" aria-invalid={Boolean(galat('password_baru'))} />
			<small id="h-baru">Minimal 12 karakter. Gunakan kalimat yang mudah diingat tetapi sulit ditebak.</small>
			{#if galat('password_baru')}<span class="galat" role="alert">{galat('password_baru')}</span>{/if}
		</div>
		<div class="field">
			<label for="ulangi">Ulangi password baru</label>
			<input id="ulangi" name="ulangi" type="password" autocomplete="new-password" required maxlength="128" aria-invalid={Boolean(galat('ulangi'))} />
			{#if galat('ulangi')}<span class="galat" role="alert">{galat('ulangi')}</span>{/if}
		</div>
		<div class="baris-aksi ujung">
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : 'Ganti password'}</button>
		</div>
	</form>
</div>

<style>
	.sempit {
		max-width: 640px;
	}
	dl {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: 0.5rem 1.5rem;
		margin: 0 0 0.75rem;
	}
	dt {
		color: var(--teks-redup);
	}
	dd {
		margin: 0;
		font-weight: 600;
	}
	.kecil {
		font-size: 0.88rem;
	}
</style>
