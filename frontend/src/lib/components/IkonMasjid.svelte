<!--
  Logo masjid. Bila PUBLIC_MASJID_LOGO diisi, tampil gambar logo itu; bila kosong,
  ikon kubah/gerbang (garis) yang mengikuti warna teks induknya. Semua tempat yang
  memakai komponen ini ikut berganti, jadi logo cukup dipasang di satu tempat.
-->
<script lang="ts">
	import { masjid } from '$lib/masjid';

	let { ukuran = 32 }: { ukuran?: number } = $props();
</script>

{#if masjid.logo}
	<img class="logo" src={masjid.logo} width={ukuran} height={ukuran} alt="" />
{:else}
	<svg width={ukuran} height={ukuran} viewBox="0 0 32 32" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
		<path d="M6 27V14a10 10 0 0 1 20 0v13" />
		<path d="M12 27v-6a4 4 0 0 1 8 0v6" />
		<path d="M3 27h26" />
	</svg>
{/if}

<style>
	.logo {
		object-fit: contain; /* logo tidak boleh gepeng bila tidak persegi */
	}
</style>
