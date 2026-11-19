package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// RestoreOriginalPath membungkus handler agar path asli request dipulihkan.
//
// Di Vercel, rewrite meneruskan path TUJUAN ke fungsi, bukan path yang diminta klien.
// vercel.json menyertakan path asli lewat query "__p"; middleware ini
// mengembalikannya sebelum request diteruskan ke router Gin.
// Tanpa "__p" (lokal / server biasa) request tidak diubah.
func RestoreOriginalPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("debug") == "1" {
			// Mode diagnosis: tampilkan apa yang diterima server (aman dihapus setelah deploy terverifikasi).
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"entry":  "server (RestoreOriginalPath)",
				"method": r.Method,
				"url":    r.URL.String(),
				"path":   r.URL.Path,
				"host":   r.Host,
			})
			return
		}
		if p, ok := q["__p"]; ok {
			q.Del("__p")
			r.URL.RawQuery = q.Encode()
			// Placeholder yang tidak terganti (":p", "$1") diabaikan.
			if !strings.HasPrefix(p[0], ":") && !strings.HasPrefix(p[0], "$") {
				r.URL.Path = "/" + strings.TrimPrefix(p[0], "/")
				r.URL.RawPath = ""
				r.RequestURI = r.URL.RequestURI()
			}
		} else if r.URL.Path == "/api/index" || r.URL.Path == "/api/index.go" {
			r.URL.Path = "/"
		}
		next.ServeHTTP(w, r)
	})
}
