package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/models"
	"app/internal/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Server struct {
	DB     *gorm.DB
	Config config.Config
	AI     *ai.Service
}

func New(db *gorm.DB, c config.Config) *Server {
	return &Server{db, c, ai.New(c.SearchEndpoint, c.OpenAIEndpoint, c.OpenAIKey, c.OpenAIModel)}
}
func (s *Server) Router() *gin.Engine {
	if s.Config.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	r.GET("/health", func(c *gin.Context) { response.Success(c, 200, "API is healthy", gin.H{"service": "Delcom Blog"}) })
	r.GET("/swagger/*any", func(c *gin.Context) {
		c.Data(200, "text/html; charset=utf-8", []byte(`<!doctype html><html><head><title>Delcom Blog API Docs</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:"/api.json",dom_id:"#swagger-ui"})</script></body></html>`))
	})
	api := r.Group("/api")
	api.GET(".json", func(c *gin.Context) {
		c.JSON(200, gin.H{"openapi": "3.0.3", "info": gin.H{"title": "Delcom Blog API", "version": "1.0.0"}, "paths": gin.H{"/api/public/posts": gin.H{"get": gin.H{"summary": "List published posts", "responses": gin.H{"200": gin.H{"description": "OK"}}}}, "/api/auth/login": gin.H{"post": gin.H{"summary": "Login", "responses": gin.H{"200": gin.H{"description": "OK"}}}}}})
	})
	pub := api.Group("/public")
	pub.GET("/categories", func(c *gin.Context) {
		response.Success(c, 200, "Categories", []string{"Teknologi", "Pendidikan", "Gaya Hidup", "Bisnis", "Opini", "Umum"})
	})
	pub.GET("/posts", s.publicPosts)
	pub.GET("/posts/:slug", s.publicPost)
	auth := api.Group("/auth")
	auth.POST("/register", s.register)
	auth.POST("/login", s.login)
	protected := api.Group("")
	protected.Use(middleware.Auth(s.Config.AppKey))
	protected.POST("/auth/logout", func(c *gin.Context) { response.Success(c, 200, "Logout berhasil", nil) })
	protected.GET("/auth/profile", s.profile)
	protected.PUT("/auth/profile", s.updateProfile)
	protected.GET("/posts", s.myPosts)
	protected.POST("/posts", s.createPost)
	protected.GET("/posts/stats", s.stats)
	protected.POST("/posts/ai-generate", s.generate)
	protected.GET("/posts/:id", s.getPost)
	protected.PUT("/posts/:id", s.updatePost)
	protected.PATCH("/posts/:id/toggle-publish", s.togglePost)
	protected.DELETE("/posts/:id", s.deletePost)
	admin := protected.Group("/admin")
	admin.Use(middleware.RequireRole("admin"))
	admin.GET("/users", s.adminUsers)
	admin.GET("/posts", s.adminPosts)
	admin.GET("/users/:id/posts", s.adminUserPosts)
	r.NoRoute(func(c *gin.Context) { response.Fail(c, 404, "Endpoint not found") })
	return r
}
func (s *Server) register(c *gin.Context) {
	var in struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Username, email valid, dan password minimal 8 karakter wajib diisi")
		return
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if e != nil {
		response.Fail(c, 500, "Gagal membuat akun")
		return
	}
	u := models.User{Username: strings.TrimSpace(in.Username), Email: strings.ToLower(strings.TrimSpace(in.Email)), Password: string(hash), Role: "member"}
	if e = s.DB.Create(&u).Error; e != nil {
		response.Fail(c, 409, "Username atau email sudah digunakan")
		return
	}
	u.Password = ""
	response.Success(c, 201, "Registrasi berhasil", u)
}
func (s *Server) login(c *gin.Context) {
	var in struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Username dan password wajib diisi")
		return
	}
	var u models.User
	if s.DB.Where("username = ?", in.Username).First(&u).Error != nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(in.Password)) != nil {
		response.Fail(c, 401, "Username atau password salah")
		return
	}
	token, e := middleware.Token(s.Config.AppKey, u.ID, u.Username, u.Role)
	if e != nil {
		response.Fail(c, 500, "Gagal membuat token")
		return
	}
	response.Success(c, 200, "Login berhasil", gin.H{"token": token, "user": gin.H{"id": u.ID, "username": u.Username, "email": u.Email, "role": u.Role}})
}
func (s *Server) profile(c *gin.Context) {
	var u models.User
	if s.DB.First(&u, c.GetUint("userID")).Error != nil {
		response.Fail(c, 404, "User tidak ditemukan")
		return
	}
	u.Password = ""
	response.Success(c, 200, "Profil pengguna", u)
}
func (s *Server) updateProfile(c *gin.Context) {
	var in struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Data profil tidak valid")
		return
	}
	updates := map[string]any{}
	if strings.TrimSpace(in.Username) != "" {
		updates["username"] = strings.TrimSpace(in.Username)
	}
	if strings.TrimSpace(in.Email) != "" {
		updates["email"] = strings.ToLower(strings.TrimSpace(in.Email))
	}
	if e := s.DB.Model(&models.User{}).Where("id = ?", c.GetUint("userID")).Updates(updates).Error; e != nil {
		response.Fail(c, 409, "Username atau email sudah digunakan")
		return
	}
	s.profile(c)
}
func (s *Server) publicPosts(c *gin.Context) {
	q := s.DB.Preload("User").Where("is_published = ?", true)
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		like := "%" + v + "%"
		q = q.Where("title ILIKE ? OR excerpt ILIKE ? OR content ILIKE ?", like, like, like)
	}
	if v := c.Query("category"); v != "" && v != "Semua" {
		q = q.Where("category = ?", v)
	}
	var posts []models.Post
	if e := q.Order("published_at DESC").Find(&posts).Error; e != nil {
		response.Fail(c, 500, "Gagal mengambil artikel")
		return
	}
	for i := range posts {
		posts[i].ReadingTime = post.ReadingTime(posts[i].Content)
	}
	response.Success(c, 200, "Daftar artikel publik", posts)
}
func (s *Server) publicPost(c *gin.Context) {
	var p models.Post
	if s.DB.Preload("User").Where("slug = ? AND is_published = ?", c.Param("slug"), true).First(&p).Error != nil {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return
	}
	p.ReadingTime = post.ReadingTime(p.Content)
	response.Success(c, 200, "Detail artikel", p)
}
func (s *Server) myPosts(c *gin.Context) {
	var p []models.Post
	if s.DB.Where("user_id = ?", c.GetUint("userID")).Order("updated_at DESC").Find(&p).Error != nil {
		response.Fail(c, 500, "Gagal mengambil artikel")
		return
	}
	response.Success(c, 200, "Artikel penulis", p)
}
func (s *Server) uniqueSlug(title string, skip uint) string {
	base := post.Slugify(title)
	if base == "" {
		base = "artikel"
	}
	slug := base
	for i := 2; ; i++ {
		var n int64
		q := s.DB.Model(&models.Post{}).Where("slug = ?", slug)
		if skip > 0 {
			q = q.Where("id <> ?", skip)
		}
		q.Count(&n)
		if n == 0 {
			return slug
		}
		slug = base + "-" + strconv.Itoa(i)
	}
}

