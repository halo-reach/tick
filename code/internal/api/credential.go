package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/credential"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

type CredentialHandler struct {
	store    *credential.Store
	credRepo *repo.CredentialRepo
	auditRepo *repo.AuditRepo
	resolver *credential.Resolver
}

func NewCredentialHandler(store *credential.Store, credRepo *repo.CredentialRepo, auditRepo *repo.AuditRepo, resolver *credential.Resolver) *CredentialHandler {
	return &CredentialHandler{store: store, credRepo: credRepo, auditRepo: auditRepo, resolver: resolver}
}

var validCredentialTypes = map[string]bool{
	"bearer": true, "basic": true, "oauth2_cc": true,
	"dynamic": true, "hmac": true, "custom_header": true,
}

var credentialCodeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,62}[a-zA-Z0-9]$`)

func validateCredentialConfig(credType string, config json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(config, &m); err != nil {
		return "config 必须是有效的 JSON 对象"
	}
	hasStr := func(key string) bool {
		v, ok := m[key]
		if !ok {
			return false
		}
		var s string
		return json.Unmarshal(v, &s) == nil && s != ""
	}
	hasObj := func(key string) bool {
		v, ok := m[key]
		if !ok {
			return false
		}
		var obj map[string]json.RawMessage
		return json.Unmarshal(v, &obj) == nil
	}
	switch credType {
	case "bearer":
		if !hasStr("token") {
			return "bearer 类型需要 token 字段"
		}
	case "basic":
		if !hasStr("username") || !hasStr("password") {
			return "basic 类型需要 username 和 password 字段"
		}
	case "oauth2_cc":
		if !hasStr("token_url") || !hasStr("client_id") || !hasStr("client_secret") {
			return "oauth2_cc 类型需要 token_url、client_id、client_secret 字段"
		}
	case "dynamic":
		if !hasObj("token_request") {
			return "dynamic 类型需要 token_request 对象（包含 url 和 method）"
		}
		var tr map[string]json.RawMessage
		_ = json.Unmarshal(m["token_request"], &tr)
		checkStr := func(key string) bool {
			v, ok := tr[key]
			if !ok {
				return false
			}
			var s string
			return json.Unmarshal(v, &s) == nil && s != ""
		}
		if !checkStr("url") || !checkStr("method") {
			return "dynamic 类型的 token_request 需要 url 和 method 字段"
		}
	case "hmac":
		if !hasStr("secret") || !hasStr("algorithm") {
			return "hmac 类型需要 secret 和 algorithm 字段"
		}
	case "custom_header":
		if !hasObj("headers") {
			return "custom_header 类型需要 headers 对象"
		}
	}
	return ""
}

type CreateCredentialRequest struct {
	Name        string          `json:"name" binding:"required"`
	Code        string          `json:"code" binding:"required"`
	Type        string          `json:"type" binding:"required"`
	Config      json.RawMessage `json:"config" binding:"required"`
	TimeoutSecs int             `json:"timeout_secs"`
}

func (h *CredentialHandler) Create(c *gin.Context) {
	var req CreateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	if !validCredentialTypes[req.Type] {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "无效的凭证类型，支持：bearer, basic, oauth2_cc, dynamic, hmac, custom_header")
		return
	}
	if len(req.Name) < 1 || len(req.Name) > 128 {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "凭证名称长度须为 1-128 个字符")
		return
	}
	if !credentialCodeRegex.MatchString(req.Code) {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", "凭证编码只允许字母、数字、下划线和连字符，以字母开头，长度 2-64")
		return
	}
	if msg := validateCredentialConfig(req.Type, req.Config); msg != "" {
		RespondErrorMsg(c, http.StatusBadRequest, "VALIDATION_ERROR", msg)
		return
	}

	tenantID := middleware.GetTenantID(c)

	count, err := h.credRepo.CountByTenant(c.Request.Context(), tenantID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	if count >= 50 {
		RespondErrorMsg(c, http.StatusForbidden, "QUOTA_EXCEEDED", "凭证数量已达上限（50）")
		return
	}

	cred, err := h.store.Create(c.Request.Context(), tenantID, req.Name, req.Code, domain.CredentialType(req.Type), req.Config, req.TimeoutSecs)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}

	h.audit(c, "credential.create", cred.ID, nil)
	RespondData(c, http.StatusCreated, cred)
}

func (h *CredentialHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, total, err := h.store.List(c.Request.Context(), tenantID, status, limit, offset)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	RespondData(c, http.StatusOK, gin.H{"items": items, "total": total})
}

func (h *CredentialHandler) Get(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	cred, err := h.store.Get(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}
	RespondData(c, http.StatusOK, cred)
}

func (h *CredentialHandler) GetDecrypted(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	cred, config, err := h.store.GetDecrypted(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}
	RespondData(c, http.StatusOK, gin.H{
		"id":           cred.ID,
		"name":         cred.Name,
		"code":         cred.Code,
		"type":         cred.Type,
		"status":       cred.Status,
		"timeout_secs": cred.TimeoutSecs,
		"config":       json.RawMessage(config),
		"created_at":   cred.CreatedAt,
	})
}

type UpdateCredentialRequest struct {
	Name        *string         `json:"name"`
	Config      json.RawMessage `json:"config"`
	TimeoutSecs *int            `json:"timeout_secs"`
}

func (h *CredentialHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var req UpdateCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	cred, err := h.store.Update(c.Request.Context(), id, tenantID, req.Name, req.Config, req.TimeoutSecs)
	if err != nil {
		RespondError(c, http.StatusNotFound, ErrNotFound)
		return
	}
	h.audit(c, "credential.update", id, nil)
	RespondData(c, http.StatusOK, cred)
}

type PatchCredentialStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *CredentialHandler) PatchStatus(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var req PatchCredentialStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, ErrValidation)
		return
	}

	if err := h.credRepo.UpdateStatus(c.Request.Context(), id, tenantID, domain.CredentialStatus(req.Status)); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	h.audit(c, "credential.status_change", id, json.RawMessage(`{"status":"`+req.Status+`"}`))
	RespondData(c, http.StatusOK, gin.H{"updated": true})
}

func (h *CredentialHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	if err := h.store.Delete(c.Request.Context(), id, tenantID); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrInternal)
		return
	}
	h.audit(c, "credential.delete", id, nil)
	RespondData(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *CredentialHandler) Test(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	if h.resolver == nil {
		RespondErrorMsg(c, http.StatusInternalServerError, "INTERNAL_ERROR", "resolver not available")
		return
	}

	resolved, err := h.resolver.Resolve(c.Request.Context(), id, tenantID)
	if err != nil {
		RespondErrorMsg(c, http.StatusUnprocessableEntity, "CREDENTIAL_RESOLVE_FAILED", err.Error())
		return
	}

	injections := credential.BuildInjections(resolved)

	result := gin.H{
		"credential_name": resolved.Name,
		"credential_type": resolved.Type,
		"injections":      injections,
	}

	RespondData(c, http.StatusOK, result)
}

func (h *CredentialHandler) audit(c *gin.Context, action, resourceID string, payload json.RawMessage) {
	if h.auditRepo == nil {
		return
	}
	tenantID := middleware.GetTenantID(c)
	log := &domain.AuditLog{
		TenantID:     tenantID,
		Actor:        middleware.GetKeyPrefix(c),
		Action:       action,
		ResourceType: "credential",
		ResourceID:   resourceID,
		Payload:      payload,
		CreatedAt:    time.Now(),
	}
	_ = h.auditRepo.Create(c.Request.Context(), log)
}
