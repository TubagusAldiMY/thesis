#!/usr/bin/env bash
# terapkan-kriteria-final.sh — pasang kriteria hasil konfirmasi Ketua Yayasan (Oktober 2026) lewat API.
#
# Isi konfigurasi final:
#   C1 Penghasilan   benefit  0,30  SKALA TINGKAT 1–4 (4 = paling membutuhkan); dulu rupiah bertipe cost
#   C2 Tanggungan    benefit  0,30  (dulu 0,20)
#   C3 Tempat tinggal benefit 0,10  (dulu 0,20)
#   C4 Kesehatan     benefit  0,15  (tetap)
#   C5 Pekerjaan     benefit  0,15  (tetap)
#
# Kenapa lewat API dan bukan migrasi: V002__seed_kriteria.sql sudah ter-apply di banyak database dan tidak boleh
# diedit; tes basis data memakainya sebagai fixture. Fitur Kriteria memang dibuat agar pengurus mengatur ini sendiri.
# Aman dijalankan berulang (hasil akhirnya sama).
#
# PERINGATAN: nilai C1 yang sudah terlanjur diisi dalam rupiah (mis. 800000) TIDAK dikonversi. Periode lama yang
# masih draft perlu dinilai ulang; periode yang sudah dihitung menyimpan potret bobot lamanya.
#
# Prasyarat sama dengan seed-demo.sh: server Go berjalan, akun admin ada, .env dimuat:
#   set -a && . ./.env && set +a && ./scripts/terapkan-kriteria-final.sh
# Variabel: API_URL (bawaan http://127.0.0.1:8090), DEMO_USER (bawaan admin), DEMO_PASSWORD (wajib).
set -euo pipefail

command -v jq >/dev/null || { echo "butuh jq (sudo apt install jq)" >&2; exit 1; }
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

echo "→ membaca kriteria sekarang"
sekarang="$(minta GET /api/kriteria)"
id_dari() { jq -r --arg k "$1" '.items[] | select(.kode == $k) | .id' <<<"$sekarang"; }
for k in C1 C2 C3; do
	[[ -n "$(id_dari "$k")" ]] || { echo "kriteria $k tidak ada di database" >&2; exit 1; }
done

# Tiap item memuat SEMUA kolom (penuh-ganti), sesuai kontrak PUT /api/kriteria.
badan="$(jq -n \
	--arg c1 "$(id_dari C1)" --arg c2 "$(id_dari C2)" --arg c3 "$(id_dari C3)" '
{items: [
  {id: $c1, nama: "Penghasilan per bulan", tipe: "benefit", bobot: 0.30, aktif: true,
   skala: {"1": "cukup",
           "2": "terbatas (kebutuhan pokok tercukupi, tetapi pas-pasan)",
           "3": "sangat terbatas (kebutuhan pokok sering tidak tercukupi)",
           "4": "tidak berpenghasilan / sangat tidak menentu"}},
  {id: $c2, nama: "Jumlah tanggungan", tipe: "benefit", bobot: 0.30, aktif: true, skala: null},
  {id: $c3, nama: "Kondisi tempat tinggal", tipe: "benefit", bobot: 0.10, aktif: true,
   skala: {"1": "layak, milik sendiri", "2": "layak, sewa/menumpang", "3": "kurang layak", "4": "tidak layak / tanpa tempat tinggal"}}
]}')"

echo "→ menyimpan kriteria final"
hasil="$(minta PUT /api/kriteria "$badan")"
jq -r '.items[] | select(.aktif) | "   \(.kode) \(.nama) — \(.tipe), bobot \(.bobot)"' <<<"$hasil"
echo "   total bobot: $(jq -r '.total_persen' <<<"$hasil")%"
