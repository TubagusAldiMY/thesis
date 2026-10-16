# Arsitektur

Rancangan lengkap (kebutuhan, ARAS, UML, UI) ada di `docs/` — mulai dari [docs/README.md](docs/README.md). Dokumen ini menjelaskan **apa yang ada di kode** dan cara membacanya.

## Gambaran besar
Monolith sederhana, dua proses + satu database:

```
Browser ──HTTPS──► SvelteKit (frontend/, Node)  ──fetch + cookie sesi──► Go API (backend/, 127.0.0.1:8090) ──SQL berparameter──► MySQL 8
                   form action, SSR, CSP           X-Request-Id, X-Real-IP     http → service → store
```

- Browser **tidak pernah** memanggil Go langsung; token sesi hanya ada di cookie HttpOnly milik SvelteKit lalu diteruskan ke Go.
- Go API listen di `127.0.0.1` (tidak publik), jadi tidak perlu CORS.
- Tanpa OpenTelemetry (monolith): korelasi lewat `request_id` yang sama di log SvelteKit dan Go.

## Peta folder
| Path | Tanggung jawab |
|------|----------------|
| `docs/` | Dokumen rancangan 01–04 |
| `docs/penjelasan/` | Catatan belajar per pekerjaan (pribadi, **tidak di-commit**) |
| `Design/` | Mockup UI (HTML, PDF, PNG) — acuan visual |
| `docker-compose.yml` | Tiga service, semua `restart: unless-stopped`: `mysql` (`127.0.0.1:33070`), `api` (Go, hanya di jaringan compose, DSN ke host `mysql`), `web` (SvelteKit, `127.0.0.1:3101`) |
| `backend/Dockerfile` | Build statis `server`+`migrate`+`useradmin` → image alpine non-root; start = `migrate` lalu `server` |
| `frontend/Dockerfile` | Build SvelteKit (adapter-node) → image node non-root, `node build` |
| `.env.example` | Variabel backend; `frontend/.env.example` untuk frontend |
| `scripts/seed-demo.sh` | Data contoh dokumen 02 untuk demo sidang (lewat API) |
| **Backend** `backend/` | Modul Go `spkzakat` (Go 1.23) |
| `backend/cmd/server/` | **Entry point API**: baca env → buka MySQL → rakit crypto + service + httpapi → listen + graceful shutdown |
| `backend/cmd/migrate/` | CLI migrasi (`DATABASE_URL`) |
| `backend/cmd/useradmin/` | CLI buat akun / reset password (FR-17), password dibaca tanpa echo |
| `backend/migrations/` | `V001` skema 12 tabel, `V002` seed kriteria, `V003` kolom `aras_run.langkah`, `V004` tabel `donasi` + `donasi_urut` (penomor kuitansi); disematkan ke biner (`go:embed`) |
| `backend/internal/aras/` | **Inti skripsi**: fungsi murni `Hitung` (6 langkah ARAS) |
| `backend/internal/domain/` | Tipe inti + error (`ErrNotFound`, `ErrConflict`, `ConflictError`, `ValidationError`), `User`, `Audit`, `Persetujuan`, `Penyaluran`. Tidak mengimpor paket lain |
| `backend/internal/crypto/` | Argon2id, AES-256-GCM (NIK/HP), HMAC-SHA256 |
| `backend/internal/payment/` | Penyedia pembayaran donasi web (`domain.GerbangBayar`): `Midtrans` (Snap + verifikasi tanda tangan notifikasi SHA512) dan `Tiruan` (palsu untuk dev/demo/test, menolak menyala di production). Hanya mengimpor `domain` |
| `backend/internal/store/` | Akses MySQL (SQL berparameter), satu tipe `Store` untuk koneksi & transaksi |
| `backend/internal/service/` | Aturan bisnis & transaksi; tidak tahu HTTP |
| `backend/internal/httpapi/` | Router (`net/http.ServeMux` Go 1.22+), middleware, handler, pemetaan error → status |
| `backend/internal/migrate/`, `internal/testdb/` | Pelaksana migrasi; database sementara untuk test |
| **Frontend** `frontend/` | SvelteKit 2 + Svelte 5 (runes) + TypeScript strict, adapter-node |
| `frontend/src/hooks.server.ts` | Tiap request: `request_id`, baca sesi (`/api/auth/me` → `locals.user`), header keamanan, log JSON |
| `frontend/src/lib/server/api.ts` | Satu-satunya jalur ke Go: `mentah`, `api`, `apiOpsional`, `aksi`, `wajibAdmin` |
| `frontend/src/lib/format.ts` | Format id-ID (ribuan titik, desimal koma, dd/mm/yyyy) + label enum |
| `frontend/src/lib/types.ts` | Tipe = bentuk JSON Go |
| `frontend/src/lib/kriteria.ts` | `parseSkala`/`teksSkala`: mengubah skala nilai antara teks isian ("1 = sehat" per baris) dan objek; hanya memeriksa bentuk, aturan ketat dijaga Go |
| `frontend/src/lib/salat.ts` | Jadwal salat dihitung di aplikasi (posisi matahari, parameter Kemenag RI: subuh 20°, isya 18°); fungsi murni, tanpa API luar. Diuji ke pembanding independen (selisih ≤ 1 menit). Tampil lewat `components/JadwalSalat.svelte` |
| `frontend/src/lib/masjid.ts` | Identitas masjid dari env publik: nama, alamat, kontak, logo, foto hero, koordinat (jadwal salat) |
| `frontend/src/routes/` | Halaman (lihat tabel rute di bawah) |
| `frontend/svelte.config.js` | adapter-node + CSP otomatis |

