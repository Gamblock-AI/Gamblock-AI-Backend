package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
)

func (m *Middleware) AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.Next()
			return
		}
		if claims, err := m.auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer ")); err == nil {
			if role, active := m.auth.ActiveIdentity(c.Request.Context(), claims.UserID); active {
				setAuthenticatedContext(c, claims)
				c.Set("role", role)
			}
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
			m.respondError(c, http.StatusUnauthorized, "auth_required")
			c.Abort()
			return
		}
		claims, err := m.auth.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			m.respondError(c, http.StatusUnauthorized, "invalid_token")
			c.Abort()
			return
		}
		role, active := m.auth.ActiveIdentity(c.Request.Context(), claims.UserID)
		if !active {
			m.respondError(c, http.StatusUnauthorized, "invalid_token")
			c.Abort()
			return
		}
		setAuthenticatedContext(c, claims)
		c.Set("role", role)
		c.Next()
	}
}

func (m *Middleware) RequireRecentAuth(maxAge time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		authTime, ok := c.Get("auth_time")
		issuedAt, valid := authTime.(time.Time)
		if !ok || !valid || time.Since(issuedAt) > maxAge {
			m.respondError(c, http.StatusUnauthorized, "recent_auth_required")
			c.Abort()
			return
		}
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
		m.respondError(c, http.StatusForbidden, "forbidden")
		c.Abort()
	}
}

func setAuthenticatedContext(c *gin.Context, claims *model.Claims) {
	c.Set("user_id", claims.UserID)
	c.Set("email", claims.Email)
	c.Set("role", claims.Role)
	if claims.AuthTime != nil {
		c.Set("auth_time", claims.AuthTime.Time)
	}
}
