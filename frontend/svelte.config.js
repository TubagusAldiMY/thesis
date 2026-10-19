import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// adapter-node: hasil build = server Node biasa (`node build`), cocok untuk
		// satu server masjid / VPS kecil.
		adapter: adapter(),
		// CSP dibuat otomatis oleh SvelteKit (hash/nonce untuk skrip miliknya).
		// style-src perlu 'unsafe-inline' karena atribut style (lebar bar K,
		// transisi Svelte); skrip tetap hanya dari origin sendiri, KECUALI snap.js Midtrans
		// (popup donasi, lihat donasi/lanjut): hanya dua host Midtrans, bukan wildcard.
		csp: {
			mode: 'auto',
			directives: {
				'default-src': ['self'],
				'script-src': ['self', 'https://app.sandbox.midtrans.com', 'https://app.midtrans.com'],
				'style-src': ['self', 'unsafe-inline', 'https://*.midtrans.com'],
				'img-src': ['self', 'data:', 'https://*.midtrans.com'],
				'font-src': ['self', 'https://*.midtrans.com'],
				'connect-src': ['self', 'https://*.midtrans.com'],
				'frame-src': ['https://app.sandbox.midtrans.com', 'https://app.midtrans.com'],
				'frame-ancestors': ['none'],
				'form-action': ['self'],
				'base-uri': ['self'],
				'object-src': ['none']
			}
		}
		// csrf.checkOrigin bawaan = true: form action dari origin lain ditolak.
	}
};

export default config;
