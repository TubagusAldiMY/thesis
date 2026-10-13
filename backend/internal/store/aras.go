package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

const kolomRunTanpaLangkah = `id, periode_id, versi, bobot_snapshot, s0, is_current, dijalankan_oleh, created_at`
const kolomRun = `id, periode_id, versi, bobot_snapshot, s0, langkah, is_current, dijalankan_oleh, created_at`

type runRow struct {
	ID             string    `db:"id"`
	PeriodeID      string    `db:"periode_id"`
	Versi          int       `db:"versi"`
	BobotSnapshot  []byte    `db:"bobot_snapshot"`
	S0             float64   `db:"s0"`
	Langkah        []byte    `db:"langkah"`
	IsCurrent      bool      `db:"is_current"`
	DijalankanOleh string    `db:"dijalankan_oleh"`
	CreatedAt      time.Time `db:"created_at"`
}

func (r runRow) domain() (domain.ArasRun, error) {
	run := domain.ArasRun{
		ID: r.ID, PeriodeID: r.PeriodeID, Versi: r.Versi, S0: r.S0, Langkah: r.Langkah,
		IsCurrent: r.IsCurrent, DijalankanOleh: r.DijalankanOleh, CreatedAt: r.CreatedAt,
	}
	if err := json.Unmarshal(r.BobotSnapshot, &run.Bobot); err != nil {
		return domain.ArasRun{}, fmt.Errorf("bobot_snapshot run %s tidak valid: %w", r.ID, err)
	}
	return run, nil
}

// desimal8 menulis angka 8 desimal sesuai DECIMAL(12,8) (pembulatan di sini,
// eksplisit, bukan bergantung perilaku server).
func desimal8(f float64) string { return strconv.FormatFloat(f, 'f', 8, 64) }

