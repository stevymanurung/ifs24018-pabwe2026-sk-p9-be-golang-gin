package main

import (
	"fmt"
	"log"
	"net/http"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}

	if err := database.EnsureSchema(db, cfg); err != nil {
		log.Fatalf("Gagal menyiapkan database: %v", err)
	}

	r := server.SetupRouter(cfg, db)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Server berjalan di http://localhost%s", addr)
	if err := http.ListenAndServe(addr, server.RestoreOriginalPath(r)); err != nil {
		log.Fatalf("Server berhenti karena error: %v", err)
	}
}
