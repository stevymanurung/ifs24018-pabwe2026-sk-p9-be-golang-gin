package post

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"app/internal/database"
)

var (
	ErrInvalidInput = errors.New("input tidak valid")
	ErrPostNotFound = errors.New("artikel tidak ditemukan")
	ErrForbidden    = errors.New("anda tidak memiliki izin untuk mengakses artikel ini")
)

var DefaultCategories = []string{
	"Teknologi",
	"Pendidikan",
	"Gaya Hidup",
	"Bisnis",
	"Opini",
	"Umum",
}

type Service interface {
	GetCategories() []string
	GetPublicPosts(category, query string) ([]PublicPostResponse, error)
	GetPublicPostBySlug(slug string) (*PublicPostResponse, error)
	CreatePost(userID uint, req *CreatePostRequest) (*PostResponse, error)
	GetAuthorPosts(userID uint) ([]PostResponse, error)
	GetPostByID(id uint, userID uint, role string) (*PostResponse, error)
	GetAuthorStats(userID uint) (*AuthorStatsResponse, error)
	UpdatePost(id uint, userID uint, role string, req *UpdatePostRequest) (*PostResponse, error)
	TogglePublish(id uint, userID uint, role string) (*PostResponse, error)
	DeletePost(id uint, userID uint, role string) error
	GetAdminPosts() ([]PostResponse, error)
	GetAdminUsers() ([]AdminUserStatsResponse, error)
	GetAdminUserPosts(targetUserID uint) ([]PostResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetCategories() []string {
	return DefaultCategories
}

func CalculateReadingTime(content string) int {
	words := len(strings.Fields(content))
	readingTime := words / 200
	if readingTime < 1 {
		return 1
	}
	return readingTime
}

var nonAlphanumericRegex = regexp.MustCompile(`[^a-z0-9]+`)

func GenerateSlug(title string) string {
	lower := strings.ToLower(title)
	clean := nonAlphanumericRegex.ReplaceAllString(lower, "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		clean = "artikel"
	}
	uniqueSuffix := fmt.Sprintf("%d%03d", time.Now().UnixNano()%1000000, rand.Intn(1000))
	return fmt.Sprintf("%s-%s", clean, uniqueSuffix)
}

func toPublicPostResponse(p *database.Post) PublicPostResponse {
	author := "Anonim"
	if p.User.Username != "" {
		author = p.User.Username
	}
	return PublicPostResponse{
		ID:             p.ID,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		ReadingTime:    CalculateReadingTime(p.Content),
		AuthorName:     author,
		PublishedAt:    p.PublishedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func toPostResponse(p *database.Post) PostResponse {
	author := ""
	if p.User.Username != "" {
		author = p.User.Username
	}
	return PostResponse{
		ID:             p.ID,
		UserID:         p.UserID,
		Title:          p.Title,
		Slug:           p.Slug,
		Category:       p.Category,
		Excerpt:        p.Excerpt,
		Content:        p.Content,
		CoverImagePath: p.CoverImagePath,
		IsPublished:    p.IsPublished,
		ReadingTime:    CalculateReadingTime(p.Content),
		AuthorName:     author,
		PublishedAt:    p.PublishedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func (s *service) GetPublicPosts(category, query string) ([]PublicPostResponse, error) {
	posts, err := s.repo.FindAllPublic(category, query)
	if err != nil {
		return nil, err
	}

	result := make([]PublicPostResponse, 0, len(posts))
	for i := range posts {
		result = append(result, toPublicPostResponse(&posts[i]))
	}
	return result, nil
}

func (s *service) GetPublicPostBySlug(slug string) (*PublicPostResponse, error) {
	post, err := s.repo.FindBySlug(slug)
	if err != nil {
		return nil, ErrPostNotFound
	}
	res := toPublicPostResponse(post)
	return &res, nil
}

func (s *service) CreatePost(userID uint, req *CreatePostRequest) (*PostResponse, error) {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)

	if title == "" || content == "" {
		return nil, ErrInvalidInput
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}

	isPublished := false
	if req.IsPublished != nil && *req.IsPublished {
		isPublished = true
	}

	var publishedAt *time.Time
	if isPublished {
		now := time.Now()
		publishedAt = &now
	}

	post := &database.Post{
		UserID:         userID,
		Title:          title,
		Slug:           GenerateSlug(title),
		Category:       category,
		Excerpt:        req.Excerpt,
		Content:        content,
		CoverImagePath: req.CoverImagePath,
		IsPublished:    isPublished,
		PublishedAt:    publishedAt,
	}

	if err := s.repo.Create(post); err != nil {
		return nil, err
	}

	res := toPostResponse(post)
	return &res, nil
}

func (s *service) GetAuthorPosts(userID uint) ([]PostResponse, error) {
	posts, err := s.repo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	result := make([]PostResponse, 0, len(posts))
	for i := range posts {
		result = append(result, toPostResponse(&posts[i]))
	}
	return result, nil
}

func (s *service) GetPostByID(id uint, userID uint, role string) (*PostResponse, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrPostNotFound
	}

	if post.UserID != userID && !strings.EqualFold(role, "admin") {
		return nil, ErrForbidden
	}

	res := toPostResponse(post)
	return &res, nil
}

func (s *service) GetAuthorStats(userID uint) (*AuthorStatsResponse, error) {
	total, selesai, belumSelesai, err := s.repo.CountStatsByUserID(userID)
	if err != nil {
		return nil, err
	}
	return &AuthorStatsResponse{
		TotalArticles: total,
		TotalSelesai:  selesai,
		BelumSelesai:  belumSelesai,
	}, nil
}

func (s *service) UpdatePost(id uint, userID uint, role string, req *UpdatePostRequest) (*PostResponse, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrPostNotFound
	}

	if post.UserID != userID && !strings.EqualFold(role, "admin") {
		return nil, ErrForbidden
	}

	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		post.Title = strings.TrimSpace(*req.Title)
	}
	if req.Category != nil && strings.TrimSpace(*req.Category) != "" {
		post.Category = strings.TrimSpace(*req.Category)
	}
	if req.Excerpt != nil && strings.TrimSpace(*req.Excerpt) != "" {
		post.Excerpt = strings.TrimSpace(*req.Excerpt)
	}
	if req.Content != nil && strings.TrimSpace(*req.Content) != "" {
		post.Content = strings.TrimSpace(*req.Content)
	}
	if req.CoverImagePath != nil && strings.TrimSpace(*req.CoverImagePath) != "" {
		post.CoverImagePath = strings.TrimSpace(*req.CoverImagePath)
	}
	if req.IsPublished != nil {
		post.IsPublished = *req.IsPublished
		if post.IsPublished && post.PublishedAt == nil {
			now := time.Now()
			post.PublishedAt = &now
		} else if !post.IsPublished {
			post.PublishedAt = nil
		}
	}

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	res := toPostResponse(post)
	return &res, nil
}

func (s *service) TogglePublish(id uint, userID uint, role string) (*PostResponse, error) {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrPostNotFound
	}

	if post.UserID != userID && !strings.EqualFold(role, "admin") {
		return nil, ErrForbidden
	}

	post.IsPublished = !post.IsPublished
	if post.IsPublished {
		now := time.Now()
		post.PublishedAt = &now
	} else {
		post.PublishedAt = nil
	}

	if err := s.repo.Update(post); err != nil {
		return nil, err
	}

	res := toPostResponse(post)
	return &res, nil
}

func (s *service) DeletePost(id uint, userID uint, role string) error {
	post, err := s.repo.FindByID(id)
	if err != nil {
		return ErrPostNotFound
	}

	if post.UserID != userID && !strings.EqualFold(role, "admin") {
		return ErrForbidden
	}

	return s.repo.Delete(id)
}

func (s *service) GetAdminPosts() ([]PostResponse, error) {
	posts, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}

	result := make([]PostResponse, 0, len(posts))
	for i := range posts {
		result = append(result, toPostResponse(&posts[i]))
	}
	return result, nil
}

func (s *service) GetAdminUsers() ([]AdminUserStatsResponse, error) {
	return s.repo.FindAdminUsers()
}

func (s *service) GetAdminUserPosts(targetUserID uint) ([]PostResponse, error) {
	posts, err := s.repo.FindByUserID(targetUserID)
	if err != nil {
		return nil, err
	}

	result := make([]PostResponse, 0, len(posts))
	for i := range posts {
		result = append(result, toPostResponse(&posts[i]))
	}
	return result, nil
}
