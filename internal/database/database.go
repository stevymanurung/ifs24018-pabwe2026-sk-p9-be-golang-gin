package database

import (
	"fmt"
	"time"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:text;not null" json:"-"`
	Role      string    `gorm:"type:varchar(50);not null;default:'member'" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Posts     []Post    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"posts,omitempty"`
}

type Post struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Slug           string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"type:varchar(100);not null;default:'Umum'" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"not null;default:false" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func ConnectDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	if cfg.DBConnection == "sqlite" {
		dialector = sqlite.Open(cfg.DBDatabase)
	} else {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta connect_timeout=10",
			cfg.DBHost, cfg.DBUsername, cfg.DBPassword, cfg.DBDatabase, cfg.DBPort, cfg.DBSSLMode)
		// PreferSimpleProtocol: kompatibel dengan connection pooler (Supabase/Neon/PgBouncer)
		// pada lingkungan serverless sehingga tidak muncul error "prepared statement already exists".
		dialector = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ConnectTestDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, _ := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	_ = Migrate(db)
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Post{})
}

// schemaReady true bila tabel dan akun bawaan (admin & member) sudah ada.
func schemaReady(db *gorm.DB, cfg *config.Config) bool {
	m := db.Migrator()
	if !m.HasTable(&User{}) || !m.HasTable(&Post{}) {
		return false
	}
	adminUsername := cfg.AdminUsername
	if adminUsername == "" {
		adminUsername = "admin"
	}
	var count int64
	db.Model(&User{}).Where("username IN ?", []string{adminUsername, "member"}).Count(&count)
	return count >= 2
}

// EnsureSchema menjalankan Migrate + Seed hanya bila belum pernah dilakukan.
// Berguna di lingkungan serverless agar cold start tidak lambat dan tidak saling bentrok.
func EnsureSchema(db *gorm.DB, cfg *config.Config) error {
	if schemaReady(db, cfg) {
		return nil
	}
	if err := Migrate(db); err != nil {
		return err
	}
	return Seed(db, cfg)
}

func Seed(db *gorm.DB, cfg *config.Config) error {
	adminPassword := cfg.AdminPassword
	if adminPassword == "" {
		adminPassword = "admin"
	}
	hashedAdminPass, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	hashedMemberPass, _ := bcrypt.GenerateFromPassword([]byte("member"), bcrypt.DefaultCost)

	adminUsername := cfg.AdminUsername
	if adminUsername == "" {
		adminUsername = "admin"
	}
	adminEmail := fmt.Sprintf("%s@delcom.org", adminUsername)

	var adminUser User
	if err := db.Where("username = ? OR email = ?", adminUsername, adminEmail).First(&adminUser).Error; err != nil {
		adminUser = User{
			Username: adminUsername,
			Email:    adminEmail,
			Password: string(hashedAdminPass),
			Role:     "admin",
		}
		_ = db.Create(&adminUser)
	}

	var memberUser User
	if err := db.Where("username = ?", "member").First(&memberUser).Error; err != nil {
		memberUser = User{
			Username: "member",
			Email:    "member@delcom.org",
			Password: string(hashedMemberPass),
			Role:     "member",
		}
		_ = db.Create(&memberUser)
	}

	var postCount int64
	db.Model(&Post{}).Count(&postCount)
	if postCount == 0 {
		now := time.Now()
		posts := []Post{
			{
				UserID:         memberUser.ID,
				Title:          "Pengenalan Bahasa Pemrograman Golang Modern",
				Slug:           "pengenalan-bahasa-pemrograman-golang-modern-1",
				Category:       "Teknologi",
				Excerpt:        "Panduan komprehensif memahami Golang dan Gin REST API untuk arsitektur modern.",
				Content:        "# Belajar Golang\n\nGolang adalah bahasa pemrograman open source yang dikembangkan di Google untuk performa tinggi, efisiensi memori, dan kemudahan konkurensi dengan goroutine. Dalam artikel ini kita membahas fitur-fitur utama Golang dan Gin web framework.",
				IsPublished:    true,
				PublishedAt:    &now,
				CoverImagePath: "",
			},
			{
				UserID:         adminUser.ID,
				Title:          "Penerapan AI dalam Kurikulum Pendidikan 2026",
				Slug:           "penerapan-ai-dalam-kurikulum-pendidikan-2026-2",
				Category:       "Pendidikan",
				Excerpt:        "Transformasi digital di dunia pendidikan tinggi berbasis teknologi AI terkini.",
				Content:        "# Pendidikan Modern Berbasis AI\n\nPendidikan di era modern mengintegrasikan kecerdasan buatan untuk membantu proses belajar mengajar secara interaktif dan adaptif.",
				IsPublished:    true,
				PublishedAt:    &now,
				CoverImagePath: "",
			},
		}
		for _, p := range posts {
			_ = db.Create(&p)
		}
	}

	return nil
}
