package seed

import (
	"context"
	"fmt"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	entuser "github.com/gamblock-ai/gamblock-ai-backend/ent/user"
)

type demoAccount struct {
	email string
	role  entuser.Role
}

// ExpectedDemoAccounts is the owner-approved four-account fixture. Both the
// users-only production seeder and the full demo seeder share this invariant.
var ExpectedDemoAccounts = map[string]demoAccount{
	"usr_gading": {email: "gading@gmail.com", role: entuser.RoleUser},
	"usr_dery":   {email: "dery@gmail.com", role: entuser.RoleUser},
	"usr_suci":   {email: "suci@gmail.com", role: entuser.RolePartner},
	"usr_nasywa": {email: "nasywa@gmail.com", role: entuser.RoleAdmin},
}

// ValidateDemoSeedTarget ensures the database is empty (when allowEmpty is
// true) or contains exactly the four approved demo accounts. It fails closed
// against any account outside the fixture.
func ValidateDemoSeedTarget(ctx context.Context, client *ent.Client, allowEmpty bool) error {
	rows, err := client.User.Query().All(ctx)
	if err != nil {
		return fmt.Errorf("query demo seed target: %w", err)
	}
	if len(rows) == 0 && allowEmpty {
		return nil
	}
	if len(rows) != len(ExpectedDemoAccounts) {
		return fmt.Errorf("demo seeder requires an empty database or exactly four known demo accounts")
	}
	for _, row := range rows {
		expected, ok := ExpectedDemoAccounts[row.ID]
		if !ok || row.Email != expected.email || row.Role != expected.role {
			return fmt.Errorf("database contains an account outside the approved demo fixture")
		}
	}
	return nil
}
