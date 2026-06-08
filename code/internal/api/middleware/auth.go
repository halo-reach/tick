package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/auth"
	"github.com/tickplatform/tick/internal/repo"
)

const (
	ContextTenantID  = "tenant_id"
	ContextKeyPrefix = "key_prefix"
	ContextUserID    = "user_id"
	ContextRole      = "role"
)

func AuthRequired(keyRepo *repo.ApiKeyRepo, tenantRepo *repo.TenantRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Missing or invalid credentials"}})
			c.Abort()
			return
		}
		plaintext := strings.TrimPrefix(header, "Bearer ")

		// JWT tokens start with "eyJ"
		if strings.HasPrefix(plaintext, "eyJ") {
			claims, err := auth.ValidateJWTClaims(plaintext)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"}})
				c.Abort()
				return
			}

			if claims.UserID == "" {
				// Old format: sub=tenantID
				tenant, err := tenantRepo.GetByID(c.Request.Context(), claims.TenantID)
				if err != nil || tenant.Status != "active" {
					c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied"}})
					c.Abort()
					return
				}
				c.Set(ContextTenantID, claims.TenantID)
			} else if claims.TenantID != "" {
				// Scoped token: user + tenant
				tenant, err := tenantRepo.GetByID(c.Request.Context(), claims.TenantID)
				if err != nil || tenant.Status != "active" {
					c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied"}})
					c.Abort()
					return
				}
				c.Set(ContextUserID, claims.UserID)
				c.Set(ContextTenantID, claims.TenantID)
				c.Set(ContextRole, claims.Role)
			} else {
				// User-level token (no tenant)
				c.Set(ContextUserID, claims.UserID)
			}
			c.Next()
			return
		}

		// API Key auth
		hash := auth.HashKey(plaintext)
		key, err := keyRepo.FindByHash(c.Request.Context(), hash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Missing or invalid API Key"}})
			c.Abort()
			return
		}

		tenant, err := tenantRepo.GetByID(c.Request.Context(), key.TenantID)
		if err != nil || tenant.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied"}})
			c.Abort()
			return
		}

		c.Set(ContextTenantID, key.TenantID)
		c.Set(ContextKeyPrefix, key.KeyPrefix)
		c.Next()
	}
}

func GetTenantID(c *gin.Context) string {
	return c.GetString(ContextTenantID)
}

func GetKeyPrefix(c *gin.Context) string {
	return c.GetString(ContextKeyPrefix)
}

func GetUserID(c *gin.Context) string {
	return c.GetString(ContextUserID)
}

func GetRole(c *gin.Context) string {
	return c.GetString(ContextRole)
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString(ContextRole)
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Insufficient permissions"}})
		c.Abort()
	}
}

func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(ContextTenantID) == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "No tenant selected"}})
			c.Abort()
			return
		}
		c.Next()
	}
}
