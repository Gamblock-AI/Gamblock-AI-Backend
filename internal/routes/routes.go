package routes

import (
	"github.com/gamblock-ai/gamblock-ai-backend/internal/handler"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, h *handler.Handler, mid *middleware.Middleware) {
	r.GET("/healthz", h.Health)
	r.GET("/readyz", h.Ready)

	v1 := r.Group("/v1")
	v1.Use(mid.AuthOptional())

	// Auth
	v1.POST("/auth/login", mid.RateLimitMiddleware("5-M"), h.Login)
	v1.POST("/auth/register", mid.RateLimitMiddleware("5-M"), h.Register)
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
	v1.POST("/approval-requests/:id/approve", mid.AuthRequired(), h.ApproveApprovalRequest)
	v1.POST("/approval-requests/:id/deny", mid.AuthRequired(), h.DenyApprovalRequest)
	v1.POST("/approval-requests/:id/apply", mid.AuthRequired(), h.ApplyApprovalRequest)

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
	v1.POST("/missions/claim", mid.AuthRequired(), h.ClaimMission)

	// Reflections / Psychoeducation
	v1.GET("/reflections", mid.AuthRequired(), h.GetReflections)
	v1.POST("/reflections", mid.AuthRequired(), h.CreateReflection)
	v1.GET("/psychoeducation/modules", mid.AuthRequired(), h.GetModules)
	v1.GET("/psychoeducation/modules/:slug", mid.AuthRequired(), h.GetModuleDetail)
	v1.PUT("/psychoeducation/modules/:id/revisions/:revision/progress", mid.AuthRequired(), h.UpdateEducationProgress)
	v1.POST("/psychoeducation/modules/:id/revisions/:revision/checks/:check_id/answer", mid.AuthRequired(), h.AnswerEducationCheck)
	v1.GET("/education/media/:id", h.PublishedEducationMedia)

	// Recovery (Intentions & Check-ins)
	v1.GET("/intentions", mid.AuthRequired(), h.GetIntention)
	v1.POST("/intentions", mid.AuthRequired(), h.SaveIntention)
	v1.GET("/check-ins", mid.AuthRequired(), h.GetCheckIns)
	v1.POST("/check-ins", mid.AuthRequired(), h.CreateCheckIn)

	// Support cases
	v1.GET("/support-cases", mid.AuthRequired(), h.GetSupportCases)
	v1.POST("/support-cases", mid.AuthRequired(), h.CreateSupportCase)

	// Data requests
	v1.GET("/data-requests", mid.AuthRequired(), h.GetDataRequests)
	v1.POST("/data-requests", mid.AuthRequired(), h.CreateDataRequest)

	// Emergency recovery request/status and single-use unlock.
	v1.GET("/emergency-key-requests/current", mid.AuthRequired(), h.CurrentEmergencyKeyRequest)
	v1.POST("/emergency-key-requests", mid.AuthRequired(), h.RequestEmergencyKey)
	v1.POST("/devices/unlock", h.EmergencyUnlock)

	// Client Dashboard / Protection Info
	v1.GET("/me", mid.AuthRequired(), h.GetProfile)
	v1.PATCH("/me", mid.AuthRequired(), h.UpdateProfile)
	v1.PATCH("/me/password", mid.AuthRequired(), h.UpdatePassword)
	v1.GET("/client/dashboard-summary", mid.AuthRequired(), h.ClientDashboardSummary)
	v1.GET("/client/protection-status", mid.AuthRequired(), h.ClientProtectionStatus)
	v1.GET("/client/progress", mid.AuthRequired(), h.ClientProgressSnapshot)
	v1.POST("/client/aggregate-events", mid.AuthRequired(), h.RecordAggregateEvent)
	v1.GET("/client/protection-analytics", mid.AuthRequired(), h.ProtectionAnalytics)

	// Portal Overview
	v1.GET("/portal/overview", mid.AuthRequired(), mid.RequireRoles("platform_admin", "support_operator", "model_release_operator", "content_admin"), h.PortalOverview)

	// Admin Control Portal
	admin := v1.Group("/admin")
	admin.Use(mid.AuthRequired())
	{
		admin.GET("/content/modules", mid.RequireRoles("content_admin", "platform_admin"), h.AdminModules)
		admin.POST("/content/modules", mid.RequireRoles("content_admin", "platform_admin"), h.CreateAdminModule)
		admin.GET("/content/modules/:id", mid.RequireRoles("content_admin", "platform_admin"), h.AdminModuleDetail)
		admin.PUT("/content/modules/:id", mid.RequireRoles("content_admin", "platform_admin"), h.UpdateAdminModule)
		admin.POST("/content/modules/:id/submit-review", mid.RequireRoles("content_admin", "platform_admin"), h.SubmitAdminModuleReview)
		admin.POST("/content/modules/:id/publish", mid.RequireRoles("content_admin", "platform_admin"), h.PublishAdminModule)
		admin.POST("/content/modules/:id/archive", mid.RequireRoles("content_admin", "platform_admin"), h.ArchiveAdminModule)
		admin.POST("/content/media", mid.RequireRoles("content_admin", "platform_admin"), h.UploadAdminEducationMedia)
		admin.POST("/content/media/external", mid.RequireRoles("content_admin", "platform_admin"), h.RegisterAdminExternalMedia)
		admin.GET("/content/media/:id", mid.RequireRoles("content_admin", "platform_admin"), h.AdminEducationMedia)
		admin.GET("/model-releases", mid.RequireRoles("model_release_operator", "platform_admin"), h.AdminModelReleases)
		admin.GET("/support-cases", mid.RequireRoles("support_operator", "platform_admin"), h.AdminSupportCases)
		admin.GET("/emergency-key-requests", mid.RequireRoles("platform_admin"), h.PendingEmergencyKeyRequests)
		admin.POST("/emergency-key-requests/:id/review", mid.RequireRoles("platform_admin"), h.ReviewEmergencyKeyRequest)
		admin.POST("/emergency-key-requests/:id/approve", mid.RequireRoles("platform_admin"), h.ApproveEmergencyKeyRequest)
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
