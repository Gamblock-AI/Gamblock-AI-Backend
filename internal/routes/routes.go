package routes

import (
	"time"

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
	v1.POST("/auth/email-verification/confirm", mid.RateLimitMiddleware("10-M"), h.ConfirmEmailVerification)
	v1.POST("/auth/email-verification/resend", mid.AuthRequired(), mid.RateLimitMiddleware("3-M"), h.ResendEmailVerification)
	v1.POST("/auth/phone-verification/start", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RateLimitMiddleware("3-M"), h.BeginPhoneVerification)
	v1.POST("/auth/phone-verification/confirm", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RateLimitMiddleware("8-M"), h.ConfirmPhoneVerification)
	v1.GET("/operator/invitations/:token", mid.RateLimitMiddleware("20-M"), h.VerifyOperatorInvitation)
	v1.POST("/operator/invitations/:token/accept", mid.RateLimitMiddleware("5-M"), h.AcceptOperatorInvitation)
	v1.GET("/public/site-social-links", h.PublicSiteSocialLinks)

	// Devices
	v1.POST("/devices", mid.AuthRequired(), h.CreateDevice)
	v1.PATCH("/devices/:device_id", mid.AuthRequired(), h.UpdateDevice)
	v1.POST("/devices/:device_id/heartbeat", mid.AuthRequired(), h.DeviceHeartbeat)

	// Accountability groups. Role gates protect data and actions server-side;
	// website navigation visibility is only a presentation concern.
	accountability := v1.Group("/accountability")
	accountability.Use(mid.AuthRequired(), mid.RequireRoles("user", "partner"))
	{
		accountability.GET("/workspace", h.AccountabilityWorkspace)
		accountability.POST("/groups", mid.RequireRoles("partner"), h.CreateAccountabilityGroup)
		accountability.POST("/groups/preview", mid.RequireRoles("user"), mid.RateLimitMiddleware("12-M"), h.PreviewAccountabilityGroup)
		accountability.POST("/groups/join", mid.RequireRoles("user"), mid.RateLimitMiddleware("6-M"), h.JoinAccountabilityGroup)
		accountability.POST("/groups/:group_id/rotate-code", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.RotateAccountabilityGroupCode)
		accountability.POST("/groups/:group_id/archive", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.ArchiveAccountabilityGroup)
		accountability.PATCH("/memberships/:membership_id/sharing", mid.RequireRoles("user"), h.UpdateAccountabilitySharing)
		accountability.POST("/memberships/:membership_id/leave", mid.RequireRoles("user"), h.RequestAccountabilityLeave)
		accountability.POST("/exit-requests/:request_id/cancel", mid.RequireRoles("user"), h.CancelAccountabilityLeave)
		accountability.POST("/memberships/:membership_id/remove", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.RemoveAccountabilityMember)
		accountability.POST("/exit-requests/:request_id/resolve", mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.ResolveAccountabilityLeave)
		accountability.POST("/contact-requests", mid.RequireRoles("user"), h.CreatePartnerContactRequest)
		accountability.POST("/contact-requests/:request_id/transition", h.TransitionPartnerContactRequest)
	}

	// Approval Requests
	v1.GET("/approval-requests", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.GetApprovalRequests)
	v1.POST("/approval-requests", mid.AuthRequired(), mid.RequireRoles("user"), h.CreateApprovalRequest)
	v1.POST("/approval-requests/:id/cancel", mid.AuthRequired(), mid.RequireRoles("user"), h.CancelApprovalRequest)
	v1.POST("/approval-requests/:id/approve", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.ApproveApprovalRequest)
	v1.POST("/approval-requests/:id/deny", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RequireRecentAuth(15*time.Minute), h.DenyApprovalRequest)
	v1.POST("/approval-requests/:id/apply", mid.AuthRequired(), mid.RequireRoles("user"), h.ApplyApprovalRequest)

	// Quick Approval (no auth)
	v1.GET("/approval-requests/verify/:token", h.VerifyApprovalToken)
	v1.POST("/approval-requests/:id/resolve-by-token", h.ResolveApprovalByToken)

	// Daily Missions
	v1.GET("/missions/today", mid.AuthRequired(), mid.RequireRoles("user"), h.GetTodayMission)
	v1.PATCH("/missions", mid.AuthRequired(), mid.RequireRoles("user"), h.UpdateMission)
	v1.POST("/missions/claim", mid.AuthRequired(), mid.RequireRoles("user"), h.ClaimMission)
	v1.POST("/missions/adjust", mid.AuthRequired(), mid.RequireRoles("user"), h.AdjustMission)

	// Reflections / Psychoeducation
	v1.GET("/reflections", mid.AuthRequired(), mid.RequireRoles("user"), h.GetReflections)
	v1.POST("/reflections", mid.AuthRequired(), mid.RequireRoles("user"), h.CreateReflection)
	v1.PATCH("/reflections/:id", mid.AuthRequired(), mid.RequireRoles("user"), h.UpdateReflection)
	v1.GET("/psychoeducation/modules", mid.AuthRequired(), h.GetModules)
	v1.GET("/psychoeducation/modules/:slug", mid.AuthRequired(), h.GetModuleDetail)
	v1.PUT("/psychoeducation/modules/:id/revisions/:revision/progress", mid.AuthRequired(), h.UpdateEducationProgress)
	v1.POST("/psychoeducation/modules/:id/revisions/:revision/checks/:check_id/answer", mid.AuthRequired(), h.AnswerEducationCheck)
	v1.GET("/education/media/:id", h.PublishedEducationMedia)

	// Recovery (Intentions & Check-ins)
	v1.GET("/intentions", mid.AuthRequired(), mid.RequireRoles("user"), h.GetIntention)
	v1.POST("/intentions", mid.AuthRequired(), mid.RequireRoles("user"), h.SaveIntention)
	v1.GET("/check-ins", mid.AuthRequired(), mid.RequireRoles("user"), h.GetCheckIns)
	v1.POST("/check-ins", mid.AuthRequired(), mid.RequireRoles("user"), h.CreateCheckIn)
	v1.GET("/recovery-records", mid.AuthRequired(), mid.RequireRoles("user"), h.GetRecoveryRecords)
	v1.PUT("/recovery-records", mid.AuthRequired(), mid.RequireRoles("user"), h.SaveRecoveryRecord)
	v1.GET("/recovery-practices", mid.AuthRequired(), mid.RequireRoles("user"), h.GetRecoveryPractices)
	v1.POST("/recovery-practices", mid.AuthRequired(), mid.RequireRoles("user"), h.CreateRecoveryPractice)
	v1.GET("/recovery-space", mid.AuthRequired(), mid.RequireRoles("user"), h.GetRecoverySpace)
	v1.PATCH("/recovery-space", mid.AuthRequired(), mid.RequireRoles("user"), h.UpdateRecoverySpace)
	v1.GET("/weekly-reviews/current", mid.AuthRequired(), mid.RequireRoles("user"), h.GetCurrentWeeklyReview)
	v1.PUT("/weekly-reviews/current", mid.AuthRequired(), mid.RequireRoles("user"), h.SaveCurrentWeeklyReview)

	// Support cases
	v1.GET("/support-cases", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.GetSupportCases)
	v1.POST("/support-cases", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.CreateSupportCase)
	v1.GET("/support-cases/:id", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.GetSupportCaseDetail)
	v1.POST("/support-cases/:id/messages", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.ReplySupportCase)
	v1.POST("/support-cases/:id/transition", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.TransitionSupportCase)

	// Data requests
	v1.GET("/data-requests", mid.AuthRequired(), h.GetDataRequests)
	v1.POST("/data-requests", mid.AuthRequired(), h.CreateDataRequest)
	v1.GET("/data-requests/:id/download", mid.AuthRequired(), mid.RequireRecentAuth(15*time.Minute), h.DownloadDataExport)
	v1.POST("/data-requests/confirm-delete", mid.AuthRequired(), mid.RequireRecentAuth(15*time.Minute), h.ConfirmAccountDeletion)

	// Emergency recovery request/status and single-use unlock.
	v1.GET("/emergency-key-requests/current", mid.AuthRequired(), h.CurrentEmergencyKeyRequest)
	v1.POST("/emergency-key-requests", mid.AuthRequired(), h.RequestEmergencyKey)
	v1.POST("/devices/unlock", h.EmergencyUnlock)

	// Client Dashboard / Protection Info
	v1.GET("/me", mid.AuthRequired(), h.GetProfile)
	v1.PATCH("/me", mid.AuthRequired(), h.UpdateProfile)
	v1.POST("/me/avatar", mid.AuthRequired(), h.UploadAvatar)
	v1.DELETE("/me/avatar", mid.AuthRequired(), h.DeleteAvatar)
	v1.PATCH("/me/password", mid.AuthRequired(), h.UpdatePassword)
	v1.GET("/users/:id/avatar", mid.AuthRequired(), h.UserAvatar)
	v1.GET("/client/dashboard-summary", mid.AuthRequired(), h.ClientDashboardSummary)
	v1.GET("/client/protection-status", mid.AuthRequired(), h.ClientProtectionStatus)
	v1.GET("/client/progress", mid.AuthRequired(), mid.RequireRoles("user"), h.ClientProgressSnapshot)
	v1.POST("/client/aggregate-events", mid.AuthRequired(), h.RecordAggregateEvent)
	v1.GET("/client/protection-analytics", mid.AuthRequired(), h.ProtectionAnalytics)

	// Portal Overview
	v1.GET("/portal/overview", mid.AuthRequired(), mid.RequireRoles("platform_admin"), h.PortalOverview)

	// Admin Control Portal
	admin := v1.Group("/admin")
	admin.Use(mid.AuthRequired())
	{
		admin.GET("/overview", mid.RequireRoles("platform_admin", "content_admin", "model_release_operator", "support_operator"), h.AdminOverview)
		admin.GET("/content/modules", mid.RequireRoles("content_admin"), h.AdminModules)
		admin.POST("/content/modules", mid.RequireRoles("content_admin"), h.CreateAdminModule)
		admin.GET("/content/modules/:id", mid.RequireRoles("content_admin"), h.AdminModuleDetail)
		admin.PUT("/content/modules/:id", mid.RequireRoles("content_admin"), h.UpdateAdminModule)
		admin.POST("/content/modules/:id/submit-review", mid.RequireRoles("content_admin"), h.SubmitAdminModuleReview)
		admin.POST("/content/modules/:id/publish", mid.RequireRoles("content_admin"), h.PublishAdminModule)
		admin.POST("/content/modules/:id/archive", mid.RequireRoles("content_admin"), h.ArchiveAdminModule)
		admin.GET("/content/modules/:id/revisions", mid.RequireRoles("content_admin"), h.AdminModuleRevisions)
		admin.POST("/content/modules/:id/revisions/:revision_id/rollback", mid.RequireRoles("content_admin"), mid.RequireRecentAuth(15*time.Minute), h.RollbackAdminModule)
		admin.POST("/content/media", mid.RequireRoles("content_admin"), h.UploadAdminEducationMedia)
		admin.POST("/content/media/external", mid.RequireRoles("content_admin"), h.RegisterAdminExternalMedia)
		admin.GET("/content/media/:id", mid.RequireRoles("content_admin"), h.AdminEducationMedia)
		admin.GET("/model-releases", mid.RequireRoles("model_release_operator"), h.AdminModelReleases)
		admin.GET("/releases", mid.RequireRoles("model_release_operator"), h.AdminReleases)
		admin.POST("/release-artifacts", mid.RequireRoles("model_release_operator"), mid.RequireRecentAuth(15*time.Minute), h.UploadAdminReleaseArtifact)
		admin.POST("/releases/rollouts", mid.RequireRoles("model_release_operator"), mid.RequireRecentAuth(15*time.Minute), h.CreateAdminRollout)
		admin.POST("/releases/rollouts/:id/transition", mid.RequireRoles("model_release_operator"), mid.RequireRecentAuth(15*time.Minute), h.TransitionAdminRollout)
		admin.GET("/support-cases", mid.RequireRoles("support_operator"), h.AdminSupportCases)
		admin.GET("/support-cases/:id", mid.RequireRoles("support_operator"), h.GetSupportCaseDetail)
		admin.POST("/support-cases/:id/claim", mid.RequireRoles("support_operator"), h.ClaimAdminSupportCase)
		admin.POST("/support-cases/:id/release", mid.RequireRoles("support_operator"), h.ReleaseAdminSupportCase)
		admin.POST("/support-cases/:id/messages", mid.RequireRoles("support_operator"), h.ReplySupportCase)
		admin.POST("/support-cases/:id/transition", mid.RequireRoles("support_operator"), h.TransitionSupportCase)
		admin.GET("/data-requests", mid.RequireRoles("support_operator"), h.AdminDataRequests)
		admin.POST("/data-requests/:id/retry", mid.RequireRoles("support_operator"), h.RetryAdminDataRequest)
		admin.POST("/data-requests/:id/reject", mid.RequireRoles("support_operator"), h.RejectAdminDataRequest)
		admin.GET("/site-social-links", mid.RequireRoles("platform_admin"), h.AdminSiteSocialLinks)
		admin.PUT("/site-social-links", mid.RequireRoles("platform_admin"), mid.RequireRecentAuth(15*time.Minute), h.ReplaceAdminSiteSocialLinks)
		admin.GET("/audit-events", mid.RequireRoles("platform_admin"), h.AdminAuditEvents)
		admin.GET("/operators", mid.RequireRoles("platform_admin"), h.AdminOperators)
		admin.POST("/operators/invitations", mid.RequireRoles("platform_admin"), mid.RequireRecentAuth(15*time.Minute), h.InviteAdminOperator)
		admin.POST("/operators/invitations/:id/revoke", mid.RequireRoles("platform_admin"), mid.RequireRecentAuth(15*time.Minute), h.RevokeAdminOperatorInvitation)
		admin.PATCH("/operators/:id", mid.RequireRoles("platform_admin"), mid.RequireRecentAuth(15*time.Minute), h.UpdateAdminOperator)
		admin.GET("/emergency-key-requests", mid.RequireRoles("platform_admin"), h.PendingEmergencyKeyRequests)
		admin.POST("/emergency-key-requests/:id/review", mid.RequireRoles("platform_admin"), mid.RequireRecentAuth(15*time.Minute), h.ReviewEmergencyKeyRequest)
		admin.POST("/emergency-key-requests/:id/approve", mid.RequireRoles("platform_admin"), mid.RequireRecentAuth(15*time.Minute), h.ApproveEmergencyKeyRequest)
	}

	// Releases Creation
	releasesGroup := v1.Group("/releases")
	releasesGroup.Use(mid.AuthRequired())
	{
		releasesGroup.POST("/model", mid.RequireRoles("model_release_operator"), h.CreateModelRelease)
		releasesGroup.POST("/ruleset", mid.RequireRoles("model_release_operator"), h.CreateRulesetRelease)
		releasesGroup.POST("/network-rulesets", mid.RequireRoles("model_release_operator"), h.CreateNetworkRulesetRelease)
	}

	// Downloads
	v1.GET("/releases/model/:version/download", h.DownloadModelRelease)
	v1.GET("/releases/ruleset/:version/download", h.DownloadRulesetRelease)
	v1.GET("/releases/network-rulesets/:version/download", h.DownloadNetworkRulesetRelease)

	v1.GET("/releases/model/latest", h.LatestModelRelease)
	v1.GET("/releases/ruleset/latest", h.LatestRulesetRelease)
	v1.GET("/releases/network-rulesets/latest", h.LatestNetworkRulesetRelease)
}
