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

type UserHandler struct {
	userRepo       *repo.UserRepo
	memberRepo     *repo.MemberRepo
	tenantRepo     *repo.TenantRepo
	invitationRepo *repo.InvitationRepo
}

func NewUserHandler(userRepo *repo.UserRepo, memberRepo *repo.MemberRepo, tenantRepo *repo.TenantRepo, invitationRepo *repo.InvitationRepo) *UserHandler {
	return &UserHandler{userRepo: userRepo, memberRepo: memberRepo, tenantRepo: tenantRepo, invitationRepo: invitationRepo}
}

type UserRegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name"`
}

func (h *UserHandler) Register(c *gin.Context) {
	var req UserRegisterRequest
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

	existing, _ := h.userRepo.GetByUsername(c.Request.Context(), req.Username)
	if existing != nil {
		RespondError(c, http.StatusConflict, ErrConflict)
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	user := &domain.User{
		ID:           generateID("usr_", 12),
		Username:     req.Username,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		Status:       domain.UserActive,
		CreatedAt:    time.Now(),
	}
	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusCreated, gin.H{
		"user_id":      user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
	})
}

type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) Login(c *gin.Context) {
	var req UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	user, err := h.userRepo.GetByUsername(c.Request.Context(), req.Username)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	now := time.Now()
	if user.Status == domain.UserLocked || (user.LockedUntil != nil && user.LockedUntil.After(now)) {
		RespondErrorMsg(c, http.StatusForbidden, "ACCOUNT_LOCKED", "Account is temporarily locked")
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, req.Password) {
		user.FailedAttempts++
		if user.FailedAttempts >= 5 {
			lockUntil := now.Add(30 * time.Minute)
			h.userRepo.UpdateFailedAttempts(c.Request.Context(), user.ID, user.FailedAttempts, &lockUntil)
		} else {
			h.userRepo.UpdateFailedAttempts(c.Request.Context(), user.ID, user.FailedAttempts, nil)
		}
		RespondError(c, http.StatusUnauthorized, ErrUnauthorized)
		return
	}

	h.userRepo.ResetFailedAttempts(c.Request.Context(), user.ID)

	tenants, _ := h.memberRepo.ListByUser(c.Request.Context(), user.ID)

	var token string
	if len(tenants) == 1 {
		token, _ = auth.GenerateScopedJWT(user.ID, tenants[0].TenantID, string(tenants[0].Role))
	} else {
		token, _ = auth.GenerateUserJWT(user.ID)
	}

	RespondData(c, http.StatusOK, gin.H{
		"user_id":      user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"tenants":      tenants,
		"token":        token,
	})
}

func (h *UserHandler) ListTenants(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenants, err := h.memberRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"tenants": tenants})
}

type SelectTenantRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
}

func (h *UserHandler) SelectTenant(c *gin.Context) {
	var req SelectTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	userID := middleware.GetUserID(c)
	member, err := h.memberRepo.GetByTenantAndUser(c.Request.Context(), req.TenantID, userID)
	if err != nil {
		RespondErrorMsg(c, http.StatusForbidden, "FORBIDDEN", "Not a member of this tenant")
		return
	}

	tenant, err := h.tenantRepo.GetByID(c.Request.Context(), req.TenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	token, err := auth.GenerateScopedJWT(userID, req.TenantID, string(member.Role))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusOK, gin.H{
		"token": token,
		"tenant": gin.H{
			"id":   tenant.ID,
			"name": tenant.Name,
			"role": member.Role,
		},
	})
}

type JoinTenantRequest struct {
	Code string `json:"code" binding:"required"`
}

func (h *UserHandler) JoinTenant(c *gin.Context) {
	var req JoinTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	userID := middleware.GetUserID(c)

	inv, err := h.invitationRepo.GetByCode(c.Request.Context(), req.Code)
	if err != nil {
		RespondErrorMsg(c, http.StatusNotFound, "NOT_FOUND", "Invalid invitation code")
		return
	}

	if inv.ExpiresAt.Before(time.Now()) {
		RespondErrorMsg(c, http.StatusBadRequest, "EXPIRED", "Invitation has expired")
		return
	}
	if inv.MaxUses > 0 && inv.UsedCount >= inv.MaxUses {
		RespondErrorMsg(c, http.StatusBadRequest, "EXHAUSTED", "Invitation has reached max uses")
		return
	}

	existing, _ := h.memberRepo.GetByTenantAndUser(c.Request.Context(), inv.TenantID, userID)
	if existing != nil {
		RespondErrorMsg(c, http.StatusConflict, "CONFLICT", "Already a member of this tenant")
		return
	}

	member := &domain.TenantMember{
		ID:       generateID("mbr_", 12),
		TenantID: inv.TenantID,
		UserID:   userID,
		Role:     inv.Role,
		JoinedAt: time.Now(),
	}
	if err := h.memberRepo.Create(c.Request.Context(), member); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	h.invitationRepo.IncrementUsedCount(c.Request.Context(), inv.ID)

	tenant, _ := h.tenantRepo.GetByID(c.Request.Context(), inv.TenantID)
	token, _ := auth.GenerateScopedJWT(userID, inv.TenantID, string(inv.Role))

	tenantName := ""
	if tenant != nil {
		tenantName = tenant.Name
	}

	RespondData(c, http.StatusOK, gin.H{
		"tenant": gin.H{
			"id":   inv.TenantID,
			"name": tenantName,
			"role": inv.Role,
		},
		"token": token,
	})
}
