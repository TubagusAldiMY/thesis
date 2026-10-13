// Package migrations menyematkan file migrasi SQL ke dalam biner (go:embed),
// sehingga server dan CLI migrasi tidak bergantung pada folder di disk.
//
// Aturan penamaan: V<nomor>__<deskripsi>.sql, mis. V001__initial_schema.sql.
// File yang sudah pernah dijalankan TIDAK boleh diedit; perubahan skema =
// file baru (migrate.Up menolak bila isi file lama berubah).
package migrations

import "embed"

// FS berisi semua file V*.sql di folder ini.
//
//go:embed V*.sql
var FS embed.FS
