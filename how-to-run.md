# Cara Menjalankan

Aplikasi terdiri dari tiga bagian: **MySQL** (docker), **Go API** (`backend/`, port 8090), dan **frontend SvelteKit** (`frontend/`, port 5173 saat development). Browser hanya bicara ke SvelteKit; SvelteKit yang memanggil Go.

```
Browser ──► SvelteKit :5173 ──► Go API 127.0.0.1:8090 ──► MySQL 127.0.0.1:33070
```

## Cara tercepat: semuanya di Docker (hidup lagi otomatis setelah laptop dinyalakan)

Tiga container (`spkzakat-mysql`, `spkzakat-api`, `spkzakat-web`) semuanya `restart: unless-stopped`, dan service Docker aktif saat boot, jadi setelah laptop dimatikan lalu dinyalakan lagi aplikasi menyala sendiri. Hanya perlu Docker + `.env` (lihat langkah 1 di bagian Development).

```bash
docker compose --env-file .env up -d --build
```

Buka <http://localhost:3101> (bukan `127.0.0.1`: `ORIGIN` harus sama persis dengan alamat di browser). Port 3101 dipilih karena 3100 dipakai Loki project lain di mesin ini; ganti lewat `WEB_PORT` **dan** `WEB_ORIGIN` di `.env` bila perlu. Go API hanya ada di jaringan internal compose (tidak dibuka ke host), migrasi dijalankan otomatis setiap API start.

Status, log, dan akun:

```bash
docker compose --env-file .env ps
docker compose --env-file .env logs -f api web
docker exec -it spkzakat-api /app/useradmin buat --username admin2 --nama "Nama Lengkap" --role admin
docker exec -it spkzakat-api /app/useradmin reset-password --username admin
```

Setelah mengubah kode: ulangi perintah `up -d --build`. Menghentikan tanpa menghapus data: `docker compose --env-file .env stop`. Jangan jalankan `go run ./cmd/server` / `npm run dev` bersamaan kecuali container `api`/`web` dihentikan dulu (port dan penghitung login berbeda).

Catatan: di Docker, IP klien yang tercatat di audit log dan penguncian login adalah gateway Docker (`172.30.0.1`), bukan IP browser, karena akses datang lewat port yang dipetakan ke host. Untuk satu laptop ini tidak masalah; di server sungguhan pasang reverse proxy (lihat Production).

