package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
)

type Middleware struct {
	auth   *service.AuthService
	logger *zap.Logger
}

func New(auth *service.AuthService, logger *zap.Logger) *Middleware {
	return &Middleware{auth: auth, logger: logger}
}

type envelope struct {
	Data      any       `json:"data"`
	Error     *apiError `json:"error"`
	RequestID string    `json:"request_id"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (m *Middleware) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

func (m *Middleware) PrivacyGuard() gin.HandlerFunc {
	forbidden := []string{"raw_url", "url", "domain", "dom", "screenshot", "browser_history", "history", "keystroke", "password", "otp", "token"}
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v1/auth/") {
			c.Next()
			return
		}
		// Quick-approval verify/resolve is token-authenticated (PRD §5.2) and its
		// body legitimately carries a "token" field. Exempt it from the guard so the
		// forbidden-key "token" does not block legitimate approvals.
		if strings.HasPrefix(c.Request.URL.Path, "/v1/approval-requests/verify/") ||
			strings.Contains(c.Request.URL.Path, "/resolve-by-token") {
			c.Next()
			return
		}
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		for key, values := range c.Request.URL.Query() {
			if unsafeKey(key, forbidden) || unsafeValue(strings.Join(values, " ")) {
				m.respondError(c, http.StatusBadRequest, "privacy_payload_rejected", "Request includes forbidden browsing or secret fields")
				c.Abort()
				return
			}
		}
		if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") && c.Request.Body != nil {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				m.respondError(c, http.StatusBadRequest, "invalid_body", "Unable to read request body")
				c.Abort()
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			if len(bytes.TrimSpace(body)) > 0 {
				var payload any
				if err := json.Unmarshal(body, &payload); err == nil && unsafePayload(payload, forbidden) {
					m.respondError(c, http.StatusBadRequest, "privacy_payload_rejected", "Request includes forbidden browsing or secret fields")
					c.Abort()
					return
				}
			}
		}
		c.Next()
	}
}

func (m *Middleware) AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}
		parsedClaims, err := m.auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err == nil {
			c.Set("user_id", parsedClaims.UserID)
			c.Set("email", parsedClaims.Email)
			c.Set("role", parsedClaims.Role)
		}
		c.Next()
	}
}

func (m *Middleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := c.Get("user_id"); ok {
			c.Next()
			return
		}
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			m.respondError(c, http.StatusUnauthorized, "auth_required", "Bearer token is required")
			c.Abort()
			return
		}
		parsedClaims, err := m.auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			m.respondError(c, http.StatusUnauthorized, "invalid_token", "Access token is invalid or expired")
			c.Abort()
			return
		}
		c.Set("user_id", parsedClaims.UserID)
		c.Set("email", parsedClaims.Email)
		c.Set("role", parsedClaims.Role)
		c.Next()
	}
}

func (m *Middleware) RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if _, ok := allowed[fmt.Sprint(role)]; ok {
			c.Next()
			return
		}
		m.respondError(c, http.StatusForbidden, "forbidden", "Role is not allowed for this action")
		c.Abort()
	}
}

func (m *Middleware) respondError(c *gin.Context, status int, code, message string) {
	reqID := uuid.NewString()
	if value, ok := c.Get("request_id"); ok {
		if id, ok := value.(string); ok {
			reqID = id
		}
	}
	c.JSON(status, envelope{Data: nil, Error: &apiError{Code: code, Message: message}, RequestID: reqID})
}

func unsafeKey(key string, forbidden []string) bool {
	lower := strings.ToLower(key)
	for _, term := range forbidden {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

// unsafeValue historically rejected values containing URLs or exceeding 4000
// chars. That censored legitimate reflective journal text (a user mentioning a
// URL) and broke the quick-approval flow's token field. Privacy is now enforced
// purely via forbidden KEYS (url, domain, dom, history, …) in unsafeKey; values
// are no longer censored. Kept as a stable hook in case future value-based
// rules are needed (return false = do not reject on value alone).
func unsafeValue(_ string) bool {
	return false
}

func unsafePayload(value any, forbidden []string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if unsafeKey(key, forbidden) || unsafePayload(nested, forbidden) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if unsafePayload(nested, forbidden) {
				return true
			}
		}
	case string:
		return unsafeValue(typed)
	}
	return false
}
