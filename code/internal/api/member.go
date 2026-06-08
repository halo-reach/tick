package api

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

type MemberHandler struct {
	memberRepo     *repo.MemberRepo
	userRepo       *repo.UserRepo
	invitationRepo *repo.InvitationRepo
}

func NewMemberHandler(memberRepo *repo.MemberRepo, userRepo *repo.UserRepo, invitationRepo *repo.InvitationRepo) *MemberHandler {
	return &MemberHandler{memberRepo: memberRepo, userRepo: userRepo, invitationRepo: invitationRepo}
}

const inviteCodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateInviteCode() string {
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		b[i] = inviteCodeChars[n.Int64()]
	}
	return string(b)
}

type CreateInviteRequest struct {
	Role          string `json:"role"`
	MaxUses       int    `json:"max_uses"`
	ExpiresInDays int    `json:"expires_in_days"`
}

func (h *MemberHandler) CreateInvite(c *gin.Context) {
	var req CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	role := domain.RoleMember
	if req.Role == "owner" {
		role = domain.RoleOwner
	}
	expiresInDays := req.ExpiresInDays
	if expiresInDays <= 0 {
		expiresInDays = 7
	}

	now := time.Now()
	inv := &domain.Invitation{
		ID:        generateID("inv_", 12),
		TenantID:  middleware.GetTenantID(c),
		Code:      generateInviteCode(),
		CreatedBy: middleware.GetUserID(c),
		Role:      role,
		MaxUses:   req.MaxUses,
		UsedCount: 0,
		ExpiresAt: now.AddDate(0, 0, expiresInDays),
		CreatedAt: now,
	}
	if err := h.invitationRepo.Create(c.Request.Context(), inv); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusCreated, gin.H{
		"id":         inv.ID,
		"code":       inv.Code,
		"role":       inv.Role,
		"max_uses":   inv.MaxUses,
		"expires_at": inv.ExpiresAt,
	})
}

func (h *MemberHandler) ListMembers(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	members, err := h.memberRepo.ListMembersWithUser(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"members": members})
}

func (h *MemberHandler) RemoveMember(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	targetUserID := c.Param("user_id")
	currentUserID := middleware.GetUserID(c)

	if targetUserID == currentUserID {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Cannot remove yourself")
		return
	}

	target, err := h.memberRepo.GetByTenantAndUser(c.Request.Context(), tenantID, targetUserID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}

	if target.Role == domain.RoleOwner {
		count, _ := h.memberRepo.CountOwners(c.Request.Context(), tenantID)
		if count <= 1 {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Cannot remove the last owner")
			return
		}
	}

	if err := h.memberRepo.Delete(c.Request.Context(), tenantID, targetUserID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	c.Status(http.StatusNoContent)
}

type ChangeRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

func (h *MemberHandler) ChangeRole(c *gin.Context) {
	var req ChangeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	tenantID := middleware.GetTenantID(c)
	targetUserID := c.Param("user_id")

	target, err := h.memberRepo.GetByTenantAndUser(c.Request.Context(), tenantID, targetUserID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}

	if target.Role == domain.RoleOwner && domain.MemberRole(req.Role) != domain.RoleOwner {
		count, _ := h.memberRepo.CountOwners(c.Request.Context(), tenantID)
		if count <= 1 {
			RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "Cannot demote the last owner")
			return
		}
	}

	if err := h.memberRepo.UpdateRole(c.Request.Context(), tenantID, targetUserID, domain.MemberRole(req.Role)); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusOK, gin.H{"user_id": targetUserID, "role": req.Role})
}

func (h *MemberHandler) ListInvitations(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	invitations, err := h.invitationRepo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"invitations": invitations})
}

func (h *MemberHandler) RevokeInvitation(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")
	if err := h.invitationRepo.Delete(c.Request.Context(), id, tenantID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *MemberHandler) SearchUsers(c *gin.Context) {
	q := c.Query("q")
	if len(q) < 1 {
		RespondData(c, http.StatusOK, gin.H{"users": []interface{}{}})
		return
	}
	tenantID := middleware.GetTenantID(c)
	users, err := h.userRepo.SearchExcludingTenant(c.Request.Context(), q, tenantID, 10)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if users == nil {
		users = []*repo.UserSearchResult{}
	}
	RespondData(c, http.StatusOK, gin.H{"users": users})
}

type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role"`
}

func (h *MemberHandler) AddMember(c *gin.Context) {
	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	tenantID := middleware.GetTenantID(c)

	_, err := h.memberRepo.GetByTenantAndUser(c.Request.Context(), tenantID, req.UserID)
	if err == nil {
		RespondErrorMsg(c, http.StatusConflict, "VALIDATION_ERROR", "用户已是该租户成员")
		return
	}

	targetUser, err := h.userRepo.GetByID(c.Request.Context(), req.UserID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}

	role := domain.RoleMember
	if req.Role == "owner" {
		role = domain.RoleOwner
	}

	member := &domain.TenantMember{
		ID:       generateID("tm_", 12),
		TenantID: tenantID,
		UserID:   req.UserID,
		Role:     role,
		JoinedAt: time.Now(),
	}
	if err := h.memberRepo.Create(c.Request.Context(), member); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	RespondData(c, http.StatusCreated, gin.H{
		"user_id":      req.UserID,
		"username":     targetUser.Username,
		"display_name": targetUser.DisplayName,
		"role":         role,
	})
}
