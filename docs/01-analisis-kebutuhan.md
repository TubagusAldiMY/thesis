# 01 — Analisis Kebutuhan

## 1. Latar Masalah
Penyaluran zakat dan infak di masjid umumnya ditentukan secara manual oleh pengurus berdasarkan pengamatan dan kedekatan. Akibatnya:
- Penilaian tidak seragam antar pengurus/antar periode (subjektif).
- Sulit dipertanggungjawabkan kepada jamaah dan donatur: *kenapa A dapat, B tidak?*
- Data mustahik tersebar (buku/kertas/chat), sulit direkap.

Sistem ini membantu pengurus **meranking calon penerima secara terukur** memakai metode **ARAS (Additive Ratio Assessment)**, lalu mencatat penyalurannya. Sistem bersifat *pendukung keputusan*: keputusan akhir tetap di tangan pengurus/Ketua DKM.

## 2. Aktor
| Aktor | Peran |
|-------|-------|
| **Admin / Petugas Amil** | Kelola data mustahik, kriteria & bobot, periode penyaluran, input penilaian, jalankan perhitungan ARAS, catat penyaluran |
| **Ketua DKM** | Melihat hasil ranking, menyetujui/menolak daftar penerima, melihat laporan |

Halaman **Landing** dan **Login** terbuka untuk umum (landing hanya penjelasan metode + contoh ranking statis, tanpa data mustahik asli); semua halaman lain butuh login. Tidak ada registrasi mandiri: akun pertama dibuat lewat CLI, akun berikutnya dikelola Ketua lewat halaman Pengguna (FR-17).

Menu per peran (acuan: folder `App/Design/`):
- **Admin**: Dashboard, Mustahik, Kriteria, Periode, Audit log.
- **Ketua DKM**: Dashboard, Kriteria (lihat), Periode (lihat hasil + putuskan), Audit log. Menu Mustahik **disembunyikan** karena memuat data pribadi; membuka URL-nya langsung menghasilkan halaman 403 ("Halaman ini hanya untuk Admin. Anda masuk sebagai Ketua DKM."). *Catatan: desain layar 403 masih menampilkan sidebar penuh dan belum ada varian layar Ketua — perlu disesuaikan.*

## 3. Daftar Fitur
0. Landing publik (penjelasan metode, kriteria, contoh ranking).
1. Login/logout + hak akses per peran + penguncian setelah 5 kali gagal.
1b. Dashboard ringkasan + stepper langkah periode aktif.
1c. Profil & password (ganti password sendiri), daftar Laporan, Tentang aplikasi.
2. Kelola data mustahik (calon penerima).
3. Kelola kriteria, tipe (benefit/cost), bobot, dan skala nilai.
4. Kelola periode penyaluran (jenis zakat/infak, dana tersedia, nominal per penerima).
5. Pilih kandidat & input penilaian per kriteria.
6. Hitung ARAS → tampilkan langkah hitung + ranking.
7. Penetapan penerima (kuota dari dana) + persetujuan Ketua DKM.
8. Catat realisasi penyaluran.
9. Laporan per periode (cetak/CSV).

