package post

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"app/internal/config"
	"app/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestPost(t *testing.T) (*Handler, Service, Repository, *database.User, *database.User) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)

	cfg := &config.Config{
		AdminUsername: "admin",
		AdminPassword: "admin",
	}
	err = database.Seed(db, cfg)
	assert.NoError(t, err)

	var adminUser, memberUser database.User
	db.Where("role = ?", "admin").First(&adminUser)
	db.Where("role = ?", "member").First(&memberUser)

	repo := NewRepository(db)
	svc := NewService(repo)
	handler := NewHandler(svc)
	return handler, svc, repo, &adminUser, &memberUser
}

func TestCalculateReadingTime(t *testing.T) {
	assert.Equal(t, 1, CalculateReadingTime(""))
	assert.Equal(t, 1, CalculateReadingTime("Sepuluh kata saja untuk artikel pendek."))
	// 500 words
	content := ""
	for i := 0; i < 500; i++ {
		content += "kata "
	}
	assert.Equal(t, 2, CalculateReadingTime(content))
}

func TestGenerateSlug(t *testing.T) {
	slug1 := GenerateSlug("Judul Artikel Uji 2026!")
	assert.Contains(t, slug1, "judul-artikel-uji-2026")
	assert.Regexp(t, "^[a-z0-9-]+$", slug1)

	slugEmpty := GenerateSlug("!@#$%^")
	assert.Contains(t, slugEmpty, "artikel-")
}

func TestPost_Categories(t *testing.T) {
	handler, _, _, _, _ := setupTestPost(t)

	r := gin.New()
	r.GET("/categories", handler.GetCategories)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	assert.Equal(t, "success", res["status"])
	cats := res["data"].([]interface{})
	assert.GreaterOrEqual(t, len(cats), 6)
}

func TestPost_PublicListAndDetail(t *testing.T) {
	handler, _, _, _, _ := setupTestPost(t)

	r := gin.New()
	r.GET("/public/posts", handler.GetPublicPosts)
	r.GET("/public/posts/:slug", handler.GetPublicPostBySlug)

	// 1. List all public posts
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/public/posts", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	var res1 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	posts := res1["data"].([]interface{})
	assert.NotEmpty(t, posts)

	firstPost := posts[0].(map[string]interface{})
	slug := firstPost["slug"].(string)

	// Verify no password or user_id in serialized public json
	serialized := string(w1.Body.Bytes())
	assert.NotContains(t, serialized, "password")
	assert.NotContains(t, serialized, `"user_id"`)

	// 2. Filter by category
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/public/posts?category=Teknologi", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// 3. Filter category="Semua"
	wSemua := httptest.NewRecorder()
	reqSemua, _ := http.NewRequest(http.MethodGet, "/public/posts?category=Semua", nil)
	r.ServeHTTP(wSemua, reqSemua)
	assert.Equal(t, http.StatusOK, wSemua.Code)

	// 4. Search query matching "Golang"
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/public/posts?q=Golang", nil)
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	// 5. Search query not matching
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodGet, "/public/posts?q=zzznothere9999", nil)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
	var res4 map[string]interface{}
	_ = json.Unmarshal(w4.Body.Bytes(), &res4)
	emptyList := res4["data"].([]interface{})
	assert.Empty(t, emptyList)

	// 6. Get detail by slug
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodGet, "/public/posts/"+slug, nil)
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusOK, w5.Code)

	// 7. Get detail unknown slug -> 404
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodGet, "/public/posts/non-existent-slug", nil)
	r.ServeHTTP(w6, req6)
	assert.Equal(t, http.StatusNotFound, w6.Code)
}

