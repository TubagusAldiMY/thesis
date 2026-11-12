#!/usr/bin/env bash
# seed-mustahik-100.sh — isi 100 data mustahik lewat Go API, untuk demo volume
# data ke pihak masjid (daftar lengkap + fitur Ekspor CSV/Excel di /mustahik).
#
# 46 nama pertama diambil dari "DAFTAR JANDA YATIM DUAFA.xlsx" (daftar janda/
# yatim/duafa riil di root repo); 54 nama berikutnya adalah tambahan FIKTIF
# bergaya nama lokal yang sama (Ibu/Bapak/Mpo + nama) untuk menggenapi 100,
# BUKAN data orang sungguhan. NIK, No. HP, dan alamat semuanya sintetis
# (dibuat dari nomor urut) karena excel aslinya hanya berisi kolom nama.
#
# Prasyarat: server Go berjalan dengan COOKIE_SECURE=false (curl lewat
# http://), akun admin sudah dibuat (cmd/useradmin), .env sudah dimuat:
#   set -a && . ./.env && set +a && ./scripts/seed-mustahik-100.sh
# Variabel: API_URL (bawaan http://127.0.0.1:8090), DEMO_USER (bawaan admin),
# DEMO_PASSWORD (wajib). Dijalankan dua kali → NIK dobel ditolak (422), aman
# (skrip berhenti di baris yang gagal, tidak ada data yang rusak).
set -euo pipefail

API="${API_URL:-http://127.0.0.1:8090}"
USER_DEMO="${DEMO_USER:-admin}"
: "${DEMO_PASSWORD:?isi DEMO_PASSWORD (lihat .env)}"

JAR="$(mktemp)"
trap 'rm -f "$JAR"' EXIT

# minta METHOD PATH [JSON] — cetak body; gagal bila status bukan 2xx.
minta() {
	local out status
	out="$(curl -sS -b "$JAR" -c "$JAR" -X "$1" -H 'Content-Type: application/json' \
		${3:+--data "$3"} -w '\n%{http_code}' "$API$2")"
	status="${out##*$'\n'}"
	out="${out%$'\n'*}"
	if [[ "$status" != 2* ]]; then
		echo "GAGAL $1 $2 → $status: $out" >&2
		return 1
	fi
	printf '%s' "$out"
}

echo "→ masuk sebagai $USER_DEMO"
minta POST /api/auth/login "{\"username\":\"$USER_DEMO\",\"password\":\"$DEMO_PASSWORD\"}" >/dev/null

