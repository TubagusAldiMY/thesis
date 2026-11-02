-- V005__program_donasi.sql
-- Program donasi ala penggalangan dana daring (permintaan Ketua Yayasan, Nov 2026):
-- pengurus membuat program bertarget (mis. renovasi atap, santunan yatim), muzaki
-- berdonasi infak ke program tertentu, dan halaman publik menampilkan kemajuan,
-- daftar donatur, serta kabar penyaluran. Zakat tidak diarahkan ke program
-- (peruntukannya ditentukan syariat dan disalurkan lewat ARAS), dijaga di service.
-- Konvensi sama dengan V001/V004: PK UUID, DATETIME(3), soft delete lewat deleted_at.

CREATE TABLE program_donasi (
  id           CHAR(36)      NOT NULL PRIMARY KEY,
  slug         VARCHAR(90)   NOT NULL,                -- alamat publik /program/<slug>; tetap setelah dibuat
  judul        VARCHAR(120)  NOT NULL,
  ringkasan    VARCHAR(200)  NOT NULL,                -- satu kalimat di kartu daftar program
  cerita       TEXT          NOT NULL,                -- uraian lengkap (teks biasa, paragraf dipisah baris kosong)
  target       DECIMAL(15,2) NOT NULL CHECK (target > 0),
  batas_waktu  DATE          NULL,                    -- NULL = tanpa batas waktu; donasi ditolak setelah tanggal ini
  status       ENUM('draf','aktif','selesai') NOT NULL DEFAULT 'draf',
  -- Gambar sampul disimpan di database, bukan di folder: ikut tercadang bersama
  -- data (mysqldump) dan tidak butuh volume tambahan. Maks 2 MB, dijaga service.
  gambar       MEDIUMBLOB    NULL,
  gambar_tipe  VARCHAR(20)   NULL,                    -- image/jpeg | image/png | image/webp (hasil deteksi isi berkas)
  dibuat_oleh  CHAR(36)      NULL,
  created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at   DATETIME(3)   NULL,
  UNIQUE KEY uq_program_slug (slug),
  KEY idx_program_status (status, created_at),
  CONSTRAINT fk_program_pembuat FOREIGN KEY (dibuat_oleh) REFERENCES users(id)
);

-- Kabar terbaru: catatan perkembangan/penyaluran yang ditulis pengurus.
CREATE TABLE program_kabar (
  id           CHAR(36)     NOT NULL PRIMARY KEY,
  program_id   CHAR(36)     NOT NULL,
  judul        VARCHAR(120) NOT NULL,
  isi          TEXT         NOT NULL,
  dibuat_oleh  CHAR(36)     NULL,
  created_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at   DATETIME(3)  NULL,
  KEY idx_kabar_program (program_id, created_at),
  CONSTRAINT fk_kabar_program FOREIGN KEY (program_id) REFERENCES program_donasi(id),
  CONSTRAINT fk_kabar_pembuat FOREIGN KEY (dibuat_oleh) REFERENCES users(id)
);

-- Donasi boleh terikat ke satu program (NULL = donasi umum seperti sebelumnya).
-- anonim = donatur memilih namanya tidak tampil di daftar donatur publik
-- (kuitansi tetap memakai nama yang diisi).
ALTER TABLE donasi
  ADD COLUMN program_id CHAR(36) NULL AFTER jenis,
  ADD COLUMN anonim BOOLEAN NOT NULL DEFAULT FALSE AFTER nama,
  ADD KEY idx_donasi_program (program_id, status),
  ADD CONSTRAINT fk_donasi_program FOREIGN KEY (program_id) REFERENCES program_donasi(id);
