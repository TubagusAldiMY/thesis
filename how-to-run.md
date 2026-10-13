# Cara Menjalankan

Aplikasi terdiri dari tiga bagian: **MySQL** (docker), **Go API** (`backend/`, port 8090), dan **frontend SvelteKit** (`frontend/`, port 5173 saat development). Browser hanya bicara ke SvelteKit; SvelteKit yang memanggil Go.

```
Browser ──► SvelteKit :5173 ──► Go API 127.0.0.1:8090 ──► MySQL 127.0.0.1:33070
```

## Prasyarat
| Kebutuhan | Versi | Cek |
|-----------|-------|-----|
| Go | 1.23 atau lebih baru | `go version` |
| Node.js | 20 LTS atau lebih baru (diuji dengan 24) | `node --version` |
| Docker + Docker Compose | Compose v2 | `docker compose version` |
| OpenSSL (membuat password/kunci acak) | bebas | `openssl version` |

Image `mysql:8.0` diunduh otomatis oleh `docker compose up` bila belum ada (±1 GB, sekali saja).

## Konfigurasi
Backend memakai `App/.env` (contoh: [.env.example](.env.example)); frontend memakai `App/frontend/.env` atau environment variable (contoh: [frontend/.env.example](frontend/.env.example)). Kedua `.env` berisi rahasia dan **tidak boleh di-commit** (sudah di `.gitignore`).

| Variabel | Dipakai | Fungsi |
|----------|---------|--------|
| `MYSQL_ROOT_PASSWORD` | docker | Password root MySQL di container (wajib) |
| `MYSQL_PORT` | docker | Port MySQL di host, hanya `127.0.0.1` (bawaan `33070`) |
| `DATABASE_URL` | Go | DSN aplikasi, mis. `root:PASSWORD@tcp(127.0.0.1:33070)/spkzakat` |
| `TEST_DATABASE_URL` | test Go | DSN test integrasi **tanpa nama database**. Kosong = test integrasi dilewati |
| `NIK_ENCRYPTION_KEY` | Go | Kunci AES-256 NIK/No. HP (32 byte base64). **Hilang = data terenkripsi tidak terbaca lagi**; simpan cadangannya terpisah dari database |
| `NIK_HMAC_KEY` | Go | Kunci HMAC pencocokan NIK, **berbeda** dari kunci enkripsi |
| `ADDR` | Go | Alamat listen API (bawaan `127.0.0.1:8090`, hanya lokal) |
| `COOKIE_SECURE` | Go | `false` hanya untuk development lewat curl/http; production `true` (bawaan) |
| `LOG_LEVEL` | Go | `debug`/`info`/`warn`/`error` (bawaan `info`) |
| `DEMO_PASSWORD` | skrip | Password akun demo lokal untuk `scripts/seed-demo.sh` |
| `API_URL` | SvelteKit | Alamat Go API (bawaan `http://127.0.0.1:8090`) |
| `PUBLIC_MASJID_NAMA`, `PUBLIC_MASJID_ALAMAT`, `PUBLIC_MASJID_KONTAK` | SvelteKit | Identitas masjid di sidebar, landing, dan kop laporan |
| `ORIGIN`, `PORT`, `ADDRESS_HEADER`, `XFF_DEPTH` | SvelteKit (production) | Lihat bagian Production |

## Development

Semua perintah di bawah dijalankan dari folder `App`, kecuali disebut lain.

### 1. Buat `.env` (sekali saja)

```bash
pw=$(openssl rand -hex 16)
printf "MYSQL_ROOT_PASSWORD=%s\nMYSQL_PORT=33070\nDATABASE_URL='root:%s@tcp(127.0.0.1:33070)/spkzakat'\nTEST_DATABASE_URL='root:%s@tcp(127.0.0.1:33070)/'\n" "$pw" "$pw" "$pw" > .env
printf 'NIK_ENCRYPTION_KEY=%s\nNIK_HMAC_KEY=%s\n' "$(openssl rand -base64 32)" "$(openssl rand -base64 32)" >> .env
printf 'ADDR=127.0.0.1:8090\nCOOKIE_SECURE=false\nDEMO_PASSWORD=%s\n' "$(openssl rand -hex 12)" >> .env
chmod 600 .env
```

### 2. Jalankan MySQL

```bash
docker compose --env-file .env up -d mysql
```

Tunggu sampai status `healthy`:

```bash
docker compose --env-file .env ps
```

### 3. Migrasi database

```bash
set -a && . ./.env && set +a
cd backend && go run ./cmd/migrate && cd ..
```

Hasil: log JSON `"migrasi selesai"`. Aman dijalankan berulang (`"diterapkan":[]` bila sudah terkini).

