package api

import (
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/auth"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

type TenantHandler struct {
	tenantRepo *repo.TenantRepo
	keyRepo    *repo.ApiKeyRepo
	secretRepo *repo.SecretRepo
	memberRepo *repo.MemberRepo
}

func NewTenantHandler(tr *repo.TenantRepo, kr *repo.ApiKeyRepo, sr *repo.SecretRepo, mr *repo.MemberRepo) *TenantHandler {
	return &TenantHandler{tenantRepo: tr, keyRepo: kr, secretRepo: sr, memberRepo: mr}
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *TenantHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	if !usernameRegex.MatchString(req.Username) {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Username must be 3-64 chars, alphanumeric, _ or -")
		return
	}
	if len(req.Password) < 8 {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters")
		return
	}

	existing, _ := h.tenantRepo.GetByUsername(c.Request.Context(), req.Username)
	if existing != nil {
		RespondError(c, http.StatusConflict, ErrConflict)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	tenantID := generateID("ten_", 12)
	now := time.Now()
	username := req.Username
	tenant := &domain.Tenant{
		ID:            tenantID,
		Name:          req.Username,
		Username:      &username,
		PasswordHash:  &passwordHash,
		Status:        domain.TenantActive,
		QuotaMaxTasks: 100,
		QuotaMaxRPS:   50,
		CreatedAt:     now,
	}
	if err := h.tenantRepo.Create(c.Request.Context(), tenant); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	plainKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	apiKey := &domain.ApiKey{
		ID:        generateID("key_", 12),
		TenantID:  tenantID,
		Name:      "default",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Status:    domain.KeyActive,
		CreatedAt: now,
	}
	if err := h.keyRepo.Create(c.Request.Context(), apiKey); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	token, err := auth.GenerateJWT(tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusCreated, gin.H{
		"token":     token,
		"tenant_id": tenantID,
		"api_key":   plainKey,
	})
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *TenantHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	tenant, err := h.tenantRepo.GetByUsername(c.Request.Context(), req.Username)
	if err != nil || tenant.PasswordHash == nil {
		RespondError(c, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	if !auth.VerifyPassword(*tenant.PasswordHash, req.Password) {
		RespondError(c, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	token, err := auth.GenerateJWT(tenant.ID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusOK, gin.H{
		"token":                token,
		"tenant_id":           tenant.ID,
		"must_change_password": tenant.MustChangePassword,
	})
}

func (h *TenantHandler) Me(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{
		"tenant_id":      tenant.ID,
		"username":       tenant.Username,
		"name":           tenant.Name,
		"quota_max_tasks": tenant.QuotaMaxTasks,
		"quota_max_rps":  tenant.QuotaMaxRPS,
		"created_at":     tenant.CreatedAt,
	})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *TenantHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}
	if len(req.NewPassword) < 8 {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "New password must be at least 8 characters")
		return
	}

	tenantID := middleware.GetTenantID(c)
	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	if tenant.PasswordHash == nil || !auth.VerifyPassword(*tenant.PasswordHash, req.CurrentPassword) {
		RespondErrorMsg(c, http.StatusUnauthorized, "UNAUTHORIZED", "Current password is incorrect")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	if err := h.tenantRepo.UpdatePassword(c.Request.Context(), tenantID, newHash); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusOK, gin.H{"changed": true})
}

func (h *TenantHandler) ListKeys(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	keys, err := h.keyRepo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, keys)
}

type CreateKeyRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *TenantHandler) CreateKey(c *gin.Context) {
	var req CreateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	tenantID := middleware.GetTenantID(c)

	count, err := h.keyRepo.CountActiveByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if count >= 20 {
		RespondErrorMsg(c, http.StatusUnprocessableEntity, "LIMIT_REACHED", "Maximum 20 active API keys per tenant")
		return
	}

	plainKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	apiKey := &domain.ApiKey{
		ID:        generateID("key_", 12),
		TenantID:  tenantID,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Status:    domain.KeyActive,
		CreatedAt: time.Now(),
	}
	if err := h.keyRepo.Create(c.Request.Context(), apiKey); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusCreated, gin.H{"id": apiKey.ID, "name": apiKey.Name, "api_key": plainKey})
}

func (h *TenantHandler) RevokeKey(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	if err := h.keyRepo.Revoke(c.Request.Context(), id, tenantID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"revoked": true})
}

func (h *TenantHandler) Quota(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	taskCount, err := h.tenantRepo.CountTasksByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{
		"max_tasks":     tenant.QuotaMaxTasks,
		"used_tasks":    taskCount,
		"max_rps":       tenant.QuotaMaxRPS,
		"tenant_status": tenant.Status,
	})
}

func (h *TenantHandler) Status(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{
		"tenant_id": tenant.ID,
		"name":      tenant.Name,
		"status":    tenant.Status,
	})
}

type CreateTenantForUserRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *TenantHandler) CreateTenantForUser(c *gin.Context) {
	var req CreateTenantForUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	tenantID := generateID("ten_", 12)

	tenant := &domain.Tenant{
		ID:            tenantID,
		Name:          req.Name,
		Status:        domain.TenantActive,
		QuotaMaxTasks: 100,
		QuotaMaxRPS:   50,
		CreatedAt:     now,
	}
	if err := h.tenantRepo.Create(c.Request.Context(), tenant); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	member := &domain.TenantMember{
		ID:       generateID("mbr_", 12),
		TenantID: tenantID,
		UserID:   userID,
		Role:     domain.RoleOwner,
		JoinedAt: now,
	}
	if err := h.memberRepo.Create(c.Request.Context(), member); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	token, err := auth.GenerateScopedJWT(userID, tenantID, string(domain.RoleOwner))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusCreated, gin.H{
		"id":    tenantID,
		"name":  req.Name,
		"role":  domain.RoleOwner,
		"token": token,
	})
}

type RenameTenantRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *TenantHandler) Rename(c *gin.Context) {
	var req RenameTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	tenantID := middleware.GetTenantID(c)
	if err := h.tenantRepo.UpdateName(c.Request.Context(), tenantID, req.Name); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusOK, gin.H{"id": tenantID, "name": req.Name})
}
