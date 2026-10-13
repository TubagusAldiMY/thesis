package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"spkzakat/internal/domain"
)

// nullStr: string kosong → NULL (kolom opsional seperti actor_id).
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// InsertAudit mencatat satu baris audit log. Pemanggil bertanggung jawab
// memastikan Detail tidak memuat data pribadi/rahasia.
func (s *Store) InsertAudit(ctx context.Context, a domain.Audit) error {
	var detail any
	if len(a.Detail) > 0 {
		b, err := json.Marshal(a.Detail)
		if err != nil {
			return fmt.Errorf("store.InsertAudit: detail: %w", err)
		}
		detail = string(b)
	}
	_, err := s.q.ExecContext(ctx,
		`INSERT INTO audit_logs (id, actor_id, action, resource, resource_id, detail, ip_address)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		newID(), nullStr(a.ActorID), a.Action, a.Resource, nullStr(a.ResourceID), detail, nullStr(a.IP))
	if err != nil {
		return fmt.Errorf("store.InsertAudit: %w", mapErr(err))
	}
	return nil
}

type auditRow struct {
	ID         string         `db:"id"`
	ActorID    sql.NullString `db:"actor_id"`
	ActorNama  sql.NullString `db:"actor_nama"`
	Action     string         `db:"action"`
	Resource   string         `db:"resource"`
	ResourceID sql.NullString `db:"resource_id"`
	Detail     []byte         `db:"detail"`
	IP         sql.NullString `db:"ip_address"`
	CreatedAt  time.Time      `db:"created_at"`
}

// ListAudit mengembalikan satu halaman audit log terbaru dulu, beserta total.
// actions kosong = semua aksi. Nama pelaku diambil dari users (LEFT JOIN:
// login gagal dengan username tak dikenal tidak punya pelaku).
func (s *Store) ListAudit(ctx context.Context, actions []string, page, perPage int) ([]domain.Audit, int, error) {
	where, args := `1=1`, []any{}
	if len(actions) > 0 {
		q, a, err := sqlx.In(`a.action IN (?)`, actions)
		if err != nil {
			return nil, 0, fmt.Errorf("store.ListAudit: %w", err)
		}
		where, args = q, a
	}
	var total int
	if err := sqlx.GetContext(ctx, s.q, &total, `SELECT COUNT(*) FROM audit_logs a WHERE `+where, args...); err != nil {
		return nil, 0, fmt.Errorf("store.ListAudit: hitung: %w", mapErr(err))
	}
	var rows []auditRow
	err := sqlx.SelectContext(ctx, s.q, &rows,
		`SELECT a.id, a.actor_id, u.nama AS actor_nama, a.action, a.resource, a.resource_id,
		        a.detail, a.ip_address, a.created_at
		   FROM audit_logs a LEFT JOIN users u ON u.id = a.actor_id
		  WHERE `+where+`
		  ORDER BY a.created_at DESC, a.id
		  LIMIT ? OFFSET ?`, append(args, perPage, (page-1)*perPage)...)
	if err != nil {
		return nil, 0, fmt.Errorf("store.ListAudit: %w", mapErr(err))
	}
	hasil := make([]domain.Audit, 0, len(rows))
	for _, r := range rows {
		a := domain.Audit{
			ID: r.ID, ActorID: r.ActorID.String, ActorNama: r.ActorNama.String, Action: r.Action,
			Resource: r.Resource, ResourceID: r.ResourceID.String, IP: r.IP.String, CreatedAt: r.CreatedAt,
		}
		if len(r.Detail) > 0 {
			if err := json.Unmarshal(r.Detail, &a.Detail); err != nil {
				return nil, 0, fmt.Errorf("store.ListAudit: detail %s rusak: %w", r.ID, err)
			}
		}
		hasil = append(hasil, a)
	}
	return hasil, total, nil
}
