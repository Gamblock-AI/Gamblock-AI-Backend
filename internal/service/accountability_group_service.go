package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	appcrypto "github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

const accountabilityCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type AccountabilityGroupService struct {
	repo *repository.Repository
	cfg  config.Config
}

func NewAccountabilityGroupService(repo *repository.Repository, cfg config.Config) *AccountabilityGroupService {
	return &AccountabilityGroupService{repo: repo, cfg: cfg}
}

func (s *AccountabilityGroupService) Workspace(ctx context.Context, userID string) (model.AccountabilityWorkspace, error) {
	if err := s.repo.EscalateOverdueExitRequests(ctx, time.Now().UTC()); err != nil {
		return model.AccountabilityWorkspace{}, err
	}
	user, ok := s.repo.UserByID(ctx, userID)
	if !ok || (user.Role != "user" && user.Role != "partner") {
		return model.AccountabilityWorkspace{}, fmt.Errorf("accountability workspace is unavailable for this role")
	}
	workspace := model.AccountabilityWorkspace{
		Role: user.Role, Groups: []model.AccountabilityGroup{}, Members: []model.AccountabilityMembership{},
		ExitRequests: []model.MembershipExitRequest{}, ContactRequests: []model.PartnerContactRequest{},
	}
	if user.Role == "user" {
		membership, err := s.repo.ActiveMembershipForStudent(ctx, userID)
		if err != nil {
			return workspace, err
		}
		workspace.Membership = membership
		if membership != nil {
			group, groupErr := s.repo.AccountabilityGroupByID(ctx, membership.GroupID)
			if groupErr != nil {
				return workspace, groupErr
			}
			if owner, found := s.repo.UserByID(ctx, group.OwnerPartnerID); found {
				applyGroupOwner(&group, owner)
			}
			workspace.Groups = append(workspace.Groups, group)
			workspace.ExitRequests, err = s.repo.ListExitRequests(ctx, []string{membership.ID})
			if err != nil {
				return workspace, err
			}
		}
	} else {
		groups, err := s.repo.ListAccountabilityGroups(ctx, userID)
		if err != nil {
			return workspace, err
		}
		for i := range groups {
			applyGroupOwner(&groups[i], user)
			if groups[i].JoinCodeEncrypted != "" && s.cfg.JournalEncryptionKey != "" {
				if plain, decErr := appcrypto.Decrypt(groups[i].JoinCodeEncrypted, s.cfg.JournalEncryptionKey); decErr == nil {
					groups[i].JoinCode = plain
				}
			}
		}
		workspace.Groups = groups
		membershipIDs := []string{}
		for _, group := range groups {
			members, memberErr := s.repo.ListMembershipsForGroup(ctx, group.ID)
			if memberErr != nil {
				return workspace, memberErr
			}
			workspace.Members = append(workspace.Members, members...)
			for _, member := range members {
				membershipIDs = append(membershipIDs, member.ID)
			}
		}
		workspace.ExitRequests, err = s.repo.ListExitRequests(ctx, membershipIDs)
		if err != nil {
			return workspace, err
		}
	}
	contacts, err := s.repo.ListPartnerContactRequests(ctx, userID, user.Role)
	if err != nil {
		return workspace, err
	}
	if err := s.decryptContactMessages(contacts); err != nil {
		return workspace, err
	}
	workspace.ContactRequests = contacts
	for _, request := range workspace.ExitRequests {
		if request.Status == "pending" {
			workspace.PendingActions++
		}
	}
	for _, request := range contacts {
		if request.Status == "pending" {
			workspace.PendingActions++
		}
	}
	return workspace, nil
}

func (s *AccountabilityGroupService) CreateGroup(ctx context.Context, partnerID, name, description string) (model.AccountabilityGroup, error) {
	partner, ok := s.repo.UserByID(ctx, partnerID)
	if !ok || partner.Role != "partner" {
		return model.AccountabilityGroup{}, fmt.Errorf("only a partner can create an accountability group")
	}
	if partner.PhoneVerifiedAt == nil {
		return model.AccountabilityGroup{}, fmt.Errorf("verified WhatsApp number is required before creating a group")
	}
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if len(name) < 3 || len(name) > 80 || len(description) > 240 {
		return model.AccountabilityGroup{}, fmt.Errorf("group name or description is invalid")
	}
	rawCode, err := generateAccountabilityCode(10)
	if err != nil {
		return model.AccountabilityGroup{}, err
	}
	encryptedCode := ""
	if s.cfg.JournalEncryptionKey != "" {
		if enc, encErr := appcrypto.Encrypt(rawCode, s.cfg.JournalEncryptionKey); encErr == nil {
			encryptedCode = enc
		}
	}
	now := time.Now().UTC()
	group := model.AccountabilityGroup{
		ID: "grp_" + uuid.NewString()[:12], OwnerPartnerID: partnerID,
		OwnerName: partner.DisplayName, Name: name, Description: description,
		JoinCode: rawCode, JoinCodeHash: HashRefreshToken(rawCode),
		JoinCodeHint: rawCode[len(rawCode)-4:], JoinCodeEncrypted: encryptedCode, Status: "active", CodeRotatedAt: now,
		CreatedAt: now, UpdatedAt: now,
	}
	created, err := s.repo.CreateAccountabilityGroup(ctx, group)
	if err != nil {
		return model.AccountabilityGroup{}, err
	}
	created.JoinCode = rawCode
	applyGroupOwner(&created, partner)
	return created, nil
}

