// Format tampilan Indonesia (NFR-08): ribuan titik, desimal koma, tanggal
// dd/mm/yyyy. Nilai di API tetap angka & ISO 8601; hanya tampilan yang diubah.
// Juga label ramah untuk enum dari API (asnaf, status, aksi audit).

import type { Asnaf, JenisDonasi, StatusDonasi, StatusPeriode } from './types';

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
	TUTUP: 'Tutup periode',
	DONASI_TERIMA: 'Donasi web diterima',
	DONASI_CATAT: 'Catat donasi',
	DONASI_BATAL: 'Batalkan kuitansi'
};

export const labelJenisDonasi: Record<JenisDonasi, string> = {
	zakat_fitrah: 'Zakat Fitrah',
	zakat_mal: 'Zakat Mal',
	zakat_profesi: 'Zakat Profesi',
	infak: 'Infak'
};
export const semuaJenisDonasi = Object.keys(labelJenisDonasi) as JenisDonasi[];

export const labelStatusDonasi: Record<StatusDonasi, string> = {
	menunggu: 'Menunggu pembayaran',
	diterima: 'Diterima',
	gagal: 'Gagal',
	kedaluwarsa: 'Kedaluwarsa',
	dibatalkan: 'Dibatalkan'
};
export const warnaStatusDonasi: Record<StatusDonasi, string> = {
	menunggu: 'kuning',
	diterima: 'hijau',
	gagal: 'merah',
	kedaluwarsa: '',
	dibatalkan: 'merah'
};

const SATUAN = ['', 'satu', 'dua', 'tiga', 'empat', 'lima', 'enam', 'tujuh', 'delapan', 'sembilan', 'sepuluh', 'sebelas'];
const SKALA: [number, string][] = [
	[1e12, 'triliun'],
	[1e9, 'miliar'],
	[1e6, 'juta'],
	[1e3, 'ribu']
];

/** Eja bilangan bulat > 0 dalam bahasa Indonesia ("seratus dua puluh tiga"). */
function bilang(n: number): string {
	if (n < 12) return SATUAN[n];
	if (n < 20) return SATUAN[n - 10] + ' belas';
	if (n < 100) return SATUAN[Math.floor(n / 10)] + ' puluh' + (n % 10 ? ' ' + SATUAN[n % 10] : '');
	if (n < 200) return 'seratus' + (n > 100 ? ' ' + bilang(n - 100) : '');
	if (n < 1000) return SATUAN[Math.floor(n / 100)] + ' ratus' + (n % 100 ? ' ' + bilang(n % 100) : '');
	if (n < 2000) return 'seribu' + (n > 1000 ? ' ' + bilang(n - 1000) : '');
	for (const [nilai, nama] of SKALA) {
		if (n >= nilai) return bilang(Math.floor(n / nilai)) + ' ' + nama + (n % nilai ? ' ' + bilang(n % nilai) : '');
	}
	return '';
}

/** 125000 → "Seratus dua puluh lima ribu rupiah" (baris "Terbilang" pada kuitansi). */
export function terbilang(n: number): string {
	if (!Number.isSafeInteger(n) || n < 0) throw new RangeError('terbilang: hanya bilangan bulat ≥ 0');
	const t = n === 0 ? 'nol' : bilang(n);
	return t.charAt(0).toUpperCase() + t.slice(1) + ' rupiah';
}

/**
 * Nomor HP → format wa.me (tanpa +): "0812-3456-7890" → "6281234567890",
 * "+62 812…" tetap, "+1 555…" tetap (nomor luar negeri harus berawalan +).
 * null bila tidak bisa dipastikan sebagai nomor.
 */
export function nomorWA(hp: string): string | null {
	const d = hp.replace(/\D/g, '');
	let n: string;
	if (hp.trim().startsWith('+')) n = d;
	else if (d.startsWith('0')) n = '62' + d.slice(1);
	else if (d.startsWith('62')) n = d;
	else return null;
	return /^\d{9,15}$/.test(n) ? n : null;
}

const labelMetode: Record<string, string> = {
	tunai: 'Tunai',
	transfer: 'Transfer bank',
	qris: 'QRIS',
	gopay: 'GoPay',
	shopeepay: 'ShopeePay',
	bank_transfer: 'Transfer bank (virtual account)',
	echannel: 'Mandiri Bill',
	credit_card: 'Kartu kredit/debit',
	cstore: 'Gerai retail'
};

/** Metode petugas atau payment_type gateway → label ramah; nilai tak dikenal tampil apa adanya, kosong → "-". */
export function namaMetode(m: string): string {
	return labelMetode[m] ?? (m || '-');
}

/** Tautan "kirim via WhatsApp" dengan teks terisi; null bila nomornya tidak valid. */
export function tautanWA(hp: string, teks: string): string | null {
	const n = nomorWA(hp);
	return n ? `https://wa.me/${n}?text=${encodeURIComponent(teks)}` : null;
}