type postInput struct {
	Title          string `json:"title" binding:"required"`
	Category       string `json:"category" binding:"required"`
	Excerpt        string `json:"excerpt"`
	Content        string `json:"content"`
	CoverImagePath string `json:"cover_image_path"`
}

func (s *Server) createPost(c *gin.Context) {
	var in postInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Judul dan kategori wajib diisi")
		return
	}
	p := models.Post{UserID: c.GetUint("userID"), Title: strings.TrimSpace(in.Title), Slug: s.uniqueSlug(in.Title, 0), Category: in.Category, Excerpt: in.Excerpt, Content: in.Content, CoverImagePath: in.CoverImagePath}
	if e := s.DB.Create(&p).Error; e != nil {
		response.Fail(c, 500, "Gagal menyimpan artikel")
		return
	}
	response.Success(c, 201, "Artikel berhasil dibuat", p)
}
func (s *Server) ownedPost(c *gin.Context) (models.Post, bool) {
	var p models.Post
	id, e := strconv.Atoi(c.Param("id"))
	if e != nil || id < 1 {
		return p, false
	}
	if s.DB.Where("id = ? AND user_id = ?", id, c.GetUint("userID")).First(&p).Error != nil {
		return p, false
	}
	return p, true
}
func (s *Server) getPost(c *gin.Context) {
	p, ok := s.ownedPost(c)
	if !ok {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return
	}
	response.Success(c, 200, "Detail artikel", p)
}
func (s *Server) updatePost(c *gin.Context) {
	p, ok := s.ownedPost(c)
	if !ok {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return
	}
	var in postInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Data artikel tidak valid")
		return
	}
	p.Title = strings.TrimSpace(in.Title)
	p.Slug = s.uniqueSlug(in.Title, p.ID)
	p.Category = in.Category
	p.Excerpt = in.Excerpt
	p.Content = in.Content
	p.CoverImagePath = in.CoverImagePath
	if e := s.DB.Save(&p).Error; e != nil {
		response.Fail(c, 500, "Gagal memperbarui artikel")
		return
	}
	response.Success(c, 200, "Artikel diperbarui", p)
}
func (s *Server) togglePost(c *gin.Context) {
	p, ok := s.ownedPost(c)
	if !ok {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return
	}
	p.IsPublished = !p.IsPublished
	if p.IsPublished {
		now := time.Now()
		p.PublishedAt = &now
	} else {
		p.PublishedAt = nil
	}
	if e := s.DB.Save(&p).Error; e != nil {
		response.Fail(c, 500, "Gagal mengubah status artikel")
		return
	}
	response.Success(c, 200, "Status publikasi diperbarui", p)
}
func (s *Server) deletePost(c *gin.Context) {
	p, ok := s.ownedPost(c)
	if !ok {
		response.Fail(c, 404, "Artikel tidak ditemukan")
		return
	}
	s.DB.Delete(&p)
	response.Success(c, 200, "Artikel dihapus", nil)
}
func (s *Server) stats(c *gin.Context) {
	var total, published int64
	s.DB.Model(&models.Post{}).Where("user_id = ?", c.GetUint("userID")).Count(&total)
	s.DB.Model(&models.Post{}).Where("user_id = ? AND is_published = ?", c.GetUint("userID"), true).Count(&published)
	response.Success(c, 200, "Statistik artikel", gin.H{"total": total, "published": published, "draft": total - published})
}
func (s *Server) generate(c *gin.Context) {
	var in struct {
		Topic            string `json:"topic" binding:"required"`
		Category         string `json:"category"`
		IncludeWebSearch bool   `json:"include_web_search"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "Topik wajib diisi")
		return
	}
	d, e := s.AI.Generate(in.Topic, in.Category, in.IncludeWebSearch)
	if e != nil {
		response.Fail(c, 502, e.Error())
		return
	}
	response.Success(c, 200, "Draf artikel berhasil dibuat", d)
}
func (s *Server) adminUsers(c *gin.Context) {
	var users []models.User
	s.DB.Find(&users)
	out := make([]gin.H, 0, len(users))
	for _, u := range users {
		var total, pub int64
		s.DB.Model(&models.Post{}).Where("user_id = ?", u.ID).Count(&total)
		s.DB.Model(&models.Post{}).Where("user_id = ? AND is_published = ?", u.ID, true).Count(&pub)
		out = append(out, gin.H{"id": u.ID, "username": u.Username, "email": u.Email, "role": u.Role, "total_posts": total, "total_published": pub})
	}
	response.Success(c, 200, "Daftar pengguna", out)
}
func (s *Server) adminPosts(c *gin.Context) {
	var p []models.Post
	s.DB.Preload("User").Order("created_at DESC").Find(&p)
	response.Success(c, 200, "Arsip seluruh artikel", p)
}
func (s *Server) adminUserPosts(c *gin.Context) {
	var p []models.Post
	s.DB.Preload("User").Where("user_id = ?", c.Param("id")).Find(&p)
	response.Success(c, 200, "Artikel pengguna", p)
}

var _ = http.StatusOK
