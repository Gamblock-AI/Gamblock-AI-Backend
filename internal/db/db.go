package db

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/operatorinvitation"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/supportmessage"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/seed"
)

func Open(databaseURL string) (*ent.Client, func() error, error) {
	if databaseURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL is empty")
	}
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, nil, err
	}
	driver := entsql.OpenDB(dialect.Postgres, sqlDB)
	return ent.NewClient(ent.Driver(driver)), sqlDB.Close, nil
}

func Migrate(ctx context.Context, client *ent.Client) error {
	if err := client.Schema.Create(ctx); err != nil {
		return err
	}
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	adminLegacyRoles := []entuser.Role{
		entuser.Role("content_admin"), entuser.Role("model_release_operator"),
		entuser.Role("support_operator"), entuser.Role("research_evaluator"),
		entuser.Role("platform_admin"),
	}
	if _, err = tx.User.Update().Where(entuser.RoleIn(adminLegacyRoles...)).SetRole(entuser.RoleAdmin).Save(ctx); err != nil {
		return rollback(fmt.Errorf("migrate legacy admin roles: %w", err))
	}
	if _, err = tx.User.Update().Where(entuser.RoleIn(entuser.Role("organization_owner"), entuser.Role("organization_admin"))).SetRole(entuser.RolePartner).Save(ctx); err != nil {
		return rollback(fmt.Errorf("migrate organization account roles: %w", err))
	}
	if _, err = tx.SupportMessage.Update().Where(supportmessage.AuthorRoleEQ(supportmessage.AuthorRole("support_operator"))).SetAuthorRole(supportmessage.AuthorRoleAdmin).Save(ctx); err != nil {
		return rollback(fmt.Errorf("migrate support author roles: %w", err))
	}
	if _, err = tx.OperatorInvitation.Update().SetRole(operatorinvitation.RoleAdmin).Save(ctx); err != nil {
		return rollback(fmt.Errorf("normalize retired operator invitations: %w", err))
	}
	if _, err = tx.OperatorInvitation.Update().Where(operatorinvitation.StatusEQ(operatorinvitation.StatusPending)).SetStatus(operatorinvitation.StatusRevoked).Save(ctx); err != nil {
		return rollback(fmt.Errorf("revoke retired operator invitations: %w", err))
	}
	return tx.Commit()
}

func Seed(ctx context.Context, client *ent.Client, mediaPath ...string) error {
	return seed.Seed(ctx, client, mediaPath...)
}
