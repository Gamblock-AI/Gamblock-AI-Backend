package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWhatsAppServiceFonnteRequestContract(t *testing.T) {
	const token = "fonnte-test-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read request body: %v", readErr)
			return
		}
		assert.NotContains(t, string(raw), token, "provider token must stay in the authorization header")
		r.Body = io.NopCloser(bytes.NewReader(raw))
		if err := r.ParseMultipartForm(64 * 1024); err != nil {
			t.Errorf("parse multipart form: %v", err)
			return
		}
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/send", r.URL.Path)
		assert.Equal(t, token, r.Header.Get("Authorization"))
		assert.Equal(t, "6281234567890", r.FormValue("target"))
		assert.Contains(t, r.FormValue("message"), "123456")
		assert.Equal(t, "0", r.FormValue("countryCode"))
		assert.Equal(t, "false", r.FormValue("preview"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"detail":"queued"}`))
	}))
	defer server.Close()

	cfg := testCfg()
	cfg.NotificationMode = "production"
	cfg.FonnteToken = token
	cfg.FonnteBaseURL = server.URL
	cfg.FonnteCountryCode = "62"
	svc := NewWhatsAppService(cfg, zap.NewNop())
	svc.client = server.Client()

	err := svc.SendPhoneVerification(context.Background(), "081234567890", "123456")
	require.NoError(t, err)
}

func TestWhatsAppServiceFonnteRejectsProviderFailures(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "http failure", status: http.StatusBadGateway, body: `{"status":false}`},
		{name: "provider rejection", status: http.StatusOK, body: `{"status":false}`},
		{name: "malformed response", status: http.StatusOK, body: `not-json`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			cfg := testCfg()
			cfg.NotificationMode = "production"
			cfg.FonnteToken = "fonnte-test-token"
			cfg.FonnteBaseURL = server.URL
			svc := NewWhatsAppService(cfg, zap.NewNop())
			svc.client = server.Client()

			err := svc.SendPasswordReset(context.Background(), "+628123456789", "reset-code")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "delivery")
		})
	}
}

func TestWhatsAppServiceFonnteTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	cfg := testCfg()
	cfg.NotificationMode = "production"
	cfg.FonnteToken = "fonnte-test-token"
	cfg.FonnteBaseURL = server.URL
	svc := NewWhatsAppService(cfg, zap.NewNop())
	svc.client = &http.Client{Timeout: 20 * time.Millisecond}

	err := svc.SendEmergencyKey(context.Background(), "081234567890", strings.Repeat("safe", 3))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivery")
}