## 4. Kebutuhan Fungsional
| ID | Kebutuhan | Peran |
|----|-----------|-------|
| FR-01 | Pengguna dapat login dengan username + password; sesi berakhir saat logout. Setelah 5 kali gagal berturut-turut (per username + IP) login dikunci 15 menit; pesan error generik dengan "sisa percobaan", juga untuk username yang tidak terdaftar | Semua |
| FR-02 | Sistem membatasi akses sesuai peran (deny-by-default): menu yang bukan haknya disembunyikan, dan akses langsung ke URL-nya dibalas 403 dengan halaman "Akses tidak diizinkan"; alamat yang tidak ada dibalas halaman 404 | Semua |
| FR-03 | Admin dapat menambah dan mengubah mustahik (nama, NIK, alamat, no. HP, golongan asnaf) serta menonaktifkannya lewat kotak centang "Mustahik aktif". Tidak ada tombol hapus; mustahik nonaktif tidak muncul sebagai kandidat baru, tetapi riwayat hasil dan penyaluran lamanya tetap tersimpan | Admin |
| FR-04 | Admin dapat mengubah nama, tipe (benefit/cost), bobot (kelipatan 0,05), keterangan skala, dan status aktif kriteria, serta menambah kriteria baru; kriteria tidak dihapus, hanya dinonaktifkan (2–10 kriteria aktif) | Admin |
| FR-05 | Sistem menolak simpan bila total bobot kriteria aktif ≠ 1 (dihitung sebagai bilangan bulat persen, Σ = 100) | Admin |
| FR-06 | Admin dapat membuat periode: nama, jenis (zakat/infak), dana tersedia, nominal per penerima | Admin |
| FR-07 | Admin dapat memilih mustahik kandidat untuk suatu periode (zakat: hanya asnaf yang sesuai) | Admin |
| FR-08 | Admin menginput nilai tiap kandidat untuk tiap kriteria; sistem memvalidasi rentang nilai | Admin |
| FR-09 | Sistem menghitung ARAS (matriks, normalisasi, terbobot, S, K) dan menyimpan hasil | Admin |
| FR-10 | Sistem menampilkan tabel ranking (S, K, status usulan/cadangan) dan tombol "Lihat langkah hitung" yang membuka matriks ternormalisasi, matriks terbobot, dan S | Admin, Ketua |
| FR-11 | Sistem menghitung kuota penerima = ⌊dana tersedia ÷ nominal per penerima⌋ dan menandai N teratas sebagai usulan penerima | Admin |
| FR-12 | Ketua DKM dapat menyetujui atau menolak usulan penerima (dengan catatan) | Ketua |
| FR-13 | Admin mencatat realisasi penyaluran (tanggal, nominal) hanya untuk periode berstatus *disetujui*; tiap penerima sekali; total nominal tercatat tidak boleh melebihi dana tersedia periode | Admin |
| FR-14 | Sistem membuat laporan periode dengan **satu template** untuk semua ukuran layar: kop masjid, ringkasan, A. kriteria & bobot, B. ranking (S, K, keterangan penerima/cadangan), C. realisasi penyaluran + total, kolom tanda tangan Admin & Ketua. Keluaran: cetak/Simpan PDF (lewat dialog cetak browser) dan CSV | Admin, Ketua |
| FR-15 | Perubahan data penting (nilai, bobot, persetujuan, penyaluran) tercatat di audit log | Sistem |
| FR-16 | Landing publik menampilkan profil masjid, transparansi dana (angka agregat tanpa data pribadi), jadwal salat, dan ringkasan penyaluran; penjelasan metode (5 kriteria + bobot, rumus ARAS, contoh ranking **statis**, bukan data database) ada di halaman publik `/cara-kerja` | Umum |
| FR-17 | Ketua (Yayasan/DKM) mengelola akun lewat halaman Pengguna: membuat akun Admin/Ketua, mengubah nama dan peran, menonaktifkan/mengaktifkan kembali, dan mereset password. Akun tidak dihapus; menonaktifkan langsung mencabut semua sesinya; tidak boleh menyentuh akun sendiri; selalu tersisa minimal satu Admin aktif. Akun pertama dan pemulihan darurat tetap lewat CLI `cmd/useradmin` | Ketua, Pengembang |
| FR-18 | Dashboard pengurus menampilkan kartu ringkasan penerimaan, penyaluran, dan saldo tercatat, jadwal salat, ringkasan dana zakat dan infak, grafik penerimaan-penyaluran 12 bulan, penerimaan menurut jenis, donasi terbaru, serta bagian penyaluran metode ARAS (mustahik aktif per asnaf, stepper 5 langkah periode aktif, dan periode terbaru) | Admin, Ketua |
| FR-19 | Periode berstatus *ditolak* menampilkan catatan Ketua + riwayat run; Admin dapat "Buka kembali" (kembali ke draft) dan tetap melihat ranking run lama | Admin |
| FR-20 | Audit log bisa difilter per kategori (Login, Data & penilaian, Hitung, Keputusan, Penyaluran), menampilkan **nama pengguna** pelaku, hanya-baca | Admin, Ketua |
| FR-21 | Pengguna dapat mengganti password sendiri ("Profil & password"): wajib password lama, password baru ≥ 12 karakter; sesi lain milik user itu dicabut | Admin, Ketua |
| FR-22 | Tersedia daftar **Laporan** (periode berstatus *selesai*) dan halaman statis **Tentang aplikasi** (versi + ringkasan metode ARAS) | Admin, Ketua |