## Arah dependensi (backend)
```
cmd/server ──► httpapi ──► service ──► store ──► domain
                               │  └──► crypto, aras (murni)
cmd/useradmin ──► service      └──► domain
cmd/server ──► payment ──► domain   (gateway dipasang ke service lewat AturGerbang; httpapi hanya mengenal payment.Tiruan untuk simulasi dev)
cmd/migrate ──► migrate ──► migrations (embed)
```
`domain`, `aras`, `crypto` tidak mengimpor paket project lain. `service` tidak tahu HTTP; `httpapi` tidak berisi aturan bisnis.

## Alur program (start → listen)
`cmd/server/main.go`:
1. `muatConfig()` membaca `DATABASE_URL`, `NIK_ENCRYPTION_KEY`, `NIK_HMAC_KEY`, `ADDR` (bawaan `127.0.0.1:8090`), `COOKIE_SECURE`, `LOG_LEVEL`. Rahasia tidak pernah dilog.
2. Logger `slog` JSON ke stdout dijadikan default.
3. `store.Open` (pool 25/5, UTC, `multiStatements` mati, ping).
4. `crypto.NewCipher` + `crypto.NewHasher` → `service.New(store, cipher, hasher)` (sekaligus membuat pembatas login di memori).
5. `httpapi.New(svc, log, cookieSecure, st.Ping)` → `http.Server` dengan timeout baca/tulis.
6. `ListenAndServe` di goroutine; SIGINT/SIGTERM → `Shutdown` maks. 30 detik.

## Alur data satu permintaan
```
Browser ─► SvelteKit hooks.server.ts (request_id, sesi, header keamanan)
        ─► +page.server.ts load / form action ─► lib/server/api.ts (Cookie sesi, X-Request-Id, X-Real-IP)
        ─► Go: requestID ─► logAkses ─► recover ─► ServeMux ─► peran() [401 tanpa sesi, 403 peran salah]
        ─► handler: bacaJSON (1 MiB, field tak dikenal = 400) ─► service (validasi + transaksi + FOR UPDATE)
        ─► store (SQL berparameter) ─► MySQL
        ◄─ JSON ◄─ gagal(): ValidationError→422 {fields}, ConflictError→409 {pesan}, ErrNotFound→404, lainnya→500 (detail hanya di log)
        ◄─ handler menulis audit log (aksi yang mengubah data)
        ◄─ api.ts: 401→/login, 403/404→halaman error, 400/409/422/429→fail() ke formulir
```

