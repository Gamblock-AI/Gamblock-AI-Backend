package seedscale

import (
	"context"
	"fmt"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitygroup"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitymembership"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/aggregateevent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/approvalrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/contactverification"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/dailymission"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/datarequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/device"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/educationmedia"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/educationrevision"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/emergencykeyrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/institution"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/intention"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/interventionrecord"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningitem"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningprogress"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/learningrevision"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/membershipexitrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/notificationdelivery"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/operatorinvitation"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organization"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationinvite"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organizationmember"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnercontactrequest"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/partnerlink"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/psychoeducationmodule"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoverypracticesession"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoveryrecord"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/recoveryspace"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/reflection"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/reportrollup"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/sitesociallink"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportcase"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportmessage"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

type ScaleSeedOptions struct {
	BaseCount            int    // Target number of rows per table (typically 500-1200)
	JournalEncryptionKey string // 64-character hex AES-256 key for encrypted recovery/support payloads
}

type TableReport struct {
	TableName string
	Count     int
}

// SeedScaleDatabase populates all 49 database tables with 500 to 2,000 realistic records for local benchmarking.
func SeedScaleDatabase(ctx context.Context, client *ent.Client, opts ScaleSeedOptions) ([]TableReport, error) {
	if opts.BaseCount < 500 {
		opts.BaseCount = 600
	} else if opts.BaseCount > 2000 {
		opts.BaseCount = 2000
	}

	encKey := opts.JournalEncryptionKey
	if encKey == "" {
		encKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}
	encryptText := func(plain string) string {
		enc, err := crypto.Encrypt(plain, encKey)
		if err != nil {
			return plain
		}
		return enc
	}

	n := opts.BaseCount
	reports := make([]TableReport, 0, 49)
	now := time.Now().UTC()
	today := FormatDate(now)

	passwordHash, err := authn.HashPassword("password")
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Ensure core local accounts exist first
	if err := SeedLocalAccounts(ctx, client); err != nil {
		return nil, fmt.Errorf("seed core accounts: %w", err)
	}

	// 1. Users (n scale users + 4 core accounts)
	userIDs := []string{"usr_gading", "usr_dery", "usr_suci", "usr_nasywa"}
	userEmails := []string{"gading@gmail.com", "dery@gmail.com", "suci@gmail.com", "nasywa@gmail.com"}
	partnerIDs := []string{"usr_suci"}
	studentIDs := []string{"usr_gading", "usr_dery"}
	adminIDs := []string{"usr_nasywa"}

	userBuilders := make([]*ent.UserCreate, 0, n)
	for i := 0; i < n; i++ {
		uid := fmt.Sprintf("usr_scale_%04d", i+1)
		email := fmt.Sprintf("scale_user_%04d@test.local", i+1)
		userIDs = append(userIDs, uid)
		userEmails = append(userEmails, email)

		role := user.RoleUser
		if i%3 == 1 {
			role = user.RolePartner
			partnerIDs = append(partnerIDs, uid)
		} else if i%20 == 0 {
			role = user.RoleAdmin
			adminIDs = append(adminIDs, uid)
		} else {
			studentIDs = append(studentIDs, uid)
		}

		userBuilders = append(userBuilders, client.User.Create().
			SetID(uid).
			SetEmail(email).
			SetDisplayName(fmt.Sprintf("Scale User %04d", i+1)).
			SetRole(role).
			SetPasswordHash(passwordHash).
			SetEmailVerifiedAt(now).
			SetPhoneE164(fmt.Sprintf("+62812%08d", i+1)).
			SetPhoneVerifiedAt(now).
			SetAvatarURL("avatar/gading.webp").
			SetExperiencePoints(RandomInt(0, 500)))
	}
	if err := saveInBatches(ctx, 200, len(userBuilders), func(start, end int) error {
		_, err := client.User.CreateBulk(userBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed users: %w", err)
	}
	reports = append(reports, TableReport{"User", len(userIDs)})

	// 2. ContactVerification (n)
	cvBuilders := make([]*ent.ContactVerificationCreate, 0, n)
	for i := 0; i < n; i++ {
		kind := contactverification.KindEmail
		if i%3 == 1 {
			kind = contactverification.KindPhone
		} else if i%3 == 2 {
			kind = contactverification.KindPasswordReset
		}
		cvBuilders = append(cvBuilders, client.ContactVerification.Create().
			SetID(fmt.Sprintf("cver_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetKind(kind).
			SetDestination(userEmails[i%len(userEmails)]).
			SetTokenHash(Sha256Hash(fmt.Sprintf("cv_token_%04d", i+1))).
			SetAttemptCount(RandomInt(0, 3)).
			SetExpiresAt(now.Add(24*time.Hour)))
	}
	if err := saveInBatches(ctx, 200, len(cvBuilders), func(start, end int) error {
		_, err := client.ContactVerification.CreateBulk(cvBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed ContactVerification: %w", err)
	}
	reports = append(reports, TableReport{"ContactVerification", n})

	// 3. RefreshToken (n)
	rtBuilders := make([]*ent.RefreshTokenCreate, 0, n)
	for i := 0; i < n; i++ {
		rtBuilders = append(rtBuilders, client.RefreshToken.Create().
			SetID(fmt.Sprintf("rt_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetTokenHash(Sha256Hash(fmt.Sprintf("rt_token_%04d", i+1))).
			SetAuthTime(now.Add(-time.Duration(i)*time.Minute)).
			SetExpiresAt(now.Add(30*24*time.Hour)))
	}
	if err := saveInBatches(ctx, 200, len(rtBuilders), func(start, end int) error {
		_, err := client.RefreshToken.CreateBulk(rtBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed RefreshToken: %w", err)
	}
	reports = append(reports, TableReport{"RefreshToken", n})

	// 4. Device (len(userIDs))
	deviceIDs := make([]string, len(userIDs))
	devBuilders := make([]*ent.DeviceCreate, 0, len(userIDs))
	for i := 0; i < len(userIDs); i++ {
		did := fmt.Sprintf("dev_scale_%04d", i+1)
		deviceIDs[i] = did
		plat := device.PlatformAndroid
		if i%2 == 1 {
			plat = device.PlatformWindows
		}
		// Realistic distribution: 85% active, 8% degraded, 5% paused, 2% inactive
		pStatus := device.ProtectionStatusActive
		if i%50 == 49 {
			pStatus = device.ProtectionStatusInactive
		} else if i%20 == 19 {
			pStatus = device.ProtectionStatusPaused
		} else if i%12 == 11 {
			pStatus = device.ProtectionStatusDegraded
		}
		devBuilders = append(devBuilders, client.Device.Create().
			SetID(did).
			SetUserID(userIDs[i]).
			SetClientInstanceID(fmt.Sprintf("inst_%04d", i+1)).
			SetPlatform(plat).
			SetLabel(fmt.Sprintf("Device %04d", i+1)).
			SetAppVersion("1.0.0").
			SetOsVersion("Linux/Android").
			SetGrantKeyThumbprint(fmt.Sprintf("thumbprint_%04d", i+1)).
			SetProtectionStatus(pStatus).
			SetLastSeenAt(now))
	}
	if err := saveInBatches(ctx, 200, len(devBuilders), func(start, end int) error {
		_, err := client.Device.CreateBulk(devBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed Device: %w", err)
	}
	reports = append(reports, TableReport{"Device", len(deviceIDs)})

	// 5. PartnerLink (n)
	plinkIDs := make([]string, n)
	plBuilders := make([]*ent.PartnerLinkCreate, 0, n)
	for i := 0; i < n; i++ {
		pid := fmt.Sprintf("plink_scale_%04d", i+1)
		plinkIDs[i] = pid
		pUserID := partnerIDs[i%len(partnerIDs)]
		sUserID := studentIDs[i%len(studentIDs)]
		if i == 0 {
			pUserID = "usr_suci"
			sUserID = "usr_gading"
		} else if i == 1 {
			pUserID = "usr_suci"
			sUserID = "usr_dery"
		}
		plBuilders = append(plBuilders, client.PartnerLink.Create().
			SetID(pid).
			SetUserID(sUserID).
			SetPartnerUserID(pUserID).
			SetPartnerEmail(fmt.Sprintf("partner_%04d@test.local", i+1)).
			SetPartnerPhone(fmt.Sprintf("+62821%08d", i+1)).
			SetStatus(partnerlink.StatusActive).
			SetAcceptedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(plBuilders), func(start, end int) error {
		_, err := client.PartnerLink.CreateBulk(plBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed PartnerLink: %w", err)
	}
	reports = append(reports, TableReport{"PartnerLink", n})

	// 6. AccountabilityGroup (n)
	groupIDs := make([]string, n)
	grpBuilders := make([]*ent.AccountabilityGroupCreate, 0, n)
	for i := 0; i < n; i++ {
		gid := fmt.Sprintf("grp_scale_%04d", i+1)
		groupIDs[i] = gid
		owner := partnerIDs[i%len(partnerIDs)]
		gName := fmt.Sprintf("Group Focus %04d", i+1)
		if i == 0 {
			owner = "usr_suci"
			gName = "Kelompok Pemulihan Kampus A"
		} else if i == 1 {
			owner = "usr_suci"
			gName = "Kelompok Fokus Belajar B"
		} else if i == 2 {
			owner = "usr_suci"
			gName = "Kelompok Dukungan Sebaya C"
		}
		grpBuilders = append(grpBuilders, client.AccountabilityGroup.Create().
			SetID(gid).
			SetOwnerPartnerID(owner).
			SetName(gName).
			SetDescription("Akuntabilitas dan pemulihan bersama").
			SetJoinCodeHash(Sha256Hash(fmt.Sprintf("join_code_%04d", i+1))).
			SetJoinCodeHint("12**").
			SetJoinCodeEncrypted(encryptText(fmt.Sprintf("JOIN%04d", i+1))).
			SetStatus(accountabilitygroup.StatusActive).
			SetCodeRotatedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(grpBuilders), func(start, end int) error {
		_, err := client.AccountabilityGroup.CreateBulk(grpBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed AccountabilityGroup: %w", err)
	}
	reports = append(reports, TableReport{"AccountabilityGroup", n})

	// 7. AccountabilityMembership (n)
	membershipIDs := make([]string, n)
	memBuilders := make([]*ent.AccountabilityMembershipCreate, 0, n)

	suciAssignments := []struct {
		groupID   string
		studentID string
	}{
		{groupIDs[0], "usr_gading"},
		{groupIDs[0], "usr_scale_0001"},
		{groupIDs[0], "usr_scale_0002"},
		{groupIDs[0], "usr_scale_0003"},
		{groupIDs[1], "usr_dery"},
		{groupIDs[1], "usr_scale_0004"},
		{groupIDs[1], "usr_scale_0005"},
		{groupIDs[2], "usr_scale_0006"},
		{groupIDs[2], "usr_scale_0007"},
		{groupIDs[2], "usr_scale_0008"},
	}

	for i := 0; i < n; i++ {
		mid := fmt.Sprintf("mem_scale_%04d", i+1)
		membershipIDs[i] = mid

		var gID, sID string
		if i < len(suciAssignments) {
			gID = suciAssignments[i].groupID
			sID = suciAssignments[i].studentID
		} else {
			// Offset to guarantee unique (group_id, student_id)
			gIndex := (i % (len(groupIDs) - 3)) + 3
			sIndex := (i + 5) % len(studentIDs)
			gID = groupIDs[gIndex]
			sID = studentIDs[sIndex]
		}

		mStatus := accountabilitymembership.StatusActive
		if i%25 == 24 {
			mStatus = accountabilitymembership.StatusLeavePending
		} else if i%30 == 29 {
			mStatus = accountabilitymembership.StatusSupportReview
		}

		memBuilders = append(memBuilders, client.AccountabilityMembership.Create().
			SetID(mid).
			SetGroupID(gID).
			SetStudentID(sID).
			SetStatus(mStatus).
			SetShareProtectionHealth(true).
			SetShareProtectionActivity(true).
			SetShareRecoveryEngagement(true).
			SetShareEducationProgress(true).
			SetJoinedAt(now.Add(-14*24*time.Hour)))
	}
	if err := saveInBatches(ctx, 200, len(memBuilders), func(start, end int) error {
		_, err := client.AccountabilityMembership.CreateBulk(memBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed AccountabilityMembership: %w", err)
	}
	reports = append(reports, TableReport{"AccountabilityMembership", n})

	// 8. MembershipExitRequest (n)
	merBuilders := make([]*ent.MembershipExitRequestCreate, 0, n)
	for i := 0; i < n; i++ {
		mID := membershipIDs[i%len(membershipIDs)]
		reqBy := userIDs[i%len(userIDs)]
		exitStatus := membershipexitrequest.StatusPending
		if i == 0 {
			mID = membershipIDs[0] // usr_gading in Suci's group
			reqBy = "usr_gading"
			exitStatus = membershipexitrequest.StatusPending
		} else {
			switch i % 5 {
			case 0:
				exitStatus = membershipexitrequest.StatusPending
			case 1, 2:
				exitStatus = membershipexitrequest.StatusApproved
			case 3:
				exitStatus = membershipexitrequest.StatusDenied
			default:
				exitStatus = membershipexitrequest.StatusCancelled
			}
		}

		merBuilders = append(merBuilders, client.MembershipExitRequest.Create().
			SetID(fmt.Sprintf("exit_scale_%04d", i+1)).
			SetMembershipID(mID).
			SetRequestedBy(reqBy).
			SetKind(membershipexitrequest.KindNormal).
			SetStatus(exitStatus).
			SetReason("Selesai masa pendampingan mandiri"))
	}
	if err := saveInBatches(ctx, 200, len(merBuilders), func(start, end int) error {
		_, err := client.MembershipExitRequest.CreateBulk(merBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed MembershipExitRequest: %w", err)
	}
	reports = append(reports, TableReport{"MembershipExitRequest", n})

	// 9. PartnerContactRequest (n)
	pcrBuilders := make([]*ent.PartnerContactRequestCreate, 0, n)
	for i := 0; i < n; i++ {
		pID := partnerIDs[i%len(partnerIDs)]
		sID := studentIDs[i%len(studentIDs)]
		mID := membershipIDs[i%len(membershipIDs)]
		pcrStatus := partnercontactrequest.StatusPending

		if i == 0 {
			pID = "usr_suci"
			sID = "usr_gading"
			mID = membershipIDs[0]
			pcrStatus = partnercontactrequest.StatusPending
		} else if i == 1 {
			pID = "usr_suci"
			sID = "usr_dery"
			mID = membershipIDs[4]
			pcrStatus = partnercontactrequest.StatusPending
		} else {
			switch i % 4 {
			case 0:
				pcrStatus = partnercontactrequest.StatusPending
			case 1:
				pcrStatus = partnercontactrequest.StatusAcknowledged
			case 2:
				pcrStatus = partnercontactrequest.StatusClosed
			default:
				pcrStatus = partnercontactrequest.StatusCancelled
			}
		}

		pcrBuilders = append(pcrBuilders, client.PartnerContactRequest.Create().
			SetID(fmt.Sprintf("pcr_scale_%04d", i+1)).
			SetMembershipID(mID).
			SetStudentID(sID).
			SetPartnerID(pID).
			SetCategory(partnercontactrequest.CategoryCheckIn).
			SetMessageEncrypted(encryptText(fmt.Sprintf("Pesan kontak konsultasi mitra pendampingan #%04d", i+1))).
			SetStatus(pcrStatus))
	}
	if err := saveInBatches(ctx, 200, len(pcrBuilders), func(start, end int) error {
		_, err := client.PartnerContactRequest.CreateBulk(pcrBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed PartnerContactRequest: %w", err)
	}
	reports = append(reports, TableReport{"PartnerContactRequest", n})

	// 10. ApprovalRequest (n)
	apprIDs := make([]string, n)
	apprBuilders := make([]*ent.ApprovalRequestCreate, 0, n)
	for i := 0; i < n; i++ {
		aid := fmt.Sprintf("appr_scale_%04d", i+1)
		apprIDs[i] = aid

		uID := userIDs[i%len(userIDs)]
		mID := membershipIDs[i%len(membershipIDs)]
		apprStatus := approvalrequest.StatusApproved

		if i == 0 {
			uID = "usr_gading"
			mID = membershipIDs[0]
			apprStatus = approvalrequest.StatusPending
		} else if i == 1 {
			uID = "usr_dery"
			mID = membershipIDs[4]
			apprStatus = approvalrequest.StatusPending
		} else if i == 2 {
			uID = "usr_scale_0001"
			mID = membershipIDs[1]
			apprStatus = approvalrequest.StatusApproved
		} else {
			switch i % 10 {
			case 0, 1:
				apprStatus = approvalrequest.StatusPending
			case 2, 3, 4, 5, 6:
				apprStatus = approvalrequest.StatusApproved
			case 7, 8:
				apprStatus = approvalrequest.StatusDenied
			default:
				apprStatus = approvalrequest.StatusExpired
			}
		}

		act := approvalrequest.ActionPauseProtection
		if i%2 == 1 {
			act = approvalrequest.ActionUninstallDetected
		}

		apprBuilders = append(apprBuilders, client.ApprovalRequest.Create().
			SetID(aid).
			SetUserID(uID).
			SetMembershipID(mID).
			SetDeviceID(deviceIDs[i%len(deviceIDs)]).
			SetQuickTokenHash(Sha256Hash(fmt.Sprintf("qtok_%04d", i+1))).
			SetAction(act).
			SetStatus(apprStatus).
			SetExpiresAt(now.Add(24*time.Hour)).
			SetGrantJti(fmt.Sprintf("jti_scale_%04d", i+1)))
	}
	if err := saveInBatches(ctx, 200, len(apprBuilders), func(start, end int) error {
		_, err := client.ApprovalRequest.CreateBulk(apprBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed ApprovalRequest: %w", err)
	}
	reports = append(reports, TableReport{"ApprovalRequest", n})

	// 11. NotificationDelivery (n)
	ndBuilders := make([]*ent.NotificationDeliveryCreate, 0, n)
	for i := 0; i < n; i++ {
		ndBuilders = append(ndBuilders, client.NotificationDelivery.Create().
			SetID(fmt.Sprintf("notif_scale_%04d", i+1)).
			SetApprovalRequestID(apprIDs[i%len(apprIDs)]).
			SetChannel(notificationdelivery.ChannelWhatsapp).
			SetRecipient(fmt.Sprintf("+62812%08d", i+1)).
			SetStatus(notificationdelivery.StatusSent).
			SetAttemptCount(1))
	}
	if err := saveInBatches(ctx, 200, len(ndBuilders), func(start, end int) error {
		_, err := client.NotificationDelivery.CreateBulk(ndBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed NotificationDelivery: %w", err)
	}
	reports = append(reports, TableReport{"NotificationDelivery", n})

	// 12. PsychoeducationModule (n)
	moduleIDs := make([]string, n)
	modBuilders := make([]*ent.PsychoeducationModuleCreate, 0, n)
	for i := 0; i < n; i++ {
		mid := fmt.Sprintf("pmod_scale_%04d", i+1)
		moduleIDs[i] = mid
		// Realistic distribution: 80% published, 15% in_review (for review queue), 5% draft
		modStatus := psychoeducationmodule.StatusPublished
		switch i % 20 {
		case 0, 1, 2:
			modStatus = psychoeducationmodule.StatusInReview
		case 3:
			modStatus = psychoeducationmodule.StatusDraft
		default:
			modStatus = psychoeducationmodule.StatusPublished
		}
		modBuilders = append(modBuilders, client.PsychoeducationModule.Create().
			SetID(mid).
			SetSlug(fmt.Sprintf("slug-module-%04d", i+1)).
			SetTitle(fmt.Sprintf("Psikoedukasi Modul %04d", i+1)).
			SetSummary("Ringkasan edukasi pola perilaku impulsif").
			SetBodyMarkdown("# Materi Edukasi\nPenjelasan lengkap.").
			SetEstimatedMinutes(10).
			SetStatus(modStatus).
			SetDraftDocumentJSON(model.EducationDocument{
				Audience:         "all",
				ExperienceType:   "article",
				Category:         "impulse-awareness",
				EstimatedMinutes: 10,
				Translations: map[string]model.EducationTranslation{
					"id": {
						Title:             fmt.Sprintf("Psikoedukasi Modul %04d", i+1),
						Summary:           "Ringkasan edukasi pola perilaku impulsif",
						LearningObjective: "Memahami pola dorongan perilaku impulsif.",
						Disclaimer:        "Materi ini bersifat psikoedukasi mandiri.",
					},
					"en": {
						Title:             fmt.Sprintf("Psychoeducation Module %04d", i+1),
						Summary:           "Summary of impulsive behavior patterns",
						LearningObjective: "Understand impulsive urge patterns.",
						Disclaimer:        "This material is for self-guided psychoeducation.",
					},
				},
				Sections: []model.EducationSection{
					{
						ID:        "sec1",
						SortOrder: 1,
						Required:  true,
						Translations: map[string]model.EducationSectionTranslation{
							"id": {
								Title: "Pengantar",
								KnowledgeCheck: &model.EducationKnowledgeCheck{
									ID:              "check-sec1",
									Question:        "Apakah dorongan impulsif dapat dikendalikan?",
									Choices:         []model.EducationChoice{{ID: "a", Text: "Ya"}, {ID: "b", Text: "Tidak"}},
									CorrectChoiceID: "a",
									Explanation:     "Dorongan dapat dikenali dan dikelola dengan jeda sadar.",
									Required:        true,
								},
							},
							"en": {
								Title: "Introduction",
								KnowledgeCheck: &model.EducationKnowledgeCheck{
									ID:              "check-sec1",
									Question:        "Can impulsive urges be managed?",
									Choices:         []model.EducationChoice{{ID: "a", Text: "Yes"}, {ID: "b", Text: "No"}},
									CorrectChoiceID: "a",
									Explanation:     "Urges can be recognized and managed with mindful pauses.",
									Required:        true,
								},
							},
						},
					},
				},
				Sources: []model.EducationSource{
					{
						Title:      "Panduan Psikoedukasi Perilaku",
						Publisher:  "Gamblock AI",
						URL:        "https://example.org/guide",
						AccessedAt: now,
					},
				},
			}).
			SetDraftRevision(1).
			SetPublishedRevision(1).
			SetPublishedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(modBuilders), func(start, end int) error {
		_, err := client.PsychoeducationModule.CreateBulk(modBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed PsychoeducationModule: %w", err)
	}
	reports = append(reports, TableReport{"PsychoeducationModule", n})

	// 13. EducationRevision (n)
	edurBuilders := make([]*ent.EducationRevisionCreate, 0, n)
	for i := 0; i < n; i++ {
		edurBuilders = append(edurBuilders, client.EducationRevision.Create().
			SetID(fmt.Sprintf("edurev_scale_%04d", i+1)).
			SetModuleID(moduleIDs[i%len(moduleIDs)]).
			SetRevision(1).
			SetDocumentJSON(model.EducationDocument{Category: "impulse"}).
			SetSlug(fmt.Sprintf("slug-module-%04d", i+1)).
			SetKind(educationrevision.KindPublished).
			SetCreatedBy("usr_nasywa"))
	}
	if err := saveInBatches(ctx, 200, len(edurBuilders), func(start, end int) error {
		_, err := client.EducationRevision.CreateBulk(edurBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed EducationRevision: %w", err)
	}
	reports = append(reports, TableReport{"EducationRevision", n})

	// 14. EducationMedia (n)
	mediaBuilders := make([]*ent.EducationMediaCreate, 0, n)
	for i := 0; i < n; i++ {
		mediaBuilders = append(mediaBuilders, client.EducationMedia.Create().
			SetID(fmt.Sprintf("edumed_scale_%04d", i+1)).
			SetKind(educationmedia.KindUpload).
			SetPurpose(educationmedia.PurposeThumbnail).
			SetMediaType(educationmedia.MediaTypeImage).
			SetMimeType("image/webp").
			SetStorageKey(fmt.Sprintf("education/media_%04d.webp", i+1)).
			SetSizeBytes(102400).
			SetStatus(educationmedia.StatusPublished))
	}
	if err := saveInBatches(ctx, 200, len(mediaBuilders), func(start, end int) error {
		_, err := client.EducationMedia.CreateBulk(mediaBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed EducationMedia: %w", err)
	}
	reports = append(reports, TableReport{"EducationMedia", n})

	// 15. PsychoeducationProgress (n)
	pprogBuilders := make([]*ent.PsychoeducationProgressCreate, 0, n)
	for i := 0; i < n; i++ {
		percent := 100
		switch i % 5 {
		case 0:
			percent = 0
		case 1:
			percent = 25
		case 2:
			percent = 50
		case 3:
			percent = 75
		default:
			percent = 100
		}
		pprogBuilders = append(pprogBuilders, client.PsychoeducationProgress.Create().
			SetID(fmt.Sprintf("pprog_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetModuleID(moduleIDs[i%len(moduleIDs)]).
			SetRevision(1).
			SetCompletedSectionIds([]string{"sec1"}).
			SetProgressPercent(percent).
			SetCompletedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(pprogBuilders), func(start, end int) error {
		_, err := client.PsychoeducationProgress.CreateBulk(pprogBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed PsychoeducationProgress: %w", err)
	}
	reports = append(reports, TableReport{"PsychoeducationProgress", n})

	// 16. Institution (n)
	instIDs := make([]string, n)
	instBuilders := make([]*ent.InstitutionCreate, 0, n)
	for i := 0; i < n; i++ {
		iid := fmt.Sprintf("inst_scale_%04d", i+1)
		instIDs[i] = iid
		instBuilders = append(instBuilders, client.Institution.Create().
			SetID(iid).
			SetSlug(fmt.Sprintf("slug-inst-%04d", i+1)).
			SetName(fmt.Sprintf("Universitas Mitra %04d", i+1)).
			SetStatus(institution.StatusActive))
	}
	if err := saveInBatches(ctx, 200, len(instBuilders), func(start, end int) error {
		_, err := client.Institution.CreateBulk(instBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed Institution: %w", err)
	}
	reports = append(reports, TableReport{"Institution", n})

	// 17. AcademicProgram (n)
	acadBuilders := make([]*ent.AcademicProgramCreate, 0, n)
	for i := 0; i < n; i++ {
		acadBuilders = append(acadBuilders, client.AcademicProgram.Create().
			SetID(fmt.Sprintf("acad_scale_%04d", i+1)).
			SetInstitutionID(instIDs[i%len(instIDs)]).
			SetSlug(fmt.Sprintf("slug-prog-%04d", i+1)).
			SetName(fmt.Sprintf("Program Studi %04d", i+1)).
			SetPrimaryClusterSlug("technology").
			SetSortOrder(i+1).
			SetActive(true))
	}
	if err := saveInBatches(ctx, 200, len(acadBuilders), func(start, end int) error {
		_, err := client.AcademicProgram.CreateBulk(acadBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed AcademicProgram: %w", err)
	}
	reports = append(reports, TableReport{"AcademicProgram", n})

	// 18. LearningCluster (n)
	clustBuilders := make([]*ent.LearningClusterCreate, 0, n)
	for i := 0; i < n; i++ {
		clustBuilders = append(clustBuilders, client.LearningCluster.Create().
			SetID(fmt.Sprintf("lclust_scale_%04d", i+1)).
			SetSlug(fmt.Sprintf("slug-cluster-%04d", i+1)).
			SetTitleID(fmt.Sprintf("Kluster Belajar %04d", i+1)).
			SetTitleEn(fmt.Sprintf("Learning Cluster %04d", i+1)).
			SetSortOrder(i+1).
			SetActive(true))
	}
	if err := saveInBatches(ctx, 200, len(clustBuilders), func(start, end int) error {
		_, err := client.LearningCluster.CreateBulk(clustBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed LearningCluster: %w", err)
	}
	reports = append(reports, TableReport{"LearningCluster", n})

	// 19. LearningItem (n)
	itemIDs := make([]string, n)
	itemBuilders := make([]*ent.LearningItemCreate, 0, n)
	for i := 0; i < n; i++ {
		lid := fmt.Sprintf("litem_scale_%04d", i+1)
		itemIDs[i] = lid
		lStatus := learningitem.StatusPublished
		switch i % 20 {
		case 0, 1:
			lStatus = learningitem.StatusInReview
		case 2:
			lStatus = learningitem.StatusDraft
		case 3:
			lStatus = learningitem.StatusArchived
		default:
			lStatus = learningitem.StatusPublished
		}
		itemBuilders = append(itemBuilders, client.LearningItem.Create().
			SetID(lid).
			SetSlug(fmt.Sprintf("slug-item-%04d", i+1)).
			SetKind(learningitem.KindCourse).
			SetTitleID(fmt.Sprintf("Kursus Pengembangan Diri %04d", i+1)).
			SetTitleEn(fmt.Sprintf("Self Growth Course %04d", i+1)).
			SetStatus(lStatus).
			SetDraftRevision(1).
			SetPublishedRevision(1).
			SetPublishedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(itemBuilders), func(start, end int) error {
		_, err := client.LearningItem.CreateBulk(itemBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed LearningItem: %w", err)
	}
	reports = append(reports, TableReport{"LearningItem", n})

	// 20. LearningRevision (n)
	lrevBuilders := make([]*ent.LearningRevisionCreate, 0, n)
	for i := 0; i < n; i++ {
		lrevBuilders = append(lrevBuilders, client.LearningRevision.Create().
			SetID(fmt.Sprintf("lrev_scale_%04d", i+1)).
			SetItemID(itemIDs[i%len(itemIDs)]).
			SetRevision(1).
			SetDocumentJSON(map[string]any{"overview": "Konten silabus pembelajaran"}).
			SetKind(learningrevision.KindPublished).
			SetCreatedBy("usr_nasywa"))
	}
	if err := saveInBatches(ctx, 200, len(lrevBuilders), func(start, end int) error {
		_, err := client.LearningRevision.CreateBulk(lrevBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed LearningRevision: %w", err)
	}
	reports = append(reports, TableReport{"LearningRevision", n})

	// 21. LearningProgress (n)
	lprogBuilders := make([]*ent.LearningProgressCreate, 0, n)
	for i := 0; i < n; i++ {
		lprogBuilders = append(lprogBuilders, client.LearningProgress.Create().
			SetID(fmt.Sprintf("lprog_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetItemID(itemIDs[i%len(itemIDs)]).
			SetState(learningprogress.StateCompleted).
			SetCompletedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(lprogBuilders), func(start, end int) error {
		_, err := client.LearningProgress.CreateBulk(lprogBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed LearningProgress: %w", err)
	}
	reports = append(reports, TableReport{"LearningProgress", n})

	// 22. ExperienceGrant (n)
	expBuilders := make([]*ent.ExperienceGrantCreate, 0, n)
	for i := 0; i < n; i++ {
		expBuilders = append(expBuilders, client.ExperienceGrant.Create().
			SetID(fmt.Sprintf("exp_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetSourceKind("mission").
			SetSourceID(fmt.Sprintf("src_%04d", i+1)).
			SetGrantDate(today).
			SetAmount(50).
			SetIdempotencyKey(fmt.Sprintf("exp_key_%04d", i+1)))
	}
	if err := saveInBatches(ctx, 200, len(expBuilders), func(start, end int) error {
		_, err := client.ExperienceGrant.CreateBulk(expBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed ExperienceGrant: %w", err)
	}
	reports = append(reports, TableReport{"ExperienceGrant", n})

	// 23. AggregateEvent (n*2, e.g. 1200)
	aggTarget := n * 2
	if aggTarget > 1800 {
		aggTarget = 1800
	} else if aggTarget < 600 {
		aggTarget = 600
	}

	aggBuilders := make([]*ent.AggregateEventCreate, 0, aggTarget)
	evIndex := 0

	makeHourly := func() []any {
		hourly := make([]any, 24)
		for h := 0; h < 24; h++ {
			hCount := 0
			if (h >= 21 || h <= 2) && RandomInt(0, 1) == 1 {
				hCount = RandomInt(1, 4)
			} else if (h >= 13 && h <= 15) && RandomInt(0, 2) == 1 {
				hCount = RandomInt(1, 2)
			} else if RandomInt(0, 4) == 0 {
				hCount = 1
			}
			hourly[h] = float64(hCount)
		}
		return hourly
	}

	suciStudents := []string{
		"usr_gading", "usr_dery",
		"usr_scale_0001", "usr_scale_0002", "usr_scale_0003",
		"usr_scale_0004", "usr_scale_0005", "usr_scale_0006",
		"usr_scale_0007", "usr_scale_0008",
	}

	// 1. Time-series events for Suci's accountability students across past 30 days
	for _, sID := range suciStudents {
		devID := deviceIDs[0]
		for i, uid := range userIDs {
			if uid == sID {
				devID = deviceIDs[i]
				break
			}
		}

		for d := 0; d < 30; d++ {
			evDate := now.AddDate(0, 0, -d)
			evIndex++
			// Daily block count sync with 24-hour histogram
			aggBuilders = append(aggBuilders, client.AggregateEvent.Create().
				SetID(fmt.Sprintf("aggev_scale_%04d", evIndex)).
				SetUserID(sID).
				SetDeviceID(devID).
				SetIdempotencyKey(fmt.Sprintf("aggev_key_%04d", evIndex)).
				SetEventType(aggregateevent.EventTypeBlockCountSync).
				SetEventDate(evDate).
				SetCount(RandomInt(2, 7)).
				SetMetadataJSON(map[string]any{"hourly": makeHourly()}))

			// Periodic interventions (every 3 days)
			if d%3 == 0 {
				evIndex++
				aggBuilders = append(aggBuilders, client.AggregateEvent.Create().
					SetID(fmt.Sprintf("aggev_scale_%04d", evIndex)).
					SetUserID(sID).
					SetDeviceID(devID).
					SetIdempotencyKey(fmt.Sprintf("aggev_key_%04d", evIndex)).
					SetEventType(aggregateevent.EventTypeInterventionShown).
					SetEventDate(evDate).
					SetCount(RandomInt(1, 4)).
					SetMetadataJSON(map[string]any{}))
			}

			// Periodic tamper detections
			if d == 2 || d == 9 || d == 17 || d == 24 {
				evIndex++
				aggBuilders = append(aggBuilders, client.AggregateEvent.Create().
					SetID(fmt.Sprintf("aggev_scale_%04d", evIndex)).
					SetUserID(sID).
					SetDeviceID(devID).
					SetIdempotencyKey(fmt.Sprintf("aggev_key_%04d", evIndex)).
					SetEventType(aggregateevent.EventTypeTamperDetected).
					SetEventDate(evDate).
					SetCount(RandomInt(1, 3)).
					SetMetadataJSON(map[string]any{}))
			}

			// Periodic permission revocations
			if d == 5 || d == 20 {
				evIndex++
				aggBuilders = append(aggBuilders, client.AggregateEvent.Create().
					SetID(fmt.Sprintf("aggev_scale_%04d", evIndex)).
					SetUserID(sID).
					SetDeviceID(devID).
					SetIdempotencyKey(fmt.Sprintf("aggev_key_%04d", evIndex)).
					SetEventType(aggregateevent.EventTypePermissionRevoked).
					SetEventDate(evDate).
					SetCount(1).
					SetMetadataJSON(map[string]any{}))
			}
		}
	}

	// 2. Fill remaining events for scale students across the 30-day window
	for evIndex < aggTarget {
		evIndex++
		uIdx := (evIndex + 10) % len(userIDs)
		dIdx := evIndex % len(deviceIDs)
		dayOffset := evIndex % 30

		evType := aggregateevent.EventTypeBlockCountSync
		metaJSON := map[string]any{}
		evCount := RandomInt(1, 8)

		switch evIndex % 10 {
		case 0, 1:
			evType = aggregateevent.EventTypeInterventionShown
			evCount = RandomInt(1, 4)
		case 2:
			evType = aggregateevent.EventTypeTamperDetected
			evCount = RandomInt(1, 2)
		case 3:
			evType = aggregateevent.EventTypePermissionRevoked
			evCount = 1
		default:
			evType = aggregateevent.EventTypeBlockCountSync
			metaJSON["hourly"] = makeHourly()
		}

		aggBuilders = append(aggBuilders, client.AggregateEvent.Create().
			SetID(fmt.Sprintf("aggev_scale_%04d", evIndex)).
			SetUserID(userIDs[uIdx]).
			SetDeviceID(deviceIDs[dIdx]).
			SetIdempotencyKey(fmt.Sprintf("aggev_key_%04d", evIndex)).
			SetEventType(evType).
			SetEventDate(now.AddDate(0, 0, -dayOffset)).
			SetCount(evCount).
			SetMetadataJSON(metaJSON))
	}

	if err := saveInBatches(ctx, 200, len(aggBuilders), func(start, end int) error {
		_, err := client.AggregateEvent.CreateBulk(aggBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed AggregateEvent: %w", err)
	}
	reports = append(reports, TableReport{"AggregateEvent", len(aggBuilders)})

	// 24. Organization (n)
	orgIDs := make([]string, n)
	orgBuilders := make([]*ent.OrganizationCreate, 0, n)
	for i := 0; i < n; i++ {
		oid := fmt.Sprintf("org_scale_%04d", i+1)
		orgIDs[i] = oid
		orgBuilders = append(orgBuilders, client.Organization.Create().
			SetID(oid).
			SetName(fmt.Sprintf("Organisasi Kampus %04d", i+1)).
			SetSlug(fmt.Sprintf("slug-org-%04d", i+1)).
			SetStatus(organization.StatusActive).
			SetCreatedBy("usr_nasywa"))
	}
	if err := saveInBatches(ctx, 200, len(orgBuilders), func(start, end int) error {
		_, err := client.Organization.CreateBulk(orgBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed Organization: %w", err)
	}
	reports = append(reports, TableReport{"Organization", n})

	// 25. OrganizationMember (n)
	omBuilders := make([]*ent.OrganizationMemberCreate, 0, n)
	for i := 0; i < n; i++ {
		omBuilders = append(omBuilders, client.OrganizationMember.Create().
			SetID(fmt.Sprintf("orgmem_scale_%04d", i+1)).
			SetOrganizationID(orgIDs[i%len(orgIDs)]).
			SetUserID(userIDs[i%len(userIDs)]).
			SetRole(organizationmember.RoleMember).
			SetStatus(organizationmember.StatusActive).
			SetJoinedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(omBuilders), func(start, end int) error {
		_, err := client.OrganizationMember.CreateBulk(omBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed OrganizationMember: %w", err)
	}
	reports = append(reports, TableReport{"OrganizationMember", n})

	// 26. OrganizationInvite (n)
	oiBuilders := make([]*ent.OrganizationInviteCreate, 0, n)
	for i := 0; i < n; i++ {
		oiBuilders = append(oiBuilders, client.OrganizationInvite.Create().
			SetID(fmt.Sprintf("orginv_scale_%04d", i+1)).
			SetOrganizationID(orgIDs[i%len(orgIDs)]).
			SetEmail(fmt.Sprintf("invite_%04d@test.local", i+1)).
			SetRole(organizationinvite.RoleMember).
			SetTokenHash(Sha256Hash(fmt.Sprintf("inv_token_%04d", i+1))).
			SetStatus(organizationinvite.StatusPending).
			SetExpiresAt(now.Add(7*24*time.Hour)))
	}
	if err := saveInBatches(ctx, 200, len(oiBuilders), func(start, end int) error {
		_, err := client.OrganizationInvite.CreateBulk(oiBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed OrganizationInvite: %w", err)
	}
	reports = append(reports, TableReport{"OrganizationInvite", n})

	// 27. OrganizationPolicy (n)
	opBuilders := make([]*ent.OrganizationPolicyCreate, 0, n)
	for i := 0; i < n; i++ {
		opBuilders = append(opBuilders, client.OrganizationPolicy.Create().
			SetID(fmt.Sprintf("orgpol_scale_%04d", i+1)).
			SetOrganizationID(orgIDs[i%len(orgIDs)]).
			SetKey(fmt.Sprintf("policy_key_%04d", i+1)).
			SetValueJSON(map[string]any{"retention_days": 30}).
			SetCreatedBy("usr_nasywa"))
	}
	if err := saveInBatches(ctx, 200, len(opBuilders), func(start, end int) error {
		_, err := client.OrganizationPolicy.CreateBulk(opBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed OrganizationPolicy: %w", err)
	}
	reports = append(reports, TableReport{"OrganizationPolicy", n})

	// 28. ReportRollup (n)
	rrBuilders := make([]*ent.ReportRollupCreate, 0, n)
	for i := 0; i < n; i++ {
		rrBuilders = append(rrBuilders, client.ReportRollup.Create().
			SetID(fmt.Sprintf("report_scale_%04d", i+1)).
			SetScope(reportrollup.ScopeUser).
			SetScopeID(userIDs[i%len(userIDs)]).
			SetPeriod(reportrollup.PeriodDaily).
			SetPeriodStart(now.AddDate(0, 0, -i%30)).
			SetMetricsJSON(map[string]any{"blocks": RandomInt(1, 15), "interventions": RandomInt(1, 5)}))
	}
	if err := saveInBatches(ctx, 200, len(rrBuilders), func(start, end int) error {
		_, err := client.ReportRollup.CreateBulk(rrBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed ReportRollup: %w", err)
	}
	reports = append(reports, TableReport{"ReportRollup", n})

	// 29. SupportCase (n)
	caseIDs := make([]string, n)
	scBuilders := make([]*ent.SupportCaseCreate, 0, n)
	for i := 0; i < n; i++ {
		cid := fmt.Sprintf("scase_scale_%04d", i+1)
		caseIDs[i] = cid

		// Realistic status: 35% waiting_support, 25% waiting_user, 25% resolved, 15% closed
		scStatus := supportcase.StatusWaitingSupport
		switch i % 20 {
		case 0, 1, 2, 3, 4, 5, 6:
			scStatus = supportcase.StatusWaitingSupport
		case 7, 8, 9, 10, 11:
			scStatus = supportcase.StatusWaitingUser
		case 12, 13, 14, 15, 16:
			scStatus = supportcase.StatusResolved
		default:
			scStatus = supportcase.StatusClosed
		}

		// Priority distribution
		scPrio := supportcase.PriorityNormal
		switch i % 10 {
		case 0:
			scPrio = supportcase.PriorityUrgent
		case 1, 2:
			scPrio = supportcase.PriorityHigh
		case 8, 9:
			scPrio = supportcase.PriorityLow
		default:
			scPrio = supportcase.PriorityNormal
		}

		// Type distribution
		scType := supportcase.TypeTechnicalSupport
		switch i % 5 {
		case 0:
			scType = supportcase.TypeTechnicalSupport
		case 1:
			scType = supportcase.TypeAccountRecovery
		case 2:
			scType = supportcase.TypePartnerAbuse
		case 3:
			scType = supportcase.TypePrivacyRequest
		case 4:
			scType = supportcase.TypeSafetySupport
		}

		builder := client.SupportCase.Create().
			SetID(cid).
			SetUserID(userIDs[i%len(userIDs)]).
			SetOrganizationID(orgIDs[i%len(orgIDs)]).
			SetType(scType).
			SetStatus(scStatus).
			SetPriority(scPrio).
			SetSummary(fmt.Sprintf("Tiket Bantuan %s #%04d", humanSupportTypeLabel(scType), i+1))

		if scStatus != supportcase.StatusWaitingSupport || i%2 == 0 {
			builder.SetAssignedOperatorID("usr_nasywa")
		}

		scBuilders = append(scBuilders, builder)
	}
	if err := saveInBatches(ctx, 200, len(scBuilders), func(start, end int) error {
		_, err := client.SupportCase.CreateBulk(scBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed SupportCase: %w", err)
	}
	reports = append(reports, TableReport{"SupportCase", n})

	// 30. SupportMessage (n*2)
	msgCount := n * 2
	if msgCount > 2000 {
		msgCount = 2000
	}
	smsgBuilders := make([]*ent.SupportMessageCreate, 0, msgCount)
	for i := 0; i < msgCount; i++ {
		smsgBuilders = append(smsgBuilders, client.SupportMessage.Create().
			SetID(fmt.Sprintf("smsg_scale_%04d", i+1)).
			SetSupportCaseID(caseIDs[i%len(caseIDs)]).
			SetAuthorID(userIDs[i%len(userIDs)]).
			SetAuthorRole(supportmessage.AuthorRoleRequester).
			SetContentEncrypted(encryptText(fmt.Sprintf("Pesan dukungan resmi untuk tiket %s nomor %d. Tim kami siap membantu.", caseIDs[i%len(caseIDs)], i+1))))
	}
	if err := saveInBatches(ctx, 200, len(smsgBuilders), func(start, end int) error {
		_, err := client.SupportMessage.CreateBulk(smsgBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed SupportMessage: %w", err)
	}
	reports = append(reports, TableReport{"SupportMessage", msgCount})

	// 31. SupportActionAudit (n)
	saudBuilders := make([]*ent.SupportActionAuditCreate, 0, n)
	for i := 0; i < n; i++ {
		saudBuilders = append(saudBuilders, client.SupportActionAudit.Create().
			SetID(fmt.Sprintf("saud_scale_%04d", i+1)).
			SetSupportCaseID(caseIDs[i%len(caseIDs)]).
			SetOperatorID("usr_nasywa").
			SetAction("status_updated").
			SetReason("Respon otomatis sistem"))
	}
	if err := saveInBatches(ctx, 200, len(saudBuilders), func(start, end int) error {
		_, err := client.SupportActionAudit.CreateBulk(saudBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed SupportActionAudit: %w", err)
	}
	reports = append(reports, TableReport{"SupportActionAudit", n})

	// 32. EmergencyKeyRequest (n)
	emkBuilders := make([]*ent.EmergencyKeyRequestCreate, 0, n)
	for i := 0; i < n; i++ {
		emkStatus := emergencykeyrequest.StatusApproved
		switch i % 20 {
		case 0, 1, 2: // 15% pending (awaits admin action)
			emkStatus = emergencykeyrequest.StatusPending
		case 3, 4: // 10% reviewed
			emkStatus = emergencykeyrequest.StatusReviewed
		case 5, 6, 7, 8: // 20% used
			emkStatus = emergencykeyrequest.StatusUsed
		case 9: // 5% expired
			emkStatus = emergencykeyrequest.StatusExpired
		default: // 50% approved
			emkStatus = emergencykeyrequest.StatusApproved
		}

		builder := client.EmergencyKeyRequest.Create().
			SetID(fmt.Sprintf("emk_scale_%04d", i+1)).
			SetRequestedBy(userIDs[i%len(userIDs)]).
			SetDeviceID(deviceIDs[i%len(deviceIDs)]).
			SetStatus(emkStatus).
			SetGrantJti(fmt.Sprintf("emk_jti_%04d", i+1)).
			SetRequestExpiresAt(now.Add(24 * time.Hour))

		if emkStatus == emergencykeyrequest.StatusReviewed || emkStatus == emergencykeyrequest.StatusApproved || emkStatus == emergencykeyrequest.StatusUsed {
			builder.SetReviewedBy("usr_nasywa").SetReviewedAt(now)
		}
		if emkStatus == emergencykeyrequest.StatusApproved || emkStatus == emergencykeyrequest.StatusUsed {
			builder.SetApprovedBy("usr_nasywa").SetApprovedAt(now)
		}
		if emkStatus == emergencykeyrequest.StatusUsed {
			builder.SetUsedAt(now)
		}

		emkBuilders = append(emkBuilders, builder)
	}
	if err := saveInBatches(ctx, 200, len(emkBuilders), func(start, end int) error {
		_, err := client.EmergencyKeyRequest.CreateBulk(emkBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed EmergencyKeyRequest: %w", err)
	}
	reports = append(reports, TableReport{"EmergencyKeyRequest", n})

	// 33. DataRequest (n)
	dreqBuilders := make([]*ent.DataRequestCreate, 0, n)
	for i := 0; i < n; i++ {
		reqType := datarequest.TypeExport
		if i%3 == 1 {
			reqType = datarequest.TypeDelete
		}

		reqStatus := datarequest.StatusCompleted
		switch i % 20 {
		case 0, 1: // 10% failed
			reqStatus = datarequest.StatusFailed
		case 2: // 5% processing
			reqStatus = datarequest.StatusProcessing
		case 3: // 5% queued
			reqStatus = datarequest.StatusQueued
		default: // 80% completed
			reqStatus = datarequest.StatusCompleted
		}

		dreqBuilders = append(dreqBuilders, client.DataRequest.Create().
			SetID(fmt.Sprintf("dreq_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetType(reqType).
			SetStatus(reqStatus).
			SetRequestedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(dreqBuilders), func(start, end int) error {
		_, err := client.DataRequest.CreateBulk(dreqBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed DataRequest: %w", err)
	}
	reports = append(reports, TableReport{"DataRequest", n})

	// 34. OperatorInvitation (n)
	opinvBuilders := make([]*ent.OperatorInvitationCreate, 0, n)
	for i := 0; i < n; i++ {
		opinvBuilders = append(opinvBuilders, client.OperatorInvitation.Create().
			SetID(fmt.Sprintf("opinv_scale_%04d", i+1)).
			SetEmail(fmt.Sprintf("admin_invite_%04d@test.local", i+1)).
			SetRole(operatorinvitation.RoleAdmin).
			SetTokenHash(Sha256Hash(fmt.Sprintf("opinv_token_%04d", i+1))).
			SetStatus(operatorinvitation.StatusPending).
			SetInvitedBy("usr_nasywa").
			SetExpiresAt(now.Add(7*24*time.Hour)))
	}
	if err := saveInBatches(ctx, 200, len(opinvBuilders), func(start, end int) error {
		_, err := client.OperatorInvitation.CreateBulk(opinvBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed OperatorInvitation: %w", err)
	}
	reports = append(reports, TableReport{"OperatorInvitation", n})

	// 35. SiteSocialLink (8 platform enum items due to strict unique index on platform)
	platforms := []sitesociallink.Platform{
		sitesociallink.PlatformInstagram,
		sitesociallink.PlatformTiktok,
		sitesociallink.PlatformYoutube,
		sitesociallink.PlatformFacebook,
		sitesociallink.PlatformLinkedin,
		sitesociallink.PlatformX,
		sitesociallink.PlatformThreads,
		sitesociallink.PlatformGithub,
	}
	socBuilders := make([]*ent.SiteSocialLinkCreate, 0, len(platforms))
	for i, plat := range platforms {
		socBuilders = append(socBuilders, client.SiteSocialLink.Create().
			SetID(fmt.Sprintf("soc_scale_%02d", i+1)).
			SetPlatform(plat).
			SetLabel(fmt.Sprintf("Gamblock-AI on %s", plat)).
			SetURL(fmt.Sprintf("https://%s.com/gamblockai", plat)).
			SetEnabled(true).
			SetSortOrder(i+1).
			SetUpdatedBy("usr_nasywa"))
	}
	if err := saveInBatches(ctx, 200, len(socBuilders), func(start, end int) error {
		_, err := client.SiteSocialLink.CreateBulk(socBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed SiteSocialLink: %w", err)
	}
	reports = append(reports, TableReport{"SiteSocialLink", len(platforms)})

	// 36. AuditLog (n*2)
	auditCount := n * 2
	if auditCount > 2000 {
		auditCount = 2000
	}
	actions := []struct {
		Action string
		Target string
	}{
		{"user.session.login", "User"},
		{"device.protection.status_changed", "Device"},
		{"accountability.group.joined", "AccountabilityGroup"},
		{"emergency.key.requested", "EmergencyKeyRequest"},
		{"support.case.created", "SupportCase"},
		{"education.module.completed", "PsychoeducationModule"},
		{"admin.content.published", "LearningItem"},
		{"admin.social_links.updated", "SiteSocialLink"},
	}
	auditBuilders := make([]*ent.AuditLogCreate, 0, auditCount)
	for i := 0; i < auditCount; i++ {
		act := actions[i%len(actions)]
		auditBuilders = append(auditBuilders, client.AuditLog.Create().
			SetID(fmt.Sprintf("audit_scale_%04d", i+1)).
			SetActorID(userIDs[i%len(userIDs)]).
			SetActorEmail(userEmails[i%len(userEmails)]).
			SetAction(act.Action).
			SetTargetType(act.Target).
			SetTargetID(fmt.Sprintf("target_%04d", i+1)).
			SetReason("Sesi lokal pengujian skala"))
	}
	if err := saveInBatches(ctx, 200, len(auditBuilders), func(start, end int) error {
		_, err := client.AuditLog.CreateBulk(auditBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed AuditLog: %w", err)
	}
	reports = append(reports, TableReport{"AuditLog", auditCount})

	// 37. ContentProgress (n)
	cprogBuilders := make([]*ent.ContentProgressCreate, 0, n)
	for i := 0; i < n; i++ {
		cprogBuilders = append(cprogBuilders, client.ContentProgress.Create().
			SetID(fmt.Sprintf("cprog_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetModuleSlug(fmt.Sprintf("slug-module-%04d", i+1)).
			SetProgress(0.85).
			SetCompletedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(cprogBuilders), func(start, end int) error {
		_, err := client.ContentProgress.CreateBulk(cprogBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed ContentProgress: %w", err)
	}
	reports = append(reports, TableReport{"ContentProgress", n})

	// 38. Intention (len(userIDs), 1 per user)
	intBuilders := make([]*ent.IntentionCreate, 0, len(userIDs))
	for i := 0; i < len(userIDs); i++ {
		intBuilders = append(intBuilders, client.Intention.Create().
			SetID(fmt.Sprintf("int_scale_%04d", i+1)).
			SetUserID(userIDs[i]).
			SetIntentionText("Komitmen pemulihan dan peningkatan fokus belajar.").
			SetStatus(intention.StatusActive).
			SetSchoolImpact(intention.SchoolImpactNever).
			SetMoneySpent(intention.MoneySpentUnder500k).
			SetScreenTime(intention.ScreenTimeUnder1h).
			SetQuitAttempts(intention.QuitAttemptsNever).
			SetQuitMotivation(intention.QuitMotivationDetermined))
	}
	if err := saveInBatches(ctx, 200, len(intBuilders), func(start, end int) error {
		_, err := client.Intention.CreateBulk(intBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed Intention: %w", err)
	}
	reports = append(reports, TableReport{"Intention", len(userIDs)})

	// 39. CheckIn (n*2)
	checkCount := n * 2
	if checkCount > 2000 {
		checkCount = 2000
	}
	chkBuilders := make([]*ent.CheckInCreate, 0, checkCount)
	for i := 0; i < checkCount; i++ {
		uIndex := i % len(userIDs)
		dayOffset := (i / len(userIDs)) % 30
		chkDate := now.AddDate(0, 0, -dayOffset)
		chkBuilders = append(chkBuilders, client.CheckIn.Create().
			SetID(fmt.Sprintf("chk_scale_%04d", i+1)).
			SetUserID(userIDs[uIndex]).
			SetMoodScore(RandomInt(1, 5)).
			SetUrgeScore(RandomInt(0, 5)).
			SetContextText("Kondisi emosi stabil.").
			SetCreatedAt(chkDate))
	}
	if err := saveInBatches(ctx, 200, len(chkBuilders), func(start, end int) error {
		_, err := client.CheckIn.CreateBulk(chkBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed CheckIn: %w", err)
	}
	reports = append(reports, TableReport{"CheckIn", checkCount})

	// 40. DailyMission (n*2)
	missionCount := n * 2
	if missionCount > 2000 {
		missionCount = 2000
	}
	dmsBuilders := make([]*ent.DailyMissionCreate, 0, missionCount)
	for i := 0; i < missionCount; i++ {
		uIndex := i % len(userIDs)
		dayOffset := (i / len(userIDs)) % 14
		mDate := FormatDate(now.AddDate(0, 0, -dayOffset))
		mKey := fmt.Sprintf("mission_slot_%d", (i/len(userIDs))%3+1)
		dmsBuilders = append(dmsBuilders, client.DailyMission.Create().
			SetID(fmt.Sprintf("dms_scale_%04d", i+1)).
			SetUserID(userIDs[uIndex]).
			SetMissionDate(mDate).
			SetMissionKey(mKey).
			SetSource(dailymission.SourceSystem).
			SetStatus(dailymission.StatusCompleted).
			SetExpReward(15).
			SetCompletedAt(now.AddDate(0, 0, -dayOffset)))
	}
	if err := saveInBatches(ctx, 200, len(dmsBuilders), func(start, end int) error {
		_, err := client.DailyMission.CreateBulk(dmsBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed DailyMission: %w", err)
	}
	reports = append(reports, TableReport{"DailyMission", missionCount})

	// 41. Reflection (n)
	reflBuilders := make([]*ent.ReflectionCreate, 0, n)
	for i := 0; i < n; i++ {
		reflBuilders = append(reflBuilders, client.Reflection.Create().
			SetID(fmt.Sprintf("refl_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetContentEncrypted(encryptText(fmt.Sprintf("Jurnal refleksi pemulihan hari ke-%d: Merasa lebih tenang, fokus, dan produktif.", i+1))).
			SetStatus(reflection.StatusActive).
			SetIsFocus(false))
	}
	if err := saveInBatches(ctx, 200, len(reflBuilders), func(start, end int) error {
		_, err := client.Reflection.CreateBulk(reflBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed Reflection: %w", err)
	}
	reports = append(reports, TableReport{"Reflection", n})

	// 42. RecoveryPracticeSession (n)
	rpsBuilders := make([]*ent.RecoveryPracticeSessionCreate, 0, n)
	for i := 0; i < n; i++ {
		rpsBuilders = append(rpsBuilders, client.RecoveryPracticeSession.Create().
			SetID(fmt.Sprintf("rps_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetPracticeKind(recoverypracticesession.PracticeKindUrgeSurfing).
			SetDurationSeconds(300).
			SetCompletedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(rpsBuilders), func(start, end int) error {
		_, err := client.RecoveryPracticeSession.CreateBulk(rpsBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed RecoveryPracticeSession: %w", err)
	}
	reports = append(reports, TableReport{"RecoveryPracticeSession", n})

	// 43. RecoverySpace (len(userIDs), 1 per user)
	rspBuilders := make([]*ent.RecoverySpaceCreate, 0, len(userIDs))
	for i := 0; i < len(userIDs); i++ {
		rspBuilders = append(rspBuilders, client.RecoverySpace.Create().
			SetID(fmt.Sprintf("rsp_scale_%04d", i+1)).
			SetUserID(userIDs[i]).
			SetTheme(recoveryspace.ThemeDormRoom).
			SetUnlockedItemsJSON([]string{"desk", "plant"}).
			SetUnlockRuleVersion(1))
	}
	if err := saveInBatches(ctx, 200, len(rspBuilders), func(start, end int) error {
		_, err := client.RecoverySpace.CreateBulk(rspBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed RecoverySpace: %w", err)
	}
	reports = append(reports, TableReport{"RecoverySpace", len(userIDs)})

	// 44. RecoveryRecord (n)
	rrecBuilders := make([]*ent.RecoveryRecordCreate, 0, n)
	for i := 0; i < n; i++ {
		rrecBuilders = append(rrecBuilders, client.RecoveryRecord.Create().
			SetID(fmt.Sprintf("rrec_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetKind(recoveryrecord.KindCopingPlan).
			SetRecordDate(today).
			SetStatus(recoveryrecord.StatusActive).
			SetContentEncrypted(encryptText("Rencana koping: Ambil jeda 5 menit, lakukan latihan pernapasan terarah, dan hubungi kontak akuntabilitas.")))
	}
	if err := saveInBatches(ctx, 200, len(rrecBuilders), func(start, end int) error {
		_, err := client.RecoveryRecord.CreateBulk(rrecBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed RecoveryRecord: %w", err)
	}
	reports = append(reports, TableReport{"RecoveryRecord", n})

	// 45. ReminderPreference (len(userIDs), 1 per user)
	remBuilders := make([]*ent.ReminderPreferenceCreate, 0, len(userIDs))
	for i := 0; i < len(userIDs); i++ {
		remBuilders = append(remBuilders, client.ReminderPreference.Create().
			SetID(fmt.Sprintf("rem_scale_%04d", i+1)).
			SetUserID(userIDs[i]).
			SetEnabled(true).
			SetLocalTime("19:00").
			SetTimezone("Asia/Jakarta").
			SetLocale("id"))
	}
	if err := saveInBatches(ctx, 200, len(remBuilders), func(start, end int) error {
		_, err := client.ReminderPreference.CreateBulk(remBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed ReminderPreference: %w", err)
	}
	reports = append(reports, TableReport{"ReminderPreference", len(userIDs)})

	// 46. PushSubscription (n)
	psubBuilders := make([]*ent.PushSubscriptionCreate, 0, n)
	for i := 0; i < n; i++ {
		psubBuilders = append(psubBuilders, client.PushSubscription.Create().
			SetID(fmt.Sprintf("psub_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetEndpoint(fmt.Sprintf("https://fcm.googleapis.com/fcm/send/scale_endpoint_%04d", i+1)).
			SetP256dh(fmt.Sprintf("p256dh_mock_key_%04d", i+1)).
			SetAuthKey(fmt.Sprintf("auth_mock_key_%04d", i+1)))
	}
	if err := saveInBatches(ctx, 200, len(psubBuilders), func(start, end int) error {
		_, err := client.PushSubscription.CreateBulk(psubBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed PushSubscription: %w", err)
	}
	reports = append(reports, TableReport{"PushSubscription", n})

	// 47. InterventionRecord (n)
	intrecBuilders := make([]*ent.InterventionRecordCreate, 0, n)
	for i := 0; i < n; i++ {
		intrecBuilders = append(intrecBuilders, client.InterventionRecord.Create().
			SetID(fmt.Sprintf("intrec_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetInterventionKey("interv_grounding_01").
			SetResponseType("modal").
			SetSupportLevel(interventionrecord.SupportLevelLOW).
			SetEngagementLevel(interventionrecord.EngagementLevelHIGH).
			SetStatus(interventionrecord.StatusRecommended).
			SetRecommendedAt(now))
	}
	if err := saveInBatches(ctx, 200, len(intrecBuilders), func(start, end int) error {
		_, err := client.InterventionRecord.CreateBulk(intrecBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed InterventionRecord: %w", err)
	}
	reports = append(reports, TableReport{"InterventionRecord", n})

	// 48. BlockedEvent (n*2)
	blockCount := n * 2
	if blockCount > 2000 {
		blockCount = 2000
	}
	blkBuilders := make([]*ent.BlockedEventCreate, 0, blockCount)
	for i := 0; i < blockCount; i++ {
		blkBuilders = append(blkBuilders, client.BlockedEvent.Create().
			SetID(fmt.Sprintf("blk_scale_%04d", i+1)).
			SetUserID(userIDs[i%len(userIDs)]).
			SetDeviceID(deviceIDs[i%len(deviceIDs)]).
			SetOccurredAt(now.AddDate(0, 0, -i%14)))
	}
	if err := saveInBatches(ctx, 200, len(blkBuilders), func(start, end int) error {
		_, err := client.BlockedEvent.CreateBulk(blkBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed BlockedEvent: %w", err)
	}
	reports = append(reports, TableReport{"BlockedEvent", blockCount})

	// 49. SpkPreference (len(userIDs), 1 per user)
	spkBuilders := make([]*ent.SpkPreferenceCreate, 0, len(userIDs))
	for i := 0; i < len(userIDs); i++ {
		spkBuilders = append(spkBuilders, client.SpkPreference.Create().
			SetID(fmt.Sprintf("spk_scale_%04d", i+1)).
			SetUserID(userIDs[i]).
			SetSpkRecommendationEnabled(true).
			SetSpkUseProtection(true).
			SetSpkUseRecovery(true).
			SetSpkUsePersonal(true).
			SetLlmPersonalizationEnabled(true))
	}
	if err := saveInBatches(ctx, 200, len(spkBuilders), func(start, end int) error {
		_, err := client.SpkPreference.CreateBulk(spkBuilders[start:end]...).Save(ctx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("bulk seed SpkPreference: %w", err)
	}
	reports = append(reports, TableReport{"SpkPreference", len(userIDs)})

	return reports, nil
}

func saveInBatches(ctx context.Context, batchSize, total int, saveFunc func(start, end int) error) error {
	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}
		if err := saveFunc(i, end); err != nil {
			return err
		}
	}
	return nil
}

// humanSupportTypeLabel returns a human-readable Indonesian label for each
// support case type, aligned with the dynamicLabels.supportType catalog in the
// website frontend. Used so that seeded ticket summaries do not expose raw enum
// values such as "partner_abuse".
func humanSupportTypeLabel(t supportcase.Type) string {
	switch t {
	case supportcase.TypeTechnicalSupport:
		return "Dukungan teknis"
	case supportcase.TypeAccountRecovery:
		return "Pemulihan akun"
	case supportcase.TypePartnerAbuse:
		return "Keamanan pendampingan"
	case supportcase.TypeStuckApproval:
		return "Kendala persetujuan"
	case supportcase.TypeDeviceRecovery:
		return "Pemulihan perangkat"
	case supportcase.TypeNotificationFailure:
		return "Kendala notifikasi"
	case supportcase.TypeOrganizationDispute:
		return "Kendala organisasi"
	case supportcase.TypeAccountabilityGuidance:
		return "Panduan pendampingan"
	case supportcase.TypePrivacyRequest:
		return "Permintaan privasi"
	case supportcase.TypeSafetySupport:
		return "Dukungan keamanan"
	default:
		return string(t)
	}
}
