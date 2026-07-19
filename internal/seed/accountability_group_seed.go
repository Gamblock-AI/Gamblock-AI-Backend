package seed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitymembership"
)

func SeedAccountabilityGroups(ctx context.Context, client *ent.Client, now time.Time) error {
	sum := sha256.Sum256([]byte("GAMBLOCK42"))
	if _, err := client.AccountabilityGroup.Create().SetID("grp_demo").SetOwnerPartnerID("usr_suci").
		SetName("Ruang dukungan Gading").SetDescription("Pendampingan pribadi yang berfokus pada dukungan dan keputusan proteksi.").
		SetJoinCodeHash(hex.EncodeToString(sum[:])).SetJoinCodeHint("CK42").SetCodeRotatedAt(now).Save(ctx); err != nil {
		return err
	}
	_, err := client.AccountabilityMembership.Create().SetID("mbr_active").SetGroupID("grp_demo").
		SetStudentID("usr_gading").SetStatus(accountabilitymembership.StatusActive).SetJoinedAt(now).Save(ctx)
	return err
}