## Paket `internal/service`
| Berkas | Use case | Aturan penting |
|--------|----------|----------------|
| `auth.go` | `Login`, `Sesi`, `Logout`, `GantiPassword`, `BuatUser`, `ResetPassword` | Argon2id; pesan sama untuk username tak dikenal (dan verifikasi palsu agar waktunya sama); 5 gagal per username+IP → kunci 15 menit (`pembatas`, di memori); token 32 byte acak, yang disimpan hanya SHA-256-nya; sesi 8 jam; ganti password mencabut sesi lain |
| `service.go` | `Catat`, `DaftarAudit`, `KategoriAudit` | Audit tanpa NIK/HP/password; kategori filter → kode aksi |
| `mustahik.go` | Buat/ubah/ambil/daftar mustahik | NIK 16 digit, terenkripsi + HMAC duplikat; keluaran hanya tersamar; tidak bisa dinonaktifkan saat jadi kandidat periode berjalan |
| `kriteria.go` | `DaftarKriteria`, `SimpanKriteria` (edit nama/tipe/bobot/skala/aktif + tambah kriteria baru), `UbahKriteria` (hanya tipe+bobot; dipakai tes) | Atas satu paket: bobot kelipatan 0,05, Σ bobot **aktif** = 100%, 2–10 kriteria aktif, nama unik (tanpa membedakan huruf besar-kecil), nama 3–100 huruf, skala 2–9 pilihan berkunci angka 1–9. Satu pelanggaran = rollback. Kriteria **tidak dihapus**, hanya dinonaktifkan (penilaian dan potret run menunjuk kriteria itu). Kode kriteria baru (`C6`, `C7`, …) dibuat store dan tidak berubah setelahnya |
| `periode.go` | Buat/daftar/ambil periode, atur & daftar kandidat | Nominal ≤ dana; zakat wajib asnaf; kandidat hanya saat draft, aktif, golongan sah |
| `penilaian.go` | `SimpanPenilaian`, `AmbilPenilaian` | Hanya draft; skala C3–C5 hanya kunci 1–4; satu sel salah membatalkan paket |
| `aras.go` | `HitungAras`, `AmbilHasilAras`, `DaftarRun` | Dari draft/dihitung; lengkap; potret bobot + langkah disimpan; run baru = versi naik |
| `keputusan.go` | `AmbilDetailPeriode`, `Putuskan`, `BukaKembali`, `CatatPenyaluran`, `DaftarPenyaluran`, `TutupPeriode` | Keputusan hanya saat dihitung, catatan wajib bila tolak, satu run sekali diputuskan; penyaluran hanya saat disetujui, hanya usulan run terkini, sekali per penerima, Σ ≤ dana; tutup bila semua usulan tercatat |
| `laporan.go` | `AmbilDashboard` (+ `ambilKeuangan`), `AmbilRingkasanPublik` (agregat tanpa data pribadi untuk landing), `AmbilLaporan`, `Laporan.TulisCSV` | Stepper 1–5; ringkasan keuangan Beranda: zakat dan infak dibukukan **terpisah** (saldo = diterima − tersalur per jenis, bisa negatif), grafik 12 bulan kalender WIB; laporan memakai potret bobot run; CSV dengan pencegahan formula injection |
| `donasi.go` | `BuatDonasiWeb`, `TerimaNotifikasi`, `CatatDonasiManual`, `DaftarDonasi`, `AmbilDonasi`, `BatalkanDonasi`, `RingkasDonasi`, `TandaiDonasiDilihat`, `AmbilDonasiPublik`, `AturGerbang` | Donasi web jadi *diterima* **hanya** dari notifikasi gateway yang tanda tangannya sah dan jumlahnya cocok (baris dikunci `FOR UPDATE`, idempoten: notifikasi ganda/terlambat diabaikan). Nomor kuitansi `KWT/<tahun WIB>/<urut>` tanpa celah (penghitung `donasi_urut` di transaksi yang sama). Kuitansi tidak dihapus, hanya *dibatalkan*. No. HP terenkripsi; penuh hanya untuk Admin pada detail. Jumlah Rp 10.000–100.000.000 |

Setiap transisi status periode memakai `GetPeriodeForUpdate` (kunci baris) + `SetStatusPeriode` (compare-and-set), sehingga dua pengguna yang menekan tombol bersamaan tidak bisa sama-sama menang.

