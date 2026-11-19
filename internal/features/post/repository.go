package post

import (
	"strings"

	"app/internal/database"

	"gorm.io/gorm"
)

type Repository interface {
	Create(post *database.Post) error
	FindByID(id uint) (*database.Post, error)
	FindBySlug(slug string) (*database.Post, error)
	FindAllPublic(category, query string) ([]database.Post, error)
	FindByUserID(userID uint) ([]database.Post, error)
	FindAll() ([]database.Post, error)
	Update(post *database.Post) error
	Delete(id uint) error
	CountStatsByUserID(userID uint) (total int64, selesai int64, belumSelesai int64, err error)
	FindAdminUsers() ([]AdminUserStatsResponse, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(post *database.Post) error {
	return r.db.Create(post).Error
}

func (r *repository) FindByID(id uint) (*database.Post, error) {
	var post database.Post
	if err := r.db.Preload("User").First(&post, id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *repository) FindBySlug(slug string) (*database.Post, error) {
	var post database.Post
	if err := r.db.Preload("User").Where("slug = ? AND is_published = ?", slug, true).First(&post).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *repository) FindAllPublic(category, query string) ([]database.Post, error) {
	var posts []database.Post
	tx := r.db.Preload("User").Where("is_published = ?", true)

	category = strings.TrimSpace(category)
	if category != "" && !strings.EqualFold(category, "Semua") {
		tx = tx.Where("LOWER(category) = LOWER(?)", category)
	}

	query = strings.TrimSpace(query)
	if query != "" {
		likeQuery := "%" + strings.ToLower(query) + "%"
		tx = tx.Where("LOWER(title) LIKE ? OR LOWER(excerpt) LIKE ? OR LOWER(content) LIKE ?", likeQuery, likeQuery, likeQuery)
	}

	if err := tx.Order("published_at DESC, created_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *repository) FindByUserID(userID uint) ([]database.Post, error) {
	var posts []database.Post
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *repository) FindAll() ([]database.Post, error) {
	var posts []database.Post
	if err := r.db.Order("created_at DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *repository) Update(post *database.Post) error {
	return r.db.Save(post).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&database.Post{}, id).Error
}

func (r *repository) CountStatsByUserID(userID uint) (int64, int64, int64, error) {
	var total, selesai int64

	if err := r.db.Model(&database.Post{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return 0, 0, 0, err
	}
	_ = r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", userID, true).Count(&selesai).Error
	return total, selesai, total - selesai, nil
}

func (r *repository) FindAdminUsers() ([]AdminUserStatsResponse, error) {
	var users []database.User
	if err := r.db.Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}

	var results []AdminUserStatsResponse
	for _, u := range users {
		var total, published int64
		_ = r.db.Model(&database.Post{}).Where("user_id = ?", u.ID).Count(&total).Error
		_ = r.db.Model(&database.Post{}).Where("user_id = ? AND is_published = ?", u.ID, true).Count(&published).Error

		results = append(results, AdminUserStatsResponse{
			ID:             u.ID,
			Username:       u.Username,
			Email:          u.Email,
			Role:           u.Role,
			TotalArticles:  total,
			TotalPublished: published,
			CreatedAt:      u.CreatedAt,
		})
	}
	return results, nil
}
