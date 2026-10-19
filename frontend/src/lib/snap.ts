// Pembantu Snap popup Midtrans. Tautan bayar yang disimpan Go berbentuk
// https://app.sandbox.midtrans.com/snap/v4/redirection/<token>, jadi alamat snap.js
// dan token popup cukup diturunkan dari situ (tanpa kolom/ubah skema baru).
// ponytail: bergantung pada bentuk tautan Midtrans; bila berubah, infoSnap null dan
// halaman jatuh ke pengalihan biasa (tetap berfungsi), lalu simpan `token` di DB.

/** Hanya dua host ini yang boleh dimuat skripnya (selaras dengan CSP di svelte.config.js). */
const hostSnap = new Set(['app.sandbox.midtrans.com', 'app.midtrans.com']);
const reToken = /^[0-9a-f-]{36}$/;

export interface InfoSnap {
	/** alamat snap.js untuk lingkungan yang sama dengan transaksi (sandbox/production) */
	skrip: string;
	token: string;
}

/** null bila tautan bukan tautan Snap Midtrans yang dikenali. */
export function infoSnap(bayarUrl: string | undefined): InfoSnap | null {
	if (!bayarUrl) return null;
	let u: URL;
	try {
		u = new URL(bayarUrl);
	} catch {
		return null;
	}
	if (u.protocol !== 'https:' || !hostSnap.has(u.hostname)) return null;
	const token = u.pathname.split('/').filter(Boolean).pop() ?? '';
	if (!reToken.test(token)) return null;
	return { skrip: `${u.origin}/snap/snap.js`, token };
}