func TestPost_AuthorCRUD(t *testing.T) {
	handler, _, _, adminUser, memberUser := setupTestPost(t)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", memberUser.ID)
		c.Set("role", "member")
		c.Next()
	})
	r.POST("/posts", handler.CreatePost)
	r.GET("/posts", handler.GetAuthorPosts)
	r.GET("/posts/stats", handler.GetAuthorStats)
	r.GET("/posts/:id", handler.GetPostByID)
	r.PUT("/posts/:id", handler.UpdatePost)
	r.PATCH("/posts/:id/toggle-publish", handler.TogglePublish)
	r.DELETE("/posts/:id", handler.DeletePost)

	// 1. Create published post
	isPub := true
	body1 := CreatePostRequest{
		Title:       "Artikel Uji Penulis 1",
		Category:    "Teknologi",
		Excerpt:     "Ringkasan artikel uji",
		Content:     "# Konten Uji\n\nIsi artikel pengujian komprehensif.",
		IsPublished: &isPub,
	}
	b1, _ := json.Marshal(body1)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(b1))
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)
	var res1 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &res1)
	data1 := res1["data"].(map[string]interface{})
	postID := uint(data1["id"].(float64))
	assert.Equal(t, "Artikel Uji Penulis 1", data1["title"])
	assert.Equal(t, true, data1["is_published"])
	assert.NotEmpty(t, data1["published_at"])

	// 2. Create draft post without category (defaults to Umum)
	isDraft := false
	bodyDraft := CreatePostRequest{
		Title:       "Artikel Draft Tanpa Kategori",
		Content:     "Konten draf.",
		IsPublished: &isDraft,
	}
	bDraft, _ := json.Marshal(bodyDraft)
	wDraft := httptest.NewRecorder()
	reqDraft, _ := http.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(bDraft))
	r.ServeHTTP(wDraft, reqDraft)
	assert.Equal(t, http.StatusCreated, wDraft.Code)
	var resDraft map[string]interface{}
	_ = json.Unmarshal(wDraft.Body.Bytes(), &resDraft)
	dataDraft := resDraft["data"].(map[string]interface{})
	assert.Equal(t, "Umum", dataDraft["category"])
	assert.Equal(t, false, dataDraft["is_published"])
	assert.Nil(t, dataDraft["published_at"])

	// 3. Create post missing title or content -> 400
	badPost, _ := json.Marshal(CreatePostRequest{Title: "", Content: "Konten"})
	wBad := httptest.NewRecorder()
	reqBad, _ := http.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(badPost))
	r.ServeHTTP(wBad, reqBad)
	assert.Equal(t, http.StatusBadRequest, wBad.Code)

	// Create post bad JSON
	wBadJSON := httptest.NewRecorder()
	reqBadJSON, _ := http.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString("{bad"))
	r.ServeHTTP(wBadJSON, reqBadJSON)
	assert.Equal(t, http.StatusBadRequest, wBadJSON.Code)

	// 4. List author posts
	wList := httptest.NewRecorder()
	reqList, _ := http.NewRequest(http.MethodGet, "/posts", nil)
	r.ServeHTTP(wList, reqList)
	assert.Equal(t, http.StatusOK, wList.Code)

	// 5. Get author stats
	wStats := httptest.NewRecorder()
	reqStats, _ := http.NewRequest(http.MethodGet, "/posts/stats", nil)
	r.ServeHTTP(wStats, reqStats)
	assert.Equal(t, http.StatusOK, wStats.Code)
	var resStats map[string]interface{}
	_ = json.Unmarshal(wStats.Body.Bytes(), &resStats)
	st := resStats["data"].(map[string]interface{})
	assert.GreaterOrEqual(t, st["total_articles"].(float64), float64(2))

	// 6. Get post by ID OK
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest(http.MethodGet, "/posts/"+formatUint(postID), nil)
	r.ServeHTTP(wGet, reqGet)
	assert.Equal(t, http.StatusOK, wGet.Code)

	// 7. Get post by non-number ID -> 400
	wBadID := httptest.NewRecorder()
	reqBadID, _ := http.NewRequest(http.MethodGet, "/posts/bukan-angka", nil)
	r.ServeHTTP(wBadID, reqBadID)
	assert.Equal(t, http.StatusBadRequest, wBadID.Code)

	// 8. Get post non-existent ID -> 404
	wNF := httptest.NewRecorder()
	reqNF, _ := http.NewRequest(http.MethodGet, "/posts/999999", nil)
	r.ServeHTTP(wNF, reqNF)
	assert.Equal(t, http.StatusNotFound, wNF.Code)

	// 9. Update post OK
	newTitle := "Judul Telah Diperbarui"
	newContent := "Konten baru yang lebih panjang."
	newCover := "img/test.jpg"
	newCat := "Bisnis"
	newEx := "Excerpt baru"
	newPub := true
	updateBody, _ := json.Marshal(UpdatePostRequest{
		Title:          &newTitle,
		Content:        &newContent,
		CoverImagePath: &newCover,
		Category:       &newCat,
		Excerpt:        &newEx,
		IsPublished:    &newPub,
	})
	wUp := httptest.NewRecorder()
	reqUp, _ := http.NewRequest(http.MethodPut, "/posts/"+formatUint(postID), bytes.NewBuffer(updateBody))
	r.ServeHTTP(wUp, reqUp)
	assert.Equal(t, http.StatusOK, wUp.Code)

	// Update post unpublish
	unpub := false
	upUnpub, _ := json.Marshal(UpdatePostRequest{IsPublished: &unpub})
	wUpUnpub := httptest.NewRecorder()
	reqUpUnpub, _ := http.NewRequest(http.MethodPut, "/posts/"+formatUint(postID), bytes.NewBuffer(upUnpub))
	r.ServeHTTP(wUpUnpub, reqUpUnpub)
	assert.Equal(t, http.StatusOK, wUpUnpub.Code)

	// Update post bad ID & bad JSON & not found
	wUpBadID := httptest.NewRecorder()
	reqUpBadID, _ := http.NewRequest(http.MethodPut, "/posts/bad", nil)
	r.ServeHTTP(wUpBadID, reqUpBadID)
	assert.Equal(t, http.StatusBadRequest, wUpBadID.Code)

	wUpBadJSON := httptest.NewRecorder()
	reqUpBadJSON, _ := http.NewRequest(http.MethodPut, "/posts/"+formatUint(postID), bytes.NewBufferString("{bad"))
	r.ServeHTTP(wUpBadJSON, reqUpBadJSON)
	assert.Equal(t, http.StatusBadRequest, wUpBadJSON.Code)

	wUpNF := httptest.NewRecorder()
	reqUpNF, _ := http.NewRequest(http.MethodPut, "/posts/999999", bytes.NewBuffer(updateBody))
	r.ServeHTTP(wUpNF, reqUpNF)
	assert.Equal(t, http.StatusNotFound, wUpNF.Code)

	// 10. Toggle publish
	wTog := httptest.NewRecorder()
	reqTog, _ := http.NewRequest(http.MethodPatch, "/posts/"+formatUint(postID)+"/toggle-publish", nil)
	r.ServeHTTP(wTog, reqTog)
	assert.Equal(t, http.StatusOK, wTog.Code)

	// Toggle again to unpublish
	wTog2 := httptest.NewRecorder()
	reqTog2, _ := http.NewRequest(http.MethodPatch, "/posts/"+formatUint(postID)+"/toggle-publish", nil)
	r.ServeHTTP(wTog2, reqTog2)
	assert.Equal(t, http.StatusOK, wTog2.Code)

	wTogBadID := httptest.NewRecorder()
	reqTogBadID, _ := http.NewRequest(http.MethodPatch, "/posts/bad/toggle-publish", nil)
	r.ServeHTTP(wTogBadID, reqTogBadID)
	assert.Equal(t, http.StatusBadRequest, wTogBadID.Code)

	wTogNF := httptest.NewRecorder()
	reqTogNF, _ := http.NewRequest(http.MethodPatch, "/posts/999999/toggle-publish", nil)
	r.ServeHTTP(wTogNF, reqTogNF)
	assert.Equal(t, http.StatusNotFound, wTogNF.Code)

	// 11. Delete post
	wDelBadID := httptest.NewRecorder()
	reqDelBadID, _ := http.NewRequest(http.MethodDelete, "/posts/bad", nil)
	r.ServeHTTP(wDelBadID, reqDelBadID)
	assert.Equal(t, http.StatusBadRequest, wDelBadID.Code)

	wDelNF := httptest.NewRecorder()
	reqDelNF, _ := http.NewRequest(http.MethodDelete, "/posts/999999", nil)
	r.ServeHTTP(wDelNF, reqDelNF)
	assert.Equal(t, http.StatusNotFound, wDelNF.Code)

	wDel := httptest.NewRecorder()
	reqDel, _ := http.NewRequest(http.MethodDelete, "/posts/"+formatUint(postID), nil)
	r.ServeHTTP(wDel, reqDel)
	assert.Equal(t, http.StatusOK, wDel.Code)

	// 12. Forbidden access tests (other user)
	rOther := gin.New()
	rOther.Use(func(c *gin.Context) {
		c.Set("user_id", uint(8888))
		c.Set("role", "member")
		c.Next()
	})
	rOther.GET("/posts/:id", handler.GetPostByID)
	rOther.PUT("/posts/:id", handler.UpdatePost)
	rOther.PATCH("/posts/:id/toggle-publish", handler.TogglePublish)
	rOther.DELETE("/posts/:id", handler.DeletePost)

	draftID := uint(dataDraft["id"].(float64))

	wForb1 := httptest.NewRecorder()
	reqForb1, _ := http.NewRequest(http.MethodGet, "/posts/"+formatUint(draftID), nil)
	rOther.ServeHTTP(wForb1, reqForb1)
	assert.Equal(t, http.StatusForbidden, wForb1.Code)

	wForb2 := httptest.NewRecorder()
	reqForb2, _ := http.NewRequest(http.MethodPut, "/posts/"+formatUint(draftID), bytes.NewBuffer(updateBody))
	rOther.ServeHTTP(wForb2, reqForb2)
	assert.Equal(t, http.StatusForbidden, wForb2.Code)

	wForb3 := httptest.NewRecorder()
	reqForb3, _ := http.NewRequest(http.MethodPatch, "/posts/"+formatUint(draftID)+"/toggle-publish", nil)
	rOther.ServeHTTP(wForb3, reqForb3)
	assert.Equal(t, http.StatusForbidden, wForb3.Code)

	wForb4 := httptest.NewRecorder()
	reqForb4, _ := http.NewRequest(http.MethodDelete, "/posts/"+formatUint(draftID), nil)
	rOther.ServeHTTP(wForb4, reqForb4)
	assert.Equal(t, http.StatusForbidden, wForb4.Code)

	// 13. Admin access override on other user's post
	rAdmin := gin.New()
	rAdmin.Use(func(c *gin.Context) {
		c.Set("user_id", adminUser.ID)
		c.Set("role", "admin")
		c.Next()
	})
	rAdmin.GET("/posts/:id", handler.GetPostByID)
	rAdmin.PUT("/posts/:id", handler.UpdatePost)
	rAdmin.PATCH("/posts/:id/toggle-publish", handler.TogglePublish)
	rAdmin.DELETE("/posts/:id", handler.DeletePost)

	wAdmGet := httptest.NewRecorder()
	reqAdmGet, _ := http.NewRequest(http.MethodGet, "/posts/"+formatUint(draftID), nil)
	rAdmin.ServeHTTP(wAdmGet, reqAdmGet)
	assert.Equal(t, http.StatusOK, wAdmGet.Code)

	wAdmPut := httptest.NewRecorder()
	reqAdmPut, _ := http.NewRequest(http.MethodPut, "/posts/"+formatUint(draftID), bytes.NewBuffer(updateBody))
	rAdmin.ServeHTTP(wAdmPut, reqAdmPut)
	assert.Equal(t, http.StatusOK, wAdmPut.Code)

	wAdmTog := httptest.NewRecorder()
	reqAdmTog, _ := http.NewRequest(http.MethodPatch, "/posts/"+formatUint(draftID)+"/toggle-publish", nil)
	rAdmin.ServeHTTP(wAdmTog, reqAdmTog)
	assert.Equal(t, http.StatusOK, wAdmTog.Code)

	wAdmDel := httptest.NewRecorder()
	reqAdmDel, _ := http.NewRequest(http.MethodDelete, "/posts/"+formatUint(draftID), nil)
	rAdmin.ServeHTTP(wAdmDel, reqAdmDel)
	assert.Equal(t, http.StatusOK, wAdmDel.Code)
}

