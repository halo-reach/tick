package api

import (
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

var variableKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

type VariableHandler struct {
	variableRepo *repo.VariableRepo
}

func NewVariableHandler(vr *repo.VariableRepo) *VariableHandler {
	return &VariableHandler{variableRepo: vr}
}

type CreateVariableRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

func (h *VariableHandler) Create(c *gin.Context) {
	var req CreateVariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}
	tenantID := middleware.GetTenantID(c)

	if !variableKeyRegex.MatchString(req.Key) {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Variable key must contain only letters, numbers, underscores, and hyphens")
		return
	}

	exists, err := h.variableRepo.ExistsByKey(c.Request.Context(), tenantID, req.Key)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if exists {
		RespondErrorMsg(c, http.StatusConflict, "CONFLICT", "Variable key already exists")
		return
	}

	now := time.Now()
	v := &domain.Variable{
		ID:        generateID("var_", 12),
		TenantID:  tenantID,
		Key:       req.Key,
		Value:     req.Value,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.variableRepo.Create(c.Request.Context(), v); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusCreated, v)
}

func (h *VariableHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	vars, err := h.variableRepo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, vars)
}

type UpdateVariableRequest struct {
	Key   *string `json:"key"`
	Value *string `json:"value"`
}

func (h *VariableHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	v, err := h.variableRepo.GetByID(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}

	var req UpdateVariableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	if req.Key != nil {
		if !variableKeyRegex.MatchString(*req.Key) {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Variable key must contain only letters, numbers, underscores, and hyphens")
			return
		}
		if *req.Key != v.Key {
			exists, err := h.variableRepo.ExistsByKey(c.Request.Context(), tenantID, *req.Key)
			if err != nil {
				RespondError(c, http.StatusInternalServerError, ErrInternal)
				return
			}
			if exists {
				RespondErrorMsg(c, http.StatusConflict, "CONFLICT", "Variable key already exists")
				return
			}
		}
		v.Key = *req.Key
	}
	if req.Value != nil {
		v.Value = *req.Value
	}

	if err := h.variableRepo.Update(c.Request.Context(), v); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, v)
}

func (h *VariableHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	if err := h.variableRepo.Delete(c.Request.Context(), id, tenantID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"deleted": true})
}
