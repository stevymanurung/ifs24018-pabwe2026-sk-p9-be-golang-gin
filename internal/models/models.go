package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"size:80;uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"size:180;uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"size:20;default:member" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Posts     []Post    `json:"-"`
}
type Post struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	User           User       `gorm:"foreignKey:UserID" json:"author,omitempty"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Slug           string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"size:100;not null" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"default:false" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	ReadingTime    int        `gorm:"-" json:"reading_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
