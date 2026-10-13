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
		// transisi Svelte); skrip tetap hanya dari origin sendiri.
		csp: {
			mode: 'auto',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self', 'unsafe-inline'],
				'img-src': ['self', 'data:'],
				'font-src': ['self'],
				'connect-src': ['self'],
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
