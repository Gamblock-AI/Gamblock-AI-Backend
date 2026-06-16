package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/handler"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
)

func Register(r *gin.Engine, h *handler.Handler, mid *middleware.Middleware) {
	r.GET("/healthz", h.Health)
	r.GET("/readyz", h.Ready)

	v1 := r.Group("/v1")
	v1.Use(mid.AuthOptional())

	// Auth
	v1.POST("/auth/login", h.Login)
	v1.POST("/auth/register", h.Register)
	v1.POST("/auth/dev-login", h.DevLogin)
	v1.POST("/auth/google", h.GoogleLogin)
	v1.POST("/auth/refresh", h.Refresh)
	v1.POST("/auth/logout", h.Logout)

	// Devices
	v1.POST("/devices", mid.AuthRequired(), h.CreateDevice)
	v1.PATCH("/devices/:device_id", mid.AuthRequired(), h.UpdateDevice)
	v1.POST("/devices/:device_id/heartbeat", mid.AuthRequired(), h.DeviceHeartbeat)

	// Partners / Accountability
	v1.GET("/partners", mid.AuthRequired(), h.GetPartners)
	v1.POST("/partners/invitations", mid.AuthRequired(), h.CreatePartnerInvitation)
	v1.POST("/partners/invitations/:token/accept", mid.AuthRequired(), h.AcceptPartnerInvitation)
	v1.POST("/partners/:partner_link_id/revoke", mid.AuthRequired(), h.RevokePartner)

	// Approval Requests
	v1.GET("/approval-requests", mid.AuthRequired(), h.GetApprovalRequests)
	v1.POST("/approval-requests", mid.AuthRequired(), h.CreateApprovalRequest)
	v1.POST("/approval-requests/:id/cancel", mid.AuthRequired(), h.CancelApprovalRequest)
	v1.POST("/approval-requests/:id/approve", mid.AuthRequired(), mid.RequireRoles("partner", "platform_admin"), h.ApproveApprovalRequest)
	v1.POST("/approval-requests/:id/deny", mid.AuthRequired(), mid.RequireRoles("partner", "platform_admin"), h.DenyApprovalRequest)

	// Quick Approval (no auth)
	v1.GET("/approval-requests/verify/:token", h.VerifyApprovalToken)
	v1.POST("/approval-requests/:id/resolve-by-token", h.ResolveApprovalByToken)

	// Organizations
	orgs := v1.Group("/organizations")
	orgs.POST("", mid.AuthRequired(), mid.RequireRoles("user", "partner", "platform_admin"), h.CreateOrganization)
	orgs.GET("/mine", mid.AuthRequired(), h.GetCurrentUserOrganization)
	orgs.GET("/:id", mid.AuthRequired(), h.GetOrganization)
	orgs.POST("/join", mid.AuthRequired(), h.JoinOrganization)
	orgs.GET("/:id/members", mid.AuthRequired(), h.ListOrganizationMembers)
	orgs.GET("/:id/analytics", mid.AuthRequired(), h.GetOrganizationAnalytics)
	orgs.DELETE("/:id/members/:user_id", mid.AuthRequired(), mid.RequireRoles("partner", "platform_admin"), h.RemoveOrganizationMember)

	// Daily Missions
	v1.GET("/missions/today", mid.AuthRequired(), h.GetTodayMission)
	v1.PATCH("/missions", mid.AuthRequired(), h.UpdateMission)

	// Reflections / Psychoeducation
	v1.GET("/reflections", mid.AuthRequired(), h.GetReflections)
	v1.POST("/reflections", mid.AuthRequired(), h.CreateReflection)
	v1.GET("/psychoeducation/modules", mid.AuthRequired(), h.GetModules)
	v1.GET("/psychoeducation/modules/:slug", mid.AuthRequired(), h.GetModuleDetail)

	// Support cases
	v1.GET("/support-cases", mid.AuthRequired(), h.GetSupportCases)
	v1.POST("/support-cases", mid.AuthRequired(), h.CreateSupportCase)

	// Data requests
	v1.GET("/data-requests", mid.AuthRequired(), h.GetDataRequests)
	v1.POST("/data-requests", mid.AuthRequired(), h.CreateDataRequest)

	// Emergency Unlock (no auth)
	v1.POST("/devices/unlock", h.EmergencyUnlock)

	// Client Dashboard / Protection Info
	v1.GET("/client/dashboard-summary", mid.AuthRequired(), h.ClientDashboardSummary)
	v1.GET("/client/protection-status", mid.AuthRequired(), h.ClientProtectionStatus)
	v1.GET("/client/progress", mid.AuthRequired(), h.ClientProgressSnapshot)

	// Portal Overview
	v1.GET("/portal/overview", mid.AuthRequired(), h.PortalOverview)

	// Admin Control Portal
	admin := v1.Group("/admin")
	admin.Use(mid.AuthRequired(), mid.RequireRoles("platform_admin", "support_operator", "model_release_operator", "content_admin"))
	{
		admin.GET("/content/modules", h.AdminModules)
		admin.GET("/model-releases", h.AdminModelReleases)
		admin.GET("/support-cases", h.AdminSupportCases)
		admin.POST("/emergency-key", h.GenerateEmergencyKey)
	}

	// Releases Creation
	releasesGroup := v1.Group("/releases")
	releasesGroup.Use(mid.AuthRequired())
	{
		releasesGroup.POST("/model", mid.RequireRoles("model_release_operator", "platform_admin"), h.CreateModelRelease)
		releasesGroup.POST("/ruleset", mid.RequireRoles("model_release_operator", "platform_admin"), h.CreateRulesetRelease)
		releasesGroup.POST("/network-rulesets", mid.RequireRoles("model_release_operator", "platform_admin"), h.CreateNetworkRulesetRelease)
	}

	// Downloads
	v1.GET("/releases/model/:version/download", h.DownloadModelRelease)
	v1.GET("/releases/ruleset/:version/download", h.DownloadRulesetRelease)
	v1.GET("/releases/network-rulesets/:version/download", h.DownloadNetworkRulesetRelease)

	v1.GET("/releases/model/latest", h.LatestModelRelease)
	v1.GET("/releases/ruleset/latest", h.LatestRulesetRelease)
	v1.GET("/releases/network-rulesets/latest", h.LatestNetworkRulesetRelease)
}
