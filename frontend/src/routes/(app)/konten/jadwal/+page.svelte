<!-- Jadwal rutin mingguan: daftar dan formulir tambah/ubah. Yang aktif tampil di halaman publik /kegiatan dan landing. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import TabKonten from '$lib/components/TabKonten.svelte';
	import { namaHari } from '$lib/kegiatan';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);

	type Nilai = { id: string; judul: string; pemateri: string; hari: string; jam: string; lokasi: string; keterangan: string; aktif: boolean };
	const awal = $derived<Nilai>(
		data.sedangDiubah
			? { ...data.sedangDiubah, hari: String(data.sedangDiubah.hari) }
			: { id: '', judul: '', pemateri: '', hari: '0', jam: '', lokasi: '', keterangan: '', aktif: true }
	);
	// Setelah galat, isian yang dipantulkan menang atas isian awal.
	const nilai = $derived<Nilai>((form?.nilai as Nilai | undefined) ?? awal);
	const galat = (f: string) => (form?.fields as Record<string, string> | undefined)?.[f] ?? '';
	const galatUmum = $derived(form?.error && Object.keys(form.fields ?? {}).length === 0 ? form.error : '');
</script>

<svelte:head><title>Jadwal rutin · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Jadwal rutin</h1>
			<p class="redup">Kajian dan kegiatan mingguan yang berulang. Yang aktif tampil di halaman publik <a href="/kegiatan">/kegiatan</a> dan di halaman depan.</p>
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
		<h2>{nilai.id ? 'Ubah jadwal' : 'Tambah jadwal'}</h2>
		{#if galatUmum}<p class="kotak-galat" role="alert">{galatUmum}</p>{/if}
		<input type="hidden" name="id" value={nilai.id} />

		<div class="dua">
			<div class="field">
				<label for="judul">Judul *</label>
				<input id="judul" name="judul" type="text" required maxlength="120" placeholder="mis. Kajian Subuh" value={nilai.judul} aria-invalid={Boolean(galat('judul'))} />
				{#if galat('judul')}<span class="galat" role="alert">{galat('judul')}</span>{/if}
			</div>
			<div class="field">
				<label for="pemateri">Pemateri (opsional)</label>
				<input id="pemateri" name="pemateri" type="text" maxlength="100" value={nilai.pemateri} aria-invalid={Boolean(galat('pemateri'))} />
				{#if galat('pemateri')}<span class="galat" role="alert">{galat('pemateri')}</span>{/if}
			</div>
		</div>
		<div class="tiga">
			<div class="field">
				<label for="hari">Hari *</label>
				<select id="hari" name="hari" value={nilai.hari} aria-invalid={Boolean(galat('hari'))}>
					{#each namaHari as h, i (h)}<option value={String(i)}>{h}</option>{/each}
				</select>
				{#if galat('hari')}<span class="galat" role="alert">{galat('hari')}</span>{/if}
			</div>
			<div class="field">
				<label for="jam">Jam (WIB) *</label>
				<input id="jam" name="jam" type="time" required value={nilai.jam} aria-invalid={Boolean(galat('jam'))} />
				{#if galat('jam')}<span class="galat" role="alert">{galat('jam')}</span>{/if}
			</div>
			<div class="field">
				<label for="lokasi">Lokasi (opsional)</label>
				<input id="lokasi" name="lokasi" type="text" maxlength="120" value={nilai.lokasi} aria-invalid={Boolean(galat('lokasi'))} />
				{#if galat('lokasi')}<span class="galat" role="alert">{galat('lokasi')}</span>{/if}
			</div>
		</div>
		<div class="field">
			<label for="keterangan">Keterangan (opsional)</label>
			<input id="keterangan" name="keterangan" type="text" maxlength="200" placeholder="mis. Kitab Riyadhus Shalihin, terbuka untuk umum" value={nilai.keterangan} aria-invalid={Boolean(galat('keterangan'))} />
			{#if galat('keterangan')}<span class="galat" role="alert">{galat('keterangan')}</span>{/if}
		</div>
		<label class="centang">
			<input type="checkbox" name="aktif" checked={nilai.aktif} />
			<span>Aktif (tampil di halaman publik)</span>
		</label>
		<div class="baris-aksi ujung">
			{#if nilai.id}<a class="tombol" href="/konten/jadwal">Batal</a>{/if}
			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Menyimpan…' : nilai.id ? 'Simpan perubahan' : 'Tambah jadwal'}</button>
		</div>
	</form>

	{#if data.jadwal.length === 0}
		<p class="kartu redup">Belum ada jadwal rutin.</p>
	{:else}
		<div class="kartu tabel-wadah">
			<table>
				<thead>
					<tr><th>Hari</th><th>Jam</th><th>Kegiatan</th><th class="lebar-saja">Lokasi</th><th>Status</th><th></th></tr>
				</thead>
				<tbody>
					{#each data.jadwal as j (j.id)}
						<tr class:redup={!j.aktif}>
							<td>{j.hari_label}</td>
							<td class="angka">{j.jam.replace(':', '.')}</td>
							<td>
								<strong>{j.judul}</strong>
								{#if j.pemateri}<small class="redup blok">{j.pemateri}</small>{/if}
							</td>
							<td class="lebar-saja">{j.lokasi || '-'}</td>
							<td><span class="badge {j.aktif ? 'hijau' : ''}">{j.aktif ? 'Aktif' : 'Nonaktif'}</span></td>
							<td class="aksi">
								<a class="tombol kecil" href="/konten/jadwal?ubah={j.id}#judul">Ubah</a>
								<form
									method="POST"
									action="?/hapus"
									onsubmit={(e) => {
										if (!confirm(`Hapus jadwal "${j.judul}"?`)) e.preventDefault();
									}}
								>
									<input type="hidden" name="id" value={j.id} />
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
	.dua {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
	}
	.tiga {
		display: grid;
		grid-template-columns: 1fr 1fr 2fr;
		gap: 1rem;
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
	}
	.aksi {
		display: flex;
		gap: 0.4rem;
		justify-content: flex-end;
	}
	@media (max-width: 700px) {
		.dua,
		.tiga {
			grid-template-columns: 1fr;
		}
		.lebar-saja {
			display: none;
		}
	}
</style>
