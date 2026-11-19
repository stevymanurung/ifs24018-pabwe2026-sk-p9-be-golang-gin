package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

var (
	mu     sync.Mutex
	router http.Handler
)

// initRouter menyiapkan koneksi DB, migrate/seed (bila perlu), dan router sekali per instance.
// Jika gagal, request berikutnya akan mencoba lagi.
func initRouter() (http.Handler, error) {
	mu.Lock()
	defer mu.Unlock()

	if router != nil {
		return router, nil
	}

	cfg := config.LoadConfig()

	db, err := database.ConnectDB(cfg)
	if err != nil {
		return nil, err
	}

	// Serverless: banyak instance berjalan paralel, jaga agar jumlah koneksi per instance kecil.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(3)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	}

	if err := database.EnsureSchema(db, cfg); err != nil {
		return nil, err
	}

	router = server.RestoreOriginalPath(server.SetupRouter(cfg, db))
	return router, nil
}

// Handler adalah entry point yang dipanggil Vercel.
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("debug") == "1" {
		// Mode diagnosis: tampilkan persis apa yang diterima fungsi dari Vercel.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"entry":       "api/index.go",
			"method":      r.Method,
			"url":         r.URL.String(),
			"path":        r.URL.Path,
			"request_uri": r.RequestURI,
			"host":        r.Host,
			"headers":     r.Header,
		})
		return
	}

	rt, err := initRouter()
	if err != nil {
		log.Printf("init gagal: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","message":"Server gagal inisialisasi"}`))
		return
	}
	rt.ServeHTTP(w, r)
}
