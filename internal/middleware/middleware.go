package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"app/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uint, username, email, role, secretKey string, expDuration ...time.Duration) (string, error) {
	dur := 24 * time.Hour
	if len(expDuration) > 0 && expDuration[0] > 0 {
		dur = expDuration[0]
	}

	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(dur)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Accept, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func Auth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.AbortFail(c, http.StatusUnauthorized, "Token autentikasi tidak ditemukan atau tidak valid")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode signing tidak valid: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			response.AbortFail(c, http.StatusUnauthorized, "Token tidak valid atau telah kedaluwarsa")
			return
		}

		claims := token.Claims.(*JWTClaims)

		c.Set("user", claims)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("role")
		for _, role := range roles {
			if strings.EqualFold(userRole, role) {
				c.Next()
				return
			}
		}
		response.AbortFail(c, http.StatusForbidden, "Akses ditolak: role tidak memiliki izin")
	}
}

type clientRecord struct {
	count     int
	resetTime time.Time
}

var (
	rateLimitStore = make(map[string]*clientRecord)
	rateLimitMutex sync.Mutex
)

func RateLimiter(limit ...int) gin.HandlerFunc {
	maxRequests := 1000
	if len(limit) > 0 && limit[0] > 0 {
		maxRequests = limit[0]
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rateLimitMutex.Lock()
		record, exists := rateLimitStore[ip]
		if !exists || now.After(record.resetTime) {
			rateLimitStore[ip] = &clientRecord{
				count:     1,
				resetTime: now.Add(time.Minute),
			}
			rateLimitMutex.Unlock()
			c.Next()
			return
		}

		record.count++
		if record.count > maxRequests {
			rateLimitMutex.Unlock()
			response.AbortFail(c, http.StatusTooManyRequests, "Terlalu banyak permintaan, coba lagi nanti")
			return
		}
		rateLimitMutex.Unlock()

		c.Next()
	}
}
