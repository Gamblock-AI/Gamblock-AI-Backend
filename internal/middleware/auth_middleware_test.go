package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/service"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testJWTSecret = "test-secret-very-long-please-32bytes!"

func newAuthTestMiddleware(t *testing.T) (*Middleware, *store.Store, string) {
	t.Helper()
	cfg := config.Config{
		AppEnv:          "test",
		JWTAccessSecret: testJWTSecret,
		JWTAccessTTL:    time.Hour,
		JWTRefreshTTL:   24 * time.Hour,
	}
	st := store.NewSeeded()
	repo := repository.New(nil, st)
	auth := service.NewAuthService(repo, cfg, zap.NewNop())
	return New(auth, zap.NewNop()), st, cfg.JWTAccessSecret
}

func signedTestAccessToken(t *testing.T, secret, userID, email, role string, authTime time.Time) string {
	t.Helper()
	now := time.Now().UTC()
	claims := model.Claims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		AuthTime: jwt.NewNumericDate(authTime),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			Issuer:    "gamblock-ai-backend",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return token
}

func setUserState(t *testing.T, st *store.Store, userID string, mutate func(*model.User)) {
	t.Helper()
	st.Lock()
	defer st.Unlock()
	for index := range st.Users {
		if st.Users[index].ID == userID {
			mutate(&st.Users[index])
			return
		}
	}
	t.Fatalf("user %q not found", userID)
}

func TestAuthRequiredRejectsMissingAndMalformedAuthorization(t *testing.T) {
	m, _, secret := newAuthTestMiddleware(t)
	cases := []struct {
		name      string
		header    string
		status    int
		errorCode string
	}{
		{name: "missing", status: http.StatusUnauthorized, errorCode: "auth_required"},
		{name: "non bearer", header: "Basic abc", status: http.StatusUnauthorized, errorCode: "auth_required"},
		{name: "invalid bearer", header: "Bearer not-a-jwt", status: http.StatusUnauthorized, errorCode: "invalid_token"},
		{name: "wrong signing secret", header: "Bearer " + signedTestAccessToken(t, secret+"wrong", "usr_gading", "gading@gmail.com", "user", time.Now().UTC()), status: http.StatusUnauthorized, errorCode: "invalid_token"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(m.AuthRequired())
			r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)

			assert.Equal(t, tc.status, response.Code)
			assert.Contains(t, response.Body.String(), tc.errorCode)
		})
	}
}

func TestAuthRequiredAcceptsActiveIdentityAndRefreshesRole(t *testing.T) {
	m, _, secret := newAuthTestMiddleware(t)
	// Deliberately put a stale role in the token. The role from ActiveIdentity
	// must be authoritative for every bearer request.
	token := signedTestAccessToken(t, secret, "usr_suci", "suci@gmail.com", "user", time.Now().UTC())

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(m.AuthRequired())
	r.GET("/protected", func(c *gin.Context) {
		userID, userIDOK := c.Get("user_id")
		email, emailOK := c.Get("email")
		role, roleOK := c.Get("role")
		_, authTimeOK := c.Get("auth_time")
		c.String(http.StatusOK, fmt.Sprintf("%s|%s|%s|%t|%t|%t|%t", userID, email, role, userIDOK, emailOK, roleOK, authTimeOK))
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "usr_suci|suci@gmail.com|partner|true|true|true|true", response.Body.String())
}

func TestAuthRequiredBypassesWhenEarlierMiddlewareSetUser(t *testing.T) {
	m := newTestMiddleware(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "already-authenticated")
		c.Next()
	})
	r.Use(m.AuthRequired())
	r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))

	assert.Equal(t, http.StatusOK, response.Code)
}

func TestAuthRequiredRejectsInactiveAndPasswordChangeRequiredUsers(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*model.User)
	}{
		{name: "disabled", mutate: func(user *model.User) {
			now := time.Now().UTC()
			user.DisabledAt = &now
		}},
		{name: "must change password", mutate: func(user *model.User) {
			user.MustChangePassword = true
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, st, secret := newAuthTestMiddleware(t)
			setUserState(t, st, "usr_gading", tc.mutate)
			token := signedTestAccessToken(t, secret, "usr_gading", "gading@gmail.com", "user", time.Now().UTC())

			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(m.AuthRequired())
			r.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)

			assert.Equal(t, http.StatusUnauthorized, response.Code)
			assert.Contains(t, response.Body.String(), "invalid_token")
		})
	}
}

