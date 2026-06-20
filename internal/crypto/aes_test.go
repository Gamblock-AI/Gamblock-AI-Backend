package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32) // AES-256
	_, err := rand.Read(b)
	require.NoError(t, err)
	return hex.EncodeToString(b)
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := newKey(t)
	plain := "refleksi hari ini: saya hampir tergoda"
	enc, err := Encrypt(plain, key)
	require.NoError(t, err)
	assert.NotEqual(t, plain, enc, "ciphertext must differ from plaintext")

	dec, err := Decrypt(enc, key)
	require.NoError(t, err)
	assert.Equal(t, plain, dec)
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	key := newKey(t)
	a, _ := Encrypt("same text", key)
	b, _ := Encrypt("same text", key)
	assert.NotEqual(t, a, b, "random nonce must make ciphertexts differ")
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	enc, err := Encrypt("secret", newKey(t))
	require.NoError(t, err)
	_, err = Decrypt(enc, newKey(t))
	assert.Error(t, err, "decrypt with wrong key must fail")
}

func TestEncrypt_InvalidHexKey(t *testing.T) {
	_, err := Encrypt("x", "not-hex-zzz")
	assert.Error(t, err)
}

func TestDecrypt_InvalidHexKey(t *testing.T) {
	_, err := Decrypt("abcd", "not-hex-zzz")
	assert.Error(t, err)
}

func TestDecrypt_TruncatedCiphertext(t *testing.T) {
	_, err := Decrypt("ab", newKey(t)) // shorter than nonce
	assert.Error(t, err)
}