## Paket `internal/store`
| Berkas | Isi |
|--------|-----|
| `store.go` | `Open`, `WithTx`, `inTx`, `mapErr` (error MySQL → error domain), `newID` (UUID v4) |
| `users.go` | `CreateUser`, `GetUserByUsername`, `GetUser`, `UpdatePasswordHash`, `CreateSession`, `GetUserBySession` (cek kedaluwarsa + akun aktif), `DeleteSession`, `DeleteSessionsUser`, `DeleteExpiredSessions` |
| `audit.go` | `InsertAudit`, `ListAudit` (LEFT JOIN users untuk nama pelaku, paging) |
| `keputusan.go` | `CreatePersetujuan`, `ListPersetujuan`, `CreatePenyaluran`, `ListPenyaluran` |
| `kriteria.go`, `mustahik.go`, `periode.go`, `penilaian.go`, `aras.go`, `lookup.go` | Kriteria, mustahik, periode & kandidat, penilaian, run ARAS, query bantu (`GetPeriodeForUpdate`, dll.) |
| `donasi.go` | `CreateDonasi`, `GetDonasi`/`ByKode`/`ForUpdate`, `UpdateDonasi` (hanya kolom status, bukan data donatur/jumlah), `ListDonasi`, `RingkasDonasi`, `TandaiDonasiDilihat`, `NextNomorKuitansi` (wajib di dalam transaksi) |
| `keuangan.go` | `PenerimaanPerJenis`, `PenyaluranPerJenis`, `PenerimaanPerBulan`, `PenyaluranPerBulan`, `JumlahPenerimaManfaat` (orang berbeda, bukan jumlah baris): agregat baca-saja untuk Beranda dan landing. Hanya donasi `diterima`; bulan donasi digeser +7 jam (WIB) karena `dibayar_pada` disimpan UTC |

Kontrak: store menerima NIK/HP yang **sudah terenkripsi**; setiap query membaca baris `deleted_at IS NULL`; uang `int64` rupiah utuh; bobot `float64` pecahan.

## Alur per endpoint (Go, prefix `/api`)
Semua kecuali `login` dan `/healthz` butuh cookie `sesi`. Error: `{"error","request_id", "fields"?}`.

