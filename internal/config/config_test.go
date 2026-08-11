package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"testing"
)

const validTestEncryptionKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func TestIsProduction(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"PRODUCTION", true},
		{"", true}, // safe default = production
		{"anything", true},
		{"development", false},
		{"staging", false},
		{"test", false},
		{"local", false},
	}
	for _, tc := range cases {
		c := Config{AppEnv: tc.env}
		assert := func(got bool) {
			if got != tc.want {
				t.Errorf("AppEnv=%q: got %v want %v", tc.env, got, tc.want)
			}
		}
		assert(c.IsProduction())
	}
}

func TestValidateRequiresEncryptionKeyInDevelopment(t *testing.T) {
	cfg := Config{AppEnv: "development"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate should reject an empty JOURNAL_ENCRYPTION_KEY")
	}
	cfg.JournalEncryptionKey = validTestEncryptionKey
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected a valid development encryption key: %v", err)
	}
}

func TestValidateRequiresFonnteInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:                           "production",
		DatabaseURL:                      "postgres://gamblock@example/gamblock",
		JWTAccessSecret:                  "0123456789abcdef0123456789abcdef",
		JournalEncryptionKey:             validTestEncryptionKey,
		NotificationMode:                 "production",
		ProtectionGrantSigningPrivateKey: validProtectionGrantSigningKey(),
		ProtectionGrantSigningKeyID:      "grant-key-test",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate should require FONNTE_TOKEN in production")
	}
	cfg.FonnteToken = "test-token"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected production with Fonnte: %v", err)
	}
}

func TestValidateRequiresProtectionGrantSigningKeyInProduction(t *testing.T) {
	cfg := Config{AppEnv: "production", JournalEncryptionKey: validTestEncryptionKey}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate should require protection grant signing configuration in production")
	}
	cfg.ProtectionGrantSigningPrivateKey = validProtectionGrantSigningKey()
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate should require a protection grant signing key ID")
	}
}

func validProtectionGrantSigningKey() string {
	curve := elliptic.P256()
	d := big.NewInt(1)
	x, y := curve.ScalarBaseMult(d.Bytes())
	privateKey := &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y}, D: d}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(der)
}