func TestPost_AdminHandlers(t *testing.T) {
	handler, _, _, _, memberUser := setupTestPost(t)

	r := gin.New()
	r.GET("/admin/posts", handler.GetAdminPosts)
	r.GET("/admin/users", handler.GetAdminUsers)
	r.GET("/admin/users/:id/posts", handler.GetAdminUserPosts)

	// 1. Admin get all posts
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/admin/posts", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 2. Admin get all users
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// 3. Admin get user posts OK
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/admin/users/"+formatUint(memberUser.ID)+"/posts", nil)
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	// 4. Admin get user posts unknown user -> returns empty list 200
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodGet, "/admin/users/999999/posts", nil)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)

	// 5. Admin get user posts bad ID -> 400
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodGet, "/admin/users/bukan-angka/posts", nil)
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusBadRequest, w5.Code)
}

func formatUint(n uint) string {
	return strconv.FormatUint(uint64(n), 10)
}

// Mock repository for service tests
type mockPostRepo struct {
	createErr         error
	findByIDErr       error
	findByIDPost      *database.Post
	findBySlugErr     error
	findAllPublicErr  error
	findByUserIDErr   error
	findAllErr        error
	updateErr         error
	deleteErr         error
	countStatsErr     error
	findAdminUsersErr error
}

func (m *mockPostRepo) Create(post *database.Post) error { return m.createErr }
func (m *mockPostRepo) FindByID(id uint) (*database.Post, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDPost, nil
}
func (m *mockPostRepo) FindBySlug(slug string) (*database.Post, error) {
	return nil, m.findBySlugErr
}
func (m *mockPostRepo) FindAllPublic(category, query string) ([]database.Post, error) {
	return nil, m.findAllPublicErr
}
func (m *mockPostRepo) FindByUserID(userID uint) ([]database.Post, error) {
	return nil, m.findByUserIDErr
}
func (m *mockPostRepo) FindAll() ([]database.Post, error) {
	return nil, m.findAllErr
}
func (m *mockPostRepo) Update(post *database.Post) error { return m.updateErr }
func (m *mockPostRepo) Delete(id uint) error             { return m.deleteErr }
func (m *mockPostRepo) CountStatsByUserID(userID uint) (int64, int64, int64, error) {
	return 0, 0, 0, m.countStatsErr
}
func (m *mockPostRepo) FindAdminUsers() ([]AdminUserStatsResponse, error) {
	return nil, m.findAdminUsersErr
}