| Method + path | Peran | Validasi / aturan | Query / aksi | Sukses → error |
|---|---|---|---|---|
| `GET /healthz` | publik | – | `PING` MySQL | 200 `{ok:true}` → 503 |
| `POST /auth/login` | publik | username & password wajib, ≤ 50/128 | cari user aktif, Argon2id, insert `sessions`, audit LOGIN/LOGIN_GAGAL | 200 `{user}` + cookie → 401 `{sisa_percobaan}`, 429 `{retry_after}` |
| `POST /auth/logout` | semua | – | delete `sessions` | 204 |
| `GET /auth/me` | semua | – | join sessions→users | 200 `{id,username,nama,role}` |
| `PUT /auth/password` | semua | lama benar, baru ≥ 12 & ≠ lama, kunci 5 gagal | update hash, hapus sesi lain | 204 → 422, 429 |
| `GET /dashboard` | semua | – | jumlah per asnaf, periode berjalan, stepper, `total_tersalur_semua` (SUM penyaluran semua periode), dan `keuangan`: `zakat`/`infak` {diterima, tersalur, saldo}, total, `diterima_bulan_ini`, `per_jenis`, `bulanan` (12 bulan WIB), `donasi_terbaru` (5, tanpa No. HP) | 200 |
| `GET /mustahik?q&asnaf&aktif&page&per_page` | admin | q ≤ 100, asnaf sah | SELECT + paging + `asnaf_counts` | 200 `{items,total,asnaf_counts}` |
| `POST /mustahik` | admin | nama 2–100, NIK 16 digit unik, HP opsional, alamat 5–500, asnaf | enkripsi + INSERT, audit | 201 → 422 |
| `GET/PUT /mustahik/{id}` | admin | sama; NIK/HP kosong = tetap | UPDATE (nonaktif, bukan hapus) | 200 → 404, 409, 422 |
| `GET /kriteria` | semua | – | SELECT urut | 200 `{items,total_persen}` |
| `PUT /kriteria` | admin | badan `{items:[{id,nama,tipe,bobot,skala\|null,aktif}], baru:[{nama,tipe,bobot,skala}]}`: **penuh-ganti per baris**, semua sekaligus; aturan paket seperti di `kriteria.go` (Σ aktif = 100%, 2–10 aktif, nama unik, skala sah) | UPDATE + INSERT kriteria baru dalam satu transaksi; audit `KRITERIA_UBAH` (isi: nama/tipe/bobot/aktif/jumlah pilihan skala) | 200 → 422 `fields{nama,tipe,bobot,skala,kriteria}`, 404 (ID tak ada) |
| `GET /periode?status` | semua | status sah | SELECT + kuota | 200 `{items}` |
| `POST /periode` | admin | nama 3–100, dana/nominal > 0, nominal ≤ dana, zakat ≥ 1 asnaf | INSERT draft, audit | 201 → 422 |
| `GET /periode/{id}` | semua | – | periode + kuota + runs + keputusan + catatan penolakan | 200 → 404 |
| `GET/PUT /periode/{id}/kandidat` | admin | draft; aktif; golongan sah | ganti selisih `periode_kandidat` | 200 → 409, 422 |
| `GET/PUT /periode/{id}/penilaian` | admin | draft; nilai per kriteria; satu salah = batal semua | upsert `penilaian` | 200 → 409, 422 `fields["nilai.<id>.<kode>"]` |
| `POST /periode/{id}/hitung` | admin | draft/dihitung; Σ bobot 100%; lengkap | `aras.Hitung`, simpan run + hasil, status → dihitung | 200 hasil → 409, 422 |
| `GET /periode/{id}/hasil?versi` | semua | – | run (terkini bila versi kosong) + langkah | 200 → 404 |
| `POST /periode/{id}/keputusan` | **ketua** | dihitung; setuju/tolak; catatan wajib bila tolak | insert `persetujuan`, status → disetujui/ditolak | 200 → 409, 422 |
| `POST /periode/{id}/buka-kembali` | admin | ditolak | status → draft | 200 → 409 |
| `GET /periode/{id}/penyaluran` | semua | – | rekap dari run terkini | 200 |
| `POST /periode/{id}/penyaluran` | admin | disetujui; usulan; sekali; Σ ≤ dana | insert `penyaluran` | 201 → 409, 422 |
| `POST /periode/{id}/tutup` | admin | disetujui; semua usulan tercatat | status → selesai | 200 → 409, 422 |
| `GET /periode/{id}/laporan?format=csv` | semua | sudah pernah dihitung | gabung ranking + penyaluran + nama | 200 JSON/CSV → 404 |
| `GET /audit?kategori&page` | semua | kategori sah | SELECT + nama pelaku, 20/halaman | 200 `{items,total,page,per_page}` |
| `GET /donasi?status&jenis&q&page&per_page` | semua | status/jenis sah, q ≤ 100 | SELECT + paging + ringkasan | 200 `{items,total,ringkas}` → 422 |
| `POST /donasi` | admin | nama 2–100, jenis sah, jumlah 10 rb–100 jt, metode tunai\|transfer, tanggal ≤ hari ini (WIB), HP opsional | transaksi: nomor kuitansi + INSERT status diterima, audit `DONASI_CATAT` | 201 → 422 |
| `GET /donasi/ringkas` | semua | – | jumlah donasi web baru (belum dilihat), jumlah & total diterima | 200 `{baru,jumlah_diterima,total_diterima}` |
| `POST /donasi/dibaca` | semua | – | tandai semua donasi web diterima sebagai dilihat | 200 `{diubah}` |
| `GET /donasi/{id}` | semua | – | 1 baris; `no_hp` penuh hanya untuk Admin, Ketua tersamar | 200 → 404 |
| `POST /donasi/{id}/batal` | admin | alasan 3–200; hanya status diterima | transaksi `FOR UPDATE` → status dibatalkan (nomor tetap), audit `DONASI_BATAL` | 200 → 409, 422 |
| `GET /publik/donasi-aktif` | **publik** | – | apakah gateway terpasang | 200 `{aktif}` |
| `GET /publik/ringkasan` | **publik**, 120/menit/IP | – | angka transparansi landing: hanya enam agregat (`zakat_diterima`, `infak_diterima`, `total_diterima`, `total_tersalur`, `jumlah_donasi`, `penerima_manfaat`); donasi `diterima` saja; tanpa nama/HP. Diuji: semua nilai angka | 200 |
| `POST /publik/donasi` | **publik**, 5/10 menit/IP | nama, jenis, jumlah, HP opsional, keterangan ≤ 200 | INSERT menunggu + buat transaksi di gateway → `bayar_url` | 201 `{kode,status,bayar_url}` → 422, 429, 502, 503 |
| `GET /publik/donasi/{kode}` | **publik**, 120/menit/IP | kode = 32 heksadesimal (selain itu 404) | status + data kuitansi milik pemegang kode | 200 → 404 |
| `POST /publik/midtrans/notifikasi` | **publik**, dijaga tanda tangan | body ≤ 64 KiB; SHA512(order_id+status_code+gross_amount+ServerKey) sah; jumlah = donasi | transaksi `FOR UPDATE`: menunggu → diterima/gagal/kedaluwarsa, terbit nomor kuitansi, audit `DONASI_TERIMA` | 200 → 401 (tanda tangan), 404, 409 (jumlah beda), 503 |
| `POST /publik/tiruan/{kode}/bayar` | publik, **hanya bila `PAYMENT_PROVIDER=tiruan`** | – | memicu notifikasi settlement tiruan lewat jalur yang sama | 200 → 404 bila rute tidak terpasang |

