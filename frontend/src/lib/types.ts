// Tipe data = bentuk JSON dari Go API (lihat tag json di backend/internal/domain
// dan backend/internal/service). Uang = rupiah utuh, bobot = pecahan 0–1,
// waktu = string ISO 8601.

export type Role = 'admin' | 'ketua';
export type Asnaf =
	| 'fakir'
	| 'miskin'
	| 'amil'
	| 'muallaf'
	| 'riqab'
	| 'gharim'
	| 'fisabilillah'
	| 'ibnu_sabil';
export type StatusPeriode = 'draft' | 'dihitung' | 'disetujui' | 'ditolak' | 'selesai';
export type StatusHasil = 'usulan' | 'cadangan';

export interface User {
	id: string;
	username: string;
	nama: string;
	role: Role;
}

export interface Mustahik {
	id: string;
	nama: string;
	nik_tersamar: string;
	no_hp_tersamar: string;
	alamat: string;
	asnaf: Asnaf;
	aktif: boolean;
	dibuat_pada: string;
}

export interface MustahikHalaman {
	items: Mustahik[];
	total: number;
	asnaf_counts: Partial<Record<Asnaf, number>>;
}

export interface Kriteria {
	id: string;
	kode: string;
	nama: string;
	tipe: 'benefit' | 'cost';
	bobot: number;
	skala: Record<string, string> | null;
	urutan: number;
	aktif: boolean;
}

export interface Periode {
	id: string;
	nama: string;
	jenis: 'zakat' | 'infak';
	dana_tersedia: number;
	nominal_per_penerima: number;
	asnaf_diizinkan: Asnaf[] | null;
	status: StatusPeriode;
	created_by: string;
	created_at: string;
	updated_at: string;
	kuota?: number;
}

export interface RunRingkas {
	versi: number;
	is_current: boolean;
	dibuat_pada: string;
	dijalankan_oleh: string;
	jumlah_kandidat: number;
	teratas: string;
	teratas_k: number;
}

export interface Persetujuan {
	run_id: string;
	versi: number;
	keputusan: 'setuju' | 'tolak';
	catatan: string;
	oleh: string;
	oleh_nama: string;
	created_at: string;
}

export interface PeriodeDetail {
	periode: Periode;
	kuota: number;
	runs: RunRingkas[];
	keputusan: Persetujuan[];
	catatan_penolakan: string;
}

export interface BarisPenilaian {
	mustahik: Mustahik;
	nilai: Record<string, number>;
	lengkap: boolean;
}

export interface PenilaianMatriks {
	kriteria: Kriteria[];
	baris: BarisPenilaian[];
	lengkap: number;
	total: number;
}

export interface BobotKriteria {
	kode: string;
	tipe: 'benefit' | 'cost';
	bobot: number;
}

export interface ArasRun {
	id: string;
	periode_id: string;
	versi: number;
	bobot: BobotKriteria[];
	s0: number;
	is_current: boolean;
	dijalankan_oleh: string;
	created_at: string;
}

export interface HasilItem {
	mustahik: Mustahik;
	s: number;
	k: number;
	ranking: number;
	status: StatusHasil;
}

export interface Langkah {
	kriteria: string[];
	alternatif: string[];
	a0: number[];
	matriks: number[][];
	normalisasi: number[][];
	terbobot: number[][];
}

export interface HasilAras {
	periode: Periode;
	run: ArasRun;
	kuota: number;
	items: HasilItem[];
	langkah: Langkah | null;
}

export interface Penyaluran {
	id: string;
	periode_id: string;
	mustahik_id: string;
	nominal: number;
	tanggal: string;
	dicatat_oleh: string;
	created_at: string;
}

export interface BarisPenyaluran {
	mustahik: Mustahik;
	ranking: number;
	k: number;
	status: StatusHasil;
	penyaluran: Penyaluran | null;
}

export interface RekapPenyaluran {
	periode: Periode;
	baris: BarisPenyaluran[];
	jumlah_penerima: number;
	tercatat: number;
	total_tersalur: number;
	sisa_dana: number;
}

export interface Dashboard {
	mustahik_aktif: number;
	asnaf_counts: Partial<Record<Asnaf, number>>;
	periode_berjalan: number;
	aktif: {
		periode: Periode;
		kuota: number;
		langkah_aktif: number;
		jumlah_kandidat: number;
		penilaian_lengkap: number;
		total_tersalur: number;
	} | null;
	terbaru: Periode[];
	/** Realisasi semua periode (rupiah) — dipakai kartu Total tersalur saat tidak ada periode aktif. */
	total_tersalur_semua: number;
}

export interface Laporan {
	periode: Periode;
	kuota: number;
	run: ArasRun;
	kriteria: { kode: string; nama: string; tipe: 'benefit' | 'cost'; bobot: number }[];
	ranking: HasilItem[];
	penyaluran: BarisPenyaluran[];
	total_tersalur: number;
	admin_nama: string;
	ketua_nama: string;
	disetujui_pada: string | null;
	dibuat_pada: string;
}

export interface Audit {
	id: string;
	actor_id: string;
	actor_nama: string;
	action: string;
	resource: string;
	resource_id: string;
	detail: Record<string, unknown> | null;
	ip: string;
	created_at: string;
}

export interface AuditHalaman {
	items: Audit[];
	total: number;
	page: number;
	per_page: number;
}

/** Bentuk body error Go API. */
export interface ApiErrorBody {
	error: string;
	request_id?: string;
	fields?: Record<string, string>;
	sisa_percobaan?: number;
	retry_after?: number;
}
