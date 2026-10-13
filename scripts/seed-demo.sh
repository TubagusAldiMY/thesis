#!/usr/bin/env bash
# seed-demo.sh — isi data contoh untuk demo sidang lewat Go API.
#
# Membuat 5 mustahik (contoh hitung docs/02: A1–A5, NIK sintetis), satu periode
# "Zakat Fitrah 1448 H" (dana Rp 3.000.000, Rp 1.000.000/penerima → kuota 3),
# memilih kelimanya sebagai kandidat, dan mengisi penilaian lengkap.
# Berhenti SEBELUM Hitung ARAS supaya perhitungannya bisa didemokan langsung.
#
# Prasyarat: server Go berjalan dengan COOKIE_SECURE=false (curl lewat http://),
# akun admin sudah dibuat (cmd/useradmin), .env sudah dimuat ke shell:
#   set -a && . ./.env && set +a && ./scripts/seed-demo.sh
# Variabel: API_URL (bawaan http://127.0.0.1:8090), DEMO_USER (bawaan admin),
# DEMO_PASSWORD (wajib). Dijalankan dua kali → NIK dobel ditolak (422), aman.
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
ambil_id() { sed -n 's/^{"id":"\([^"]*\)".*/\1/p'; }

echo "→ masuk sebagai $USER_DEMO"
minta POST /api/auth/login "{\"username\":\"$USER_DEMO\",\"password\":\"$DEMO_PASSWORD\"}" >/dev/null

# nama | asnaf | C1 penghasilan | C2 tanggungan | C3 | C4 | C5   (docs/02 bagian 3)
# Nama sengaja berurutan abjad (Aminah → Endang) supaya label A1–A5 di UI,
# yang mengikuti urutan nama, sama dengan label di dokumen.
DATA=(
	"Aminah|miskin|800000|5|3|2|3"
	"Badrun|fakir|500000|4|4|3|4"
	"Cahyono|miskin|1500000|3|2|1|2"
	"Dewi Sartika|fakir|600000|2|3|4|3"
	"Endang Susilo|miskin|1000000|6|2|2|1"
)

IDS=()
NILAI=()
i=0
for baris in "${DATA[@]}"; do
	i=$((i + 1))
	IFS='|' read -r nama asnaf c1 c2 c3 c4 c5 <<<"$baris"
	nik="99990000000000$(printf '%02d' "$i")" # sintetis, bukan NIK asli
	echo "→ mustahik A$i: $nama"
	id="$(minta POST /api/mustahik "{\"nama\":\"$nama\",\"nik\":\"$nik\",\"alamat\":\"Jl. Contoh No. $i, RT 0$i\",\"asnaf\":\"$asnaf\"}" | ambil_id)"
	IDS+=("\"$id\"")
	NILAI+=("{\"mustahik_id\":\"$id\",\"nilai\":{\"C1\":$c1,\"C2\":$c2,\"C3\":$c3,\"C4\":$c4,\"C5\":$c5}}")
done

echo "→ periode Zakat Fitrah 1448 H"
pid="$(minta POST /api/periode '{"nama":"Zakat Fitrah 1448 H","jenis":"zakat","dana_tersedia":3000000,"nominal_per_penerima":1000000,"asnaf_diizinkan":["fakir","miskin"]}' | ambil_id)"

echo "→ kandidat & penilaian"
minta PUT "/api/periode/$pid/kandidat" "{\"mustahik_ids\":[$(IFS=,; echo "${IDS[*]}")]}" >/dev/null
minta PUT "/api/periode/$pid/penilaian" "{\"items\":[$(IFS=,; echo "${NILAI[*]}")]}" >/dev/null

echo "Selesai. Buka /periode/$pid/penilaian lalu tekan \"Hitung ARAS\"."
echo "Hasil yang diharapkan (docs/02): A2 0,8939 · A4 0,7327 · A1 0,6888 (penerima) · A5 0,5611 · A3 0,4099 (cadangan)."