## Rute frontend
| Rute | Isi | Peran |
|------|-----|-------|
| `/` | Landing masjid: navigasi pil melayang dengan tombol Donasi, hero hijau tua beraksen emas (foto opsional `PUBLIC_MASJID_FOTO`), **jadwal salat hari ini** + penanda salat berikutnya, ragam zakat/infak + ajakan donasi, **transparansi dana** (angka agregat dari `GET /publik/ringkasan`; bagian disembunyikan bila API gagal atau belum ada data), cara penyaluran ARAS ringkas, kontak. Logo masjid (`/logo-masjid.jpg`, bisa diganti lewat `PUBLIC_MASJID_LOGO`) dipakai juga di login, sidebar, kuitansi, kop laporan, dan favicon | publik |
| `/cara-kerja` | Penjelasan statis 6 langkah ARAS (dulu bagian landing) | publik |
| `/login` | Formulir + sisa percobaan / terkunci | publik |
| `/dashboard` | Beranda umum masjid bertema sama dengan landing (hero hijau tua berpola + ubin kaca 4 angka utama, tombol emas, **jadwal salat** hari ini): dana zakat vs infak, grafik 12 bulan, penerimaan per jenis, donasi terbaru; lalu bagian ARAS (stepper, periode terbaru). Aksi cepat hanya untuk Admin | semua |
| `/mustahik`, `/mustahik/baru`, `/mustahik/[id]` | Daftar + chip golongan dinamis; formulir | admin (Ketua → 403) |
| `/kriteria` | Edit nama, tipe, bobot, skala nilai (teks "1 = sehat", satu pilihan per baris), aktif/nonaktif, dan **tambah kriteria baru**; indikator total/aturan dihitung di layar, aturan akhir dijaga Go | admin ubah, ketua lihat |
| `/periode`, `/periode/baru` | Daftar + filter status; formulir + kartu kuota langsung | semua / admin |
| `/periode/[id]/kandidat`, `/penilaian` | Centang kandidat; grid/kartu nilai + Hitung ARAS | admin |
| `/periode/[id]/hasil` | Ranking, langkah hitung, panel keputusan, buka kembali | semua (keputusan: ketua) |
| `/periode/[id]/penyaluran` | Catat per penerima, tutup periode | semua (catat: admin) |
| `/periode/[id]/laporan`, `/laporan.csv` | Template cetak + unduh CSV | semua |
| `/audit`, `/laporan`, `/profil`, `/tentang`, `/lainnya` | Audit; daftar laporan; ganti password; tentang; menu HP | semua |
| `/penerimaan`, `/penerimaan/baru`, `/penerimaan/[id]` | Menu pengurus **Donasi**: daftar + ringkasan + penanda donasi web baru; catat donasi tunai/transfer; detail (kuitansi, WhatsApp, batal). Penanda angka di menu dari `GET /api/donasi/ringkas` (dimuat `(app)/+layout.server.ts`) | semua lihat; Admin mencatat/membatalkan |
| `/donasi`, `/donasi/lanjut`, `/donasi/selesai`, `/donasi/tiruan` | **Publik**: formulir donasi → perantara di domain sendiri (CSP `form-action 'self'` memblokir pengalihan langsung ke gateway) → halaman bayar gateway → status (diperbarui otomatis). `/donasi/tiruan` hanya berfungsi pada penyedia tiruan | publik |
| `/kuitansi/[kode]` | **Publik**: kuitansi cetak/PDF dengan terbilang; stempel DIBATALKAN bila dibatalkan. Kode di path tidak dicatat di log | pemegang kode |
| `/midtrans/notifikasi` (POST) | Penerima webhook Midtrans; hanya meneruskan body mentah ke Go (Go sendiri tidak publik) | gateway |
| `/logout` (POST) | Cabut sesi + hapus cookie | semua |

