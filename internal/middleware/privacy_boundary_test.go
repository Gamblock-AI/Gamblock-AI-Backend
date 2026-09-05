package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPrivacyGuardRejectsForbiddenQueryKeyCaseInsensitively(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.RequestID(), m.PrivacyGuard())
	r.POST("/v1/aggregate-events", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/v1/aggregate-events?DoMaIn=example.com", nil)
	req.Header.Set("X-Request-ID", "privacy-query-test")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "privacy-query-test", response.Header().Get("X-Request-ID"))
	assert.Contains(t, response.Body.String(), `"code":"privacy_payload_rejected"`)
	assert.Contains(t, response.Body.String(), `"request_id":"privacy-query-test"`)
}

func TestPrivacyGuardRejectsForbiddenKeyInsideNestedArrays(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.POST("/v1/aggregate-events", func(c *gin.Context) { c.Status(http.StatusOK) })

	body := []byte(`{"events":[{"metadata":[{"DOM":"page heading"}]}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/aggregate-events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "privacy_payload_rejected")
}

func TestPrivacyGuardPreservesSafeJSONBodyForHandler(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.POST("/v1/reflections", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Data(http.StatusOK, "application/json", body)
	})

	body := []byte(`{"text":"I remembered https://example.com without sending a URL field"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/reflections", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, string(body), response.Body.String())
}

type failingPrivacyBody struct{}

func (failingPrivacyBody) Read([]byte) (int, error) { return 0, errors.New("body read failed") }
func (failingPrivacyBody) Close() error             { return nil }

func TestPrivacyGuardRejectsBodyReadFailure(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.POST("/v1/reflections", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/v1/reflections", nil)
	req.Body = failingPrivacyBody{}
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), `"code":"invalid_body"`)
}

func TestPrivacyGuardExemptsAdminContentURLs(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.PrivacyGuard())
	r.POST("/v1/admin/content/items", func(c *gin.Context) { c.Status(http.StatusOK) })

	body := []byte(`{"provider_url":"https://www.who.int/health-topics"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/content/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)

	assert.Equal(t, http.StatusOK, response.Code)
}

func TestPrivacyExemptPathBoundaries(t *testing.T) {
	cases := []struct {
		path   string
		exempt bool
	}{
		{path: "/v1/auth/login", exempt: true},
		{path: "/v1/me/password", exempt: true},
		{path: "/v1/devices/device-1/grant-key", exempt: true},
		{path: "/v1/devices//grant-key", exempt: false},
		{path: "/v1/devices/device-1/not-grant-key", exempt: false},
		{path: "/v1/admin/site-social-links", exempt: true},
		{path: "/v1/admin/content/items", exempt: true},
		{path: "/v1/approval-requests/verify/token", exempt: true},
		{path: "/v1/approval-requests/123/resolve-by-token", exempt: true},
		{path: "/v1/reflections", exempt: false},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.exempt, privacyExemptPath(tc.path))
		})
	}
}
