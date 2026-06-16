package seed

import (
	"context"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/organization"
)

func SeedOrganizations(ctx context.Context, client *ent.Client) error {
	if _, err := client.Organization.Create().SetID("org_community").SetName("Gamblock Community Pilot").SetSlug("community-pilot").SetStatus(organization.StatusActive).SetCreatedBy("usr_nasywa").SetRetentionPolicyJSON(map[string]any{"retention_days": 365}).Save(ctx); err != nil {
		return err
	}
	return nil
}