# "Nama|Asnaf" — 1..46 dari excel (duafa/janda/yatim → fakir/miskin),
# 47..100 tambahan fiktif, golongan diputar ke semua asnaf supaya chip
# filter di /mustahik punya contoh tiap golongan.
DATA=(
	"Ibu Rohaya B. Mukhlis|miskin"
	"Mpo Mul|miskin"
	"Ibu Elis|miskin"
	"Ibu Tinah / Neneng|fakir"
	"Ibu Enca|miskin"
	"Ibu Ida Jubaedah|miskin"
	"Ibu Rohaya B. Reyab|miskin"
	"Ibu Neni Nampur|fakir"
	"Ibu Misa / Donte|miskin"
	"Bapak Saban|miskin"
	"Bapak Gerok|miskin"
	"Tinah|fakir"
	"Yanih|miskin"
	"Rodiah|miskin"
	"Dijah|miskin"
	"Nesih|fakir"
	"Try Yana|miskin"
	"Jenap|miskin"
	"Kamah|miskin"
	"Dahlia|fakir"
	"Sukarliah|miskin"
	"Rosmini|miskin"
	"Daryanti|miskin"
	"Aisah|fakir"
	"Siti Mariam|miskin"
	"Saityah|miskin"
	"Nenih|miskin"
	"Nonih|fakir"
	"Sri Sutiana|miskin"
	"Urip Rahayu Ningsih|miskin"
	"Bapak (Stroke)|miskin"
	"Mpo Arnah|fakir"
	"Mpo Aram|miskin"
	"Ibu Etha Ali|miskin"
	"Ibu Eni Gocap|miskin"
	"Ibu Engkung|fakir"
	"Ibu Nemi|miskin"
	"Mpo Marjanih|miskin"
	"Moh. Risky|miskin"
	"Rasya Muh. At Haya|fakir"
	"Sapa|miskin"
	"Marwah|miskin"
	"Erisca Della Agustine|miskin"
	"Semi|fakir"
	"Husna Malika Putri|miskin"
	"Fitri|miskin"
	"Ibu Sumiati|fakir"
	"Bapak Jumadi|miskin"
	"Mpo Inah|amil"
	"Ibu Warsih|muallaf"
	"Bapak Udin|riqab"
	"Siti Khadijah|gharim"
	"Nurhayati|fisabilillah"
	"Sarinah|ibnu_sabil"
	"Juminah|fakir"
	"Karsih|miskin"
	"Marsiah|amil"
	"Bapak Komar|muallaf"
	"Ibu Ocoh|riqab"
	"Mpo Icih|gharim"
	"Ibu Ratminah|fisabilillah"
	"Bapak Sukro|ibnu_sabil"
	"Ibu Suryati|fakir"
	"Dedeh Kurniasih|miskin"
	"Yati Rohayati|amil"
	"Ibu Nining|muallaf"
	"Bapak Salim|riqab"
	"Mpo Emeh|gharim"
	"Ibu Masnah|fisabilillah"
	"Bapak Jajang|ibnu_sabil"
	"Siti Rohani|fakir"
	"Entin Suhartini|miskin"
	"Ibu Komalasari|amil"
	"Bapak Warso|muallaf"
	"Mpo Encum|riqab"
	"Ibu Juhairiyah|gharim"
	"Suhaemi|fisabilillah"
	"Darminah|ibnu_sabil"
	"Ibu Carinah|fakir"
	"Bapak Rohman|miskin"
	"Yayah Sopiah|amil"
	"Ibu Mursidah|muallaf"
	"Mpo Nonoh|riqab"
	"Bapak Dulah|gharim"
	"Ibu Tarminah|fisabilillah"
	"Siti Fatimah|ibnu_sabil"
	"Rohimah|fakir"
	"Ibu Wartinah|miskin"
	"Bapak Engkos|amil"
	"Mpo Esih|muallaf"
	"Ibu Jamilah|riqab"
	"Bapak Jalaludin|gharim"
	"Siti Nurjanah|fisabilillah"
	"Ratna Juwita|ibnu_sabil"
	"Ibu Warkonah|fakir"
	"Bapak Samad|miskin"
	"Mpo Encih|amil"
	"Ibu Kalsum|muallaf"
	"Siti Aminah Nasution|riqab"
	"Bapak Hasbullah|gharim"
)

i=0
for baris in "${DATA[@]}"; do
	i=$((i + 1))
	IFS='|' read -r nama asnaf <<<"$baris"
	nik="97$(printf '%014d' "$i")" # sintetis, bukan NIK asli
	rt=$(printf '%02d' $(((i - 1) % 10 + 1)))
	rw=$(printf '%02d' $(((i - 1) % 3 + 1)))
	alamat="RT $rt/RW $rw, Lingkungan Masjid Jami Al-Barokah"
	body="{\"nama\":\"$nama\",\"nik\":\"$nik\",\"alamat\":\"$alamat\",\"asnaf\":\"$asnaf\""
	if ((i % 2 == 0)); then
		body+=",\"no_hp\":\"0812$(printf '%08d' "$i")\""
	fi
	body+="}"
	echo "→ mustahik $i/100: $nama ($asnaf)"
	minta POST /api/mustahik "$body" >/dev/null
done

echo "Selesai. 100 mustahik tersimpan. Lihat /mustahik, coba tombol \"Unduh CSV\"."
