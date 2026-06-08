package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/domain"
	"github.com/tickplatform/tick/internal/repo"
)

func AuditLog(auditRepo *repo.AuditRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" {
			return
		}

		tenantID := GetTenantID(c)
		if tenantID == "" {
			return
		}

		log := &domain.AuditLog{
			TenantID:     tenantID,
			Actor:        GetKeyPrefix(c),
			Action:       c.Request.Method + " " + c.FullPath(),
			ResourceType: c.FullPath(),
			CreatedAt:    time.Now(),
		}
		_ = auditRepo.Create(c.Request.Context(), log)
	}
}
