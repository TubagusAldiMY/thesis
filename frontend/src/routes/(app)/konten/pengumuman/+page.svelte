<!-- Pengumuman singkat: daftar dan formulir tambah/ubah. Yang terbit dan masih dalam masa tayang tampil di halaman depan. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import TabKonten from '$lib/components/TabKonten.svelte';
	import { tanggalSaja } from '$lib/format';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);

	type Nilai = { id: string; judul: string; isi: string; tampil_sampai: string; terbit: boolean };
	const awal = $derived<Nilai>(data.sedangDiubah ?? { id: '', judul: '', isi: '', tampil_sampai: '', terbit: true });
	// Setelah galat, isian yang dipantulkan menang atas isian awal.
	const nilai = $derived<Nilai>((form?.nilai as Nilai | undefined) ?? awal);
	const galat = (f: string) => (form?.fields as Record<string, string> | undefined)?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Pengumuman · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Pengumuman</h1>
			<p class="redup">Pengumuman singkat (mis. kerja bakti, perubahan jadwal). Tampil di halaman depan selama masa tayangnya.</p>
		</div>
	</div>

	<TabKonten />

	{#if data.pesan}<p class="kotak-sukses" role="status">{data.pesan}</p>{/if}

	<form
		method="POST"
		action="?/simpan"
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
		<h2>{nilai.id ? 'Ubah pengumuman' : 'Tambah pengumuman'}</h2>
		{#if galatUmum}<p class="kotak-galat" role="alert">{galatUmum}</p>{/if}
		<input type="hidden" name="id" value={nilai.id} />

		<div class="field">
			<label for="judul">Judul *</label>
			<input id="judul" name="judul" type="text" required maxlength="120" value={nilai.judul} aria-invalid={Boolean(galat('judul'))} />
			{#if galat('judul')}<span class="galat" role="alert">{galat('judul')}</span>{/if}
		</div>
		<div class="field">
			<label for="isi">Isi *</label>
			<textarea id="isi" name="isi" rows="3" required maxlength="500" aria-invalid={Boolean(galat('isi'))}>{nilai.isi}</textarea>
			<small>Maksimal 500 karakter.</small>
			{#if galat('isi')}<span class="galat" role="alert">{galat('isi')}</span>{/if}
		</div>
		<div class="field">
			<label for="tampil_sampai">Tayang sampai (opsional)</label>
			<input id="tampil_sampai" name="tampil_sampai" type="date" value={nilai.tampil_sampai} aria-invalid={Boolean(galat('tampil_sampai'))} />
			<small>Hari itu masih tayang. Kosongkan bila tayang terus sampai disembunyikan.</small>
			{#if galat('tampil_sampai')}<span class="galat" role="alert">{galat('tampil_sampai')}</span>{/if}
		</div>
		<label class="centang">
			<input type="checkbox" name="terbit" checked={nilai.terbit} />
			<span>Terbit (tampil di halaman depan)</span>
		</label>
		<div class="baris-aksi ujung">
			{#if nilai.id}<a class="tombol" href="/konten/pengumuman">Batal</a>{/if}
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : nilai.id ? 'Simpan perubahan' : 'Tambah pengumuman'}</button>
		</div>
	</form>

	{#if data.daftar.length === 0}
		<p class="kartu redup">Belum ada pengumuman.</p>
	{:else}
		<div class="kartu tabel-wadah">
			<table>
				<thead>
					<tr><th>Pengumuman</th><th class="lebar-saja">Tayang sampai</th><th>Status</th><th></th></tr>
				</thead>
				<tbody>
					{#each data.daftar as p (p.id)}
						<tr class:redup={!p.tayang}>
							<td>
								<strong>{p.judul}</strong>
								<small class="redup blok">{p.isi}</small>
							</td>
							<td class="lebar-saja">{p.tampil_sampai ? tanggalSaja(p.tampil_sampai) : 'Tanpa batas'}</td>
							<td>
								{#if p.tayang}<span class="badge hijau">Tayang</span>
								{:else if !p.terbit}<span class="badge">Disembunyikan</span>
								{:else}<span class="badge kuning">Sudah berakhir</span>{/if}
							</td>
							<td class="aksi">
								<a class="tombol kecil" href="/konten/pengumuman?ubah={p.id}#judul">Ubah</a>
								<form
									method="POST"
									action="?/hapus"
									onsubmit={(e) => {
										if (!confirm(`Hapus pengumuman "${p.judul}"?`)) e.preventDefault();
									}}
								>
									<input type="hidden" name="id" value={p.id} />
									<button class="tombol kecil bahaya" type="submit">Hapus</button>
								</form>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style>
	textarea {
		resize: vertical;
	}
	.centang {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		cursor: pointer;
	}
	.centang input {
		accent-color: var(--hijau);
	}
	.blok {
		display: block;
		max-width: 52ch;
	}
	.aksi {
		display: flex;
		gap: 0.4rem;
		justify-content: flex-end;
	}
	@media (max-width: 700px) {
		.lebar-saja {
			display: none;
		}
	}
</style>
