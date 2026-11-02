// Package httpapi adalah lapisan HTTP (REST JSON) di atas service: routing,
// middleware (request_id, log akses, recover, sesi, peran), dekode/enkode JSON,
// dan penerjemahan error domain → status HTTP. Tidak ada aturan bisnis di sini.
//
// Router memakai net/http.ServeMux bawaan Go 1.22+ (pola "METHOD /path/{id}"),
// jadi tidak perlu dependensi router tambahan.
package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"spkzakat/internal/domain"
	"spkzakat/internal/payment"
	"spkzakat/internal/service"
)

// NamaCookie = nama cookie sesi (isinya token acak; hash-nya di tabel sessions).
const NamaCookie = "sesi"

// API menyimpan dependensi handler.
type API struct {
	svc          *service.Service
	log          *slog.Logger
	cookieSecure bool
	ping         func(context.Context) error

	// Pembatas laju endpoint publik (donasi), per IP klien. Disimpan di memori:
	// hilang saat server restart, cukup untuk satu proses (lihat batas.go).
	batasBuat     *batasLaju
	batasBaca     *batasLaju
	simulasiBayar *payment.Tiruan // non-nil hanya bila penyedia tiruan menyala (dev/demo)
}

// Opsi mengubah API saat dibangun, supaya New tetap kompatibel dengan pemanggil lama.
type Opsi func(*API)

// DenganSimulasiBayar membuka endpoint simulasi pembayaran untuk penyedia
// tiruan. JANGAN dipasang di production: siapa pun yang tahu kode donasi
// bisa menandainya lunas.
func DenganSimulasiBayar(t *payment.Tiruan) Opsi { return func(a *API) { a.simulasiBayar = t } }

