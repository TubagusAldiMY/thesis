<!-- Kelola pengguna (Ketua): daftar akun Admin dan Ketua dengan status; akun tidak dihapus, hanya dinonaktifkan. -->
<script lang="ts">
	import { labelRole, tanggal } from '$lib/format';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
	const aktif = $derived(data.pengguna.filter((p) => p.aktif).length);
</script>

<svelte:head><title>Pengguna · SPK Zakat</title></svelte:head>

<div class="tumpuk">
	<div class="kartu-kepala">
		<div>
			<h1>Pengguna</h1>
			<p class="redup">
				{aktif} akun aktif dari {data.pengguna.length}. Akun yang dinonaktifkan tidak dihapus, supaya jejak di audit log tetap utuh.
			</p>
		</div>
		<a class="tombol utama" href="/pengguna/baru">+ Tambah pengguna</a>
	</div>

	{#if data.pesan}<p class="kotak-sukses" role="status">{data.pesan}</p>{/if}

	<div class="kartu tabel-wadah">
		<table>
			<thead>
				<tr><th>Nama</th><th>Username</th><th>Peran</th><th>Status</th><th class="lebar-saja">Dibuat</th></tr>
			</thead>
			<tbody>
				{#each data.pengguna as p (p.id)}
					<tr class:redup={!p.aktif}>
						<td>
							<a href="/pengguna/{p.id}"><strong>{p.nama}</strong></a>
							{#if p.id === data.user.id}<span class="badge">Anda</span>{/if}
						</td>
						<td class="angka">{p.username}</td>
						<td>{labelRole[p.role]}</td>
						<td><span class="badge {p.aktif ? 'hijau' : 'merah'}">{p.aktif ? 'Aktif' : 'Nonaktif'}</span></td>
						<td class="lebar-saja">{tanggal(p.dibuat_pada)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>

<style>
	@media (max-width: 700px) {
		.lebar-saja {
			display: none;
		}
	}
</style>
