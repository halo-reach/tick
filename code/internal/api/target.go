package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

type TargetHandler struct {
	targetRepo *repo.TargetRepo
}

func NewTargetHandler(tr *repo.TargetRepo) *TargetHandler {
	return &TargetHandler{targetRepo: tr}
}

type CreateTargetRequest struct {
	Name   string          `json:"name" binding:"required"`
	Type   string          `json:"type" binding:"required"`
	Config json.RawMessage `json:"config" binding:"required"`
}

func (h *TargetHandler) Create(c *gin.Context) {
	var req CreateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	tenantID := middleware.GetTenantID(c)
	now := time.Now()
	target := &domain.Target{
		ID:        generateID("tgt_", 12),
		TenantID:  tenantID,
		Name:      req.Name,
		Type:      domain.TargetType(req.Type),
		Config:    req.Config,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.targetRepo.Create(c.Request.Context(), target); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusCreated, target)
}

func (h *TargetHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	targets, err := h.targetRepo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, targets)
}

func (h *TargetHandler) Get(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	target, err := h.targetRepo.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}
	RespondData(c, http.StatusOK, target)
}

type UpdateTargetRequest struct {
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

func (h *TargetHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	target, err := h.targetRepo.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}

	var req UpdateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}
	if req.Name != "" {
		target.Name = req.Name
	}
	if req.Type != "" {
		target.Type = domain.TargetType(req.Type)
	}
	if req.Config != nil {
		target.Config = req.Config
	}

	if err := h.targetRepo.Update(c.Request.Context(), target); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, target)
}

func (h *TargetHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	hasActive, err := h.targetRepo.HasActiveTasks(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if hasActive {
		RespondErrorMsg(c, http.StatusConflict, "CONFLICT", "Target is referenced by active tasks")
		return
	}

	if err := h.targetRepo.Delete(c.Request.Context(), id, tenantID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"deleted": true})
}
