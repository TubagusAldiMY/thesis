<!-- Layout root: hanya memuat gaya global. Shell aplikasi (sidebar) ada di (app)/+layout.svelte. -->
<script lang="ts">
	import '../app.css';
	let { children } = $props();
</script>

{@render children()}
