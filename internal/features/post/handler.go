package post

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetCategories(c *gin.Context) {
	categories := h.service.GetCategories()
	response.Success(c, http.StatusOK, "Daftar kategori berhasil diambil", categories)
}

func (h *Handler) GetPublicPosts(c *gin.Context) {
	category := c.Query("category")
	query := c.Query("q")

	posts, err := h.service.GetPublicPosts(category, query)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar artikel publik: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel publik berhasil diambil", posts)
}

func (h *Handler) GetPublicPostBySlug(c *gin.Context) {
	slug := c.Param("slug")
	post, err := h.service.GetPublicPostBySlug(slug)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "Artikel tidak ditemukan")
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", post)
}

func (h *Handler) CreatePost(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	post, err := h.service.CreatePost(userID, &req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			response.Fail(c, http.StatusBadRequest, "Judul dan konten artikel wajib diisi")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal membuat artikel: "+err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Artikel berhasil dibuat", post)
}

func (h *Handler) GetAuthorPosts(c *gin.Context) {
	userID := c.GetUint("user_id")
	posts, err := h.service.GetAuthorPosts(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil artikel: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar artikel penulis berhasil diambil", posts)
}

func (h *Handler) GetPostByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID artikel tidak valid")
		return
	}

	userID := c.GetUint("user_id")
	role := c.GetString("role")

	post, err := h.service.GetPostByID(uint(id), userID, role)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Artikel tidak ditemukan")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil artikel: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Detail artikel berhasil diambil", post)
}

func (h *Handler) GetAuthorStats(c *gin.Context) {
	userID := c.GetUint("user_id")
	stats, err := h.service.GetAuthorStats(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil statistik: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Statistik artikel berhasil diambil", stats)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID artikel tidak valid")
		return
	}

	userID := c.GetUint("user_id")
	role := c.GetString("role")

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	post, err := h.service.UpdatePost(uint(id), userID, role, &req)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Artikel tidak ditemukan")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		if errors.Is(err, ErrInvalidInput) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memperbarui artikel: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil diperbarui", post)
}

func (h *Handler) TogglePublish(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID artikel tidak valid")
		return
	}

	userID := c.GetUint("user_id")
	role := c.GetString("role")

	post, err := h.service.TogglePublish(uint(id), userID, role)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Artikel tidak ditemukan")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal mengubah status publikasi: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Status publikasi artikel berhasil diubah", post)
}

func (h *Handler) DeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID artikel tidak valid")
		return
	}

	userID := c.GetUint("user_id")
	role := c.GetString("role")

	err = h.service.DeletePost(uint(id), userID, role)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.Fail(c, http.StatusNotFound, "Artikel tidak ditemukan")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Fail(c, http.StatusForbidden, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal menghapus artikel: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Artikel berhasil dihapus")
}

func (h *Handler) GetAdminPosts(c *gin.Context) {
	posts, err := h.service.GetAdminPosts()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil seluruh artikel: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Seluruh artikel berhasil diambil", posts)
}

func (h *Handler) GetAdminUsers(c *gin.Context) {
	users, err := h.service.GetAdminUsers()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil daftar pengguna: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar pengguna berhasil diambil", users)
}

func (h *Handler) GetAdminUserPosts(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}

	posts, err := h.service.GetAdminUserPosts(uint(id))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Gagal mengambil artikel pengguna: "+err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Artikel pengguna berhasil diambil", posts)
}
