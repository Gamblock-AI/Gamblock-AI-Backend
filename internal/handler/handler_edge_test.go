package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/middleware"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
)

func newPartnerRouter(t *testing.T, appEnv ...string) (*gin.Engine, string) {
	env := "test"
	if len(appEnv) > 0 {
		env = appEnv[0]
	}
	_ = env
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv: env, JWTAccessSecret: "test-secret-very-long-please", JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
		JournalEncryptionKey: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		ExportStoragePath:    t.TempDir(), NotificationMode: "demo", PublicWebBaseURL: "http://localhost:3000",
	}
	st := store.NewSeeded()
	st.JournalEntries = nil
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(gin.Recovery(), mid.RequestID(), mid.PrivacyGuard())
	v1 := r.Group("/v1")
	v1.Use(mid.AuthOptional())
	v1.POST("/auth/login", h.Login)
	v1.GET("/partners", mid.AuthRequired(), h.GetPartners)
	v1.POST("/partners/invitations", mid.AuthRequired(), h.CreatePartnerInvitation)
	v1.GET("/support-cases", mid.AuthRequired(), h.GetSupportCases)
	v1.POST("/support-cases", mid.AuthRequired(), h.CreateSupportCase)
	v1.GET("/data-requests", mid.AuthRequired(), h.GetDataRequests)
	v1.POST("/data-requests", mid.AuthRequired(), h.CreateDataRequest)
	v1.GET("/me", mid.AuthRequired(), h.GetProfile)
	v1.GET("/psychoeducation/modules/:slug", mid.AuthRequired(), h.GetModuleDetail)
	return r, loginToken(t, r)
}

func TestHandler_GetPartners(t *testing.T) {
	r, token := newPartnerRouter(t)
	w := authedGet(r, "/v1/partners", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CreatePartnerInvitation(t *testing.T) {
	r, token := newPartnerRouter(t)
	body := []byte(`{"email":"newpartner@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/partners/invitations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreatePartnerInvitation_EmailRequired(t *testing.T) {
	r, token := newPartnerRouter(t, "production")
	body := []byte(`{"email":""}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/partners/invitations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateSupportCase(t *testing.T) {
	r, token := newPartnerRouter(t)
	body := []byte(`{"summary":"tidak bisa login","type":"device_recovery","priority":"normal"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/support-cases", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateSupportCase_SummaryRequired(t *testing.T) {
	r, token := newPartnerRouter(t, "production")
	body := []byte(`{"summary":""}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/support-cases", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateDataRequest(t *testing.T) {
	r, token := newPartnerRouter(t)
	body := []byte(`{"type":"export"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/data-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateDataRequest_Conflict(t *testing.T) {
	r, token := newPartnerRouter(t)
	body := []byte(`{"type":"delete"}`)
	for index, expected := range []int{http.StatusCreated, http.StatusConflict} {
		req := httptest.NewRequest(http.MethodPost, "/v1/data-requests", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, expected, w.Code, "request %d", index+1)
	}
}

func TestHandler_GetDataRequests(t *testing.T) {
	r, token := newPartnerRouter(t)
	w := authedGet(r, "/v1/data-requests", token)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetProfileIncludesPasswordCapability(t *testing.T) {
	r, token := newPartnerRouter(t)
	w := authedGet(r, "/v1/me", token)
	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data struct {
			PasswordEnabled bool `json:"password_enabled"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Data.PasswordEnabled)
}

func TestHandler_GetModuleDetail_NotFound(t *testing.T) {
	r, token := newPartnerRouter(t)
	w := authedGet(r, "/v1/psychoeducation/modules/no-such-slug", token)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetModuleDetail_Found(t *testing.T) {
	r, token := newPartnerRouter(t)
	w := authedGet(r, "/v1/psychoeducation/modules/memahami-siklus-dorongan", token)
	assert.Equal(t, http.StatusOK, w.Code)
}