## 5. Kebutuhan Non-Fungsional
| ID | Kebutuhan |
|----|-----------|
| NFR-01 | Password di-hash **Argon2id**; cookie sesi `HttpOnly`, `Secure`, `SameSite=Strict` |
| NFR-02 | Semua query MySQL **parameterized**; input divalidasi di backend |
| NFR-03 | Data pribadi (NIK, HP) tidak boleh muncul di log |
| NFR-04 | Log JSON ke stdout dengan `request_id`; response membawa header `X-Request-Id` |
| NFR-05 | Perhitungan ARAS deterministik & teruji (table-driven test, dicocokkan dengan hitung manual di dokumen 02) |
| NFR-06 | Halaman utama merespons < 1 detik untuk ≤ 500 mustahik |
| NFR-07 | Tampilan responsif sesuai desain: desktop, tablet, Android (input penilaian di HP berupa kartu per kandidat + tombol skala 1–4) |
| NFR-08 | Format Indonesia di seluruh UI **dan di semua ukuran layar**: ribuan titik (`3.000.000`), desimal koma (`0,30`), tanggal `dd/mm/yyyy`. Nilai di API/DB tetap angka & ISO 8601. Implementasi: tampilan lewat `Intl` locale `id-ID`; input tanggal memakai `<input type="date">` bawaan (mengikuti locale perangkat, jadi mockup `mm/dd/yyyy` hanya artefak render en-US); input angka desimal berupa teks `inputmode="decimal"` yang menerima koma |
| NFR-09 | Laporan dicetak lewat CSS `@media print` (sidebar & tombol disembunyikan) dan bisa diunduh CSV |

## 6. Batasan Masalah (di luar cakupan)
- Pencatatan pemasukan zakat/infak harian (hanya "dana tersedia" per periode yang diinput).
- Multi-masjid / multi-tenant.
- ~~Halaman kelola akun~~ — sudah ada sejak permintaan Ketua Yayasan (Nov 2026), lihat FR-17. Yang tetap di luar cakupan: registrasi mandiri, hapus akun permanen, dan pemulihan password lewat email.
- Notifikasi WhatsApp/SMS, aplikasi mobile native, pembayaran online.
- Pembobotan otomatis (AHP/entropy): bobot diisi manual; validasi bobot dilakukan lewat wawancara dengan pengurus.
- Penentuan syariat per kasus (fatwa): sistem hanya memfilter golongan asnaf dasar.

## 7. Skenario Utama (ringkas)
1. Admin membuat periode "Zakat Fitrah 1448H" → dana Rp 6.000.000, nominal Rp 1.000.000/penerima → kuota 6.
2. Admin memilih 12 mustahik kandidat dan mengisi nilai 5 kriteria.
3. Admin klik **Hitung ARAS** → muncul ranking.
4. 6 teratas ditandai usulan; Ketua DKM menyetujui.
5. Admin mencatat penyaluran per penerima; laporan dicetak.

Lanjut: [02 — Perancangan ARAS](02-perancangan-aras.md).
