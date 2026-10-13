# 03 — Perancangan Sistem (UML + ERD)

Semua diagram memakai Mermaid (render otomatis di GitHub/VS Code/Obsidian; untuk skripsi ekspor ke PNG lewat <https://mermaid.live>).

## 1. Use Case Diagram
Mermaid tidak punya notasi use case bawaan, jadi dipakai flowchart: kotak = aktor, oval = use case.

```mermaid
flowchart LR
    Admin([Admin / Petugas Amil])
    Ketua([Ketua DKM])

    subgraph Sistem["SPK Penyaluran Zakat & Infak"]
        UC1(Login / Logout)
        UC2(Kelola Mustahik)
        UC3(Kelola Kriteria & Bobot)
        UC4(Kelola Periode)
        UC5(Pilih Kandidat & Input Penilaian)
        UC6(Hitung ARAS)
        UC7(Lihat Ranking & Langkah Hitung)
        UC8(Setujui / Tolak Usulan Penerima)
        UC9(Catat Penyaluran)
        UC10(Cetak Laporan)
        UC11(Buka Kembali Periode Ditolak)
        UC12(Lihat Audit Log)
    end

    Admin --- UC1 & UC2 & UC3 & UC4 & UC5 & UC6 & UC7 & UC9 & UC10 & UC11 & UC12
    Ketua --- UC1 & UC7 & UC8 & UC10 & UC12
    UC11 -. precondition: ditolak .-> UC8
    UC6 -. include .-> UC5
    UC9 -. precondition: disetujui .-> UC8
```

## 2. Status Periode (State Diagram)
```mermaid
stateDiagram-v2
    [*] --> draft: Admin buat periode
    draft --> dihitung: Hitung ARAS (FR-09)
    dihitung --> dihitung: Hitung ulang (versi run baru)
    dihitung --> disetujui: Ketua setuju (FR-12)
    dihitung --> ditolak: Ketua tolak + catatan
    ditolak --> draft: Admin buka kembali
    disetujui --> selesai: Semua penyaluran tercatat / ditutup
    selesai --> [*]
```
Aturan: nilai penilaian hanya bisa diubah saat `draft`; penyaluran hanya saat `disetujui` (FR-13).

## 3. Activity Diagram — Proses Penyaluran
```mermaid
flowchart TD
    A([Mulai]) --> B[Admin buat periode<br/>jenis, dana, nominal/penerima]
    B --> C[Pilih mustahik kandidat<br/>zakat: filter asnaf]
    C --> D[Input nilai C1..C5 tiap kandidat]
    D --> E{Semua nilai<br/>lengkap & valid?}
    E -- Tidak --> D
    E -- Ya --> F[Klik Hitung ARAS]
    F --> G[Sistem hitung K & ranking<br/>simpan sebagai run baru]
    G --> H[Tandai N teratas = usulan<br/>N = dana ÷ nominal]
    H --> I{Ketua DKM<br/>menyetujui?}
    I -- Tidak --> J[Catatan penolakan] --> D
    I -- Ya --> K[Admin catat penyaluran<br/>per penerima]
    K --> L[Cetak laporan]
    L --> M([Selesai])
```

## 4. Sequence Diagram — Hitung ARAS
```mermaid
sequenceDiagram
    actor Admin
    participant UI as SvelteKit (+page.server.ts)
    participant API as Go Handler
    participant UC as Service HitungAras
    participant AR as aras.Hitung (fungsi murni)
    participant DB as MySQL

    Admin->>UI: klik "Hitung ARAS"
    UI->>API: POST /api/periode/{id}/hitung (cookie sesi)
    API->>API: cek sesi, peran=admin, validasi id
    API->>UC: Hitung(periodeID)
    UC->>DB: SELECT periode (status=draft?)
    UC->>DB: SELECT kriteria aktif + bobot
    UC->>DB: SELECT penilaian kandidat
    UC->>UC: cek Σ bobot = 1, nilai lengkap, nilai > 0
    UC->>AR: Hitung(kriteria, alternatif)
    AR-->>UC: S0, tiap (S, K, ranking), langkah antara
    UC->>DB: transaksi: INSERT aras_run + aras_hasil, UPDATE periode.status='dihitung', INSERT audit_log
    UC-->>API: HasilAras
    API-->>UI: 200 {run_id, ranking[], kuota}
    UI-->>Admin: tabel ranking + langkah hitung
```

## 5. Class Diagram (Backend Go)
Hanya struct inti; Go tidak punya class, digambar sebagai tipe + method.
```mermaid
classDiagram
    class Kriteria {
        +string Kode
        +Tipe Tipe
        +float64 Bobot
    }
    class Alternatif {
        +string ID
        +[]float64 Nilai
    }
    class HasilAlternatif {
        +string ID
        +float64 S
        +float64 K
        +int Ranking
    }
    class Hasil {
        +float64 S0
        +[]HasilAlternatif Alternatif
        +Langkah Langkah
    }
    class ArasService {
        +Hitung(ctx, periodeID) Hasil
        +Kuota(periode) int
    }
    class PeriodeRepo {
        <<interface>>
    }
    class PenilaianRepo {
        <<interface>>
    }
    class KriteriaRepo {
        <<interface>>
    }

    ArasService --> PeriodeRepo
    ArasService --> PenilaianRepo
    ArasService --> KriteriaRepo
    ArasService ..> Hasil : memakai aras.Hitung
    Hasil "1" o-- "*" HasilAlternatif
```
Catatan: `Tipe` = `benefit | cost`; `Nilai` diurutkan sesuai urutan `Kriteria`; `Langkah` = matriks antara (normalisasi, terbobot) untuk ditampilkan di UI. Interface repo dipakai agar `ArasService` bisa di-unit-test dengan mock (rules global: unit test fokus ke service, repo diuji integrasi ke MySQL asli). `aras.Hitung` murni (tanpa I/O) → test table-driven memakai contoh dokumen 02.

## 6. ERD
```mermaid
erDiagram
    USERS ||--o{ SESSIONS : punya
    USERS ||--o{ PERIODE : membuat
    USERS ||--o{ ARAS_RUN : menjalankan
    USERS ||--o{ PERSETUJUAN : memutuskan
    USERS ||--o{ PENYALURAN : mencatat
    USERS ||--o{ AUDIT_LOG : pelaku

    PERIODE ||--o{ PERIODE_KANDIDAT : memiliki
    MUSTAHIK ||--o{ PERIODE_KANDIDAT : menjadi
    PERIODE ||--o{ PENILAIAN : memiliki
    MUSTAHIK ||--o{ PENILAIAN : dinilai
    KRITERIA ||--o{ PENILAIAN : atas

    PERIODE ||--o{ ARAS_RUN : dihitung_dalam
    ARAS_RUN ||--o{ ARAS_HASIL : berisi
    MUSTAHIK ||--o{ ARAS_HASIL : peringkat
    ARAS_RUN ||--o| PERSETUJUAN : diputuskan
    PERIODE ||--o{ PENYALURAN : menyalurkan
    MUSTAHIK ||--o{ PENYALURAN : menerima

    USERS {
        char36 id PK
        varchar username UK
        varchar password_hash
        varchar nama
        enum role "admin|ketua"
    }
    MUSTAHIK {
        char36 id PK
        varchar nama
        varbinary nik_enc
        char64 nik_hash UK
        text alamat
        varbinary no_hp_enc
        enum asnaf
    }
    KRITERIA {
        char36 id PK
        varchar kode UK
        varchar nama
        enum tipe "benefit|cost"
        decimal bobot
        json skala_keterangan
        bool is_active
    }
    PERIODE {
        char36 id PK
        varchar nama
        enum jenis "zakat|infak"
        decimal dana_tersedia
        decimal nominal_per_penerima
        set asnaf_diizinkan
        enum status
    }
    PENILAIAN {
        char36 id PK
        char36 periode_id FK
        char36 mustahik_id FK
        char36 kriteria_id FK
        decimal nilai
    }
    ARAS_RUN {
        char36 id PK
        char36 periode_id FK
        int versi
        json bobot_snapshot
        decimal s0
        bool is_current
    }
    ARAS_HASIL {
        char36 id PK
        char36 run_id FK
        char36 mustahik_id FK
        decimal s
        decimal k
        int ranking
        enum status "usulan|cadangan"
    }
    PERSETUJUAN {
        char36 id PK
        char36 run_id FK
        enum keputusan "setuju|tolak"
        text catatan
    }
    PENYALURAN {
        char36 id PK
        char36 periode_id FK
        char36 mustahik_id FK
        decimal nominal
        date tanggal
    }
```

## 7. Skema Tabel (MySQL 8, InnoDB, utf8mb4)
> **Sudah diimplementasikan** sebagai [`backend/migrations/V001__initial_schema.sql`](../backend/migrations/V001__initial_schema.sql) (skema) dan `V002__seed_kriteria.sql` (5 kriteria awal), dan terverifikasi di MySQL 8.0 asli. Bila ada selisih antara blok di bawah dan file migrasi, **file migrasi yang berlaku**. Perbedaan yang sengaja: komentar `audit_logs.action` diperpendek, dan ID dibuat oleh aplikasi (UUID v4) walau kolom tetap punya `DEFAULT (UUID())`. `V003__aras_run_langkah.sql` menambah kolom `aras_run.langkah JSON NULL` (langkah hitung tiap run: A0, matriks, normalisasi, terbobot) agar "Lihat langkah hitung" tetap bisa menampilkan run lama; kolom ini belum ada di blok di bawah. Perubahan skema berikutnya harus lewat file migrasi baru (`V004__…`), bukan mengedit blok di bawah.

Konvensi mengikuti `db-patterns.md`: snake_case plural, PK UUID, `created_at/updated_at/deleted_at` (soft delete), migrasi versioned (`V001__...sql`, tidak diedit setelah deploy).

```sql
-- V001__initial_schema.sql
CREATE TABLE users (
  id            CHAR(36)     NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  username      VARCHAR(50)  NOT NULL,
  password_hash VARCHAR(255) NOT NULL,              -- Argon2id
  nama          VARCHAR(100) NOT NULL,
  role          ENUM('admin','ketua') NOT NULL,
  is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at    DATETIME(3)  NULL,
  UNIQUE KEY uq_users_username (username)
);

CREATE TABLE sessions (                              -- sesi di DB agar logout benar-benar mencabut sesi
  id          CHAR(36)    NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  user_id     CHAR(36)    NOT NULL,
  token_hash  CHAR(64)    NOT NULL,                  -- hash token, bukan plaintext
  expires_at  DATETIME(3) NOT NULL,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_sessions_token_hash (token_hash),
  KEY idx_sessions_user_id (user_id),
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE mustahik (
  id         CHAR(36)     NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  nama       VARCHAR(100) NOT NULL,
  nik_enc    VARBINARY(255) NOT NULL,                -- AES-256-GCM di app layer (PII, UU PDP)
  nik_hash   CHAR(64)     NOT NULL,                  -- HMAC-SHA256 untuk cek duplikat
  alamat     TEXT         NOT NULL,
  no_hp_enc  VARBINARY(255) NULL,
  asnaf      ENUM('fakir','miskin','amil','muallaf','riqab','gharim','fisabilillah','ibnu_sabil') NOT NULL,
  is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3)  NULL,
  UNIQUE KEY uq_mustahik_nik_hash (nik_hash)
);

CREATE TABLE kriteria (
  id               CHAR(36)     NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  kode             VARCHAR(10)  NOT NULL,
  nama             VARCHAR(100) NOT NULL,
  tipe             ENUM('benefit','cost') NOT NULL,
  bobot            DECIMAL(5,4) NOT NULL CHECK (bobot > 0 AND bobot <= 1),
  skala_keterangan JSON         NULL,                -- {"1":"layak milik sendiri", ...}
  urutan           TINYINT      NOT NULL,
  is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
  created_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at       DATETIME(3)  NULL,
  UNIQUE KEY uq_kriteria_kode (kode)
);

CREATE TABLE periode (
  id                   CHAR(36)      NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  nama                 VARCHAR(100)  NOT NULL,
  jenis                ENUM('zakat','infak') NOT NULL,
  dana_tersedia        DECIMAL(15,2) NOT NULL CHECK (dana_tersedia > 0),
  nominal_per_penerima DECIMAL(15,2) NOT NULL CHECK (nominal_per_penerima > 0),
  asnaf_diizinkan      SET('fakir','miskin','amil','muallaf','riqab','gharim','fisabilillah','ibnu_sabil') NULL,
  status               ENUM('draft','dihitung','disetujui','ditolak','selesai') NOT NULL DEFAULT 'draft',
  created_by           CHAR(36)      NOT NULL,
  created_at           DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at           DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at           DATETIME(3)   NULL,
  KEY idx_periode_status (status),
  CONSTRAINT fk_periode_user FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE periode_kandidat (
  periode_id  CHAR(36) NOT NULL,
  mustahik_id CHAR(36) NOT NULL,
  PRIMARY KEY (periode_id, mustahik_id),
  CONSTRAINT fk_pk_periode  FOREIGN KEY (periode_id)  REFERENCES periode(id)  ON DELETE CASCADE,
  CONSTRAINT fk_pk_mustahik FOREIGN KEY (mustahik_id) REFERENCES mustahik(id)
);

CREATE TABLE penilaian (
  id          CHAR(36)      NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  periode_id  CHAR(36)      NOT NULL,
  mustahik_id CHAR(36)      NOT NULL,
  kriteria_id CHAR(36)      NOT NULL,
  nilai       DECIMAL(15,4) NOT NULL CHECK (nilai > 0),   -- cost dipakai sebagai 1/x
  created_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_penilaian (periode_id, mustahik_id, kriteria_id),
  CONSTRAINT fk_penilaian_kandidat FOREIGN KEY (periode_id, mustahik_id)
    REFERENCES periode_kandidat(periode_id, mustahik_id) ON DELETE CASCADE,
  CONSTRAINT fk_penilaian_kriteria FOREIGN KEY (kriteria_id) REFERENCES kriteria(id)
);

CREATE TABLE aras_run (                               -- tiap "Hitung ARAS" = 1 run; riwayat tidak dihapus
  id              CHAR(36)     NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  periode_id      CHAR(36)     NOT NULL,
  versi           INT          NOT NULL,
  bobot_snapshot  JSON         NOT NULL,              -- bobot & tipe saat dihitung → hasil bisa direproduksi
  s0              DECIMAL(12,8) NOT NULL,
  is_current      BOOLEAN      NOT NULL DEFAULT TRUE,
  dijalankan_oleh CHAR(36)     NOT NULL,
  created_at      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_run_versi (periode_id, versi),
  CONSTRAINT fk_run_periode FOREIGN KEY (periode_id) REFERENCES periode(id),
  CONSTRAINT fk_run_user    FOREIGN KEY (dijalankan_oleh) REFERENCES users(id)
);

CREATE TABLE aras_hasil (
  id          CHAR(36)      NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  run_id      CHAR(36)      NOT NULL,
  mustahik_id CHAR(36)      NOT NULL,
  s           DECIMAL(12,8) NOT NULL,
  k           DECIMAL(12,8) NOT NULL,
  ranking     INT           NOT NULL,
  status      ENUM('usulan','cadangan') NOT NULL,
  UNIQUE KEY uq_hasil_run_mustahik (run_id, mustahik_id),
  KEY idx_hasil_run_ranking (run_id, ranking),
  CONSTRAINT fk_hasil_run      FOREIGN KEY (run_id)      REFERENCES aras_run(id) ON DELETE CASCADE,
  CONSTRAINT fk_hasil_mustahik FOREIGN KEY (mustahik_id) REFERENCES mustahik(id)
);

CREATE TABLE persetujuan (
  id         CHAR(36)    NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  run_id     CHAR(36)    NOT NULL,
  keputusan  ENUM('setuju','tolak') NOT NULL,
  catatan    TEXT        NULL,
  oleh       CHAR(36)    NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_persetujuan_run (run_id),
  CONSTRAINT fk_setuju_run  FOREIGN KEY (run_id) REFERENCES aras_run(id),
  CONSTRAINT fk_setuju_user FOREIGN KEY (oleh)   REFERENCES users(id)
);

CREATE TABLE penyaluran (
  id           CHAR(36)      NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  periode_id   CHAR(36)      NOT NULL,
  mustahik_id  CHAR(36)      NOT NULL,
  nominal      DECIMAL(15,2) NOT NULL CHECK (nominal > 0),
  tanggal      DATE          NOT NULL,
  dicatat_oleh CHAR(36)      NOT NULL,
  created_at   DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_penyaluran (periode_id, mustahik_id),   -- 1 penerima 1x per periode
  CONSTRAINT fk_salur_periode  FOREIGN KEY (periode_id)   REFERENCES periode(id),
  CONSTRAINT fk_salur_mustahik FOREIGN KEY (mustahik_id)  REFERENCES mustahik(id),
  CONSTRAINT fk_salur_user     FOREIGN KEY (dicatat_oleh) REFERENCES users(id)
);

CREATE TABLE audit_logs (
  id          CHAR(36)    NOT NULL DEFAULT (UUID()) PRIMARY KEY,
  actor_id    CHAR(36)    NULL,
  action      VARCHAR(30) NOT NULL,                  -- LOGIN|LOGIN_GAGAL|MUSTAHIK_*|KRITERIA_UBAH|PERIODE_BUAT|KANDIDAT|PENILAIAN|HITUNG|SETUJU|TOLAK|BUKA_KEMBALI|SALUR|TUTUP (dipetakan ke 5 kategori filter UI di dokumen 04)
  resource    VARCHAR(30) NOT NULL,
  resource_id CHAR(36)    NULL,
  detail      JSON        NULL,                      -- tanpa NIK/HP/password
  ip_address  VARCHAR(45) NULL,
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_audit_actor (actor_id),
  KEY idx_audit_resource (resource, resource_id),
  KEY idx_audit_created (created_at)
);
```

### Trade-off & catatan skema
- **UUID `CHAR(36)`** mengikuti konvensi global (mudah dibaca/di-debug). Ceiling: lebih boros index daripada `BINARY(16)`; untuk skala masjid (ratusan–ribuan baris) tidak terasa.
- **`bobot_snapshot` di `aras_run`**: bobot kriteria boleh berubah kapan saja, tetapi hasil lama tetap bisa dijelaskan/direproduksi (penting untuk akuntabilitas & pembuktian di sidang).
- **NIK/HP terenkripsi** + `nik_hash` untuk deteksi duplikat tanpa membuka plaintext (UU PDP No. 27/2022). Kunci enkripsi dari env var, bukan di repo.
- MySQL tidak mendukung partial index → filter `deleted_at IS NULL` tetap ditulis di query; index biasa cukup untuk skala ini.
- Validasi Σ bobot = 1 dilakukan di service (CHECK lintas baris tidak bisa di MySQL).

Lanjut: [04 — Rancangan UI & Arsitektur](04-rancangan-ui-arsitektur.md).
