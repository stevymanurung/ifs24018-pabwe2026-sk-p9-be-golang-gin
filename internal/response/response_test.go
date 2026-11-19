package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResponse_Struct(t *testing.T) {
	r := Response{
		Status:  "success",
		Message: "ok",
		Data:    "test",
	}
	assert.Equal(t, "success", r.Status)
	assert.Equal(t, "ok", r.Message)
	assert.Equal(t, "test", r.Data)
}

func TestSuccess_WithData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, http.StatusOK, "Berhasil", map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, "Berhasil", body["message"])
	assert.NotNil(t, body["data"])
}

func TestSuccess_WithoutData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, http.StatusOK, "Berhasil tanpa data")

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, "Berhasil tanpa data", body["message"])
	_, hasData := body["data"]
	assert.False(t, hasData)
}

func TestSuccess_WithNilData(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, http.StatusOK, "Berhasil dengan nil", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "success", body["status"])
	_, hasData := body["data"]
	assert.False(t, hasData)
}

func TestFail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, http.StatusBadRequest, "Permintaan gagal")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "fail", body["status"])
	assert.Equal(t, "Permintaan gagal", body["message"])
	_, hasData := body["data"]
	assert.False(t, hasData)
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusInternalServerError, "Kesalahan server")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "error", body["status"])
	assert.Equal(t, "Kesalahan server", body["message"])
	_, hasData := body["data"]
	assert.False(t, hasData)
}

func TestAbortFail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	AbortFail(c, http.StatusUnauthorized, "Tidak diizinkan")

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "fail", body["status"])
}

func TestAbortError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	AbortError(c, http.StatusInternalServerError, "Fatal error")

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	assert.NoError(t, err)
	assert.Equal(t, "error", body["status"])
}