// Mock service for handler 500 error tests
type mockPostService struct {
	Service
	err error
}

func (m *mockPostService) GetCategories() []string { return []string{"Umum"} }
func (m *mockPostService) GetPublicPosts(category, query string) ([]PublicPostResponse, error) {
	return nil, m.err
}
func (m *mockPostService) CreatePost(userID uint, req *CreatePostRequest) (*PostResponse, error) {
	return nil, m.err
}
func (m *mockPostService) GetAuthorPosts(userID uint) ([]PostResponse, error) {
	return nil, m.err
}
func (m *mockPostService) GetPostByID(id uint, userID uint, role string) (*PostResponse, error) {
	return nil, m.err
}
func (m *mockPostService) GetAuthorStats(userID uint) (*AuthorStatsResponse, error) {
	return nil, m.err
}
func (m *mockPostService) UpdatePost(id uint, userID uint, role string, req *UpdatePostRequest) (*PostResponse, error) {
	return nil, m.err
}
func (m *mockPostService) TogglePublish(id uint, userID uint, role string) (*PostResponse, error) {
	return nil, m.err
}
func (m *mockPostService) DeletePost(id uint, userID uint, role string) error {
	return m.err
}
func (m *mockPostService) GetAdminPosts() ([]PostResponse, error) {
	return nil, m.err
}
func (m *mockPostService) GetAdminUsers() ([]AdminUserStatsResponse, error) {
	return nil, m.err
}
func (m *mockPostService) GetAdminUserPosts(targetUserID uint) ([]PostResponse, error) {
	return nil, m.err
}

