package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtectionGrantSigner_SignsDeviceBoundES256Grant(t *testing.T) {
	signer := NewProtectionGrantSigner(testCfg())
	startsAt := time.Date(2026, time.August, 12, 8, 0, 0, 0, time.UTC)
	expiresAt := startsAt.Add(30 * time.Minute)

	raw, err := signer.Sign("APR-test", "dev_android", "pause_protection", "device-jwk-thumbprint", "grant-jti", startsAt, expiresAt)
	require.NoError(t, err)

	parsed, err := jwt.ParseWithClaims(raw, &protectionGrantClaims{}, func(token *jwt.Token) (any, error) {
		return &signer.privateKey.PublicKey, nil
	}, jwt.WithValidMethods([]string{"ES256"}), jwt.WithIssuer(protectionGrantIssuer), jwt.WithAudience(protectionGrantAudience), jwt.WithExpirationRequired(), jwt.WithTimeFunc(func() time.Time { return startsAt.Add(time.Minute) }))
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	assert.Equal(t, protectionGrantType, parsed.Header["typ"])
	assert.Equal(t, "test-grant-key", parsed.Header["kid"])
	claims := parsed.Claims.(*protectionGrantClaims)
	assert.Equal(t, protectionGrantVersion, claims.GrantVersion)
	assert.Equal(t, "APR-test", claims.RequestID)
	assert.Equal(t, "dev_android", claims.DeviceID)
	assert.Equal(t, "pause_protection", claims.Action)
	assert.Equal(t, "device-jwk-thumbprint", claims.Confirmation.JWKThumbprint)
	assert.Equal(t, "grant-jti", claims.ID)
	assert.True(t, claims.IssuedAt.Time.Equal(startsAt))
	assert.True(t, claims.NotBefore.Time.Equal(startsAt))
	assert.True(t, claims.ExpiresAt.Time.Equal(expiresAt))
}

func TestProtectionGrantSigner_RejectsUnsupportedPauseDuration(t *testing.T) {
	signer := NewProtectionGrantSigner(testCfg())
	startsAt := time.Now().UTC()
	_, err := signer.Sign("APR-test", "dev_android", "pause_protection", "device-jwk-thumbprint", "grant-jti", startsAt, startsAt.Add(45*time.Minute))
	require.Error(t, err)
}
