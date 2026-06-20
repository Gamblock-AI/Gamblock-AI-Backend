package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFriendly_KnownCode(t *testing.T) {
	got := Friendly("invalid_credentials")
	assert.Equal(t, "Email atau kata sandi salah. Silakan periksa kembali.", got)
	assert.NotContains(t, got, "err")
}

func TestFriendly_UnknownCodeFallsBackGeneric(t *testing.T) {
	got := Friendly("definitely_not_a_real_code_xyz")
	assert.Equal(t, Generic, got)
}

func TestFriendly_NonEmpty(t *testing.T) {
	for code := range messages {
		assert.NotEmpty(t, Friendly(code), "code %s must map to a message", code)
		assert.NotEqual(t, Generic, Friendly(code), "code %s must have a specific message", code)
	}
}
