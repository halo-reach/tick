package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tickplatform/tick/internal/api/middleware"
	"github.com/tickplatform/tick/internal/credential"
	"github.com/tickplatform/tick/internal/repo"
	"github.com/tickplatform/tick/internal/scheduler"
)

func NewRouter(
	tenantRepo *repo.TenantRepo,
	keyRepo *repo.ApiKeyRepo,
	secretRepo *repo.SecretRepo,
	targetRepo *repo.TargetRepo,
	taskRepo *repo.TaskRepo,
	execRepo *repo.ExecutionRepo,
	auditRepo *repo.AuditRepo,
	variableRepo *repo.VariableRepo,
	sched *scheduler.Scheduler,
	credRepo *repo.CredentialRepo,
	credStore *credential.Store,
	credResolver *credential.Resolver,
	userRepo *repo.UserRepo,
	memberRepo *repo.MemberRepo,
	invitationRepo *repo.InvitationRepo,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	tenantH := NewTenantHandler(tenantRepo, keyRepo, secretRepo, memberRepo)
	targetH := NewTargetHandler(targetRepo)
	taskH := NewTaskHandler(taskRepo, targetRepo, tenantRepo, execRepo, secretRepo, credRepo, variableRepo, sched, credResolver)
	secretH := NewSecretHandler(secretRepo)
	credH := NewCredentialHandler(credStore, credRepo, auditRepo, credResolver)
	variableH := NewVariableHandler(variableRepo)
	userH := NewUserHandler(userRepo, memberRepo, tenantRepo, invitationRepo)
	memberH := NewMemberHandler(memberRepo, userRepo, invitationRepo)

	v1 := r.Group("/api/v1")
	{
		// Public (no auth)
		v1.POST("/auth/register", userH.Register)
		v1.POST("/auth/login", userH.Login)
		v1.POST("/auth/tenant-register", tenantH.Register)
		v1.POST("/auth/tenant-login", tenantH.Login)

		// User-level auth (no tenant needed)
		userAuth := v1.Group("", middleware.AuthRequired(keyRepo, tenantRepo))
		{
			userAuth.GET("/auth/tenants", userH.ListTenants)
			userAuth.POST("/auth/select-tenant", userH.SelectTenant)
			userAuth.POST("/tenants", tenantH.CreateTenantForUser)
			userAuth.POST("/tenants/join", userH.JoinTenant)
		}

		// Tenant-scoped auth
		auth := v1.Group("", middleware.AuthRequired(keyRepo, tenantRepo), middleware.RequireTenant(), middleware.AuditLog(auditRepo))
		{
			auth.GET("/auth/me", tenantH.Me)
			auth.POST("/auth/change-password", tenantH.ChangePassword)
			auth.PUT("/tenant/name", tenantH.Rename)
			auth.GET("/auth/keys", tenantH.ListKeys)
			auth.POST("/auth/keys", tenantH.CreateKey)
			auth.DELETE("/auth/keys/:id", tenantH.RevokeKey)

			auth.GET("/quota", tenantH.Quota)
			auth.GET("/status", tenantH.Status)

			auth.POST("/secrets", secretH.Create)
			auth.GET("/secrets", secretH.List)
			auth.DELETE("/secrets/:id", secretH.Revoke)

			auth.POST("/targets", targetH.Create)
			auth.GET("/targets", targetH.List)
			auth.GET("/targets/:id", targetH.Get)
			auth.PUT("/targets/:id", targetH.Update)
			auth.DELETE("/targets/:id", targetH.Delete)

			auth.POST("/tasks", taskH.Create)
			auth.GET("/tasks", taskH.List)
			auth.GET("/tasks/:id", taskH.Get)
			auth.PUT("/tasks/:id", taskH.Update)
			auth.DELETE("/tasks/:id", taskH.Delete)
			auth.POST("/tasks/:id/pause", taskH.Pause)
			auth.POST("/tasks/:id/resume", taskH.Resume)
			auth.POST("/tasks/:id/trigger", taskH.Trigger)
			auth.GET("/tasks/:id/history", taskH.History)

			auth.POST("/credentials", credH.Create)
			auth.GET("/credentials", credH.List)
			auth.GET("/credentials/:id", credH.Get)
			auth.GET("/credentials/:id/config", credH.GetDecrypted)
			auth.PUT("/credentials/:id", credH.Update)
			auth.PATCH("/credentials/:id/status", credH.PatchStatus)
			auth.DELETE("/credentials/:id", credH.Delete)
			auth.GET("/credentials/:id/test", credH.Test)

			auth.POST("/variables", variableH.Create)
			auth.GET("/variables", variableH.List)
			auth.PUT("/variables/:id", variableH.Update)
			auth.DELETE("/variables/:id", variableH.Delete)

			// Member management
			auth.GET("/members", memberH.ListMembers)
			auth.POST("/members", middleware.RequireRole("owner"), memberH.AddMember)
			auth.POST("/members/invite", middleware.RequireRole("owner"), memberH.CreateInvite)
			auth.DELETE("/members/:user_id", middleware.RequireRole("owner"), memberH.RemoveMember)
			auth.PATCH("/members/:user_id/role", middleware.RequireRole("owner"), memberH.ChangeRole)
			auth.GET("/users/search", middleware.RequireRole("owner"), memberH.SearchUsers)
			auth.GET("/invitations", middleware.RequireRole("owner"), memberH.ListInvitations)
			auth.DELETE("/invitations/:id", middleware.RequireRole("owner"), memberH.RevokeInvitation)
		}
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}
