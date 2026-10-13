<!-- Ikon kubah/gerbang masjid (garis), mengikuti warna teks induknya. -->
<script lang="ts">
	let { ukuran = 32 }: { ukuran?: number } = $props();
</script>

<svg width={ukuran} height={ukuran} viewBox="0 0 32 32" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
	<path d="M6 27V14a10 10 0 0 1 20 0v13" />
	<path d="M12 27v-6a4 4 0 0 1 8 0v6" />
	<path d="M3 27h26" />
</svg>
