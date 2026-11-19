package auth

import (
	"errors"
	"net/http"

	"app/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	res, err := h.service.Register(&req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrDuplicateUser) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memproses registrasi: "+err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "Registrasi berhasil", res)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	if req.Username == "" || req.Password == "" {
		response.Fail(c, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	res, err := h.service.Login(&req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Fail(c, http.StatusUnauthorized, "Username atau password salah")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memproses login: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Login berhasil", res)
}

func (h *Handler) Logout(c *gin.Context) {
	response.Success(c, http.StatusOK, "Logout berhasil")
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.service.GetProfile(userID)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	userResp := UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}
	response.Success(c, http.StatusOK, "Profil berhasil diambil", userResp)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Payload request tidak valid")
		return
	}

	updated, err := h.service.UpdateProfile(userID, &req)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrDuplicateUser) {
			response.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			response.Fail(c, http.StatusNotFound, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memperbarui profil: "+err.Error())
		return
	}

	userResp := UserResponse{
		ID:       updated.ID,
		Username: updated.Username,
		Email:    updated.Email,
		Role:     updated.Role,
	}
	response.Success(c, http.StatusOK, "Profil berhasil diperbarui", userResp)
}
