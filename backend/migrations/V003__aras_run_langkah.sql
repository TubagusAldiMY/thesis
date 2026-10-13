-- V003__aras_run_langkah.sql
-- Menyimpan langkah hitung tiap run (A0, matriks, normalisasi, terbobot) sebagai
-- JSON. Tanpa ini, "Lihat langkah hitung" untuk run lama tidak bisa dibuat ulang
-- bila penilaian berubah sesudahnya (periode dibuka kembali lalu diperbaiki).
-- NULL = run yang dibuat sebelum kolom ini ada.
ALTER TABLE aras_run ADD COLUMN langkah JSON NULL AFTER s0;
