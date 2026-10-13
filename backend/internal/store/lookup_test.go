package store

import (
	"errors"
	"testing"
	"time"

	"spkzakat/internal/domain"
)

func TestGetMustahikByIDs(t *testing.T) {
	s := newStore(t)
	a := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)
	b := buatMustahik(t, s, "Budi", domain.AsnafMiskin)
	c := buatMustahik(t, s, "Citra", domain.AsnafMiskin)
	if _, err := s.db.ExecContext(bg, `UPDATE mustahik SET deleted_at = NOW(3) WHERE id = ?`, c.ID); err != nil {
		t.Fatal(err)
	}
	hantu := "00000000-0000-4000-8000-000000000000"

	tests := []struct {
		nama string
		ids  []string
		mau  int
	}{
		{"semua ada", []string{a.ID, b.ID}, 2},
		{"sebagian tidak ada: yang hilang tidak ikut (tanpa error)", []string{a.ID, hantu}, 1},
		{"yang soft-deleted tidak ikut", []string{a.ID, c.ID}, 1},
		{"semua tidak ada", []string{hantu}, 0},
		{"slice kosong tidak error", nil, 0},
		{"input mencurigakan diperlakukan sebagai ID biasa", []string{`' OR '1'='1`}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			got, err := s.GetMustahikByIDs(bg, tt.ids)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tt.mau {
				t.Errorf("jumlah = %d, mau %d", len(got), tt.mau)
			}
		})
	}
}

func TestMustahikDalamPeriodeBerjalan(t *testing.T) {
	s := newStore(t)
	user := seedUser(t, s)
	m := buatMustahik(t, s, "Ahmad", domain.AsnafFakir)

	if ada, err := s.MustahikDalamPeriodeBerjalan(bg, m.ID); err != nil || ada {
		t.Fatalf("bukan kandidat mana pun: ada=%v err=%v, mau false", ada, err)
	}

	p := periodeBaru(user)
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceKandidat(bg, p.ID, []string{m.ID}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		status domain.StatusPeriode
		mau    bool
	}{
		{domain.StatusDraft, false},
		{domain.StatusDihitung, true},
		{domain.StatusDisetujui, true},
		{domain.StatusDitolak, false},
		{domain.StatusSelesai, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if _, err := s.db.ExecContext(bg, `UPDATE periode SET status = ? WHERE id = ?`, string(tt.status), p.ID); err != nil {
				t.Fatal(err)
			}
			ada, err := s.MustahikDalamPeriodeBerjalan(bg, m.ID)
			if err != nil || ada != tt.mau {
				t.Errorf("ada=%v err=%v, mau %v", ada, err, tt.mau)
			}
		})
	}

	t.Run("periode yang soft-deleted tidak dihitung", func(t *testing.T) {
		if _, err := s.db.ExecContext(bg, `UPDATE periode SET status = 'dihitung', deleted_at = NOW(3) WHERE id = ?`, p.ID); err != nil {
			t.Fatal(err)
		}
		if ada, _ := s.MustahikDalamPeriodeBerjalan(bg, m.ID); ada {
			t.Error("periode terhapus masih dihitung sebagai berjalan")
		}
	})
}

func TestGetPeriodeForUpdate(t *testing.T) {
	s := newStore(t)
	p := periodeBaru(seedUser(t, s))
	if err := s.CreatePeriode(bg, &p); err != nil {
		t.Fatal(err)
	}

	t.Run("membaca data sama dengan GetPeriode", func(t *testing.T) {
		got, err := s.GetPeriodeForUpdate(bg, p.ID)
		if err != nil || got.ID != p.ID || got.Status != domain.StatusDraft || got.DanaTersedia != 3_000_000 {
			t.Errorf("got = %+v, err = %v", got, err)
		}
	})

	t.Run("id tidak ada", func(t *testing.T) {
		if _, err := s.GetPeriodeForUpdate(bg, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})

	// Inti fungsinya: selama transaksi memegang kunci, perubahan status dari
	// luar HARUS menunggu. Tanpa FOR UPDATE ia langsung lolos di tengah transaksi.
	t.Run("mengunci baris: SetStatus dari luar menunggu sampai transaksi selesai", func(t *testing.T) {
		selesai := make(chan error, 1)
		var terkunciSaatCek bool

		err := s.WithTx(bg, func(tx *Store) error {
			if _, err := tx.GetPeriodeForUpdate(bg, p.ID); err != nil {
				return err
			}
			go func() { selesai <- s.SetStatusPeriode(bg, p.ID, domain.StatusDraft, domain.StatusDihitung) }()

			select {
			case err := <-selesai:
				t.Errorf("SetStatus lolos padahal baris dikunci (err=%v)", err)
			case <-time.After(400 * time.Millisecond):
				terkunciSaatCek = true // masih menunggu = terkunci, sesuai harapan
			}
			return nil // commit → kunci lepas
		})
		if err != nil {
			t.Fatal(err)
		}
		if !terkunciSaatCek {
			t.Fatal("tidak terbukti terkunci")
		}
		select {
		case err := <-selesai:
			if err != nil {
				t.Errorf("SetStatus setelah kunci lepas: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("SetStatus tidak selesai setelah transaksi commit")
		}
		if got, _ := s.GetPeriode(bg, p.ID); got.Status != domain.StatusDihitung {
			t.Errorf("status = %s, mau dihitung", got.Status)
		}
	})
}
