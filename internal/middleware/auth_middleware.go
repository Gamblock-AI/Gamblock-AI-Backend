package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (m *Middleware) AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}
		if claims, err := m.auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer ")); err == nil {
			setAuthenticatedContext(c, claims.UserID, claims.Email, claims.Role)
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
		claims, err := m.auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			m.respondError(c, http.StatusUnauthorized, "invalid_token", "Access token is invalid or expired")
			c.Abort()
			return
		}
		setAuthenticatedContext(c, claims.UserID, claims.Email, claims.Role)
		c.Next()
	}
}

func (m *Middleware) RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
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

func setAuthenticatedContext(c *gin.Context, userID, email, role string) {
	c.Set("user_id", userID)
	c.Set("email", email)
	c.Set("role", role)
}
