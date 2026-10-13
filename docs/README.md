# Dokumen Rancangan — SPK Penyaluran Dana Zakat dan Infak Masjid (Metode ARAS)

Judul skripsi: *Sistem Pendukung Keputusan Penyaluran Dana Zakat dan Infak Masjid Menggunakan Metode ARAS (Additive Ratio Assessment)*.

**Ringkasan sistem.** Aplikasi web untuk pengurus masjid: mencatat calon penerima (mustahik), menilai mereka dengan 5 kriteria berbobot, meranking otomatis memakai ARAS, lalu memproses persetujuan Ketua DKM dan pencatatan penyaluran. Stack: **Go** (REST API) + **SvelteKit** (UI) + **MySQL 8**, monolith sederhana.

## Daftar Dokumen
| # | Dokumen | Isi |
|---|---------|-----|
| 01 | [Analisis Kebutuhan](01-analisis-kebutuhan.md) | Latar masalah, aktor, fitur, FR/NFR, batasan masalah |
| 02 | [Perancangan ARAS](02-perancangan-aras.md) | Kriteria & bobot, rumus 6 langkah, contoh hitung 5 alternatif, aturan kuota |
| 03 | [Perancangan Sistem](03-perancangan-sistem.md) | Use case, state, activity, sequence, class diagram, ERD, DDL MySQL |
| 04 | [Rancangan UI & Arsitektur](04-rancangan-ui-arsitektur.md) | Arsitektur, struktur folder, peta halaman → file desain, aturan interaksi, endpoint REST, keamanan, strategi test |
| – | [`../Design/`](../Design/) | Mockup UI (HTML, PDF, PNG desktop/tablet/Android) — acuan visual; dokumen 04 memetakan tiap route ke file desainnya |

## Hal yang Masih Perlu Diputuskan/Divalidasi
1. **Bobot & skala kriteria** (dokumen 02) → validasi lewat wawancara dengan pengurus masjid dan dosen pembimbing.
2. Golongan asnaf yang diaktifkan untuk periode zakat (default: fakir & miskin).
3. Format laporan: desain sudah memuat kolom tanda tangan Admin & Ketua DKM — konfirmasi ke masjid; isi nama/alamat masjid lewat `.env`.
4. **Pembuatan akun lewat CLI** (FR-17). Teks desain "Akun diberikan oleh Admin" / "Hubungi Admin untuk reset akun" menyiratkan halaman kelola akun; pilih: (a) tetap CLI dan ubah teks desain, atau (b) tambah halaman Kelola Akun + endpoint.
5. **Desain yang belum ada**: varian layar Ketua DKM (sidebar tanpa Mustahik — termasuk pada layar 403 —, panel keputusan hanya aktif untuk Ketua), layar login gagal/terkunci & sesi habis di tablet/Android, varian periode Infak (apakah chip asnaf disembunyikan), audit log berpaginasi. *(Mustahik tidak dihapus tetapi dinonaktifkan; halaman 403/404 sudah ada di desain.)*
6. **Perbaikan visual di desain**: tabel Mustahik & Hasil terpotong oleh scrollbar internal, bobot `0.30` → `0,30`, input tanggal `mm/dd/yyyy` → `dd/mm/yyyy`, kolom Pengguna audit log pakai nama.
7. Tinjau tablet & Android sudah dilakukan (48 dari 49 PNG; `Ubah Tambah Mustahik.png`, HTML, dan PDF belum dibuka). **Sudah diterapkan ke dokumen**: Profil & password (FR-21, `PUT /auth/password`), daftar Laporan & Tentang (FR-22), bobot kelipatan 0,05 (FR-04/05, dokumen 02), satu template laporan (FR-14), validasi total penyaluran ≤ dana (FR-13), label menu "Beranda", format angka/tanggal id-ID di semua layar (NFR-08), filter asnaf dinamis.
8. **Pekerjaan di sisi desain** (dokumen sudah menetapkan acuannya di dokumen 04 bagian 4): ganti grid Penilaian tablet yang terpotong dengan kartu; sembunyikan scrollbar pada chip filter Android; ubah label "Dashboard" → "Beranda" di desktop; gambar halaman Profil & password, daftar Laporan, dan Tentang aplikasi; tambah tombol "Lihat ranking run lama" dan "Kembalikan" di Android; samakan isi layar laporan dengan template cetak.

## Status
Tahap: **rancangan + implementasi awal**. Yang sudah ada kodenya: paket `backend/internal/aras` (perhitungan ARAS, 100% cakupan test), migrasi MySQL (`backend/migrations`, 12 tabel + seed kriteria), `backend/internal/store` (kriteria, mustahik, periode, kandidat; diuji ke MySQL 8.0 asli), `backend/internal/crypto` (Argon2id, AES-256-GCM, HMAC), dan `backend/internal/service` (aturan bisnis kriteria, mustahik, periode, kandidat, **penilaian, dan Hitung ARAS** — alur ranking lengkap dari data sampai tersimpan, diuji dengan contoh dokumen 02). Cara menjalankan: [../how-to-run.md](../how-to-run.md); penjelasan kode: [../ARCHITECTURE.md](../ARCHITECTURE.md). Server, database, dan frontend belum dibuat.
