package ai

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

func (h *Handler) GenerateDraft(c *gin.Context) {
	var req GenerateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "Field topic wajib diisi dengan format JSON yang valid")
		return
	}

	if req.Topic == "" {
		response.Fail(c, http.StatusBadRequest, "Field topic wajib diisi")
		return
	}

	draft, err := h.service.GenerateDraft(&req)
	if err != nil {
		if errors.Is(err, ErrTopicRequired) {
			response.Fail(c, http.StatusBadRequest, "Field topic wajib diisi")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Gagal memproses draf OpenAI: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Draf artikel AI berhasil dibuat", draft)
}
