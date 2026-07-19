package seed

import (
	"context"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/authn"
)

func SeedUsers(ctx context.Context, client *ent.Client) error {
	passwordHash, err := authn.HashPassword("password")
	if err != nil {
		return err
	}
	users := []struct {
		id, email, name string
		role            user.Role
	}{
		{"usr_gading", "gading@gmail.com", "Gading", user.RoleUser},
		{"usr_dery", "dery@gmail.com", "Dery", user.RoleUser},
		{"usr_suci", "suci@gmail.com", "Suci", user.RolePartner},
		{"usr_nasywa", "nasywa@gmail.com", "Nasywa", user.RolePlatformAdmin},
		{"usr_demo_student", "student@gmail.com", "Demo Student", user.RoleUser},
		{"usr_demo_partner", "partner@gmail.com", "Demo Partner", user.RolePartner},
		{"usr_demo_organization_owner", "organization-owner@gmail.com", "Demo Organization Owner", user.RoleOrganizationOwner},
		{"usr_demo_organization_admin", "organization-admin@gmail.com", "Demo Organization Admin", user.RoleOrganizationAdmin},
		{"usr_demo_content_admin", "content-admin@gmail.com", "Demo Content Admin", user.RoleContentAdmin},
		{"usr_demo_model_release_operator", "model-release-operator@gmail.com", "Demo Model Release Operator", user.RoleModelReleaseOperator},
		{"usr_demo_support_operator", "support-operator@gmail.com", "Demo Support Operator", user.RoleSupportOperator},
		{"usr_demo_research_evaluator", "research-evaluator@gmail.com", "Demo Research Evaluator", user.RoleResearchEvaluator},
		{"usr_demo_platform_admin", "platform-admin@gmail.com", "Demo Platform Admin", user.RolePlatformAdmin},
	}
	for _, item := range users {
		create := client.User.Create().SetID(item.id).SetEmail(item.email).SetDisplayName(item.name).SetRole(item.role).SetPasswordHash(passwordHash).SetEmailVerifiedAt(time.Now().UTC())
		if item.role == user.RolePartner {
			create.SetPhoneE164("+6281200000000").SetPhoneVerifiedAt(time.Now().UTC())
		}
		if _, err := create.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}
