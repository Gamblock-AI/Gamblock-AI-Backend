package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

)

func newTestMiddleware(t *testing.T) *Middleware {
	t.Helper()
	return New(nil, zap.NewNop())
}

func setupRouter(m *Middleware, path string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.POST(path, func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

// Bug #1: quick-approval resolve sends {token, status} — token is a forbidden
// key, so the guard must NOT reject it (the endpoint is token-authenticated).
func TestPrivacyGuard_QuickApprovalTokenAccepted(t *testing.T) {
	m := newTestMiddleware(t)
	r := setupRouter(m, "/v1/approval-requests/:id/resolve-by-token")

	body := []byte(`{"token":"abc","status":"approved"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/approval-requests/APR-1/resolve-by-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "quick-approval resolve must accept a token field")
}

func TestPrivacyGuard_QuickApprovalVerifyExempt(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.GET("/v1/approval-requests/verify/:token", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/v1/approval-requests/verify/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Bug #2: a reflection journal text that mentions a URL must be allowed (privacy
// is enforced via forbidden KEYS, not by censoring the user's reflective text).
func TestPrivacyGuard_ReflectionWithURLAccepted(t *testing.T) {
	m := newTestMiddleware(t)
	r := setupRouter(m, "/v1/reflections")

	body := []byte(`{"text":"saya hampir buka https://example.com tapi tahan","mood":"cemas"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/reflections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "reflection text may contain a URL")
}

// Regression: forbidden KEYS still rejected (url, domain, history, …).
func TestPrivacyGuard_ForbiddenKeyRejected(t *testing.T) {
	m := newTestMiddleware(t)
	r := setupRouter(m, "/v1/reflections")

	body := []byte(`{"url":"https://example.com","text":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/reflections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "privacy_payload_rejected")
}

func TestPrivacyGuard_AuthPathExempt(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.POST("/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })

	body := []byte(`{"email":"a@b.com","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// RequestID middleware sets/propagates X-Request-ID.
func TestRequestID_GeneratesAndEchoes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := newTestMiddleware(t)
	r := gin.New()
	r.Use(m.RequestID())
	r.GET("/x", func(c *gin.Context) {
		rid, _ := c.Get("request_id")
		c.String(200, rid.(string))
	})

	// No incoming header -> generated.
	req := httptest.NewRequest("GET", "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	// Incoming header -> echoed.
	req2 := httptest.NewRequest("GET", "/x", nil)
	req2.Header.Set("X-Request-ID", "my-id")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, "my-id", w2.Header().Get("X-Request-ID"))
}

// RequireRoles denies a missing/forbidden role.
func TestRequireRoles_Denies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := newTestMiddleware(t)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "user"); c.Next() })
	r.GET("/admin", m.RequireRoles("platform_admin"), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// RequireRoles allows a matching role.
func TestRequireRoles_Allows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := newTestMiddleware(t)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("role", "partner"); c.Next() })
	r.GET("/approve", m.RequireRoles("partner", "platform_admin"), func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// PrivacyGuard skips GET (no body inspection).
func TestPrivacyGuard_GetSkipped(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.GET("/v1/anything", func(c *gin.Context) { c.Status(200) })

	req := httptest.NewRequest("GET", "/v1/anything?domain=evil.com", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "GET must skip the guard")
}

// PrivacyGuard skips OPTIONS (preflight).
func TestPrivacyGuard_OptionsSkipped(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.OPTIONS("/v1/anything", func(c *gin.Context) { c.Status(204) })

	req := httptest.NewRequest("OPTIONS", "/v1/anything", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
