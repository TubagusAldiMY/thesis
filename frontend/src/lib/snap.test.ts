import { describe, expect, it } from 'vitest';
import { infoSnap } from './snap';

const token = '825c2bbf-24d3-4575-8e08-edbf7fba1ce6';

describe('infoSnap', () => {
	it.each([
		[`https://app.sandbox.midtrans.com/snap/v4/redirection/${token}`, 'https://app.sandbox.midtrans.com/snap/snap.js'],
		[`https://app.midtrans.com/snap/v4/redirection/${token}`, 'https://app.midtrans.com/snap/snap.js'],
		[`https://app.midtrans.com/snap/v4/redirection/${token}/`, 'https://app.midtrans.com/snap/snap.js']
	])('mengenali tautan Snap %s', (url, skrip) => {
		expect(infoSnap(url)).toEqual({ skrip, token });
	});

	it.each([
		['kosong', undefined],
		['bukan URL', 'bukan-url'],
		['http biasa', `http://app.sandbox.midtrans.com/snap/v4/redirection/${token}`],
		['host asing', `https://evil.example.com/snap/v4/redirection/${token}`],
		['host mirip', `https://app.sandbox.midtrans.com.evil.example/snap/v4/redirection/${token}`],
		['token bukan UUID', 'https://app.midtrans.com/snap/v4/redirection/abc'],
		['tanpa token', 'https://app.midtrans.com/']
	])('menolak %s', (_nama, url) => {
		expect(infoSnap(url)).toBeNull();
	});
});
