<!-- Halaman login (desain: Login.png): panel hijau kiri + formulir kanan. -->
<script lang="ts">
	import { enhance } from '$app/forms';
	import IkonMasjid from '$lib/components/IkonMasjid.svelte';
	import { masjid } from '$lib/masjid';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
	let mengirim = $state(false);

	const pesan = $derived.by(() => {
		if (!form?.error) return '';
		if (form.retryAfter) return `${form.error}.`;
		if (form.sisa !== null && form.sisa !== undefined) return `${form.error}. Sisa percobaan: ${form.sisa} dari 5.`;
		return form.error;
	});
</script>

<svelte:head><title>Masuk · SPK Zakat</title></svelte:head>

<div class="halaman">
	<section class="panel">
		<div class="merek">
			<IkonMasjid ukuran={40} />
			<div>
				<strong>{masjid.nama}</strong>
				<small>Sistem Pendukung Keputusan</small>
			</div>
		</div>
		<div class="slogan">
			<h1>Penyaluran zakat & infak yang adil dan bisa dipertanggungjawabkan.</h1>
			<p>Calon penerima dinilai dengan kriteria berbobot dan diranking otomatis memakai metode ARAS, lalu disetujui Ketua DKM.</p>
		</div>
		<small class="kaki">Metode ARAS · Additive Ratio Assessment</small>
	</section>

	<main class="isi">
		<form
			method="POST"
			class="formulir"
			use:enhance={() => {
				mengirim = true;
				return async ({ update }) => {
					await update({ reset: false });
					mengirim = false;
				};
			}}
		>
			<div>
				<h2>Masuk</h2>
				<p class="redup">Gunakan akun pengurus (Admin atau Ketua DKM).</p>
			</div>

			{#if data.sesiHabis && !form}
				<p class="kotak-info" role="status">Sesi Anda sudah berakhir. Silakan masuk kembali.</p>
			{/if}

			<div class="field">
				<label for="username">Username</label>
				<input id="username" name="username" type="text" autocomplete="username" placeholder="mis. admin" required maxlength="50" value={form?.username ?? ''} />
			</div>
			<div class="field">
				<label for="password">Password</label>
				<input
					id="password"
					name="password"
					type="password"
					autocomplete="current-password"
					required
					maxlength="128"
					aria-invalid={Boolean(pesan)}
					aria-describedby={pesan ? 'galat-login' : undefined}
				/>
				{#if pesan}
					<span id="galat-login" class="galat" role="alert">{pesan}</span>
				{/if}
			</div>

			<button class="tombol utama" type="submit" disabled={mengirim}>{mengirim ? 'Memeriksa…' : 'Masuk'}</button>
			<p class="redup kecil">Lupa password? Hubungi Admin masjid untuk reset akun.</p>
		</form>
	</main>
</div>

<style>
	.halaman {
		min-height: 100vh;
		display: grid;
		grid-template-columns: 1fr 1fr;
	}
	.panel {
		background: var(--hijau);
		color: #fff;
		padding: 3.5rem 4.5rem;
		display: grid;
		grid-template-rows: auto 1fr auto;
		gap: 2rem;
	}
	.merek {
		display: flex;
		gap: 0.9rem;
		align-items: center;
		font-size: 1.15rem;
	}
	.merek small {
		display: block;
		opacity: 0.85;
		font-size: 0.9rem;
	}
	.slogan {
		align-self: center;
		display: grid;
		gap: 1.25rem;
	}
	.slogan h1 {
		font-size: clamp(2rem, 3.8vw, 3.3rem);
	}
	.slogan p {
		opacity: 0.9;
		font-size: 1.1rem;
		max-width: 46ch;
	}
	.kaki {
		opacity: 0.8;
	}
	.isi {
		display: grid;
		place-items: center;
		padding: 2rem 1rem;
	}
	.formulir {
		width: min(100%, 470px);
		display: grid;
		gap: 1.25rem;
	}
	.formulir h2 {
		font-size: 2.2rem;
		margin-bottom: 0.4rem;
	}
	.kecil {
		font-size: 0.9rem;
	}
	@media (max-width: 860px) {
		.halaman {
			grid-template-columns: 1fr;
		}
		.panel {
			padding: 1.25rem 1rem;
			grid-template-rows: auto;
		}
		.slogan,
		.kaki {
			display: none;
		}
		.isi {
			place-items: start center;
		}
	}
</style>
