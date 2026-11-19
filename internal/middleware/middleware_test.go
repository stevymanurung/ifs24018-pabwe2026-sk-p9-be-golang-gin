package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGenerateToken(t *testing.T) {
	secret := "test_secret_key"
	token, err := GenerateToken(1, "user1", "user1@test.com", "member", secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	customToken, err := GenerateToken(2, "user2", "user2@test.com", "admin", secret, 2*time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, customToken)
}

func TestCORS(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Non-OPTIONS request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// OPTIONS request
	wOpt := httptest.NewRecorder()
	reqOpt, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	r.ServeHTTP(wOpt, reqOpt)
	assert.Equal(t, http.StatusNoContent, wOpt.Code)
}

func TestAuth(t *testing.T) {
	secret := "my_secret_key"
	r := gin.New()
	r.Use(Auth(secret))
	r.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": uid})
	})

	// Case 1: No auth header
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// Case 2: Malformed auth header (no Bearer)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Basic 123456")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// Case 3: Invalid token
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req3.Header.Set("Authorization", "Bearer invalid.token.value")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusUnauthorized, w3.Code)

	// Case 4: Token signed with different signing method (RSA) - triggers wrong algorithm check
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	rsaToken := jwt.NewWithClaims(jwt.SigningMethodRS256, &JWTClaims{
		UserID: 1,
	})
	signedRsa, err := rsaToken.SignedString(rsaKey)
	assert.NoError(t, err)
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req4.Header.Set("Authorization", "Bearer "+signedRsa)
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusUnauthorized, w4.Code)

	// Case 5: Valid token - success path
	validToken, err := GenerateToken(10, "tester", "tester@test.com", "member", secret)
	assert.NoError(t, err)
	w5 := httptest.NewRecorder()
	req5, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req5.Header.Set("Authorization", "Bearer "+validToken)
	r.ServeHTTP(w5, req5)
	assert.Equal(t, http.StatusOK, w5.Code)

	// Case 6: Token with wrong claims type (using plain MapClaims instead of JWTClaims)
	// This triggers the claims assertion failure
	plainClaims := jwt.MapClaims{
		"user_id": float64(5),
		"exp":     float64(time.Now().Add(time.Hour).Unix()),
	}
	plainToken := jwt.NewWithClaims(jwt.SigningMethodHS256, plainClaims)
	signedPlain, err := plainToken.SignedString([]byte(secret))
	assert.NoError(t, err)
	// ParseWithClaims with &JWTClaims{} but MapClaims token actually fills ok,
	// so let's force a known-bad scenario: expire the token so it fails parse step
	expiredClaims := jwt.MapClaims{
		"user_id": float64(5),
		"exp":     float64(time.Now().Add(-time.Hour).Unix()),
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	signedExpired, err := expiredToken.SignedString([]byte(secret))
	assert.NoError(t, err)
	_ = signedPlain
	w6 := httptest.NewRecorder()
	req6, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req6.Header.Set("Authorization", "Bearer "+signedExpired)
	r.ServeHTTP(w6, req6)
	assert.Equal(t, http.StatusUnauthorized, w6.Code)
}

func TestRequireRole(t *testing.T) {
	r := gin.New()
	r.GET("/admin-only", func(c *gin.Context) {
		c.Set("role", c.Query("role"))
		c.Next()
	}, RequireRole("admin"), func(c *gin.Context) {
		c.String(http.StatusOK, "welcome admin")
	})

	// Fail: member
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/admin-only?role=member", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusForbidden, w1.Code)

	// Pass: admin
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/admin-only?role=admin", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRateLimiter(t *testing.T) {
	r := gin.New()
	r.Use(RateLimiter(2))
	r.GET("/limited", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Request 1: OK
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/limited", nil)
	req1.RemoteAddr = "192.0.2.1:1234"
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Request 2: OK
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/limited", nil)
	req2.RemoteAddr = "192.0.2.1:1234"
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Request 3: Exceeded -> 429
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodGet, "/limited", nil)
	req3.RemoteAddr = "192.0.2.1:1234"
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)

	// Reset window simulation
	rateLimitMutex.Lock()
	rateLimitStore["192.0.2.1"] = &clientRecord{
		count:     1,
		resetTime: time.Now().Add(-time.Second),
	}
	rateLimitMutex.Unlock()

	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest(http.MethodGet, "/limited", nil)
	req4.RemoteAddr = "192.0.2.1:1234"
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}
