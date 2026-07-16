package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPartner_GetAndCreateInvitation(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, _, err := repo.GetPartners(ctx, "usr_suci")
	assert.NoError(t, err)

	_, err = repo.CreatePartnerInvitation(ctx, "pl_new", "usr_gading", "partner@example.com", nil, "hashhash")
	assert.NoError(t, err)
}

func TestPartner_InvitationAcceptRevoke(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.CreatePartnerInvitation(ctx, "pl_t", "usr_gading", "partner@example.com", nil, "tokenhash123")
	require.NoError(t, err)
	_ = repo.AcceptPartnerInvitation(ctx, "pl_t", "usr_suci")
	_ = repo.RevokePartner(ctx, "pl_t", "usr_gading")
}

func TestPartner_GetPartners(t *testing.T) {
	repo, _ := newRepo(t)
	_, _, err := repo.GetPartners(context.Background(), "usr_suci")
	assert.NoError(t, err)
}

func TestPartnerLinkByToken(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.CreatePartnerInvitation(ctx, "pl_t2", "usr_gading", "p2@example.com", nil, "tokenhash456")
	require.NoError(t, err)
	_, _ = repo.GetPartnerLinkByToken(ctx, "tokenhash456")
}
