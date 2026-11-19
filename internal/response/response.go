package response

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, code int, message string, data ...interface{}) {
	resp := gin.H{
		"status":  "success",
		"message": message,
	}
	if len(data) > 0 && data[0] != nil {
		resp["data"] = data[0]
	}
	c.JSON(code, resp)
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"status":  "fail",
		"message": message,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"status":  "error",
		"message": message,
	})
}

func AbortFail(c *gin.Context, code int, message string) {
	Fail(c, code, message)
	c.Abort()
}

func AbortError(c *gin.Context, code int, message string) {
	Error(c, code, message)
	c.Abort()
}
