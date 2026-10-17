// Pembagian dana periode untuk kartu "Kuota penerima" di formulir Buat periode.
//
// Cermin rumus di backend (service.BuatPeriode): hanya untuk tampilan langsung saat mengetik. Yang menentukan
// tetap server Go; kalau aturannya berubah di sana, ubah juga di sini (dan tesnya).
//
// Dua cara membagi dana, pilih salah satu:
//   'nominal' → pengurus mengisi nominal per penerima; kuota = ⌊dana ÷ nominal⌋.
//   'jumlah'  → pengurus mengisi jumlah penerima;      nominal = ⌊dana ÷ jumlah⌋, kuota harus tepat = jumlah.

export type CaraBagi = 'nominal' | 'jumlah';

export interface Pembagian {
	/** Nominal tiap penerima (diisi pengurus, atau dihitung dari jumlah). 0 bila belum bisa dihitung. */
	nominal: number;
	/** Jumlah penerima yang bisa dibantu. 0 bila isian belum lengkap atau tidak sah. */
	kuota: number;
	/** Total dana yang terpakai = kuota × nominal. */
	terpakai: number;
	/** Dana yang tersisa karena pembagian tidak habis. */
	sisa: number;
	/** Alasan isian jumlah tidak bisa dipenuhi (kosong bila tidak ada masalah). */
	peringatan: string;
}

const kosong: Pembagian = { nominal: 0, kuota: 0, terpakai: 0, sisa: 0, peringatan: '' };

/** `isian` = nominal per penerima (cara 'nominal') atau jumlah penerima (cara 'jumlah'). NaN/≤ 0 dianggap belum diisi. */
export function hitungPembagian(cara: CaraBagi, dana: number, isian: number): Pembagian {
	if (!(dana > 0) || !(isian > 0)) return kosong;

	if (cara === 'nominal') {
		const kuota = Math.floor(dana / isian);
		return { nominal: isian, kuota, terpakai: kuota * isian, sisa: kuota > 0 ? dana - kuota * isian : 0, peringatan: '' };
	}

	const jumlah = Math.floor(isian); // jumlah orang selalu bilangan bulat
	const nominal = Math.floor(dana / jumlah);
	if (nominal < 1) return { ...kosong, peringatan: 'Jumlah penerima melebihi dana (tiap penerima minimal Rp 1).' };
	// Pembulatan nominal ke bawah bisa menambah kuota (Rp 10 ÷ 4 = Rp 2, lalu 10 ÷ 2 = 5 orang): tolak, seperti server.
	if (Math.floor(dana / nominal) !== jumlah) return { ...kosong, nominal, peringatan: `Dana terlalu kecil untuk dibagi rata ke ${jumlah} penerima.` };
	return { nominal, kuota: jumlah, terpakai: jumlah * nominal, sisa: dana - jumlah * nominal, peringatan: '' };
}
