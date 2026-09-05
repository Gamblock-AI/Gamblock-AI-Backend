package authn

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

func TestHashPasswordRejectsEmptyPassword(t *testing.T) {
	hash, err := HashPassword("")

	assert.Error(t, err)
	assert.Empty(t, hash)
}

func TestHashPasswordProducesVerifiableArgon2idHash(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	require.NoError(t, err)

	parts := strings.Split(hash, "$")
	require.Len(t, parts, 6)
	assert.Equal(t, "argon2id", parts[1])
	assert.Equal(t, fmt.Sprintf("v=%d", argon2.Version), parts[2])
	assert.Equal(t, fmt.Sprintf("m=%d,t=%d,p=%d", argonMemory, argonIterations, argonParallelism), parts[3])
	assert.True(t, VerifyPassword("correct horse battery staple", hash))
	assert.False(t, VerifyPassword("wrong password", hash))
}

func TestHashPasswordUsesFreshSalt(t *testing.T) {
	first, err := HashPassword("same password")
	require.NoError(t, err)
	second, err := HashPassword("same password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.True(t, VerifyPassword("same password", first))
	assert.True(t, VerifyPassword("same password", second))
}

func TestVerifyPasswordFailsClosedForMalformedHashes(t *testing.T) {
	valid, err := HashPassword("password")
	require.NoError(t, err)
	parts := strings.Split(valid, "$")
	require.Len(t, parts, 6)

	shortSalt := base64.RawStdEncoding.EncodeToString([]byte("short"))
	shortHash := base64.RawStdEncoding.EncodeToString([]byte("short"))
	cases := map[string]string{
		"empty":                     "",
		"wrong algorithm":           strings.Replace(valid, "$argon2id$", "$bcrypt$", 1),
		"wrong version":             strings.Replace(valid, fmt.Sprintf("v=%d", argon2.Version), "v=18", 1),
		"malformed parameters":      strings.Replace(valid, parts[3], "not-parameters", 1),
		"memory below minimum":      strings.Replace(valid, parts[3], "m=1,t=2,p=1", 1),
		"iterations below minimum":  strings.Replace(valid, parts[3], "m=19456,t=1,p=1", 1),
		"parallelism below minimum": strings.Replace(valid, parts[3], "m=19456,t=2,p=0", 1),
		"invalid salt encoding":     strings.Replace(valid, parts[4], "not-base64!", 1),
		"short salt":                strings.Replace(valid, parts[4], shortSalt, 1),
		"invalid hash encoding":     strings.Replace(valid, parts[5], "not-base64!", 1),
		"short hash":                strings.Replace(valid, parts[5], shortHash, 1),
		"missing fields":            "$argon2id$v=19$m=19456,t=2,p=1$only-four-fields",
	}

	for name, candidate := range cases {
		t.Run(name, func(t *testing.T) {
			assert.False(t, VerifyPassword("password", candidate))
		})
	}
}

func TestVerifyPasswordRejectsEmptyPasswordAgainstValidHash(t *testing.T) {
	hash, err := HashPassword("password")
	require.NoError(t, err)

	assert.False(t, VerifyPassword("", hash))
}
