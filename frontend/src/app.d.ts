// Tipe global SvelteKit. Lihat https://svelte.dev/docs/kit/types#app.d.ts
import type { User } from '$lib/types';

declare global {
	namespace App {
		interface Locals {
			/** Pengguna yang sedang masuk (dibaca hooks.server.ts dari /api/auth/me); null = belum masuk. */
			user: User | null;
			/** ID permintaan, diteruskan ke Go API sebagai X-Request-Id supaya log keduanya bisa dikaitkan. */
			requestId: string;
		}
		interface Error {
			message: string;
			requestId?: string;
		}
	}
}

export {};
