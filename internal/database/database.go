package database

import (
	"app/internal/config"
	"app/internal/models"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// User dan Post menjadi alias model bersama agar repository fitur memakai tipe yang sama.
type User = models.User
type Post = models.Post

func Open(c config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err = db.AutoMigrate(&models.User{}, &models.Post{}); err != nil {
		return nil, err
	}
	return db, nil
}