func TestPost_RepositoryErrors(t *testing.T) {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)
	sqlDB, _ := db.DB()
	sqlDB.Close()

	repo := NewRepository(db)
	_, err = repo.FindAllPublic("", "")
	assert.Error(t, err)

	_, err = repo.FindByUserID(1)
	assert.Error(t, err)

	_, err = repo.FindAll()
	assert.Error(t, err)

	_, _, _, err = repo.CountStatsByUserID(1)
	assert.Error(t, err)

	_, err = repo.FindAdminUsers()
	assert.Error(t, err)
}

func TestPost_ServiceErrors(t *testing.T) {
	dbErr := errors.New("db error")
	mock := &mockPostRepo{
		findAllPublicErr:  dbErr,
		createErr:         dbErr,
		findByUserIDErr:   dbErr,
		countStatsErr:     dbErr,
		updateErr:         dbErr,
		findAllErr:        dbErr,
		findAdminUsersErr: dbErr,
		findByIDPost: &database.Post{
			ID:          1,
			UserID:      1,
			IsPublished: false,
		},
	}
	svc := NewService(mock)

	_, err := svc.GetPublicPosts("", "")
	assert.Error(t, err)

	_, err = svc.CreatePost(1, &CreatePostRequest{Title: "Title", Content: "Content"})
	assert.Error(t, err)

	_, err = svc.GetAuthorPosts(1)
	assert.Error(t, err)

	_, err = svc.GetAuthorStats(1)
	assert.Error(t, err)

	title := "New Title"
	_, err = svc.UpdatePost(1, 1, "member", &UpdatePostRequest{Title: &title})
	assert.Error(t, err)

	_, err = svc.TogglePublish(1, 1, "member")
	assert.Error(t, err)

	_, err = svc.GetAdminPosts()
	assert.Error(t, err)

	_, err = svc.GetAdminUserPosts(1)
	assert.Error(t, err)
}

