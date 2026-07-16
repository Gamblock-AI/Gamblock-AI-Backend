package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var forbiddenPrivacyKeys = []string{
	"raw_url", "url", "domain", "dom", "screenshot", "browser_history",
	"history", "keystroke", "password", "otp", "token",
}

func (m *Middleware) PrivacyGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if privacyExemptPath(c.Request.URL.Path) || c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		for key, values := range c.Request.URL.Query() {
			if unsafeKey(key, forbiddenPrivacyKeys) || unsafeValue(strings.Join(values, " ")) {
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
				if err := json.Unmarshal(body, &payload); err == nil && unsafePayload(payload, forbiddenPrivacyKeys) {
					m.respondError(c, http.StatusBadRequest, "privacy_payload_rejected", "Request includes forbidden browsing or secret fields")
					c.Abort()
					return
				}
			}
		}
		c.Next()
	}
}

func privacyExemptPath(path string) bool {
	if strings.HasPrefix(path, "/v1/auth/") {
		return true
	}
	// Quick approvals use a single-use token in their body. The token is an
	// authorization credential, not browsing data, so these routes are exempt.
	return strings.HasPrefix(path, "/v1/approval-requests/verify/") ||
		strings.Contains(path, "/resolve-by-token")
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

// unsafeValue deliberately does not censor user-supplied content. A journal
// may legitimately mention a URL; privacy is enforced on transport keys.
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