## Prasyarat
| Kebutuhan | Versi | Cek |
|-----------|-------|-----|
| Go | 1.26.10 atau lebih baru | `go version` |
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
| `PAYMENT_PROVIDER` | Go | Donasi online: kosong = nonaktif, `midtrans`, atau `tiruan` (palsu, hanya dev/demo). Lihat bagian "Donasi online" |
| `MIDTRANS_SERVER_KEY` | Go | Server Key Midtrans (**rahasia**). Wajib bila `PAYMENT_PROVIDER=midtrans` |
| `MIDTRANS_SNAP_URL` | Go | Alamat Snap; kosong = sandbox |
| `PUBLIC_MIDTRANS_CLIENT_KEY` | SvelteKit | Client Key Midtrans (bukan rahasia). Diisi = pembayaran tampil sebagai popup Snap di `/donasi/lanjut`; kosong = muzaki dialihkan ke halaman bayar Midtrans. Untuk `npm run dev` isi juga di `frontend/.env` |
| `WEB_ORIGIN` | Go | Alamat publik frontend untuk pengalihan setelah bayar; wajib bila `PAYMENT_PROVIDER` aktif (di Docker sama dengan `WEB_ORIGIN` frontend) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | Go | Server SMTP untuk email "donasi baru" ke Ketua (port bawaan `587`/STARTTLS, `465` = TLS langsung; `SMTP_FROM` kosong = `SMTP_USER`). `SMTP_PASSWORD` **rahasia**. Lihat bagian "Email pemberitahuan ke Ketua" |
| `NOTIF_EMAIL_KETUA` | Go | Alamat email Ketua penerima pemberitahuan. Wajib diisi bersama `SMTP_HOST` (keduanya kosong = tanpa email) |
| `DEMO_PASSWORD` | skrip | Password akun demo lokal untuk `scripts/seed-demo.sh` |
| `API_URL` | SvelteKit | Alamat Go API (bawaan `http://127.0.0.1:8090`) |
| `PUBLIC_MASJID_NAMA`, `PUBLIC_MASJID_ALAMAT`, `PUBLIC_MASJID_KONTAK` | SvelteKit + Go | Identitas masjid di sidebar, landing, dan kop laporan. Go memakai nilai yang sama untuk kop email dan PDF kuitansi (di Docker diteruskan lewat `docker-compose.yml`; untuk `go run`, muat `.env` yang sama). Alamat/kontak yang diisi tampil di footer landing publik (blok Kontak) serta kop laporan dan kuitansi. Bawaannya **kosong** (tidak ada data contoh): yang kosong tidak ditampilkan. Nilai dengan spasi/koma diberi tanda kutip tunggal di `.env` |
| `PUBLIC_MASJID_LOGO` | SvelteKit | Opsional. Alamat gambar logo (berkas di `frontend/static/`); tampil di landing, login, sidebar, halaman publik, kuitansi, dan kop laporan. Bawaan: `/logo-masjid.jpg` (lambang Masjid Jami' Al-Barokah, sudah disertakan; versi lengkap dengan tulisan: `/logo-masjid-lengkap.jpeg`). Masjid lain: ganti berkasnya atau isi alamat lain. Pakai Docker: build ulang image `web` |
| `PUBLIC_MASJID_FOTO` | SvelteKit | Opsional. Foto masjid untuk latar hero landing, mis. `/foto-masjid.jpg` (berkas di `frontend/static/`); otomatis digelapkan agar teks terbaca. Kosong = latar hijau tua berpola. Pakai Docker: build ulang image `web` |
| `PUBLIC_MASJID_LINTANG`, `PUBLIC_MASJID_BUJUR` | SvelteKit | Koordinat masjid (derajat desimal, lintang selatan negatif) untuk jadwal salat di landing. Bawaan: Pamulang, Tangerang Selatan (-6.3436, 106.7365). Isi titik masjid yang sebenarnya; selisih 1° bujur ≈ 4 menit. Rumus Kemenag RI (subuh 20°, isya 18°), dihitung di aplikasi tanpa API luar |
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

Skrip ini lebih dulu menjalankan `scripts/terapkan-kriteria-final.sh` (kriteria hasil konfirmasi Ketua Yayasan: C1 penghasilan sebagai tingkat 1–4, bobot 0,30/0,30/0,10/0,15/0,15), jadi data contoh memakai tingkat, bukan rupiah. Skrip kriteria itu juga bisa dijalankan sendiri pada database yang sudah dipakai (aman diulang), mis. `set -a && . ./.env && set +a && ./scripts/terapkan-kriteria-final.sh`. Nilai C1 lama berupa rupiah tidak dikonversi; periode lama perlu dinilai ulang.

Setelah menekan Hitung ARAS, hasil yang diharapkan: A2 0,8618 · A1 0,7346 · A4 0,6671 (penerima) · A5 0,6098 · A3 0,3849 (cadangan). Alur demo lengkap: Admin hitung → keluar → masuk sebagai Ketua → Hasil → Setujui → keluar → Admin → Penyaluran → Catat ×3 → Tutup periode → Laporan (Cetak/CSV).

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

## Donasi online (opsional)
Halaman publik **`/donasi`** (tanpa login) menerima zakat/infak dari muzaki lewat payment gateway, lalu menerbitkan kuitansi yang bisa dicetak. Menu pengurus **Donasi** (`/penerimaan`) menampilkan semuanya dan memungkinkan Admin mencatat donasi tunai/transfer langsung. Tanpa `PAYMENT_PROVIDER`, halaman publik menampilkan "belum diaktifkan" dan menu pengurus tetap berfungsi.

**Mencoba tanpa akun Midtrans** (pembayaran PALSU, hanya development): di `.env` set `PAYMENT_PROVIDER=tiruan`, `COOKIE_SECURE=false`, dan `WEB_ORIGIN` ke alamat frontend (mis. `http://localhost:5173` untuk `npm run dev`), lalu jalankan ulang Go API. Buka `/donasi`, isi form, dan tombol "Simulasikan bayar berhasil" akan mengubah donasi menjadi lunas lewat jalur notifikasi yang sama dengan gateway asli. Server **menolak start** bila `tiruan` dipakai dengan `COOKIE_SECURE` bukan `false`.

**Midtrans sandbox**:
1. Daftar akun Midtrans, buka mode **Sandbox**, salin **Server Key** dan **Client Key** (Access Keys → General Credentials).
2. `.env`: `PAYMENT_PROVIDER=midtrans`, `MIDTRANS_SERVER_KEY=…`, `PUBLIC_MIDTRANS_CLIENT_KEY=…` (opsional, untuk popup; di Docker perlu `docker compose --env-file .env up -d --build web`), `WEB_ORIGIN=https://alamat-publik` (alamat yang dibuka muzaki). Biarkan `MIDTRANS_SNAP_URL` kosong untuk sandbox.
3. Di dashboard Midtrans, isi **Notification URL** (webhook) dengan `https://alamat-publik/midtrans/notifikasi`. Alamat ini harus bisa dijangkau dari internet (untuk uji lokal pakai tunnel HTTPS); Go API sendiri tidak dibuka ke publik, karena SvelteKit meneruskan notifikasinya.
4. Jalankan ulang Go API, buka `/donasi`, dan bayar memakai metode uji dari dokumentasi Midtrans.

**Alur & keamanan**: donasi tersimpan berstatus *menunggu* → muzaki membayar lewat popup Snap di atas halaman masjid (bila Client Key diisi) atau di halaman bayar Midtrans → hasil callback popup tidak dipercaya, hanya membawa muzaki ke halaman status → Midtrans memanggil webhook dengan tanda tangan SHA512 → Go memverifikasi tanda tangan **dan** kecocokan jumlah, lalu mengubah status menjadi *diterima* dan menerbitkan nomor kuitansi `KWT/<tahun>/<urut>`. Status tidak pernah berubah hanya karena muzaki kembali ke halaman "selesai". Kuitansi dibuka lewat tautan berkode rahasia 128 bit (`/kuitansi/<kode>`); kode itu tidak dicatat di log. Pembuatan donasi publik dibatasi 5 permintaan per 10 menit per IP.

## Email pemberitahuan ke Ketua (opsional)
Setiap donasi **online** yang baru berubah menjadi *diterima* mengirim satu email ke Ketua. Tampilannya email HTML (kop berlogo, kotak jumlah, rincian donasi, dan tombol ke halaman detail pengurus `/penerimaan/<id>` yang wajib login) dengan versi teks polos sebagai cadangan, **dan melampirkan PDF kuitansi** (A5 lanskap, siap cetak, senada dengan halaman `/kuitansi/<kode>`). Isi email **tidak** memuat No. HP, kode rahasia kuitansi, maupun keterangan bebas. Donasi yang dicatat petugas, masih menunggu, gagal, atau kedaluwarsa tidak mengirim email, dan notifikasi ganda dari gateway tidak menghasilkan email ganda.

1. `.env` (contoh lengkap di [.env.example](.env.example)):

   ```bash
   SMTP_HOST=smtp.gmail.com
   SMTP_PORT=587
   SMTP_USER=alamat.pengirim@gmail.com
   SMTP_PASSWORD=app-password-16-karakter
   NOTIF_EMAIL_KETUA=ketua@contoh.id
   ```

   Gmail wajib memakai **App password** (Akun Google → Keamanan → Verifikasi 2 langkah → Sandi aplikasi), bukan password login. Penyedia lain: isi host/port/akun dari dokumentasi mereka (port `587` = STARTTLS, `465` = TLS langsung). `SMTP_HOST` dan `NOTIF_EMAIL_KETUA` harus diisi bersamaan, kalau tidak server menolak start.
2. Terapkan ulang: `go run ./cmd/server` (setelah memuat `.env`), atau untuk Docker `docker compose --env-file .env up -d api`. Log awal menampilkan `pemberitahuan email donasi aktif`.
3. Coba tanpa akun SMTP sungguhan: jalankan penampung email lokal (mis. Mailpit: `docker run --rm -p 8025:8025 -p 1025:1025 axllent/mailpit`), isi `SMTP_HOST=127.0.0.1`, `SMTP_PORT=1025`, `SMTP_FROM=sistem@masjid.test`, `NOTIF_EMAIL_KETUA=ketua@masjid.test` tanpa `SMTP_USER`, lalu bayar donasi lewat `PAYMENT_PROVIDER=tiruan`. Emailnya muncul di `http://127.0.0.1:8025`. Cara ini untuk Go yang dijalankan dengan `go run`; container `api` tidak bisa memakai `127.0.0.1` milik host.

**Kop dan logo.** Nama, alamat, dan kontak masjid di email dan PDF diambil dari `PUBLIC_MASJID_*` (sama dengan frontend). Logonya salinan tersendiri di `backend/internal/kuitansi/logo-masjid.jpg` (di-embed ke binary): bila Anda mengganti logo masjid, ganti berkas itu juga lalu build ulang Go (`docker compose --env-file .env up -d --build api`). PDF memakai font bawaan PDF (Helvetica), jadi huruf di luar Latin (mis. aksara Arab pada nama donatur) tampil sebagai pengganti di PDF; nama aslinya tetap utuh di halaman web. Isian yang terlalu panjang dipotong dengan `...` di PDF. Bila PDF gagal dibuat, email tetap terkirim tanpa lampiran dan kegagalannya dicatat di log (`PDF kuitansi gagal dibuat`).

Pengiriman berjalan di latar belakang dengan batas 30 detik: SMTP yang lambat atau mati **tidak** menggagalkan donasi maupun memperlambat balasan ke gateway. Kegagalan hanya muncul di log Go (`pemberitahuan donasi ke Ketua gagal terkirim`, tanpa alamat email) dan tidak dicoba ulang; donasinya tetap terlihat di menu Penerimaan.

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
10. **Donasi online** (bila dipakai): `PAYMENT_PROVIDER=midtrans`, `MIDTRANS_SERVER_KEY` dari secret manager, `MIDTRANS_SNAP_URL` ke alamat Snap **production**, `WEB_ORIGIN=ORIGIN`. Jangan pernah `PAYMENT_PROVIDER=tiruan` (server menolaknya bila `COOKIE_SECURE=true`). Daftarkan `https://alamat-publik/midtrans/notifikasi` sebagai Notification URL di dashboard Midtrans **production**, dan pastikan reverse proxy meneruskan path itu ke frontend. Syarat akun production (dokumen lembaga, kategori donasi/zakat) ditentukan Midtrans; urus langsung ke penyedia sebelum go-live.
11. **Email ke Ketua** (bila dipakai): isi `SMTP_*` dan `NOTIF_EMAIL_KETUA` (lihat bagian "Email pemberitahuan ke Ketua"). `SMTP_PASSWORD` dari secret manager / file env ber-izin 600, jangan di-commit. Pastikan server bisa keluar ke host SMTP di port 587/465; tanpa TLS ke server non-lokal, login SMTP ditolak otomatis supaya password tidak terkirim polos.

## Troubleshooting
| Gejala | Penyebab & solusi |
|--------|-------------------|
| `go: command not found` | Go belum terpasang / belum di `PATH`. Pasang dari <https://go.dev/dl/>. |
| `go.mod requires go >= 1.2x` atau Go mengunduh toolchain | Project dikunci ke Go 1.26.10. Jangan `go get ...@latest`; pakai `GOTOOLCHAIN=local` dan kembalikan `go 1.26.10` di `go.mod`. |
| `syntax error near unexpected token '('` saat `. ./.env` (bash) | Nilai `DATABASE_URL`/`TEST_DATABASE_URL` belum diberi tanda kutip tunggal. Ubah jadi `DATABASE_URL='root:…@tcp(127.0.0.1:33070)/spkzakat'` (zsh menerima tanpa kutip, bash tidak). |
| `required variable MYSQL_ROOT_PASSWORD is missing` | `.env` belum dibuat atau compose dijalankan tanpa `--env-file .env`. |
| Docker: `Bind for 127.0.0.1:3101 failed: port is already allocated` | Port frontend dipakai proses/container lain (cek `ss -ltnp \| grep 3101`). Set `WEB_PORT` dan `WEB_ORIGIN` di `.env` ke port bebas, lalu `up -d`. |
| Docker: login admin ditolak padahal password benar | Password diubah lewat `useradmin reset-password` (cek `updated_at` tabel `users`) atau akun terkunci 15 menit. Reset password, atau `docker restart spkzakat-api` untuk membersihkan penghitung. |
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
| `/donasi` menampilkan "belum diaktifkan" | `PAYMENT_PROVIDER` kosong. Isi `midtrans` (atau `tiruan` untuk dev) di `.env` dan jalankan ulang Go API. |
| Server: `PAYMENT_PROVIDER=tiruan hanya untuk development` | Penyedia tiruan sengaja ditolak saat `COOKIE_SECURE` bukan `false`. Set `COOKIE_SECURE=false` di dev, atau pakai `midtrans`. |
| Server: `WEB_ORIGIN wajib diset` / `MIDTRANS_SERVER_KEY wajib diset` | Variabel donasi belum lengkap (lihat bagian "Donasi online"). |
| Donasi tetap "menunggu" setelah dibayar di Midtrans | Notifikasi belum sampai. Cek Notification URL di dashboard = `https://alamat-publik/midtrans/notifikasi`, alamat itu terjangkau dari internet, dan log Go: `notifikasi gateway ditolak` (tanda tangan/Server Key salah) atau status 409 (jumlah tidak cocok). |
| Muzaki melihat "Layanan pembayaran sedang bermasalah" | Go tidak bisa membuat transaksi di Midtrans (Server Key salah, jaringan, atau `MIDTRANS_SNAP_URL` keliru). Lihat log Go `payment gateway bermasalah`. |
| "Terlalu banyak permintaan" di form donasi | Batas 5 donasi baru per 10 menit per IP. Tunggu, atau restart Go API (penghitung di memori). |
| Server: `SMTP_HOST dan NOTIF_EMAIL_KETUA harus diisi bersamaan` | Hanya salah satu yang terisi. Isi keduanya, atau kosongkan keduanya. |
| Server: `pemberitahuan email: notif: alamat penerima/pengirim: bukan alamat email yang valid` | `NOTIF_EMAIL_KETUA` atau `SMTP_FROM`/`SMTP_USER` bukan satu alamat email yang benar. Bila `SMTP_FROM` kosong, `SMTP_USER` dipakai sebagai pengirim dan harus berupa alamat email. |
| Email datang tanpa lampiran PDF, atau PDF menampilkan nama masjid `Masjid` | Log Go `PDF kuitansi gagal dibuat` = pembuatan PDF gagal (email tetap terkirim). Nama `Masjid` = `PUBLIC_MASJID_NAMA` tidak sampai ke container `api`: pastikan terisi di `.env` dan `docker-compose.yml` meneruskannya, lalu `up -d api`. |
| Donasi dibayar tapi email tidak datang | Cek log Go: tidak ada baris `pemberitahuan donasi ke Ketua` = fitur nonaktif (`SMTP_HOST` kosong) atau donasinya bukan donasi online. `… gagal terkirim` + `login SMTP`: akun/App password salah, atau server tidak menawarkan STARTTLS (login ditolak demi keamanan). `… sambung ke SMTP … i/o timeout`: host/port salah atau diblokir firewall (coba port 465 bila 587 diblokir). Cek juga folder Spam. |
