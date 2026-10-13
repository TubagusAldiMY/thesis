<!-- Badge status periode: selalu teks + warna (tidak mengandalkan warna saja). -->
<script lang="ts">
	import { labelStatus, warnaStatus } from '$lib/format';
	import type { StatusPeriode } from '$lib/types';

	let { status }: { status: StatusPeriode } = $props();
</script>

<span class="badge {warnaStatus[status]}">{labelStatus[status]}</span>
