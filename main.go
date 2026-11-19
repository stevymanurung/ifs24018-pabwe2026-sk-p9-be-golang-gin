package main

import (
	"app/internal/config"
	"app/internal/database"
	"app/internal/models"
	"app/internal/server"
	"golang.org/x/crypto/bcrypt"
	"log"
)

func main() {
	cfg := config.Load()
	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	var count int64
	db.Model(&models.User{}).Where("username = ?", cfg.AdminUsername).Count(&count)
	if count == 0 {
		hash, e := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
		if e != nil {
			log.Fatal(e)
		}
		db.Create(&models.User{Username: cfg.AdminUsername, Email: "admin@delcom.local", Password: string(hash), Role: "admin"})
	}
	app := server.New(db, cfg)
	log.Printf("Delcom Blog API listening on :%s", cfg.Port)
	if err = app.Router().Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
