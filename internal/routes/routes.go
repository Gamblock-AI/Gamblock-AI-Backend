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
	v1.POST("/auth/password-reset/request", mid.RateLimitMiddleware("3-M"), h.RequestPasswordReset)
	v1.POST("/auth/password-reset/confirm", mid.RateLimitMiddleware("8-M"), h.ConfirmPasswordReset)
	v1.POST("/auth/first-login/password", mid.RateLimitMiddleware("8-M"), h.CompleteInitialPasswordChange)
	v1.POST("/auth/refresh", h.Refresh)
	v1.POST("/auth/logout", h.Logout)
	v1.POST("/auth/email-verification/confirm", mid.RateLimitMiddleware("10-M"), h.ConfirmEmailVerification)
	v1.POST("/auth/email-verification/resend", mid.AuthRequired(), mid.RateLimitMiddleware("3-M"), h.ResendEmailVerification)
	v1.POST("/auth/phone-verification/start", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RateLimitMiddleware("3-M"), h.BeginPhoneVerification)
	v1.POST("/auth/phone-verification/confirm", mid.AuthRequired(), mid.RequireRoles("partner"), mid.RateLimitMiddleware("8-M"), h.ConfirmPhoneVerification)
	v1.GET("/operator/invitations/:token", h.RetiredOperatorInvitation)
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
	v1.POST("/me/google/link", mid.AuthRequired(), mid.RequireRoles("user", "partner"), h.LinkGoogle)
	v1.GET("/users/:id/avatar", mid.AuthRequired(), h.UserAvatar)
	v1.GET("/client/dashboard-summary", mid.AuthRequired(), h.ClientDashboardSummary)
	v1.GET("/client/protection-status", mid.AuthRequired(), h.ClientProtectionStatus)
	v1.GET("/client/progress", mid.AuthRequired(), mid.RequireRoles("user"), h.ClientProgressSnapshot)
	v1.POST("/client/aggregate-events", mid.AuthRequired(), h.RecordAggregateEvent)
	v1.GET("/client/protection-analytics", mid.AuthRequired(), h.ProtectionAnalytics)

	// Portal Overview
	v1.GET("/portal/overview", mid.AuthRequired(), mid.RequireRoles("admin"), mid.RequireVerifiedEmail(), h.PortalOverview)

	// Admin Control Portal
	admin := v1.Group("/admin")
	admin.Use(mid.AuthRequired(), mid.RequireRoles("admin"), mid.RequireVerifiedEmail())
	{
		admin.GET("/overview", h.AdminOverview)
		admin.GET("/content/modules", h.AdminModules)
		admin.POST("/content/modules", h.CreateAdminModule)
		admin.GET("/content/modules/:id", h.AdminModuleDetail)
		admin.PUT("/content/modules/:id", h.UpdateAdminModule)
		admin.POST("/content/modules/:id/submit-review", h.SubmitAdminModuleReview)
		admin.POST("/content/modules/:id/publish", h.PublishAdminModule)
		admin.POST("/content/modules/:id/archive", h.ArchiveAdminModule)
		admin.GET("/content/modules/:id/revisions", h.AdminModuleRevisions)
		admin.POST("/content/modules/:id/revisions/:revision_id/rollback", mid.RequireRecentAuth(15*time.Minute), h.RollbackAdminModule)
		admin.POST("/content/media", h.UploadAdminEducationMedia)
		admin.POST("/content/media/external", h.RegisterAdminExternalMedia)
		admin.GET("/content/media/:id", h.AdminEducationMedia)
		admin.GET("/model-releases", h.AdminModelReleases)
		admin.GET("/releases", h.AdminReleases)
		admin.POST("/release-artifacts", mid.RequireRecentAuth(15*time.Minute), h.UploadAdminReleaseArtifact)
		admin.POST("/releases/rollouts", mid.RequireRecentAuth(15*time.Minute), h.CreateAdminRollout)
		admin.POST("/releases/rollouts/:id/transition", mid.RequireRecentAuth(15*time.Minute), h.TransitionAdminRollout)
		admin.GET("/support-cases", h.AdminSupportCases)
		admin.GET("/support-cases/:id", h.GetAdminSupportCaseDetail)
		admin.POST("/support-cases/:id/claim", h.ClaimAdminSupportCase)
		admin.POST("/support-cases/:id/release", h.ReleaseAdminSupportCase)
		admin.POST("/support-cases/:id/messages", h.ReplyAdminSupportCase)
		admin.POST("/support-cases/:id/transition", h.TransitionAdminSupportCase)
		admin.GET("/data-requests", h.AdminDataRequests)
		admin.POST("/data-requests/:id/retry", h.RetryAdminDataRequest)
		admin.POST("/data-requests/:id/reject", h.RejectAdminDataRequest)
		admin.GET("/site-social-links", h.AdminSiteSocialLinks)
		admin.PUT("/site-social-links", mid.RequireRecentAuth(15*time.Minute), h.ReplaceAdminSiteSocialLinks)
		admin.GET("/audit-events", h.AdminAuditEvents)
		admin.GET("/accounts", h.AdminAccounts)
		admin.POST("/accounts", mid.RequireRecentAuth(15*time.Minute), h.CreateAdminAccount)
		admin.PATCH("/accounts/:id", mid.RequireRecentAuth(15*time.Minute), h.UpdateAdminAccount)
		admin.GET("/emergency-key-requests", h.PendingEmergencyKeyRequests)
		admin.POST("/emergency-key-requests/:id/review", mid.RequireRecentAuth(15*time.Minute), h.ReviewEmergencyKeyRequest)
		admin.POST("/emergency-key-requests/:id/approve", mid.RequireRecentAuth(15*time.Minute), h.ApproveEmergencyKeyRequest)
	}

	// Releases Creation
	releasesGroup := v1.Group("/releases")
	releasesGroup.Use(mid.AuthRequired(), mid.RequireRoles("admin"), mid.RequireVerifiedEmail())
	{
		releasesGroup.POST("/model", h.CreateModelRelease)
		releasesGroup.POST("/ruleset", h.CreateRulesetRelease)
		releasesGroup.POST("/network-rulesets", h.CreateNetworkRulesetRelease)
	}

	// Downloads
	v1.GET("/releases/model/:version/download", h.DownloadModelRelease)
	v1.GET("/releases/ruleset/:version/download", h.DownloadRulesetRelease)
	v1.GET("/releases/network-rulesets/:version/download", h.DownloadNetworkRulesetRelease)

	v1.GET("/releases/model/latest", h.LatestModelRelease)
	v1.GET("/releases/ruleset/latest", h.LatestRulesetRelease)
	v1.GET("/releases/network-rulesets/latest", h.LatestNetworkRulesetRelease)
}
