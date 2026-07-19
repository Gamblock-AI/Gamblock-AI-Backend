package config

import "testing"

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

func TestValidateAllowsMissingOptionalDeliveryIntegrationsInProduction(t *testing.T) {
	cfg := Config{
		AppEnv:               "production",
		DatabaseURL:          "postgres://gamblock@example/gamblock",
		JWTAccessSecret:      "0123456789abcdef0123456789abcdef",
		JournalEncryptionKey: validTestEncryptionKey,
		NotificationMode:     "production",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate rejected production without optional delivery providers: %v", err)
	}
}
