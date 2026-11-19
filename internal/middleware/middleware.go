package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Role     string `json:"role"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
func Token(secret string, id uint, username, role string) (string, error) {
	claims := Claims{UserID: id, Username: username, Role: role, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(401, gin.H{"status": "error", "message": "Unauthorized"})
			return
		}
		t, err := jwt.ParseWithClaims(parts[1], &Claims{}, func(t *jwt.Token) (any, error) { return []byte(secret), nil })
		if err != nil || !t.Valid {
			c.AbortWithStatusJSON(401, gin.H{"status": "error", "message": "Invalid or expired token"})
			return
		}
		cl, ok := t.Claims.(*Claims)
		if !ok {
			c.AbortWithStatus(401)
			return
		}
		c.Set("userID", cl.UserID)
		c.Set("role", cl.Role)
		c.Set("username", cl.Username)
		c.Next()
	}
}
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("role") != role {
			c.AbortWithStatusJSON(403, gin.H{"status": "error", "message": "Forbidden"})
			return
		}
		c.Next()
	}
}
