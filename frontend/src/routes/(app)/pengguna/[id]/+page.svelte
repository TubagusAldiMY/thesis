<!--
  Kelola satu akun (Ketua): ubah nama & peran, reset password, nonaktifkan/aktifkan.
  Akun sendiri hanya boleh ganti nama: peran, status, dan password diatur lewat
  aturan Go (dan Profil untuk password sendiri).
-->
<script lang="ts">
	import { enhance } from '$app/forms';
	import type { SubmitFunction } from '@sveltejs/kit';
	import { labelRole, tanggal } from '$lib/format';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	const a = $derived(data.akun);
	const diriSendiri = $derived(a.id === data.user.id);
	let mengirim = $state('');

	type Hasil = { bagian?: string; sukses?: string; error?: string; fields?: Record<string, string>; nilai?: { nama: string; role: string } };
	const hasil = $derived(form as Hasil | null);
	const untuk = (bagian: string) => (hasil?.bagian === bagian ? hasil : null);
	const galat = (bagian: string) => (f: string) => untuk(bagian)?.fields?.[f] ?? '';
	const galatUmum = (bagian: string) => {
		const h = untuk(bagian);
		return h?.error && Object.keys(h.fields ?? {}).length === 0 ? h.error : '';
	};
	const nilaiUbah = $derived(untuk('ubah')?.nilai ?? { nama: a.nama, role: a.role });

	const kirim =
		(bagian: string): SubmitFunction =>
		() => {
			mengirim = bagian;
			return async ({ result, update }) => {
				// Isian password dikosongkan setelah berhasil; formulir ubah nama tidak direset.
				await update({ reset: bagian === 'reset' && result.type === 'success' });
				mengirim = '';
			};
		};
</script>

<svelte:head><title>{a.nama} · Pengguna</title></svelte:head>

<div class="tumpuk sempit">
	<a class="tautan-kembali" href="/pengguna">← Pengguna</a>

	<div class="kartu-kepala">
		<div>
			<h1>{a.nama}</h1>
			<p class="redup"><span class="angka">{a.username}</span> · {labelRole[a.role]} · dibuat {tanggal(a.dibuat_pada)}{diriSendiri ? ' · akun Anda' : ''}</p>
		</div>
		<span class="badge {a.aktif ? 'hijau' : 'merah'}">{a.aktif ? 'Aktif' : 'Nonaktif'}</span>
	</div>

	<form method="POST" action="?/ubah" class="kartu tumpuk" novalidate use:enhance={kirim('ubah')}>
		<h2>Nama dan peran</h2>
		{#if untuk('ubah')?.sukses}<p class="kotak-sukses" role="status">{untuk('ubah')?.sukses}</p>{/if}
		{#if galatUmum('ubah')}<p class="kotak-galat" role="alert">{galatUmum('ubah')}</p>{/if}
		<div class="dua">
			<div class="field">
				<label for="nama">Nama lengkap</label>
				<input id="nama" name="nama" type="text" required maxlength="100" value={nilaiUbah.nama} aria-invalid={Boolean(galat('ubah')('nama'))} />
				{#if galat('ubah')('nama')}<span class="galat" role="alert">{galat('ubah')('nama')}</span>{/if}
			</div>
			<div class="field">
				<label for="role">Peran</label>
				{#if diriSendiri}<input type="hidden" name="role" value={a.role} />{/if}
				<select id="role" name="role" value={nilaiUbah.role} disabled={diriSendiri} aria-invalid={Boolean(galat('ubah')('role'))}>
					<option value="admin">{labelRole.admin}</option>
					<option value="ketua">{labelRole.ketua}</option>
				</select>
				{#if diriSendiri}<small>Peran akun sendiri tidak bisa diubah.</small>{/if}
				{#if galat('ubah')('role')}<span class="galat" role="alert">{galat('ubah')('role')}</span>{/if}
			</div>
		</div>
		<div class="baris-aksi ujung">
			<button class="tombol utama" type="submit" disabled={mengirim === 'ubah'}>{mengirim === 'ubah' ? 'Menyimpan…' : 'Simpan'}</button>
		</div>
	</form>

	{#if diriSendiri}
		<section class="kartu tumpuk">
			<h2>Password dan status</h2>
			<p class="redup">Untuk mengganti password Anda sendiri gunakan <a href="/profil">Profil & password</a>. Akun sendiri tidak bisa dinonaktifkan dari sini.</p>
		</section>
	{:else}
		<form method="POST" action="?/reset" class="kartu tumpuk" novalidate use:enhance={kirim('reset')}>
			<h2>Reset password</h2>
			<p class="redup">Semua sesi akun ini dikeluarkan seketika. Sampaikan password baru langsung kepada yang bersangkutan.</p>
			{#if untuk('reset')?.sukses}<p class="kotak-sukses" role="status">{untuk('reset')?.sukses}</p>{/if}
			{#if galatUmum('reset')}<p class="kotak-galat" role="alert">{galatUmum('reset')}</p>{/if}
			<div class="dua">
				<div class="field">
					<label for="password">Password baru</label>
					<input id="password" name="password" type="password" autocomplete="new-password" required minlength="12" maxlength="128" aria-invalid={Boolean(galat('reset')('password'))} />
					<small>Minimal 12 karakter.</small>
					{#if galat('reset')('password')}<span class="galat" role="alert">{galat('reset')('password')}</span>{/if}
				</div>
				<div class="field">
					<label for="ulangi">Ulangi password</label>
					<input id="ulangi" name="ulangi" type="password" autocomplete="new-password" required maxlength="128" aria-invalid={Boolean(galat('reset')('ulangi'))} />
					{#if galat('reset')('ulangi')}<span class="galat" role="alert">{galat('reset')('ulangi')}</span>{/if}
				</div>
			</div>
			<div class="baris-aksi ujung">
				<button class="tombol" type="submit" disabled={mengirim === 'reset'}>{mengirim === 'reset' ? 'Menyimpan…' : 'Reset password'}</button>
			</div>
		</form>

		<section class="kartu tumpuk">
			<h2>Status akun</h2>
			{#if untuk('status')?.sukses}<p class="kotak-sukses" role="status">{untuk('status')?.sukses}</p>{/if}
			{#if galatUmum('status')}<p class="kotak-galat" role="alert">{galatUmum('status')}</p>{/if}
			{#if a.aktif}
				<p class="redup">Akun nonaktif tidak bisa masuk, dan sesi yang sedang berjalan langsung dikeluarkan. Akun tidak dihapus; Anda bisa mengaktifkannya kembali kapan saja.</p>
				<form
					method="POST"
					action="?/nonaktif"
					use:enhance={kirim('status')}
					onsubmit={(e) => {
						if (!confirm(`Nonaktifkan akun ${a.nama}? Yang bersangkutan langsung dikeluarkan.`)) e.preventDefault();
					}}
				>
					<button class="tombol bahaya" type="submit" disabled={mengirim === 'status'}>Nonaktifkan akun</button>
				</form>
			{:else}
				<p class="redup">Akun ini nonaktif. Aktifkan kembali agar yang bersangkutan bisa masuk dengan password terakhirnya (atau reset password lebih dulu).</p>
				<form method="POST" action="?/aktif" use:enhance={kirim('status')}>
					<button class="tombol utama" type="submit" disabled={mengirim === 'status'}>Aktifkan kembali</button>
				</form>
			{/if}
		</section>
	{/if}
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
