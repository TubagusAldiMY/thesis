-- V006__konten_kegiatan.sql
-- CMS konten kegiatan keagamaan Masjid Jami Al-Barokah (permintaan Ketua Yayasan, Nov 2026):
--   kegiatan     = artikel/acara (Maulid Nabi, Isra Mi'raj, kajian, dll.) berdraf & terbit
--   jadwal_rutin = jadwal kajian/kegiatan mingguan yang berulang
--   pengumuman   = pengumuman singkat dengan masa tayang
-- Konvensi sama dengan V001/V004/V005: PK UUID, DATETIME(3), soft delete lewat deleted_at.

CREATE TABLE kegiatan (
  id            CHAR(36)     NOT NULL PRIMARY KEY,
  slug          VARCHAR(90)  NOT NULL,                -- alamat publik /kegiatan/<slug>; tetap setelah dibuat
  judul         VARCHAR(120) NOT NULL,
  kategori      VARCHAR(30)  NOT NULL,                -- daftar sah ada di service (kategoriKegiatan); VARCHAR agar kategori baru tanpa migrasi
  ringkasan     VARCHAR(200) NOT NULL,
  isi           TEXT         NOT NULL,                -- teks biasa, paragraf dipisah baris kosong
  tanggal_acara DATE         NULL,                    -- NULL = artikel tanpa tanggal acara
  jam_acara     CHAR(5)      NULL,                    -- "HH:MM" waktu setempat (WIB)
  lokasi        VARCHAR(120) NULL,
  status        ENUM('draf','terbit') NOT NULL DEFAULT 'draf',
  gambar        MEDIUMBLOB   NULL,                    -- sama dengan program_donasi: di database, ≤ 2 MB, jenis dari isi berkas
  gambar_tipe   VARCHAR(20)  NULL,
  dibuat_oleh   CHAR(36)     NULL,
  created_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at    DATETIME(3)  NULL,
  UNIQUE KEY uq_kegiatan_slug (slug),
  KEY idx_kegiatan_status_tanggal (status, tanggal_acara),
  KEY idx_kegiatan_kategori (kategori),
  CONSTRAINT fk_kegiatan_pembuat FOREIGN KEY (dibuat_oleh) REFERENCES users(id)
);

CREATE TABLE jadwal_rutin (
  id         CHAR(36)     NOT NULL PRIMARY KEY,
  judul      VARCHAR(120) NOT NULL,                   -- mis. "Kajian Subuh"
  pemateri   VARCHAR(100) NULL,
  hari       TINYINT      NOT NULL CHECK (hari BETWEEN 0 AND 6), -- 0 = Ahad ... 6 = Sabtu
  jam        CHAR(5)      NOT NULL,                   -- "HH:MM" waktu setempat (WIB)
  lokasi     VARCHAR(120) NULL,
  keterangan VARCHAR(200) NULL,
  aktif      BOOLEAN      NOT NULL DEFAULT TRUE,      -- nonaktif = disembunyikan tanpa menghapus
  created_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3)  NULL,
  KEY idx_jadwal_hari_jam (aktif, hari, jam)
);

CREATE TABLE pengumuman (
  id            CHAR(36)     NOT NULL PRIMARY KEY,
  judul         VARCHAR(120) NOT NULL,
  isi           VARCHAR(500) NOT NULL,
  tampil_sampai DATE         NULL,                    -- NULL = tayang terus sampai disembunyikan; batas termasuk (hari terakhir masih tayang)
  terbit        BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at    DATETIME(3)  NULL,
  KEY idx_pengumuman_tayang (terbit, tampil_sampai)
);