// New membangun http.Handler lengkap. cookieSecure=false hanya untuk
// development lewat http:// (curl); production wajib true. ping dipakai /healthz.
func New(svc *service.Service, log *slog.Logger, cookieSecure bool, ping func(context.Context) error, opsi ...Opsi) http.Handler {
	a := &API{
		svc: svc, log: log, cookieSecure: cookieSecure, ping: ping,
		// 5 donasi baru per 10 menit dan 120 pembacaan status per menit per IP.
		batasBuat: batasLajuBaru(5, 10*time.Minute),
		batasBaca: batasLajuBaru(120, time.Minute),
	}
	for _, o := range opsi {
		o(a)
	}
	admin := []domain.Role{domain.RoleAdmin}
	semua := []domain.Role{domain.RoleAdmin, domain.RoleKetua}
	ketua := []domain.Role{domain.RoleKetua}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.Handle("POST /api/auth/logout", a.peran(a.logout, semua...))
	mux.Handle("GET /api/auth/me", a.peran(a.me, semua...))
	mux.Handle("PUT /api/auth/password", a.peran(a.gantiPassword, semua...))
	mux.Handle("GET /api/dashboard", a.peran(a.dashboard, semua...))

	mux.Handle("GET /api/mustahik", a.peran(a.daftarMustahik, admin...))
	mux.Handle("POST /api/mustahik", a.peran(a.buatMustahik, admin...))
	mux.Handle("GET /api/mustahik/{id}", a.peran(a.ambilMustahik, admin...))
	mux.Handle("PUT /api/mustahik/{id}", a.peran(a.ubahMustahik, admin...))

	mux.Handle("GET /api/kriteria", a.peran(a.daftarKriteria, semua...))
	mux.Handle("PUT /api/kriteria", a.peran(a.ubahKriteria, admin...))

	mux.Handle("GET /api/periode", a.peran(a.daftarPeriode, semua...))
	mux.Handle("POST /api/periode", a.peran(a.buatPeriode, admin...))
	mux.Handle("GET /api/periode/{id}", a.peran(a.ambilPeriode, semua...))
	mux.Handle("GET /api/periode/{id}/kandidat", a.peran(a.daftarKandidat, admin...))
	mux.Handle("PUT /api/periode/{id}/kandidat", a.peran(a.aturKandidat, admin...))
	mux.Handle("GET /api/periode/{id}/penilaian", a.peran(a.ambilPenilaian, admin...))
	mux.Handle("PUT /api/periode/{id}/penilaian", a.peran(a.simpanPenilaian, admin...))
	mux.Handle("POST /api/periode/{id}/hitung", a.peran(a.hitung, admin...))
	mux.Handle("GET /api/periode/{id}/hasil", a.peran(a.hasil, semua...))
	mux.Handle("POST /api/periode/{id}/keputusan", a.peran(a.keputusan, ketua...))
	mux.Handle("POST /api/periode/{id}/buka-kembali", a.peran(a.bukaKembali, admin...))
	mux.Handle("GET /api/periode/{id}/penyaluran", a.peran(a.daftarPenyaluran, semua...))
	mux.Handle("POST /api/periode/{id}/penyaluran", a.peran(a.catatPenyaluran, admin...))
	mux.Handle("POST /api/periode/{id}/tutup", a.peran(a.tutup, admin...))
	mux.Handle("GET /api/periode/{id}/laporan", a.peran(a.laporan, semua...))

	mux.Handle("GET /api/audit", a.peran(a.audit, semua...))

	// Donasi — pengurus (butuh sesi). Rute literal ("ringkas", "dibaca") menang
	// atas {id} karena lebih spesifik.
	mux.Handle("GET /api/donasi", a.peran(a.daftarDonasi, semua...))
	mux.Handle("POST /api/donasi", a.peran(a.catatDonasi, admin...))
	mux.Handle("GET /api/donasi/ringkas", a.peran(a.ringkasDonasi, semua...))
	mux.Handle("POST /api/donasi/dibaca", a.peran(a.tandaiDonasiDilihat, semua...))
	mux.Handle("GET /api/donasi/{id}", a.peran(a.ambilDonasi, semua...))
	mux.Handle("POST /api/donasi/{id}/batal", a.peran(a.batalkanDonasi, admin...))

	// Program donasi (penggalangan bertarget) — pengurus. Ketua boleh melihat;
	// hanya Admin yang membuat, mengubah, mengunggah gambar, dan menulis kabar.
	mux.Handle("GET /api/program", a.peran(a.daftarProgram, semua...))
	mux.Handle("POST /api/program", a.peran(a.buatProgram, admin...))
	mux.Handle("GET /api/program/{id}", a.peran(a.ambilProgram, semua...))
	mux.Handle("PUT /api/program/{id}", a.peran(a.ubahProgram, admin...))
	mux.Handle("GET /api/program/{id}/gambar", a.peran(a.gambarProgram, semua...))
	mux.Handle("PUT /api/program/{id}/gambar", a.peran(a.unggahGambarProgram, admin...))
	mux.Handle("POST /api/program/{id}/kabar", a.peran(a.tambahKabar, admin...))
	mux.Handle("DELETE /api/program/{id}/kabar/{kabar}", a.peran(a.hapusKabar, admin...))

	// Donasi — PUBLIK (tanpa sesi; ini satu-satunya kelompok rute tanpa peran()
	// selain login/healthz). Dijaga pembatas laju per IP, validasi ketat di
	// service, dan kode rahasia 128 bit untuk akses ke satu donasi. Notifikasi
	// gateway dijaga tanda tangan SHA512, bukan sesi.
	mux.HandleFunc("GET /api/publik/donasi-aktif", a.donasiAktif)
	// Angka transparansi landing: agregat saja, tanpa nama/HP siapa pun.
	mux.Handle("GET /api/publik/ringkasan", a.batasi(a.batasBaca, a.ringkasanPublik))
	mux.Handle("POST /api/publik/donasi", a.batasi(a.batasBuat, a.buatDonasiWeb))
	mux.Handle("GET /api/publik/donasi/{kode}", a.batasi(a.batasBaca, a.donasiPublik))
	// Program donasi publik: draf tidak pernah tampil; nama donatur anonim disamarkan di service.
	mux.Handle("GET /api/publik/program", a.batasi(a.batasBaca, a.daftarProgramPublik))
	mux.Handle("GET /api/publik/program/{slug}", a.batasi(a.batasBaca, a.programPublik))
	mux.Handle("GET /api/publik/program/{slug}/gambar", a.batasi(a.batasBaca, a.gambarProgramPublik))
	mux.HandleFunc("POST /api/publik/midtrans/notifikasi", a.notifikasiGerbang)
	if a.simulasiBayar != nil {
		mux.Handle("POST /api/publik/tiruan/{kode}/bayar", a.batasi(a.batasBuat, a.simulasikanBayar))
	}

	// Urutan luar → dalam: request_id paling luar supaya semua (termasuk
	// panic) punya ID; log akses mencatat status akhir; recover di dalamnya
	// mengubah panic jadi 500 yang masih tercatat log akses.
	return a.requestID(a.logAkses(a.recover(mux)))
}

