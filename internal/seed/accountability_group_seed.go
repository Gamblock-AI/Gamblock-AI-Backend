package seed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/ent"
	"github.com/gamblock-ai/gamblock-ai-backend/ent/accountabilitymembership"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/crypto"
)

func SeedAccountabilityGroups(ctx context.Context, client *ent.Client, now time.Time) error {
	sum := sha256.Sum256([]byte("GAMBLOCK42"))
	encryptedCode := ""
	if key := os.Getenv("JOURNAL_ENCRYPTION_KEY"); key != "" {
		if enc, err := crypto.Encrypt("GAMBLOCK42", key); err == nil {
			encryptedCode = enc
		}
	}
	if _, err := client.AccountabilityGroup.Create().SetID("grp_demo").SetOwnerPartnerID("usr_suci").
		SetName("Kelas Informatika C").SetDescription("Kelas pendampingan mahasiswa Informatika yang berfokus pada dukungan dan keputusan proteksi.").
		SetJoinCodeHash(hex.EncodeToString(sum[:])).SetJoinCodeHint("CK42").SetJoinCodeEncrypted(encryptedCode).SetCodeRotatedAt(now).Save(ctx); err != nil {
		return err
	}
	_, err := client.AccountabilityMembership.Create().SetID("mbr_active").SetGroupID("grp_demo").
		SetStudentID("usr_gading").SetStatus(accountabilitymembership.StatusActive).SetJoinedAt(now).Save(ctx)
	return err
}
