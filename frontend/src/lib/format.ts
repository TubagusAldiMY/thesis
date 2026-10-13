// Format tampilan Indonesia (NFR-08): ribuan titik, desimal koma, tanggal
// dd/mm/yyyy. Nilai di API tetap angka & ISO 8601; hanya tampilan yang diubah.
// Juga label ramah untuk enum dari API (asnaf, status, aksi audit).

import type { Asnaf, StatusPeriode } from './types';

const fmtAngka = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 });

/** 3000000 → "3.000.000" */
export function angka(n: number): string {
	return fmtAngka.format(n);
}

/** 3000000 → "Rp 3.000.000" */
export function rupiah(n: number): string {
	return 'Rp ' + angka(n);
}

/** 3000000 → "Rp 3,0 jt" (kartu ringkasan); di bawah sejuta tampil penuh. */
export function rupiahRingkas(n: number): string {
	if (Math.abs(n) < 1_000_000) return rupiah(n);
	return 'Rp ' + desimal(n / 1_000_000, 1) + ' jt';
}

/** 0.89391 → "0,8939" (digit desimal tetap). */
export function desimal(x: number, digit = 4): string {
	return new Intl.NumberFormat('id-ID', {
		minimumFractionDigits: digit,
		maximumFractionDigits: digit
	}).format(x);
}

const fmtTanggal = new Intl.DateTimeFormat('id-ID', {
	day: '2-digit',
	month: '2-digit',
	year: 'numeric',
	timeZone: 'Asia/Jakarta'
});
const fmtWaktu = new Intl.DateTimeFormat('id-ID', {
	day: '2-digit',
	month: '2-digit',
	year: 'numeric',
	hour: '2-digit',
	minute: '2-digit',
	timeZone: 'Asia/Jakarta'
});

/** Waktu ISO → "12/10/2026" menurut WIB. */
export function tanggal(iso: string): string {
	return fmtTanggal.format(new Date(iso));
}

/** Waktu ISO → "12/10/2026 18.20" menurut WIB. */
export function waktu(iso: string): string {
	return fmtWaktu.format(new Date(iso));
}

/** Tanggal tanpa jam ("2026-10-12" atau "2026-10-12T00:00:00Z") → "12/10/2026", tanpa geser zona waktu. */
export function tanggalSaja(iso: string): string {
	const [y, m, d] = iso.slice(0, 10).split('-');
	return `${d}/${m}/${y}`;
}

/** Hari ini sebagai "YYYY-MM-DD" menurut WIB (nilai bawaan input tanggal). */
export function hariIniISO(): string {
	return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Jakarta' }).format(new Date());
}

/** "3.000.000" / "3000000" / "Rp 3.000.000" → 3000000. Kosong/tidak ada digit → NaN. */
export function parseAngka(s: string): number {
	const digit = s.replace(/\D/g, '');
	return digit === '' ? NaN : Number(digit);
}

/**
 * Angka gaya Indonesia: titik = ribuan, koma = desimal.
 * "1.500.000" → 1500000, "2,5" → 2.5, "800000" → 800000. Tidak valid → NaN.
 * Dipakai nilai penilaian (penghasilan bisa besar, tanggungan bisa kecil).
 */
export function parseNilai(s: string): number {
	const t = s.trim().replace(/\./g, '').replace(',', '.');
	return /^\d+(\.\d+)?$/.test(t) ? Number(t) : NaN;
}

/** "0,30" atau "0.30" → 0.3. Tidak valid → NaN. */
export function parseDesimal(s: string): number {
	const t = s.trim().replace(',', '.');
	return /^\d+(\.\d+)?$/.test(t) ? Number(t) : NaN;
}

export const labelAsnaf: Record<Asnaf, string> = {
	fakir: 'Fakir',
	miskin: 'Miskin',
	amil: 'Amil',
	muallaf: 'Muallaf',
	riqab: 'Riqab',
	gharim: 'Gharim',
	fisabilillah: 'Fisabilillah',
	ibnu_sabil: 'Ibnu sabil'
};
export const semuaAsnaf = Object.keys(labelAsnaf) as Asnaf[];

export const labelStatus: Record<StatusPeriode, string> = {
	draft: 'Draft',
	dihitung: 'Dihitung',
	disetujui: 'Disetujui',
	ditolak: 'Ditolak',
	selesai: 'Selesai'
};

/** Warna badge status (status selalu tampil sebagai teks + warna). */
export const warnaStatus: Record<StatusPeriode, string> = {
	draft: '',
	dihitung: 'kuning',
	disetujui: 'hijau',
	ditolak: 'merah',
	selesai: ''
};

export const labelAksi: Record<string, string> = {
	LOGIN: 'Masuk',
	LOGIN_GAGAL: 'Gagal masuk',
	LOGOUT: 'Keluar',
	PASSWORD_UBAH: 'Ganti password',
	MUSTAHIK_BUAT: 'Tambah mustahik',
	MUSTAHIK_UBAH: 'Ubah mustahik',
	KRITERIA_UBAH: 'Ubah bobot',
	PERIODE_BUAT: 'Buat periode',
	KANDIDAT: 'Atur kandidat',
	PENILAIAN: 'Isi penilaian',
	HITUNG: 'Hitung ARAS',
	SETUJU: 'Setujui',
	TOLAK: 'Tolak',
	BUKA_KEMBALI: 'Buka kembali',
	SALUR: 'Catat penyaluran',
	TUTUP: 'Tutup periode'
};