// ---------- konteks permintaan ----------

type kunciKonteks int

const (
	kunciRequestID kunciKonteks = iota
	kunciUser
	kunciToken
	kunciLogUser // *string: middleware peran mengisi ID user untuk log akses
)

func requestIDDari(ctx context.Context) string { s, _ := ctx.Value(kunciRequestID).(string); return s }
func userDari(ctx context.Context) domain.User { u, _ := ctx.Value(kunciUser).(domain.User); return u }
func tokenDari(ctx context.Context) string     { s, _ := ctx.Value(kunciToken).(string); return s }

// logger = logger dengan request_id permintaan ini.
func (a *API) logger(r *http.Request) *slog.Logger {
	return a.log.With("request_id", requestIDDari(r.Context()))
}

// ipKlien mengambil IP pengguna untuk penguncian login & audit. API ini hanya
// diakses SvelteKit (listen di 127.0.0.1, tidak publik), yang meneruskan IP
// browser lewat X-Real-IP; tanpa header itu dipakai alamat koneksi.
func ipKlien(r *http.Request) string {
	if ip := net.ParseIP(r.Header.Get("X-Real-IP")); ip != nil {
		return ip.String()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---------- middleware ----------

var reRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// requestID menerima X-Request-Id dari klien bila formatnya aman, kalau tidak
// membuat yang baru; selalu dikembalikan di header respons (aturan logging #4).
func (a *API) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !reRequestID.MatchString(id) {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-Id", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), kunciRequestID, id)))
	})
}

// perekam menyimpan status respons untuk log akses.
type perekam struct {
	http.ResponseWriter
	status int
}

func (p *perekam) WriteHeader(code int) { p.status = code; p.ResponseWriter.WriteHeader(code) }

// logAkses mencatat satu baris JSON per permintaan. Query string sengaja TIDAK
// dicatat (bisa memuat kata kunci nama mustahik = data pribadi).
func (a *API) logAkses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mulai := time.Now()
		p := &perekam{ResponseWriter: w, status: http.StatusOK}
		var userID string
		next.ServeHTTP(p, r.WithContext(context.WithValue(r.Context(), kunciLogUser, &userID)))
		level := slog.LevelInfo
		if p.status >= 500 {
			level = slog.LevelError
		}
		a.log.Log(r.Context(), level, "http",
			"request_id", requestIDDari(r.Context()), "method", r.Method, "path", pathLog(r.URL.Path),
			"status", p.status, "duration_ms", time.Since(mulai).Milliseconds(), "user_id", userID)
	})
}

// pathLog menyembunyikan segmen rahasia sebelum path masuk log. Kode donasi
// adalah kunci tautan kuitansi muzaki (siapa pun yang memegangnya bisa
// membukanya), jadi tidak boleh tercatat di log akses maupun log galat.
func pathLog(path string) string {
	for _, awalan := range []string{"/api/publik/donasi/", "/api/publik/tiruan/"} {
		if strings.HasPrefix(path, awalan) {
			return awalan + "{kode}"
		}
	}
	return path
}

// recover mengubah panic menjadi 500 tanpa membocorkan stack trace ke klien.
func (a *API) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				a.logger(r).Error("panic", "error", fmt.Sprint(v), "stack", string(debug.Stack()))
				a.tulisError(w, r, http.StatusInternalServerError, "Terjadi kesalahan internal", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// peran membungkus handler dengan cek sesi (401) dan peran (403), deny-by-default:
// handler tanpa pembungkus ini hanyalah yang memang publik (login, healthz).
func (a *API) peran(h http.HandlerFunc, boleh ...domain.Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(NamaCookie)
		if err != nil || c.Value == "" {
			a.tulisError(w, r, http.StatusUnauthorized, "Sesi tidak ada atau sudah berakhir. Silakan masuk kembali", nil)
			return
		}
		u, err := a.svc.Sesi(r.Context(), c.Value)
		if errors.Is(err, domain.ErrNotFound) {
			a.tulisError(w, r, http.StatusUnauthorized, "Sesi tidak ada atau sudah berakhir. Silakan masuk kembali", nil)
			return
		} else if err != nil {
			a.gagal(w, r, err)
			return
		}
		if p, ok := r.Context().Value(kunciLogUser).(*string); ok {
			*p = u.ID
		}
		izin := false
		for _, role := range boleh {
			if u.Role == role {
				izin = true
			}
		}
		if !izin {
			a.tulisError(w, r, http.StatusForbidden, "Peran Anda tidak memiliki akses ke fitur ini", nil)
			return
		}
		ctx := context.WithValue(context.WithValue(r.Context(), kunciUser, u), kunciToken, c.Value)
		h(w, r.WithContext(ctx))
	})
}

