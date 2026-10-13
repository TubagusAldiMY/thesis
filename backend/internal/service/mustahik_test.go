package service

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"spkzakat/internal/crypto"
	"spkzakat/internal/domain"
)

type barisMentah struct {
	nikEnc, noHPEnc []byte
	nikHash         string
}

// mentah membaca kolom terenkripsi langsung dari tabel.
func mentah(t *testing.T, e *lingkungan, id string) barisMentah {
	t.Helper()
	var b barisMentah
	if err := e.db.QueryRow(`SELECT nik_enc, no_hp_enc, nik_hash FROM mustahik WHERE id = ?`, id).Scan(&b.nikEnc, &b.noHPEnc, &b.nikHash); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBuatMustahik_Sukses_DataPribadiTerenkripsi(t *testing.T) {
	e := baru(t)
	v, err := e.svc.BuatMustahik(bg, masukanSah(func(in *MustahikInput) { in.Nama = "  Ahmad Fauzi  " }))
	if err != nil {
		t.Fatal(err)
	}

	if v.Nama != "Ahmad Fauzi" || v.NIKTersamar != "•••• •••• •••• 8901" || v.NoHPTersamar != "•••• 7890" ||
		len(v.ID) != 36 || !v.Aktif || v.DibuatPada.IsZero() {
		t.Errorf("view = %+v", v)
	}
	// Tampilan tidak boleh memuat NIK/HP utuh di field mana pun.
	for _, s := range []string{v.Nama, v.NIKTersamar, v.NoHPTersamar, v.Alamat} {
		if strings.Contains(s, "3271012345678901") || strings.Contains(s, "081234567890") {
			t.Errorf("plaintext bocor di view: %q", s)
		}
	}

	raw := mentah(t, e, v.ID)
	if bytes.Contains(raw.nikEnc, []byte("3271012345678901")) || bytes.Contains(raw.noHPEnc, []byte("081234567890")) {
		t.Error("plaintext NIK/HP tersimpan di database")
	}
	if raw.nikHash != e.idx.Sum("3271012345678901") {
		t.Errorf("nik_hash = %q, mau HMAC dari NIK", raw.nikHash)
	}
	// Dekripsi dengan AAD yang benar mengembalikan NIK asli.
	if got, err := e.enc.Decrypt(raw.nikEnc, []byte(aadNIK)); err != nil || string(got) != "3271012345678901" {
		t.Errorf("dekripsi NIK = %q, %v", got, err)
	}
}

func TestBuatMustahik_Normalisasi(t *testing.T) {
	e := baru(t)
	v, err := e.svc.BuatMustahik(bg, masukanSah(func(in *MustahikInput) {
		in.NIK, in.NoHP = " 3271 0123 4567 8901 ", "0812-3456 7890"
	}))
	if err != nil {
		t.Fatal(err)
	}
	raw := mentah(t, e, v.ID)
	if raw.nikHash != e.idx.Sum("3271012345678901") {
		t.Error("NIK bersepasi tidak dinormalkan sebelum di-hash")
	}
	if hp, _ := e.enc.Decrypt(raw.noHPEnc, []byte(aadNoHP)); string(hp) != "081234567890" {
		t.Errorf("No. HP tersimpan %q, mau 081234567890", hp)
	}
}

func TestBuatMustahik_TanpaNoHP(t *testing.T) {
	e := baru(t)
	v, err := e.svc.BuatMustahik(bg, masukanSah(func(in *MustahikInput) { in.NoHP = "" }))
	if err != nil {
		t.Fatal(err)
	}
	if v.NoHPTersamar != "" || mentah(t, e, v.ID).noHPEnc != nil {
		t.Errorf("No. HP kosong harus NULL: view=%q", v.NoHPTersamar)
	}
}

func TestBuatMustahik_Validasi(t *testing.T) {
	tests := []struct {
		nama  string
		ubah  func(*MustahikInput)
		field string // "" = harus sukses
	}{
		{"nama kosong", func(in *MustahikInput) { in.Nama = "   " }, "nama"},
		{"nama 1 karakter", func(in *MustahikInput) { in.Nama = "A" }, "nama"},
		{"nama 101 karakter", func(in *MustahikInput) { in.Nama = strings.Repeat("a", 101) }, "nama"},
		{"nama 2 karakter sah (hitung rune, bukan byte)", func(in *MustahikInput) { in.Nama = "日本" }, ""},
		{"NIK kosong", func(in *MustahikInput) { in.NIK = "" }, "nik"},
		{"NIK 15 digit", func(in *MustahikInput) { in.NIK = "327101234567890" }, "nik"},
		{"NIK 17 digit", func(in *MustahikInput) { in.NIK = "32710123456789012" }, "nik"},
		{"NIK mengandung huruf O", func(in *MustahikInput) { in.NIK = "3271O12345678901" }, "nik"},
		{"NIK digit Arab-Indic", func(in *MustahikInput) { in.NIK = "٣٢٧١٠١٢٣٤٥٦٧٨٩٠١" }, "nik"},
		{"No. HP bukan angka", func(in *MustahikInput) { in.NoHP = "abcdefghij" }, "no_hp"},
		{"No. HP terlalu pendek", func(in *MustahikInput) { in.NoHP = "12345" }, "no_hp"},
		{"No. HP 16 digit", func(in *MustahikInput) { in.NoHP = "+1234567890123456" }, "no_hp"},
		{"No. HP +62 sah", func(in *MustahikInput) { in.NoHP = "+6281234567890" }, ""},
		{"alamat terlalu pendek", func(in *MustahikInput) { in.Alamat = "Jl" }, "alamat"},
		{"alamat 501 karakter", func(in *MustahikInput) { in.Alamat = strings.Repeat("a", 501) }, "alamat"},
		{"asnaf kosong", func(in *MustahikInput) { in.Asnaf = "" }, "asnaf"},
		{"asnaf tidak dikenal", func(in *MustahikInput) { in.Asnaf = "bukan-asnaf" }, "asnaf"},
	}
	e := baru(t)
	for i, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			in := masukanSah(tt.ubah)
			if tt.field == "" { // kasus sah disimpan sungguhan: NIK-nya harus unik
				in.NIK = fmt.Sprintf("3271012345678%03d", i)
			}
			_, err := e.svc.BuatMustahik(bg, in)
			if tt.field == "" {
				if err != nil {
					t.Fatalf("err = %v, mau sukses", err)
				}
				return
			}
			cekValidasi(t, err, tt.field)
		})
	}

	t.Run("semua kesalahan dilaporkan sekaligus", func(t *testing.T) {
		_, err := e.svc.BuatMustahik(bg, MustahikInput{})
		ve := cekValidasi(t, err, "nama")
		for _, f := range []string{"nama", "nik", "alamat", "asnaf"} {
			if ve.Fields[f] == "" {
				t.Errorf("field %q tidak dilaporkan: %v", f, ve.Fields)
			}
		}
	})
}