func TestPost_HandlerErrors(t *testing.T) {
	errFatal := errors.New("fatal server error")
	mockSvc := &mockPostService{err: errFatal}
	h := NewHandler(mockSvc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("role", "member")
		c.Next()
	})
	r.GET("/public/posts", h.GetPublicPosts)
	r.POST("/posts", h.CreatePost)
	r.GET("/posts", h.GetAuthorPosts)
	r.GET("/posts/stats", h.GetAuthorStats)
	r.GET("/posts/:id", h.GetPostByID)
	r.PUT("/posts/:id", h.UpdatePost)
	r.PATCH("/posts/:id/toggle-publish", h.TogglePublish)
	r.DELETE("/posts/:id", h.DeletePost)
	r.GET("/admin/posts", h.GetAdminPosts)
	r.GET("/admin/users", h.GetAdminUsers)
	r.GET("/admin/users/:id/posts", h.GetAdminUserPosts)

	// 1. GetPublicPosts 500
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/public/posts", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 2. CreatePost 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/posts", bytes.NewBufferString(`{"title":"T","content":"C"}`))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 3. GetAuthorPosts 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/posts", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 4. GetAuthorStats 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/posts/stats", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 5. GetPostByID 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/posts/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 6. UpdatePost 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/posts/1", bytes.NewBufferString(`{"title":"T"}`))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// UpdatePost 400 (ErrInvalidInput)
	mockSvc.err = ErrInvalidInput
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPut, "/posts/1", bytes.NewBufferString(`{"title":"T"}`))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.err = errFatal

	// 7. TogglePublish 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPatch, "/posts/1/toggle-publish", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 8. DeletePost 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/posts/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 9. GetAdminPosts 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/admin/posts", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 10. GetAdminUsers 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/admin/users", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 11. GetAdminUserPosts 500
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/admin/users/1/posts", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
