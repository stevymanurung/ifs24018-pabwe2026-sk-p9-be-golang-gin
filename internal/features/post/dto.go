package post

import "time"

type CreatePostRequest struct {
	Title          string `json:"title"`
	Category       string `json:"category"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
	IsPublished    *bool  `json:"is_published"`
}

type UpdatePostRequest struct {
	Title          *string `json:"title"`
	Category       *string `json:"category"`
	Excerpt        *string `json:"excerpt"`
	Content        *string `json:"content"`
	CoverImagePath *string `json:"cover_image_path"`
	IsPublished    *bool   `json:"is_published"`
}

type PublicPostResponse struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	IsPublished    bool       `json:"is_published"`
	ReadingTime    int        `json:"reading_time"`
	AuthorName     string     `json:"author_name"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type PostResponse struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Category       string     `json:"category"`
	Excerpt        string     `json:"excerpt"`
	Content        string     `json:"content"`
	CoverImagePath string     `json:"cover_image_path"`
	IsPublished    bool       `json:"is_published"`
	ReadingTime    int        `json:"reading_time"`
	AuthorName     string     `json:"author_name,omitempty"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type AuthorStatsResponse struct {
	TotalArticles int64 `json:"total_articles"`
	TotalSelesai  int64 `json:"total_selesai"`
	BelumSelesai  int64 `json:"belum_selesai"`
}

type AdminUserStatsResponse struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	TotalArticles  int64     `json:"total_articles"`
	TotalPublished int64     `json:"total_published"`
	CreatedAt      time.Time `json:"created_at"`
}