// ---------- JSON & error ----------

const maksBody = 1 << 20 // 1 MiB; permintaan terbesar (penilaian 500 kandidat) jauh di bawah ini

// bacaJSON mendekode body ke v. Field tak dikenal dan data sisa ditolak (400)
// supaya salah ketik nama field tidak diam-diam diabaikan.
func (a *API) bacaJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maksBody))
	dec.DisallowUnknownFields()
	err := dec.Decode(v)
	if err == nil && dec.Decode(&struct{}{}) != io.EOF {
		err = errors.New("data sisa setelah objek JSON")
	}
	if err != nil {
		a.tulisError(w, r, http.StatusBadRequest, "Format permintaan tidak valid", nil)
		return false
	}
	return true
}

func (a *API) tulisJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		a.log.Error("tulis JSON gagal", "error", err.Error())
	}
}

// tulisError membentuk format error seragam {error, request_id, ...tambahan}.
func (a *API) tulisError(w http.ResponseWriter, r *http.Request, status int, pesan string, tambahan map[string]any) {
	body := map[string]any{"error": pesan, "request_id": requestIDDari(r.Context())}
	for k, v := range tambahan {
		body[k] = v
	}
	a.tulisJSON(w, status, body)
}

// gagal menerjemahkan error service ke status HTTP. Hanya pesan yang memang
// untuk pengguna (validasi, ConflictError, penguncian) yang dikirim apa
// adanya; sisanya pesan umum, detailnya hanya di log.
func (a *API) gagal(w http.ResponseWriter, r *http.Request, err error) {
	var ve *domain.ValidationError
	var ce *domain.ConflictError
	var tk *service.TerkunciError
	switch {
	case errors.As(err, &ve):
		a.tulisError(w, r, http.StatusUnprocessableEntity, "Periksa kembali isian", map[string]any{"fields": ve.Fields})
	case errors.As(err, &tk):
		detik := int(tk.RetryAfter.Seconds()) + 1
		w.Header().Set("Retry-After", fmt.Sprint(detik))
		a.tulisError(w, r, http.StatusTooManyRequests, tk.Error(), map[string]any{"retry_after": detik})
	case errors.As(err, &ce):
		a.tulisError(w, r, http.StatusConflict, ce.Pesan, nil)
	case errors.Is(err, domain.ErrConflict):
		a.tulisError(w, r, http.StatusConflict, "Data bertentangan dengan keadaan saat ini. Muat ulang halaman lalu coba lagi", nil)
	case errors.Is(err, domain.ErrNotFound):
		a.tulisError(w, r, http.StatusNotFound, "Data tidak ditemukan", nil)
	case errors.Is(err, domain.ErrNonaktif):
		a.tulisError(w, r, http.StatusServiceUnavailable, "Fitur ini belum diaktifkan oleh pengurus masjid", nil)
	case errors.Is(err, domain.ErrGerbang):
		a.logger(r).Error("payment gateway bermasalah", "path", pathLog(r.URL.Path), "error", err.Error())
		a.tulisError(w, r, http.StatusBadGateway, "Layanan pembayaran sedang bermasalah. Coba lagi beberapa saat", nil)
	default:
		a.logger(r).Error("kesalahan internal", "path", pathLog(r.URL.Path), "error", err.Error())
		a.tulisError(w, r, http.StatusInternalServerError, "Terjadi kesalahan internal", nil)
	}
}

// catat menulis audit log setelah aksi berhasil; kegagalannya hanya dilog
// (aksi sudah tersimpan, jangan dibalas error ke pengguna).
func (a *API) catat(r *http.Request, aksi, resource, id string, detail map[string]any) {
	u := userDari(r.Context())
	err := a.svc.Catat(r.Context(), domain.Audit{
		ActorID: u.ID, Action: aksi, Resource: resource, ResourceID: id, Detail: detail, IP: ipKlien(r),
	})
	if err != nil {
		a.logger(r).Error("audit gagal ditulis", "action", aksi, "error", err.Error())
	}
}