func TestAuthOptionalIgnoresMissingInvalidAndInactiveTokens(t *testing.T) {
	cases := []struct {
		name     string
		header   string
		inactive bool
		expected string
	}{
		{name: "missing", expected: "anonymous"},
		{name: "non bearer", header: "Basic abc", expected: "anonymous"},
		{name: "invalid bearer", header: "Bearer invalid", expected: "anonymous"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, _, _ := newAuthTestMiddleware(t)
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(m.AuthOptional())
			r.GET("/optional", func(c *gin.Context) {
				if _, ok := c.Get("user_id"); !ok {
					c.String(http.StatusOK, "anonymous")
					return
				}
				c.String(http.StatusOK, "authenticated")
			})

			req := httptest.NewRequest(http.MethodGet, "/optional", nil)
			req.Header.Set("Authorization", tc.header)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			assert.Equal(t, tc.expected, response.Body.String())
		})
	}

	t.Run("inactive bearer remains optional", func(t *testing.T) {
		m, st, secret := newAuthTestMiddleware(t)
		setUserState(t, st, "usr_gading", func(user *model.User) {
			now := time.Now().UTC()
			user.DisabledAt = &now
		})
		token := signedTestAccessToken(t, secret, "usr_gading", "gading@gmail.com", "user", time.Now().UTC())

		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(m.AuthOptional())
		r.GET("/optional", func(c *gin.Context) {
			_, ok := c.Get("user_id")
			assert.False(t, ok)
			c.Status(http.StatusOK)
		})
		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		r.ServeHTTP(response, req)
		assert.Equal(t, http.StatusOK, response.Code)
	})
}

func TestRequireRecentAuthEnforcesTimestampAndType(t *testing.T) {
	cases := []struct {
		name      string
		value     any
		status    int
		errorCode string
	}{
		{name: "missing", status: http.StatusUnauthorized, errorCode: "recent_auth_required"},
		{name: "wrong type", value: "recent", status: http.StatusUnauthorized, errorCode: "recent_auth_required"},
		{name: "stale", value: time.Now().Add(-2 * time.Hour), status: http.StatusUnauthorized, errorCode: "recent_auth_required"},
		{name: "fresh", value: time.Now(), status: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestMiddleware(t)
			gin.SetMode(gin.TestMode)
			r := gin.New()
			if tc.value != nil {
				r.Use(func(c *gin.Context) {
					c.Set("auth_time", tc.value)
					c.Next()
				})
			}
			r.Use(m.RequireRecentAuth(time.Minute))
			r.GET("/sensitive", func(c *gin.Context) { c.Status(http.StatusOK) })

			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/sensitive", nil))
			assert.Equal(t, tc.status, response.Code)
			if tc.errorCode != "" {
				assert.Contains(t, response.Body.String(), tc.errorCode)
			}
		})
	}
}

func TestRequireVerifiedPhoneAllowsOnlyActiveVerifiedUsers(t *testing.T) {
	cases := []struct {
		name   string
		userID string
		mutate func(*model.User)
		status int
	}{
		{name: "verified", userID: "usr_gading", status: http.StatusOK},
		{name: "unverified", userID: "usr_dery", mutate: func(user *model.User) {
			user.PhoneVerifiedAt = nil
		}, status: http.StatusForbidden},
		{name: "disabled", userID: "usr_gading", mutate: func(user *model.User) {
			now := time.Now().UTC()
			user.DisabledAt = &now
		}, status: http.StatusForbidden},
		{name: "unknown", userID: "missing", status: http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, st, _ := newAuthTestMiddleware(t)
			if tc.mutate != nil {
				setUserState(t, st, tc.userID, tc.mutate)
			}
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(func(c *gin.Context) {
				c.Set("user_id", tc.userID)
				c.Next()
			})
			r.Use(m.RequireVerifiedPhone())
			r.GET("/phone-protected", func(c *gin.Context) { c.Status(http.StatusOK) })

			response := httptest.NewRecorder()
			r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/phone-protected", nil))
			assert.Equal(t, tc.status, response.Code)
			if tc.status != http.StatusOK {
				assert.Contains(t, response.Body.String(), "phone_verification_required")
			}
		})
	}
}