func TestBuatMustahik_NIKDuplikat(t *testing.T) {
	e := baru(t)
	if _, err := e.svc.BuatMustahik(bg, masukanSah()); err != nil {
		t.Fatal(err)
	}
	for nama, nik := range map[string]string{"sama persis": "3271012345678901", "dengan spasi": "3271 0123 4567 8901"} {
		t.Run(nama, func(t *testing.T) {
			_, err := e.svc.BuatMustahik(bg, masukanSah(func(in *MustahikInput) { in.Nama, in.NIK = "Orang Lain", nik }))
			ve := cekValidasi(t, err, "nik")
			if !strings.Contains(ve.Fields["nik"], "sudah terdaftar") {
				t.Errorf("pesan = %q", ve.Fields["nik"])
			}
		})
	}
	var n int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM mustahik`).Scan(&n); err != nil || n != 1 {
		t.Errorf("jumlah baris = %d, mau 1", n)
	}
}

func TestUbahMustahik(t *testing.T) {
	e := baru(t)
	a := e.buatMustahik(t, "Ahmad", "3271012345678901", domain.AsnafFakir, true)
	b := e.buatMustahik(t, "Budi", "3271012345678902", domain.AsnafMiskin, true)

	t.Run("NIK dan No. HP kosong = tidak diubah (ciphertext pun tak disentuh)", func(t *testing.T) {
		sebelum := mentah(t, e, a.ID)
		v, err := e.svc.UbahMustahik(bg, a.ID, MustahikInput{Nama: "Ahmad Baru", Alamat: "Alamat Baru 123", Asnaf: domain.AsnafGharim, Aktif: true})
		if err != nil {
			t.Fatal(err)
		}
		sesudah := mentah(t, e, a.ID)
		if v.Nama != "Ahmad Baru" || v.Asnaf != domain.AsnafGharim || v.NIKTersamar != a.NIKTersamar || v.NoHPTersamar != a.NoHPTersamar {
			t.Errorf("view = %+v", v)
		}
		if !bytes.Equal(sebelum.nikEnc, sesudah.nikEnc) || !bytes.Equal(sebelum.noHPEnc, sesudah.noHPEnc) || sebelum.nikHash != sesudah.nikHash {
			t.Error("kolom terenkripsi berubah padahal NIK/HP tidak dikirim")
		}
	})

	t.Run("NIK milik sendiri dikirim ulang tidak dianggap duplikat", func(t *testing.T) {
		if _, err := e.svc.UbahMustahik(bg, a.ID, masukanSah(func(in *MustahikInput) { in.NIK = "3271012345678901" })); err != nil {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("NIK milik mustahik lain ditolak dan tidak ada yang berubah", func(t *testing.T) {
		sebelum := mentah(t, e, b.ID)
		_, err := e.svc.UbahMustahik(bg, b.ID, masukanSah(func(in *MustahikInput) { in.Nama, in.NIK = "Budi Diubah", "3271012345678901" }))
		cekValidasi(t, err, "nik")
		if got, _ := e.svc.AmbilMustahik(bg, b.ID); got.Nama != "Budi" {
			t.Errorf("nama berubah menjadi %q padahal ditolak", got.Nama)
		}
		if mentah(t, e, b.ID).nikHash != sebelum.nikHash {
			t.Error("hash NIK berubah")
		}
	})

	t.Run("NIK baru: tersimpan, NIK lama bebas dipakai lagi", func(t *testing.T) {
		v, err := e.svc.UbahMustahik(bg, b.ID, masukanSah(func(in *MustahikInput) { in.Nama, in.NIK, in.NoHP = "Budi", "3271012345679999", "" }))
		if err != nil {
			t.Fatal(err)
		}
		if v.NIKTersamar != "•••• •••• •••• 9999" {
			t.Errorf("NIKTersamar = %q", v.NIKTersamar)
		}
		if _, err := e.svc.BuatMustahik(bg, masukanSah(func(in *MustahikInput) { in.Nama, in.NIK = "Pemakai NIK Lama", "3271012345678902" })); err != nil {
			t.Errorf("NIK lama seharusnya bebas: %v", err)
		}
	})

	t.Run("validasi lebih dulu daripada akses database", func(t *testing.T) {
		_, err := e.svc.UbahMustahik(bg, idHantu, masukanSah(func(in *MustahikInput) { in.Nama = "" }))
		cekValidasi(t, err, "nama")
	})

	t.Run("id tidak ada", func(t *testing.T) {
		_, err := e.svc.UbahMustahik(bg, idHantu, masukanSah(func(in *MustahikInput) { in.NIK, in.NoHP = "", "" }))
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, mau ErrNotFound", err)
		}
	})
}

func TestUbahMustahik_NonaktifDiblokirSaatPeriodeBerjalan(t *testing.T) {
	e := baru(t)
	user := e.user(t)
	m := e.buatMustahik(t, "Ahmad", "3271012345678901", domain.AsnafFakir, true)
	p, err := e.svc.BuatPeriode(bg, user, PeriodeInput{Nama: "Zakat Fitrah", Jenis: domain.JenisZakat, DanaTersedia: 3_000_000, NominalPerPenerima: 1_000_000, AsnafDiizinkan: []domain.Asnaf{domain.AsnafFakir}})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.AturKandidat(bg, p.ID, []string{m.ID}); err != nil {
		t.Fatal(err)
	}
	nonaktif := masukanSah(func(in *MustahikInput) { in.NIK, in.NoHP, in.Aktif = "", "", false })

	tests := []struct {
		status   string
		diblokir bool
	}{
		{"draft", false}, {"dihitung", true}, {"disetujui", true}, {"ditolak", false}, {"selesai", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			e.exec(t, `UPDATE periode SET status = ? WHERE id = ?`, tt.status, p.ID)
			e.exec(t, `UPDATE mustahik SET is_active = TRUE WHERE id = ?`, m.ID)

			_, err := e.svc.UbahMustahik(bg, m.ID, nonaktif)

			got, _ := e.svc.AmbilMustahik(bg, m.ID)
			if tt.diblokir {
				if !errors.Is(err, domain.ErrConflict) {
					t.Errorf("err = %v, mau ErrConflict", err)
				}
				if !got.Aktif {
					t.Error("mustahik ikut nonaktif padahal diblokir")
				}
			} else {
				if err != nil {
					t.Errorf("err = %v, mau sukses", err)
				}
				if got.Aktif {
					t.Error("mustahik masih aktif padahal berhasil dinonaktifkan")
				}
			}
		})
	}
}

func TestDaftarMustahik(t *testing.T) {
	e := baru(t)
	e.buatMustahik(t, "Ahmad", "3271012345678901", domain.AsnafFakir, true)
	e.buatMustahik(t, "Budi", "3271012345678902", domain.AsnafMiskin, true)
	e.buatMustahik(t, "Citra", "3271012345678903", domain.AsnafMiskin, false)

	t.Run("semua, tersamar, dengan hitungan golongan aktif", func(t *testing.T) {
		h, err := e.svc.DaftarMustahik(bg, MustahikQuery{})
		if err != nil {
			t.Fatal(err)
		}
		if h.Total != 3 || len(h.Items) != 3 {
			t.Fatalf("total=%d items=%d", h.Total, len(h.Items))
		}
		for _, v := range h.Items {
			if !strings.HasPrefix(v.NIKTersamar, "•••• •••• •••• ") || strings.Contains(v.NIKTersamar, "32710123456789") {
				t.Errorf("NIK tidak tersamar: %q", v.NIKTersamar)
			}
		}
		if h.AsnafCounts[domain.AsnafFakir] != 1 || h.AsnafCounts[domain.AsnafMiskin] != 1 { // Citra nonaktif tak dihitung
			t.Errorf("AsnafCounts = %v", h.AsnafCounts)
		}
	})
	t.Run("filter golongan dan pencarian", func(t *testing.T) {
		h, err := e.svc.DaftarMustahik(bg, MustahikQuery{Asnaf: domain.AsnafMiskin, Q: "  bud "})
		if err != nil || h.Total != 1 || h.Items[0].Nama != "Budi" {
			t.Errorf("hasil = %+v, %v", h, err)
		}
	})
	t.Run("golongan tidak dikenal", func(t *testing.T) {
		_, err := e.svc.DaftarMustahik(bg, MustahikQuery{Asnaf: "xx"})
		cekValidasi(t, err, "asnaf")
	})
	t.Run("kata kunci terlalu panjang", func(t *testing.T) {
		_, err := e.svc.DaftarMustahik(bg, MustahikQuery{Q: strings.Repeat("a", 101)})
		cekValidasi(t, err, "q")
	})
}

func TestAmbilMustahik_TidakAda(t *testing.T) {
	e := baru(t)
	if _, err := e.svc.AmbilMustahik(bg, idHantu); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, mau ErrNotFound", err)
	}
}

// Kunci enkripsi yang salah harus TERLIHAT sebagai error (bukan data kosong
// diam-diam), dan pesannya tidak boleh memuat data pribadi.
func TestView_KunciSalahTerdeteksi(t *testing.T) {
	e := baru(t)
	v, err := e.svc.BuatMustahik(bg, masukanSah())
	if err != nil {
		t.Fatal(err)
	}
	kunciLain, _ := crypto.NewCipher(bytes.Repeat([]byte{9}, crypto.KeySize))
	salah := New(e.st, kunciLain, e.idx)

	_, err = salah.AmbilMustahik(bg, v.ID)
	if err == nil || errors.Is(err, domain.ErrNotFound) || !errors.Is(err, crypto.ErrDekripsi) {
		t.Fatalf("err = %v, mau error dekripsi (bukan nil/NotFound)", err)
	}
	if strings.Contains(err.Error(), "3271012345678901") {
		t.Errorf("pesan error memuat NIK: %v", err)
	}
	if _, err := salah.DaftarMustahik(bg, MustahikQuery{}); err == nil {
		t.Error("DaftarMustahik dengan kunci salah harus error, bukan daftar kosong")
	}
}

// Ciphertext kolom NIK yang dipindah ke kolom No. HP (atau sebaliknya) harus
// ditolak: AAD mengikat tiap ciphertext ke kolomnya.
func TestView_CiphertextDipindahAntarKolomDitolak(t *testing.T) {
	e := baru(t)
	v, _ := e.svc.BuatMustahik(bg, masukanSah())

	e.exec(t, `UPDATE mustahik SET no_hp_enc = nik_enc WHERE id = ?`, v.ID)
	if _, err := e.svc.AmbilMustahik(bg, v.ID); !errors.Is(err, crypto.ErrDekripsi) {
		t.Errorf("NIK dipasang di kolom HP: err = %v, mau ErrDekripsi", err)
	}

	v2, _ := e.svc.BuatMustahik(bg, masukanSah(func(in *MustahikInput) { in.NIK = "3271012345678902" }))
	e.exec(t, `UPDATE mustahik SET nik_enc = no_hp_enc WHERE id = ?`, v2.ID)
	if _, err := e.svc.AmbilMustahik(bg, v2.ID); !errors.Is(err, crypto.ErrDekripsi) {
		t.Errorf("HP dipasang di kolom NIK: err = %v, mau ErrDekripsi", err)
	}
}
