package httpapi

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// batasLaju membatasi jumlah permintaan per kunci (IP klien) dalam jendela waktu
// tetap-bergeser: tiap kunci menyimpan waktu permintaan terakhirnya dan yang
// lebih tua dari jendela dibuang.
//
// ponytail: disimpan di memori satu proses (hilang saat restart, tidak dibagi
// antar-instance). Cukup untuk satu server masjid; pindah ke Redis bila API
// dijalankan lebih dari satu instance.
type batasLaju struct {
	mu       sync.Mutex
	maks     int
	jendela  time.Duration
	kejadian map[string][]time.Time
	sekarang func() time.Time
	// penyapuan terakhir: peta dibersihkan dari kunci basi paling sering sekali
	// per jendela, supaya memori tidak tumbuh tanpa batas oleh banyak IP berbeda.
	sapuTerakhir time.Time
}

func batasLajuBaru(maks int, jendela time.Duration) *batasLaju {
	return &batasLaju{maks: maks, jendela: jendela, kejadian: map[string][]time.Time{}, sekarang: time.Now}
}

// izinkan mencatat satu permintaan untuk kunci. Bila sudah mencapai batas,
// mengembalikan false beserta berapa lama lagi harus menunggu.
func (b *batasLaju) izinkan(kunci string) (bool, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	kini := b.sekarang()
	batas := kini.Add(-b.jendela)

	if kini.Sub(b.sapuTerakhir) > b.jendela {
		for k, ts := range b.kejadian {
			if len(ts) == 0 || !ts[len(ts)-1].After(batas) {
				delete(b.kejadian, k)
			}
		}
		b.sapuTerakhir = kini
	}

	ts := b.kejadian[kunci]
	i := 0
	for i < len(ts) && !ts[i].After(batas) {
		i++
	}
	ts = ts[i:]
	if len(ts) >= b.maks {
		b.kejadian[kunci] = ts
		return false, ts[0].Add(b.jendela).Sub(kini)
	}
	b.kejadian[kunci] = append(ts, kini)
	return true, 0
}

// batasi membungkus handler publik dengan pembatas laju per IP klien. Lewat
// batas → 429 dengan Retry-After (SvelteKit menampilkannya di formulir).
func (a *API) batasi(b *batasLaju, h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, tunggu := b.izinkan(ipKlien(r)); !ok {
			detik := int(tunggu.Seconds()) + 1
			w.Header().Set("Retry-After", strconv.Itoa(detik))
			a.tulisError(w, r, http.StatusTooManyRequests, "Terlalu banyak permintaan. Coba lagi beberapa saat", map[string]any{"retry_after": detik})
			return
		}
		h(w, r)
	})
}