// SimpanRun menyimpan satu run beserta seluruh hasilnya, ATOMIK:
//  1. versi = versi terbesar periode + 1 (mengisi run.Versi);
//  2. run-run sebelumnya menjadi is_current = FALSE;
//  3. run baru (is_current = TRUE) dan semua barisnya disimpan.
//
// Bila satu langkah gagal (mis. mustahik dobel, periode tidak ada), tidak ada
// yang berubah — termasuk penanda is_current run sebelumnya. Penomoran versi
// aman dari balapan hanya bila pemanggil memegang kunci baris periode
// (GetPeriodeForUpdate di transaksi yang sama); service melakukannya.
//
// run.ID, Versi, IsCurrent, dan CreatedAt diisi dari database. run.Bobot wajib
// berisi potret kriteria; run.Langkah boleh nil.
func (s *Store) SimpanRun(ctx context.Context, run *domain.ArasRun, hasil []domain.ArasHasil) error {
	bobot, err := json.Marshal(run.Bobot)
	if err != nil {
		return fmt.Errorf("store.SimpanRun: bobot snapshot: %w", err)
	}
	var langkah any // nil → NULL
	if len(run.Langkah) > 0 {
		langkah = string(run.Langkah)
	}

	id := newID()
	err = s.inTx(ctx, func(tx *Store) error {
		var versi int
		if err := sqlx.GetContext(ctx, tx.q, &versi,
			`SELECT COALESCE(MAX(versi), 0) + 1 FROM aras_run WHERE periode_id = ?`, run.PeriodeID); err != nil {
			return fmt.Errorf("store.SimpanRun: versi: %w", mapErr(err))
		}
		if _, err := tx.q.ExecContext(ctx,
			`UPDATE aras_run SET is_current = FALSE WHERE periode_id = ?`, run.PeriodeID); err != nil {
			return fmt.Errorf("store.SimpanRun: lepas run lama: %w", mapErr(err))
		}
		if _, err := tx.q.ExecContext(ctx,
			`INSERT INTO aras_run (id, periode_id, versi, bobot_snapshot, s0, langkah, is_current, dijalankan_oleh)
			 VALUES (?, ?, ?, ?, ?, ?, TRUE, ?)`,
			id, run.PeriodeID, versi, string(bobot), desimal8(run.S0), langkah, run.DijalankanOleh); err != nil {
			return fmt.Errorf("store.SimpanRun: simpan run: %w", mapErr(err))
		}
		for _, h := range hasil {
			if _, err := tx.q.ExecContext(ctx,
				`INSERT INTO aras_hasil (id, run_id, mustahik_id, s, k, ranking, status)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				newID(), id, h.MustahikID, desimal8(h.S), desimal8(h.K), h.Ranking, string(h.Status)); err != nil {
				return fmt.Errorf("store.SimpanRun: simpan hasil: %w", mapErr(err))
			}
		}
		var r runRow
		if err := sqlx.GetContext(ctx, tx.q, &r, `SELECT `+kolomRun+` FROM aras_run WHERE id = ?`, id); err != nil {
			return fmt.Errorf("store.SimpanRun: baca ulang: %w", mapErr(err))
		}
		dibuat, err := r.domain()
		if err != nil {
			return fmt.Errorf("store.SimpanRun: %w", err)
		}
		*run = dibuat
		return nil
	})
	return err
}

// GetRun mengambil satu run lengkap dengan langkah hitungnya. versi ≤ 0 =
// run terkini (is_current). ErrNotFound bila periode belum pernah dihitung
// atau versi tidak ada.
func (s *Store) GetRun(ctx context.Context, periodeID string, versi int) (domain.ArasRun, error) {
	query := `SELECT ` + kolomRun + ` FROM aras_run WHERE periode_id = ? AND is_current = TRUE`
	args := []any{periodeID}
	if versi > 0 {
		query = `SELECT ` + kolomRun + ` FROM aras_run WHERE periode_id = ? AND versi = ?`
		args = append(args, versi)
	}
	var r runRow
	if err := sqlx.GetContext(ctx, s.q, &r, query, args...); err != nil {
		return domain.ArasRun{}, fmt.Errorf("store.GetRun: %w", mapErr(err))
	}
	run, err := r.domain()
	if err != nil {
		return domain.ArasRun{}, fmt.Errorf("store.GetRun: %w", err)
	}
	return run, nil
}

// ListRun mengembalikan riwayat run periode, terbaru dulu, TANPA kolom langkah
// (bisa besar; ambil lewat GetRun bila perlu).
func (s *Store) ListRun(ctx context.Context, periodeID string) ([]domain.ArasRun, error) {
	var rows []runRow
	if err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT `+kolomRunTanpaLangkah+` FROM aras_run WHERE periode_id = ? ORDER BY versi DESC`, periodeID); err != nil {
		return nil, fmt.Errorf("store.ListRun: %w", mapErr(err))
	}
	hasil := make([]domain.ArasRun, 0, len(rows))
	for _, r := range rows {
		run, err := r.domain()
		if err != nil {
			return nil, fmt.Errorf("store.ListRun: %w", err)
		}
		hasil = append(hasil, run)
	}
	return hasil, nil
}

type hasilRow struct {
	MustahikID string  `db:"mustahik_id"`
	S          float64 `db:"s"`
	K          float64 `db:"k"`
	Ranking    int     `db:"ranking"`
	Status     string  `db:"status"`
}

// ListHasil mengembalikan hasil sebuah run menurut ranking (1, 2, 3, …).
func (s *Store) ListHasil(ctx context.Context, runID string) ([]domain.ArasHasil, error) {
	var rows []hasilRow
	if err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT mustahik_id, s, k, ranking, status FROM aras_hasil WHERE run_id = ? ORDER BY ranking`, runID); err != nil {
		return nil, fmt.Errorf("store.ListHasil: %w", mapErr(err))
	}
	hasil := make([]domain.ArasHasil, 0, len(rows))
	for _, r := range rows {
		hasil = append(hasil, domain.ArasHasil{
			MustahikID: r.MustahikID, S: r.S, K: r.K, Ranking: r.Ranking, Status: domain.StatusHasil(r.Status),
		})
	}
	return hasil, nil
}
