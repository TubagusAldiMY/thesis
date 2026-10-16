// Jadwal salat dihitung di aplikasi (tanpa API luar, jadi tetap jalan saat offline
// dan tidak mengirim lokasi pengunjung ke pihak lain). Rumus posisi matahari
// standar (kalkulasi tipe PrayTimes) dengan parameter Kemenag RI: subuh 20°,
// isya 18°, ashar bayangan = tinggi benda (mazhab Syafii), maghrib = matahari
// terbenam. Hasil = menit sejak 00.00 pada zona waktu `zona`.

const rad = (d: number) => (d * Math.PI) / 180;
const deg = (r: number) => (r * 180) / Math.PI;
const sin = (d: number) => Math.sin(rad(d));
const cos = (d: number) => Math.cos(rad(d));
const fixSudut = (a: number) => ((a % 360) + 360) % 360;
const fixJam = (h: number) => ((h % 24) + 24) % 24;

export interface JadwalSalat {
	subuh: number;
	terbit: number;
	dzuhur: number;
	ashar: number;
	maghrib: number;
	isya: number;
}

/** Hari Julian (pukul 00.00 UT) untuk tanggal Gregorian. */
function julian(tahun: number, bulan: number, tanggal: number): number {
	if (bulan <= 2) {
		tahun -= 1;
		bulan += 12;
	}
	const a = Math.floor(tahun / 100);
	const b = 2 - a + Math.floor(a / 4);
	return Math.floor(365.25 * (tahun + 4716)) + Math.floor(30.6001 * (bulan + 1)) + tanggal + b - 1524.5;
}

/** Deklinasi matahari (derajat) dan persamaan waktu (jam) pada hari Julian jd. */
function posisiMatahari(jd: number): { deklinasi: number; persamaanWaktu: number } {
	const d = jd - 2451544.5;
	const g = fixSudut(357.529 + 0.98560028 * d);
	const q = fixSudut(280.459 + 0.98564736 * d);
	const l = fixSudut(q + 1.915 * sin(g) + 0.02 * sin(2 * g));
	const e = 23.439 - 0.00000036 * d;
	const ra = fixJam(deg(Math.atan2(cos(e) * sin(l), cos(l))) / 15);
	return { deklinasi: deg(Math.asin(sin(e) * sin(l))), persamaanWaktu: q / 15 - ra };
}

/**
 * Waktu salat untuk satu tanggal dan titik. lintang selatan negatif, bujur timur
 * positif; zona = selisih jam dari UTC (WIB = 7).
 */
export function jadwalSalat(tahun: number, bulan: number, tanggal: number, lintang: number, bujur: number, zona = 7): JadwalSalat {
	const jd = julian(tahun, bulan, tanggal) - bujur / 360; // 360 = 15 derajat/jam × 24 jam
	const tengahHari = (t: number) => fixJam(12 - posisiMatahari(jd + t).persamaanWaktu);

	// Jam saat matahari berada `sudut` derajat di bawah (+) / di atas (−) horizon.
	const waktuSudut = (sudut: number, t: number, sebelumZuhur: boolean) => {
		const dek = posisiMatahari(jd + t).deklinasi;
		const h = (1 / 15) * deg(Math.acos((-sin(sudut) - sin(dek) * sin(lintang)) / (cos(dek) * cos(lintang))));
		return tengahHari(t) + (sebelumZuhur ? -h : h);
	};
	const ashar = (t: number) => {
		const dek = posisiMatahari(jd + t).deklinasi;
		const sudut = -deg(Math.atan(1 / (1 + Math.tan(rad(Math.abs(lintang - dek)))))); // bayangan = 1 × tinggi benda
		return waktuSudut(sudut, t, false);
	};

	const koreksi = zona - bujur / 15; // dari waktu matahari lokal ke zona waktu
	const menit = (jam: number) => Math.round((jam + koreksi) * 60);
	return {
		subuh: menit(waktuSudut(20, 5 / 24, true)),
		terbit: menit(waktuSudut(0.833, 6 / 24, true)),
		dzuhur: menit(tengahHari(12 / 24)),
		ashar: menit(ashar(13 / 24)),
		maghrib: menit(waktuSudut(0.833, 18 / 24, false)),
		isya: menit(waktuSudut(18, 18 / 24, false))
	};
}

/** 699 → "11:39" */
export function jamMenit(menit: number): string {
	const m = ((menit % 1440) + 1440) % 1440;
	return `${String(Math.floor(m / 60)).padStart(2, '0')}:${String(m % 60).padStart(2, '0')}`;
}
