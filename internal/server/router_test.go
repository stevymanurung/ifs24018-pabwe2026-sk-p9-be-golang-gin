package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/config"
	"app/internal/database"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := database.ConnectTestDB()
	assert.NoError(t, err)
	return db
}

func generateTestToken(secret string, userID uint, role string) string {
	token, _ := middleware.GenerateToken(userID, "testuser", "test@delcom.org", role, secret, 24*time.Hour)
	return token
}

func TestRouter_AllRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	cfg := &config.Config{
		AppKey:               "test-secret-key-1234567890123456",
		SearchEngineEndpoint: "https://mock.search",
		OpenAIEndpoint:       "https://mock.openai",
		OpenAIKey:            "sk-mock",
	}

	r := SetupRouter(cfg, db)

	// 0. Test base URL / membalas JSON { status, message, data }
	w0 := httptest.NewRecorder()
	req0, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w0, req0)
	assert.Equal(t, http.StatusOK, w0.Code)
	assert.Contains(t, w0.Body.String(), `"status":"success"`)
	assert.Contains(t, w0.Body.String(), `"data"`)

	// 1. Test 404 NoRoute
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/route-tidak-ada-uji", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "fail")

	// 2. Test /api.json (HTTP)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api.json", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "swagger")

	// 3. Test /api.json (HTTPS via TLS and via Header)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api.json", nil)
	req.TLS = &tls.ConnectionState{}
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api.json", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Test /swagger/*any
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	req.RequestURI = "/swagger/index.html"
	r.ServeHTTP(w, req)
	t.Logf("Swagger code: %d, body: %s", w.Code, w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)

	// 5. Test Public Categories
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/public/categories", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 6. Test Auth Route
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 7. Test Protected Author Post Route with Token
	user := database.User{
		Username: "author1",
		Email:    "author1@example.com",
		Role:     "member",
		Password: "hashedpassword",
	}
	db.Create(&user)
	token := generateTestToken(cfg.AppKey, user.ID, "member")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/posts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 8. Test Protected Admin Route with Admin Token
	adminUser := database.User{
		Username: "admin1",
		Email:    "admin1@example.com",
		Role:     "admin",
		Password: "hashedpassword",
	}
	db.Create(&adminUser)
	adminToken := generateTestToken(cfg.AppKey, adminUser.ID, "admin")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
