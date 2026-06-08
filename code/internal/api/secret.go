package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/auth"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

type SecretHandler struct {
	secretRepo *repo.SecretRepo
}

func NewSecretHandler(sr *repo.SecretRepo) *SecretHandler {
	return &SecretHandler{secretRepo: sr}
}

func (h *SecretHandler) Create(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	secret, err := auth.GenerateSecret()
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	s := &domain.SigningSecret{
		ID:        generateID("sec_", 12),
		TenantID:  tenantID,
		Secret:    secret,
		Status:    domain.SecretActive,
		CreatedAt: time.Now(),
	}
	if err := h.secretRepo.Create(c.Request.Context(), s); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusCreated, gin.H{"id": s.ID, "secret": secret})
}

func (h *SecretHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	secrets, err := h.secretRepo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, secrets)
}

func (h *SecretHandler) Revoke(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	if err := h.secretRepo.Revoke(c.Request.Context(), id, tenantID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"revoked": true})
}