### 4. Buat akun Admin dan Ketua DKM (FR-17)
Tidak ada halaman kelola akun; akun dibuat lewat CLI. Password diminta tanpa tampil di layar (minimal 12 karakter).

```bash
set -a && . ./.env && set +a
cd backend && go run ./cmd/useradmin buat --username admin --nama "Ahmad Fauzi" --role admin && cd ..
```

```bash
cd backend && go run ./cmd/useradmin buat --username ketua --nama "H. Abdul Rahman" --role ketua && cd ..
```

Untuk akun **demo lokal** saja, password bisa diambil dari `DEMO_PASSWORD` di `.env` (stdin bukan terminal → dibaca satu baris):

```bash
printf '%s\n' "$DEMO_PASSWORD" | (cd backend && go run ./cmd/useradmin buat --username admin --nama "Ahmad Fauzi" --role admin)
```

Reset password (semua sesi akun itu ikut dicabut):

```bash
cd backend && go run ./cmd/useradmin reset-password --username admin && cd ..
```

### 5. Jalankan Go API

```bash
set -a && . ./.env && set +a
cd backend && go run ./cmd/server
```

Log JSON `"server berjalan","addr":"127.0.0.1:8090"`. Cek di terminal lain:

```bash
curl -s http://127.0.0.1:8090/healthz
```

Hasil: `{"ok":true}`. Matikan dengan Ctrl+C (graceful shutdown, menunggu permintaan berjalan maks. 30 detik).

### 6. Jalankan frontend
Terminal baru, dari `App/frontend`:

```bash
cd frontend && npm install && npm run dev
```

Buka <http://localhost:5173>. Masuk dengan akun dari langkah 4. Untuk mengganti nama/alamat masjid, salin `frontend/.env.example` ke `frontend/.env` lalu isi, kemudian jalankan ulang `npm run dev`.

### 7. (Opsional) Data contoh untuk demo sidang
Mengisi 5 mustahik dari contoh hitung `docs/02` + satu periode zakat (kuota 3) sampai penilaian lengkap, **tanpa** menghitung, supaya tombol "Hitung ARAS" bisa didemokan. Butuh Go API berjalan dengan `COOKIE_SECURE=false` dan akun admin dengan password `DEMO_PASSWORD`.

```bash
set -a && . ./.env && set +a && ./scripts/seed-demo.sh
```

Setelah menekan Hitung ARAS, hasil yang diharapkan: A2 0,8939 · A4 0,7327 · A1 0,6888 (penerima) · A5 0,5611 · A3 0,4099 (cadangan). Alur demo lengkap: Admin hitung → keluar → masuk sebagai Ketua → Hasil → Setujui → keluar → Admin → Penyaluran → Catat ×3 → Tutup periode → Laporan (Cetak/CSV).

### 8. Test
**Backend** (dari `App/backend`, `.env` sudah dimuat):

```bash
go test -race -count=1 -cover ./...
```

±5 menit karena tiap test integrasi membuat database `spkzakat_test_<acak>` sendiri lalu menghapusnya (database `spkzakat` tidak disentuh). Tanpa `TEST_DATABASE_URL` hanya test murni yang jalan. Iterasi cepat tanpa database:

```bash
go test -count=1 ./internal/aras/ ./internal/crypto/ ./internal/domain/
```

Smoke + alur demo lewat HTTP sungguhan (login → hitung → setujui → salurkan → laporan):

```bash
go test -count=1 -v ./internal/httpapi/
```

Format & analisis statis (tidak boleh mencetak apa pun):

```bash
gofmt -l . ; go vet ./...
```

**Frontend** (dari `App/frontend`):

```bash
npm test
```

```bash
npm run check
```

`npm test` = unit test format angka/tanggal Indonesia (vitest); `npm run check` = pemeriksaan tipe TypeScript + Svelte (harus `0 ERRORS 0 WARNINGS`).

### 9. Menghentikan MySQL

```bash
docker compose --env-file .env stop mysql
```

Data tetap di volume `spkzakat-mysql-data`. `docker compose --env-file .env down -v` **menghapus semua data dev**.

## Production
Ringkas (satu server masjid / VPS kecil):

1. **Database**: buat user MySQL khusus aplikasi (bukan `root`) dengan hak `SELECT, INSERT, UPDATE, DELETE` pada `spkzakat`; user terpisah dengan hak DDL untuk migrasi.
2. **Build Go** (dari `App/backend`):

   ```bash
   CGO_ENABLED=0 go build -o bin/server ./cmd/server && go build -o bin/migrate ./cmd/migrate && go build -o bin/useradmin ./cmd/useradmin
   ```

