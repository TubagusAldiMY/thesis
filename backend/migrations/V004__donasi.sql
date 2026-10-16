-- V004__donasi.sql
-- Fitur donasi muzaki: zakat/infak yang diterima masjid, baik dari halaman publik
-- (dibayar lewat payment gateway) maupun dicatat petugas (tunai/transfer langsung).
-- Satu tabel untuk keduanya supaya kuitansi dan laporan penerimaan seragam.
-- Konvensi sama dengan V001: PK UUID, DATETIME(3), soft delete lewat deleted_at.

CREATE TABLE donasi (
  id             CHAR(36)       NOT NULL PRIMARY KEY,   -- dipakai juga sebagai order_id di gateway
  kode_publik    CHAR(32)       NOT NULL,               -- 128 bit acak (hex): tautan kuitansi & status untuk muzaki tanpa akun
  nama           VARCHAR(100)   NOT NULL,
  no_hp_enc      VARBINARY(255) NULL,                   -- AES-256-GCM di app layer (PII, UU PDP)
  jenis          ENUM('zakat_fitrah','zakat_mal','zakat_profesi','infak') NOT NULL,
  jumlah         DECIMAL(15,2)  NOT NULL CHECK (jumlah > 0),
  sumber         ENUM('web','petugas') NOT NULL,
  metode         VARCHAR(30)    NULL,                   -- tunai | transfer | payment_type dari gateway (qris, bank_transfer, ...)
  status         ENUM('menunggu','diterima','gagal','kedaluwarsa','dibatalkan') NOT NULL DEFAULT 'menunggu',
  nomor_kuitansi VARCHAR(30)    NULL,                   -- terbit saat status menjadi diterima: KWT/<tahun>/<urut>
  keterangan     VARCHAR(200)   NULL,
  bayar_url      VARCHAR(500)   NULL,                   -- tautan halaman bayar gateway selama status menunggu
  dibayar_pada   DATETIME(3)    NULL,
  dilihat_pada   DATETIME(3)    NULL,                   -- NULL = donasi web yang diterima tetapi belum dilihat pengurus (notifikasi)
  dicatat_oleh   CHAR(36)       NULL,                   -- petugas, hanya untuk sumber 'petugas'
  alasan_batal   VARCHAR(200)   NULL,
  created_at     DATETIME(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at     DATETIME(3)    NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at     DATETIME(3)    NULL,
  UNIQUE KEY uq_donasi_kode_publik (kode_publik),
  UNIQUE KEY uq_donasi_nomor_kuitansi (nomor_kuitansi), -- NULL boleh banyak; nomor yang sudah terbit tidak boleh ganda
  KEY idx_donasi_status_waktu (status, created_at),
  KEY idx_donasi_notifikasi (sumber, status, dilihat_pada),
  CONSTRAINT fk_donasi_petugas FOREIGN KEY (dicatat_oleh) REFERENCES users(id)
);

-- Penomor kuitansi per tahun tanpa celah: baris dikunci lewat
-- INSERT ... ON DUPLICATE KEY UPDATE di dalam transaksi yang sama dengan
-- perubahan status donasi, jadi nomor yang dibatalkan rollback tidak hilang.
CREATE TABLE donasi_urut (
  tahun    SMALLINT NOT NULL PRIMARY KEY,
  terakhir INT      NOT NULL
);
