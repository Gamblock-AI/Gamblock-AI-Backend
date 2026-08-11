package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

func TestDeviceService_BindsGrantKeyWithChallengeAndProof(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewDeviceService(repo, testCfg(), zap.NewNop())
	ctx := context.Background()

	challenge, err := svc.IssueGrantKeyChallenge(ctx, "usr_gading", "dev_android")
	require.NoError(t, err)
	unverified, _, err := jwt.NewParser().ParseUnverified(challenge.ChallengeToken, &deviceKeyChallengeClaims{})
	require.NoError(t, err)
	assert.Equal(t, deviceKeyChallengeType, unverified.Header["typ"])

	privateKey := testDeviceGrantKey(2)
	publicJWK := testDeviceGrantJWK(t, &privateKey.PublicKey)
	proof := testDeviceGrantProof(t, privateKey, "dev_android", challenge.ChallengeToken)
	binding, err := svc.BindGrantKey(ctx, "usr_gading", "dev_android", challenge.ChallengeToken, publicJWK, proof)
	require.NoError(t, err)
	assert.True(t, binding.Bound)
	assert.NotEmpty(t, binding.Thumbprint)

	idempotent, err := svc.BindGrantKey(ctx, "usr_gading", "dev_android", challenge.ChallengeToken, publicJWK, proof)
	require.NoError(t, err)
	assert.Equal(t, binding.Thumbprint, idempotent.Thumbprint)

	differentKey := testDeviceGrantKey(3)
	differentJWK := testDeviceGrantJWK(t, &differentKey.PublicKey)
	differentProof := testDeviceGrantProof(t, differentKey, "dev_android", challenge.ChallengeToken)
	_, err = svc.BindGrantKey(ctx, "usr_gading", "dev_android", challenge.ChallengeToken, differentJWK, differentProof)
	require.ErrorIs(t, err, repository.ErrDeviceGrantKeyConflict)
}

func TestDeviceService_RejectsGrantKeyProofForAnotherDevice(t *testing.T) {
	repo, _ := newRepo(t)
	svc := NewDeviceService(repo, testCfg(), zap.NewNop())
	challenge, err := svc.IssueGrantKeyChallenge(context.Background(), "usr_gading", "dev_android")
	require.NoError(t, err)
	privateKey := testDeviceGrantKey(4)

	_, err = svc.BindGrantKey(
		context.Background(),
		"usr_gading",
		"dev_android",
		challenge.ChallengeToken,
		testDeviceGrantJWK(t, &privateKey.PublicKey),
		testDeviceGrantProof(t, privateKey, "dev_windows", challenge.ChallengeToken),
	)
	require.Error(t, err)
}

func testDeviceGrantKey(d int64) *ecdsa.PrivateKey {
	curve := elliptic.P256()
	scalar := big.NewInt(d)
	x, y := curve.ScalarBaseMult(scalar.Bytes())
	return &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y}, D: scalar}
}

func testDeviceGrantJWK(t *testing.T, key *ecdsa.PublicKey) json.RawMessage {
	t.Helper()
	coordinate := func(value *big.Int) string {
		encoded := make([]byte, 32)
		value.FillBytes(encoded)
		return base64.RawURLEncoding.EncodeToString(encoded)
	}
	raw, err := json.Marshal(deviceGrantPublicJWK{KTY: "EC", CRV: "P-256", X: coordinate(key.X), Y: coordinate(key.Y)})
	require.NoError(t, err)
	return raw
}

func testDeviceGrantProof(t *testing.T, key *ecdsa.PrivateKey, deviceID, challengeToken string) string {
	t.Helper()
	digest := sha256.Sum256([]byte("gamblock-device-key-v1\n" + deviceID + "\n" + challengeToken))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	require.NoError(t, err)
	raw := make([]byte, 64)
	r.FillBytes(raw[:32])
	s.FillBytes(raw[32:])
	return base64.RawURLEncoding.EncodeToString(raw)
}