3. **Migrasi** sebelum server dijalankan: `DATABASE_URL=... ./bin/migrate`.
4. **Jalankan API** dengan `COOKIE_SECURE=true` (atau tanpa variabel itu), `ADDR=127.0.0.1:8090` (jangan diekspos ke internet), kunci `NIK_*` dari secret manager / file env ber-izin 600. Jalankan sebagai service (systemd) supaya hidup lagi setelah restart.
5. **Build frontend** (dari `App/frontend`):

   ```bash
   npm ci && npm run build
   ```

6. **Jalankan frontend**: `ORIGIN=https://alamat-publik PORT=3000 API_URL=http://127.0.0.1:8090 node build`. `ORIGIN` wajib sama dengan alamat yang dibuka pengguna (dipakai pemeriksaan CSRF).
7. **Reverse proxy HTTPS** (nginx/caddy) di depan port 3000 saja. Set `ADDRESS_HEADER=X-Forwarded-For` dan `XFF_DEPTH=1` supaya IP asli pengguna terbaca untuk penguncian login & audit.
8. **Health check**: `curl -s http://127.0.0.1:8090/healthz` → `{"ok":true}` (503 bila MySQL tidak menjawab).
9. **Log**: Go dan SvelteKit menulis JSON ke stdout (field `request_id` sama di keduanya). Kumpulkan dengan journald/Alloy → Loki bila perlu.

## Troubleshooting
| Gejala | Penyebab & solusi |
|--------|-------------------|
| `go: command not found` | Go belum terpasang / belum di `PATH`. Pasang dari <https://go.dev/dl/>. |
| `go.mod requires go >= 1.2x` atau Go mengunduh toolchain | Project dikunci ke Go 1.23. Jangan `go get ...@latest`; pakai `GOTOOLCHAIN=local` dan kembalikan `go 1.23` di `go.mod`. |
| `syntax error near unexpected token '('` saat `. ./.env` (bash) | Nilai `DATABASE_URL`/`TEST_DATABASE_URL` belum diberi tanda kutip tunggal. Ubah jadi `DATABASE_URL='root:…@tcp(127.0.0.1:33070)/spkzakat'` (zsh menerima tanpa kutip, bash tidak). |
| `required variable MYSQL_ROOT_PASSWORD is missing` | `.env` belum dibuat atau compose dijalankan tanpa `--env-file .env`. |
| `bind: address already in use` pada 33070 | Ubah `MYSQL_PORT` di `.env` (dan DSN), lalu `docker compose --env-file .env up -d mysql`. |
| Server: `listen tcp 127.0.0.1:8090: bind: address already in use` | Port dipakai proses lain (cek `ss -ltnp \| grep 8090`). Ubah `ADDR` di `.env` **dan** `API_URL` frontend. |
| Server: `konfigurasi tidak valid … NIK_ENCRYPTION_KEY` | `.env` belum dimuat (`set -a && . ./.env && set +a`) atau kunci bukan 32 byte base64. |
| Frontend: halaman "Layanan belum tersedia" (503) | Go API tidak berjalan atau `API_URL` salah. Cek `curl -s http://127.0.0.1:8090/healthz`. |
| Login selalu kembali ke halaman login | Di production tanpa HTTPS, cookie `Secure` tidak tersimpan. Pakai HTTPS (atau `npm run dev` untuk lokal). |
| Form ditolak "Cross-site POST form submissions are forbidden" | `ORIGIN` production tidak sama dengan alamat yang dibuka di browser. |
| "Terlalu banyak percobaan gagal" | 5 kali salah password untuk username+IP yang sama → terkunci 15 menit. Tunggu, atau restart Go API (penghitung disimpan di memori). |
| `seed-demo.sh`: `GAGAL POST /api/mustahik → 401` | Server Go dijalankan dengan `COOKIE_SECURE=true`; curl tidak mengirim cookie Secure lewat http. Jalankan server dengan `COOKIE_SECURE=false` untuk dev. |
| `seed-demo.sh`: `… 422 … NIK sudah terdaftar` | Data contoh sudah pernah dibuat. Pakai data yang ada, atau kosongkan database dev (`down -v`, migrasi ulang, buat akun ulang). |
| `TEST_DATABASE_URL kosong: test integrasi MySQL dilewati` | `.env` belum dimuat ke shell. |
| `migrate: isi migrasi yang sudah diterapkan berubah` | File `V*.sql` lama diedit. Kembalikan; perubahan skema = file migrasi **baru**. |
| `NIK mustahik … tidak terbaca (kunci enkripsi salah atau data rusak)` | Kunci di `.env` berbeda dari saat data dibuat. Pulihkan kunci aslinya. |
| Test gagal dengan selisih di digit ke-15/16 | Galat floating point; bandingkan dengan toleransi, jangan `==`. |