Pola tiap halaman: `+page.server.ts` memanggil `api()` (load) atau `aksi(() => api(...))` (form action); `+page.svelte` hanya menampilkan (`$props`, `$state`, `$derived`) dan memakai `use:enhance` supaya tetap jalan tanpa JavaScript.

## Keamanan (ringkas)
- Argon2id; sesi opaque di DB (logout benar-benar mencabut); cookie `HttpOnly; SameSite=Strict; Secure` (dev: `secure` mati di localhost).
- Deny-by-default: setiap rute Go dibungkus `peran(...)`; frontend juga `wajibAdmin` untuk halaman Admin.
- CSRF: cek `Origin` bawaan SvelteKit untuk form action; Go hanya menerima cookie SameSite=Strict.
- CSP otomatis SvelteKit (`script-src 'self'`), `X-Frame-Options: DENY`, `nosniff`, HSTS di production.
- NIK/HP AES-256-GCM, hanya 4 digit terakhir yang tampil; tidak pernah masuk log/audit/CSV.
- Log akses tidak mencatat query string (bisa berisi nama), dan segmen rahasia di path (`/api/publik/donasi/{kode}`, `/kuitansi/{kode}`) disamarkan (`pathLog`).
- **Endpoint publik donasi** (satu-satunya selain login/healthz tanpa `peran()`): pembatas laju per IP (memori), validasi ketat di service, kolom jebakan anti-bot di form, kode rahasia 128 bit untuk akses satu donasi (format salah = 404), dan HP terenkripsi AES-256-GCM. Data yang dikirim ke gateway hanya order_id, jumlah, nama item, dan nama donatur (tanpa HP/alamat). Webhook dijaga tanda tangan SHA512 (perbandingan waktu-konstan) + pencocokan jumlah, bukan sesi; status donasi tidak pernah berubah karena parameter di URL pengalihan.

## Cara membaca project
1. `docs/02-perancangan-aras.md` → `backend/internal/aras/aras.go` + `aras_test.go` — inti skripsi.
2. `backend/migrations/V001__initial_schema.sql` — tabel yang dipakai semua lapisan.
3. `backend/internal/domain/` — tipe & error bersama.
4. `backend/internal/store/store.go` lalu `periode.go` — pola SQL & transaksi.
5. `backend/internal/service/aras.go` (`HitungAras`) + `aras_test.go` (`TestHitungAras_ContohDokumen02_DariDataSampaiTersimpan`).
6. `backend/internal/service/keputusan.go` — transisi status setelah ranking.
7. `backend/internal/service/auth.go` — login, penguncian, sesi.
8. `backend/cmd/server/main.go` → `backend/internal/httpapi/server.go` (router + middleware) → `handler.go`.
9. `backend/internal/httpapi/api_test.go` → `TestAlurDemo_DariMustahikSampaiLaporan` — seluruh alur lewat HTTP.
10. `frontend/src/hooks.server.ts` → `src/lib/server/api.ts` → `src/routes/(app)/+layout.svelte` → satu halaman lengkap, mis. `src/routes/(app)/periode/[id]/penilaian/`.