func (s *AccountabilityGroupService) PreviewJoin(ctx context.Context, studentID, rawCode string) (model.AccountabilityGroup, error) {
	student, ok := s.repo.UserByID(ctx, studentID)
	if !ok || student.Role != "user" {
		return model.AccountabilityGroup{}, fmt.Errorf("only a student can join an accountability group")
	}
	group, err := s.groupByRawCode(ctx, rawCode)
	if err != nil {
		return model.AccountabilityGroup{}, err
	}
	if owner, found := s.repo.UserByID(ctx, group.OwnerPartnerID); found {
		applyGroupOwner(&group, owner)
	}
	group.JoinCodeHash = ""
	return group, nil
}

func applyGroupOwner(group *model.AccountabilityGroup, owner model.User) {
	group.OwnerName = owner.DisplayName
	group.OwnerAvatarURL = owner.AvatarURL
}

func (s *AccountabilityGroupService) Join(ctx context.Context, studentID, rawCode string, confirmed bool) (model.AccountabilityMembership, error) {
	if !confirmed {
		return model.AccountabilityMembership{}, fmt.Errorf("group confirmation is required")
	}
	student, ok := s.repo.UserByID(ctx, studentID)
	if !ok || student.Role != "user" || student.PhoneVerifiedAt == nil {
		return model.AccountabilityMembership{}, fmt.Errorf("a verified student WhatsApp number is required")
	}
	if existing, err := s.repo.ActiveMembershipForStudent(ctx, studentID); err != nil {
		return model.AccountabilityMembership{}, err
	} else if existing != nil {
		return model.AccountabilityMembership{}, fmt.Errorf("student already has an active accountability group")
	}
	group, err := s.groupByRawCode(ctx, rawCode)
	if err != nil {
		return model.AccountabilityMembership{}, err
	}
	now := time.Now().UTC()
	return s.repo.SaveAccountabilityMembership(ctx, model.AccountabilityMembership{
		ID: "mbr_" + uuid.NewString()[:12], GroupID: group.ID, StudentID: studentID,
		StudentName: student.DisplayName, Status: "active",
		Sharing:  model.SharingPreferences{ProtectionHealth: true, ProtectionActivity: true, RecoveryEngagement: true, EducationProgress: true},
		JoinedAt: now, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *AccountabilityGroupService) RotateCode(ctx context.Context, groupID, partnerID string) (string, error) {
	rawCode, err := generateAccountabilityCode(10)
	if err != nil {
		return "", err
	}
	encryptedCode := ""
	if s.cfg.JournalEncryptionKey != "" {
		if enc, encErr := appcrypto.Encrypt(rawCode, s.cfg.JournalEncryptionKey); encErr == nil {
			encryptedCode = enc
		}
	}
	if err := s.repo.RotateAccountabilityGroupCode(ctx, groupID, partnerID, HashRefreshToken(rawCode), rawCode[len(rawCode)-4:], encryptedCode, time.Now().UTC()); err != nil {
		return "", err
	}
	return rawCode, nil
}

func (s *AccountabilityGroupService) UpdateSharing(ctx context.Context, studentID, membershipID string, sharing model.SharingPreferences) (model.AccountabilityMembership, error) {
	return s.repo.UpdateMembershipSharing(ctx, membershipID, studentID, sharing)
}

func (s *AccountabilityGroupService) RequestLeave(ctx context.Context, studentID, membershipID, kind, reason string) (model.MembershipExitRequest, error) {
	membership, err := s.repo.MembershipByID(ctx, membershipID)
	if err != nil || membership.StudentID != studentID || (membership.Status != "active" && membership.Status != "leave_pending") {
		return model.MembershipExitRequest{}, fmt.Errorf("student is not authorized for this membership")
	}
	if kind != "normal" && kind != "unsafe" {
		return model.MembershipExitRequest{}, fmt.Errorf("leave kind must be normal or unsafe")
	}
	if kind == "normal" && membership.Status != "active" {
		return model.MembershipExitRequest{}, fmt.Errorf("a normal exit request is already pending")
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 500 {
		return model.MembershipExitRequest{}, fmt.Errorf("leave reason is too long")
	}
	now := time.Now().UTC()
	request := model.MembershipExitRequest{
		ID: "exit_" + uuid.NewString()[:12], MembershipID: membership.ID,
		RequestedBy: studentID, Kind: kind, Status: "pending", Reason: reason,
		CreatedAt: now, UpdatedAt: now,
	}
	if kind == "unsafe" {
		if _, err := s.repo.UpdateMembershipSharing(ctx, membership.ID, studentID, model.SharingPreferences{}); err != nil {
			return model.MembershipExitRequest{}, err
		}
		if err := s.repo.SetMembershipStatus(ctx, membership.ID, "safety_suspended", nil); err != nil {
			return model.MembershipExitRequest{}, err
		}
		if err := s.repo.CancelPendingApprovalsForMembership(ctx, membership.ID, studentID); err != nil {
			return model.MembershipExitRequest{}, err
		}
		if err := s.repo.CancelPendingNormalExitRequestsForMembership(ctx, membership.ID, studentID); err != nil {
			return model.MembershipExitRequest{}, err
		}
	} else {
		due := now.Add(72 * time.Hour)
		request.ReviewDueAt = &due
		if err := s.repo.SetMembershipStatus(ctx, membership.ID, "leave_pending", nil); err != nil {
			return model.MembershipExitRequest{}, err
		}
	}
	return s.repo.CreateMembershipExitRequest(ctx, request)
}

func (s *AccountabilityGroupService) ResolveLeave(ctx context.Context, partnerID, requestID, decision string) error {
	return s.repo.ResolveMembershipExitRequest(ctx, requestID, partnerID, decision)
}

func (s *AccountabilityGroupService) CancelLeave(ctx context.Context, studentID, requestID string) error {
	return s.repo.CancelMembershipExitRequest(ctx, requestID, studentID)
}

func (s *AccountabilityGroupService) RemoveMember(ctx context.Context, partnerID, membershipID, reason string) error {
	membership, err := s.repo.MembershipByID(ctx, membershipID)
	if err != nil {
		return err
	}
	group, err := s.repo.AccountabilityGroupByID(ctx, membership.GroupID)
	if err != nil || group.OwnerPartnerID != partnerID || !liveMembershipStatus(membership.Status) {
		return fmt.Errorf("partner is not authorized for this membership")
	}
	now := time.Now().UTC()
	if err := s.repo.SetMembershipStatus(ctx, membership.ID, "removed", &now); err != nil {
		return err
	}
	if err := s.repo.CancelPendingApprovalsForMembership(ctx, membership.ID, partnerID); err != nil {
		return err
	}
	_, err = s.repo.CreateMembershipExitRequest(ctx, model.MembershipExitRequest{
		ID: "exit_" + uuid.NewString()[:12], MembershipID: membership.ID,
		RequestedBy: partnerID, Kind: "partner_removal", Status: "approved",
		Reason: strings.TrimSpace(reason), ResolvedBy: partnerID, ResolvedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	})
	return err
}

func (s *AccountabilityGroupService) DeleteGroup(ctx context.Context, partnerID, groupID string) error {
	return s.repo.DeleteAccountabilityGroup(ctx, groupID, partnerID)
}

func (s *AccountabilityGroupService) CreateContactRequest(ctx context.Context, studentID, membershipID, category, message string) (model.PartnerContactRequest, error) {
	membership, err := s.repo.MembershipByID(ctx, membershipID)
	if err != nil || membership.StudentID != studentID || membership.Status != "active" {
		return model.PartnerContactRequest{}, fmt.Errorf("an active membership is required")
	}
	allowedCategories := map[string]bool{"check_in": true, "practical_help": true, "accountability": true, "other": true}
	if !allowedCategories[category] {
		return model.PartnerContactRequest{}, fmt.Errorf("contact request category is invalid")
	}
	message = strings.TrimSpace(message)
	if len(message) > 1000 {
		return model.PartnerContactRequest{}, fmt.Errorf("contact message is too long")
	}
	encrypted := ""
	if message != "" {
		if s.cfg.JournalEncryptionKey == "" {
			return model.PartnerContactRequest{}, fmt.Errorf("encryption is required for a shared message")
		}
		encrypted, err = appcrypto.Encrypt(message, s.cfg.JournalEncryptionKey)
		if err != nil {
			return model.PartnerContactRequest{}, fmt.Errorf("contact message encryption failed")
		}
	}
	group, err := s.repo.AccountabilityGroupByID(ctx, membership.GroupID)
	if err != nil {
		return model.PartnerContactRequest{}, err
	}
	now := time.Now().UTC()
	return s.repo.CreatePartnerContactRequest(ctx, model.PartnerContactRequest{
		ID: "contact_" + uuid.NewString()[:12], MembershipID: membership.ID,
		StudentID: studentID, StudentName: membership.StudentName, PartnerID: group.OwnerPartnerID,
		Category: category, Message: encrypted, Status: "pending", CreatedAt: now, UpdatedAt: now,
	})
}

func (s *AccountabilityGroupService) TransitionContactRequest(ctx context.Context, actorID, requestID, status string) error {
	return s.repo.TransitionPartnerContactRequest(ctx, requestID, actorID, status)
}

func (s *AccountabilityGroupService) groupByRawCode(ctx context.Context, rawCode string) (model.AccountabilityGroup, error) {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if len(code) != 10 {
		return model.AccountabilityGroup{}, fmt.Errorf("join code is invalid")
	}
	return s.repo.AccountabilityGroupByCodeHash(ctx, HashRefreshToken(code))
}

func (s *AccountabilityGroupService) decryptContactMessages(items []model.PartnerContactRequest) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if items[i].Message == "" {
			continue
		}
		if !appcrypto.IsHexCiphertext(items[i].Message) {
			// Already plaintext (e.g. unencrypted seed or plain string)
			continue
		}
		if s.cfg.JournalEncryptionKey == "" {
			items[i].Message = "[Pesan terenkripsi tidak dapat dimuat]"
			continue
		}
		plain, err := appcrypto.Decrypt(items[i].Message, s.cfg.JournalEncryptionKey)
		if err != nil {
			items[i].Message = "[Pesan terenkripsi tidak dapat dimuat]"
			continue
		}
		items[i].Message = plain
	}
	return nil
}

func generateAccountabilityCode(length int) (string, error) {
	buf := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i, value := range random {
		buf[i] = accountabilityCodeAlphabet[int(value)%len(accountabilityCodeAlphabet)]
	}
	return string(buf), nil
}

func liveMembershipStatus(status string) bool {
	return status == "active" || status == "leave_pending" || status == "support_review" || status == "safety_suspended"
}

func (s *AccountabilityGroupService) FlaggedMembers(ctx context.Context, partnerID string, query model.PaginationQuery) (model.PaginatedList[model.FlaggedAccountabilityMember], error) {
	user, ok := s.repo.UserByID(ctx, partnerID)
	if !ok || user.Role != "partner" {
		return model.PaginatedList[model.FlaggedAccountabilityMember]{}, fmt.Errorf("only a partner can view flagged members")
	}

	groups, err := s.repo.ListAccountabilityGroups(ctx, partnerID)
	if err != nil {
		return model.PaginatedList[model.FlaggedAccountabilityMember]{}, err
	}

	var flagged []model.FlaggedAccountabilityMember
	seenMembers := make(map[string]bool)

	for _, group := range groups {
		members, memberErr := s.repo.ListMembershipsForGroup(ctx, group.ID)
		if memberErr != nil {
			return model.PaginatedList[model.FlaggedAccountabilityMember]{}, memberErr
		}
		for _, member := range members {
			if !liveMembershipStatus(member.Status) || seenMembers[member.ID] {
				continue
			}
			seenMembers[member.ID] = true
			flags := computeMonitorFlags(member)
			if len(flags) > 0 {
				flagged = append(flagged, model.FlaggedAccountabilityMember{
					Member: member,
					Flags:  flags,
				})
			}
		}
	}

	sort.SliceStable(flagged, func(i, j int) bool {
		sevI := minMonitorSeverity(flagged[i].Flags)
		sevJ := minMonitorSeverity(flagged[j].Flags)
		if sevI != sevJ {
			return sevI < sevJ
		}
		return strings.ToLower(flagged[i].Member.StudentName) < strings.ToLower(flagged[j].Member.StudentName)
	})

	page, limit, offset := query.Normalize(3)
	total := len(flagged)
	if offset >= total {
		return model.NewPaginatedList([]model.FlaggedAccountabilityMember{}, total, page, limit), nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	paged := flagged[offset:end]
	return model.NewPaginatedList(paged, total, page, limit), nil
}

func computeMonitorFlags(member model.AccountabilityMembership) []string {
	var flags []string
	if member.Status != "active" {
		flags = append(flags, "status")
	}
	if member.Aggregate.ProtectionStatus == "attention" {
		flags = append(flags, "protection")
	}
	if member.Aggregate.LastHeartbeatBucket == "older" || member.Aggregate.LastHeartbeatBucket == "never" {
		flags = append(flags, "inactive")
	}
	if member.Aggregate.CheckInDays == 0 {
		flags = append(flags, "noCheckIn")
	}
	return flags
}

func minMonitorSeverity(flags []string) int {
	minSev := 99
	for _, f := range flags {
		var sev int
		switch f {
		case "status":
			sev = 0
		case "protection":
			sev = 1
		case "inactive":
			sev = 2
		case "noCheckIn":
			sev = 3
		default:
			sev = 4
		}
		if sev < minSev {
			minSev = sev
		}
	}
	return minSev
}

