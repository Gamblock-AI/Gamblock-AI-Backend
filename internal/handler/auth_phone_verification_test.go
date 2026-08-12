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

// Builds a router in demo notification mode so the WhatsApp code is returned
// as a preview and the full public OTP flow can be exercised end-to-end.
func newPhoneVerificationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{
		AppEnv: "test", NotificationMode: "demo",
		JWTAccessSecret: "test-secret-very-long-please",
		JWTAccessTTL: 3600e9, JWTRefreshTTL: 720 * 3600e9,
	}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	services := service.NewContainer(repo, cfg, zap.NewNop())
	mid := middleware.New(services.Auth, zap.NewNop())
	h := New(services, mid, cfg, zap.NewNop())

	r := gin.New()
	r.Use(gin.Recovery(), mid.RequestID(), mid.PrivacyGuard())
	v1 := r.Group("/v1")
	v1.Use(mid.AuthOptional())
	v1.POST("/auth/register", h.Register)
	v1.POST("/auth/login", h.Login)
	v1.POST("/auth/phone-verification/verify", h.VerifyPhoneVerification)
	v1.POST("/auth/phone-verification/verify/resend", h.ResendPhoneVerification)
	return r
}

func postJSON(r *gin.Engine, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPhoneVerificationFlow_RegisterVerifyThenLogin(t *testing.T) {
	r := newPhoneVerificationRouter(t)

	// Registration returns a verification token and demo preview code, no session.
	register := postJSON(r, "/v1/auth/register", []byte(`{"email":"otp@example.com","password":"password2","name":"Otp User","phone":"+6281200000099","role":"user"}`))
	require.Equal(t, http.StatusCreated, register.Code)
	var registerEnv envelopeShape
	require.NoError(t, json.Unmarshal(register.Body.Bytes(), &registerEnv))
	data := registerEnv.Data.(map[string]any)
	assert.Equal(t, true, data["verification_required"])
	assert.NotEmpty(t, data["verification_token"])
	assert.NotEmpty(t, data["phone_verification_preview_code"])
	assert.Empty(t, data["access_token"])

	token := data["verification_token"].(string)
	code := data["phone_verification_preview_code"].(string)

	// The unverified account cannot log in (no session is issued).
	login := postJSON(r, "/v1/auth/login", []byte(`{"email":"otp@example.com","password":"password2"}`))
	require.Equal(t, http.StatusOK, login.Code)
	var loginEnv envelopeShape
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &loginEnv))
	loginData := loginEnv.Data.(map[string]any)
	assert.Equal(t, true, loginData["verification_required"])
	assert.Empty(t, loginData["access_token"])

	// Public verify completes the flow without a bearer session.
	verify := postJSON(r, "/v1/auth/phone-verification/verify", []byte(`{"verification_token":"`+token+`","code":"`+code+`"}`))
	require.Equal(t, http.StatusOK, verify.Code)

	// Now login succeeds with a session.
	login2 := postJSON(r, "/v1/auth/login", []byte(`{"email":"otp@example.com","password":"password2"}`))
	require.Equal(t, http.StatusOK, login2.Code)
	var login2Env envelopeShape
	require.NoError(t, json.Unmarshal(login2.Body.Bytes(), &login2Env))
	login2Data := login2Env.Data.(map[string]any)
	assert.Equal(t, false, login2Data["verification_required"])
	assert.NotEmpty(t, login2Data["access_token"])
}

func TestPhoneVerificationFlow_WrongCode(t *testing.T) {
	r := newPhoneVerificationRouter(t)
	register := postJSON(r, "/v1/auth/register", []byte(`{"email":"otp2@example.com","password":"password2","name":"Otp Two","phone":"+6281200000098","role":"user"}`))
	require.Equal(t, http.StatusCreated, register.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(register.Body.Bytes(), &env))
	data := env.Data.(map[string]any)

	verify := postJSON(r, "/v1/auth/phone-verification/verify", []byte(`{"verification_token":"`+data["verification_token"].(string)+`","code":"000000"}`))
	require.Equal(t, http.StatusBadRequest, verify.Code)
	var verifyEnv envelopeShape
	require.NoError(t, json.Unmarshal(verify.Body.Bytes(), &verifyEnv))
	assert.Equal(t, "phone_verification_failed", verifyEnv.Error.Code)
}

func TestPhoneVerificationFlow_Resend(t *testing.T) {
	r := newPhoneVerificationRouter(t)
	register := postJSON(r, "/v1/auth/register", []byte(`{"email":"otp3@example.com","password":"password2","name":"Otp Three","phone":"+6281200000097","role":"user"}`))
	require.Equal(t, http.StatusCreated, register.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(register.Body.Bytes(), &env))
	data := env.Data.(map[string]any)

	resend := postJSON(r, "/v1/auth/phone-verification/verify/resend", []byte(`{"verification_token":"`+data["verification_token"].(string)+`"}`))
	require.Equal(t, http.StatusOK, resend.Code)
	var resendEnv envelopeShape
	require.NoError(t, json.Unmarshal(resend.Body.Bytes(), &resendEnv))
	resendData := resendEnv.Data.(map[string]any)
	assert.Equal(t, true, resendData["sent"])
	assert.NotEmpty(t, resendData["preview_code"])
}

func TestPhoneVerificationFlow_RequiresToken(t *testing.T) {
	r := newPhoneVerificationRouter(t)
	verify := postJSON(r, "/v1/auth/phone-verification/verify", []byte(`{"verification_token":"","code":"123456"}`))
	require.Equal(t, http.StatusBadRequest, verify.Code)
	var env envelopeShape
	require.NoError(t, json.Unmarshal(verify.Body.Bytes(), &env))
	assert.Equal(t, "err_validation", env.Error.Code)
}
