package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestoreOriginalPath(t *testing.T) {
	var gotPath, gotQuery string
	h := RestoreOriginalPath(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
	}))

	cases := []struct {
		name, url, path, query string
	}{
		{"path asli dipulihkan dari __p", "/api/index?__p=/api/auth/login", "/api/auth/login", ""},
		{"query asli dipertahankan", "/api/index?__p=/api/public/posts&q=go", "/api/public/posts", "q=go"},
		{"__p tanpa slash awal", "/api/index?__p=api/posts", "/api/posts", ""},
		{"__p kosong menjadi root", "/api/index?__p=", "/", ""},
		{"placeholder tidak terganti diabaikan", "/api/public/categories?__p=:path*", "/api/public/categories", ""},
		{"placeholder dollar diabaikan", "/api/public/categories?__p=$1", "/api/public/categories", ""},
		{"tanpa __p tidak diubah", "/api/public/categories?x=1", "/api/public/categories", "x=1"},
		{"/api/index tanpa __p menjadi root", "/api/index", "/", ""},
		{"/api/index.go tanpa __p menjadi root", "/api/index.go", "/", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			h.ServeHTTP(httptest.NewRecorder(), req)
			assert.Equal(t, tc.path, gotPath)
			assert.Equal(t, tc.query, gotQuery)
		})
	}
}

func TestRestoreOriginalPath_Debug(t *testing.T) {
	called := false
	h := RestoreOriginalPath(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/categories?debug=1", nil))
	assert.False(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"path":"/api/public/categories"`)
}
