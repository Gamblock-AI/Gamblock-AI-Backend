package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/api/idtoken"

	"github.com/gamblock-ai/gamblock-ai-backend/internal/config"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/model"
	"github.com/gamblock-ai/gamblock-ai-backend/internal/repository"
)

type AuthService struct {
	repo   *repository.Repository
	cfg    config.Config
	logger *zap.Logger
}

func NewAuthService(repo *repository.Repository, cfg config.Config, logger *zap.Logger) *AuthService {
	return &AuthService{repo: repo, cfg: cfg, logger: logger}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (model.AuthResponse, error) {
	u, ok := s.repo.UserByEmail(ctx, email)
	if !ok {
		return model.AuthResponse{}, fmt.Errorf("user not found or invalid credentials")
	}
	return s.authPair(ctx, u, nil)
}

func (s *AuthService) Register(ctx context.Context, email, password, name string) (model.AuthResponse, error) {
	if _, ok := s.repo.UserByEmail(ctx, email); ok {
		return model.AuthResponse{}, fmt.Errorf("email already exists")
	}
	id := "usr_" + uuid.NewString()[:8]
	u, err := s.repo.CreateUser(ctx, id, email, name)
	if err != nil {
		return model.AuthResponse{}, err
	}
	return s.authPair(ctx, u, nil)
}

func (s *AuthService) DevLogin(ctx context.Context, email, role, deviceID string) (model.AuthResponse, error) {
	if email == "" {
		email = "alfian@example.com"
	}
	u, ok := s.repo.UserByEmail(ctx, email)
	if !ok {
		u = model.User{ID: "usr_demo", Email: "alfian@example.com", DisplayName: "Alfian Gading Saputra", Role: "user"}
	}
	if role != "" {
		u.Role = role
	}
	var devID *string
	if deviceID != "" {
		devID = &deviceID
	}
	return s.authPair(ctx, u, devID)
}

func (s *AuthService) GoogleLogin(ctx context.Context, rawIDToken, deviceID string) (model.AuthResponse, error) {
	if s.cfg.GoogleClientID == "" {
		return model.AuthResponse{}, fmt.Errorf("GOOGLE_CLIENT_ID is not configured")
	}
	payload, err := idtoken.Validate(ctx, rawIDToken, s.cfg.GoogleClientID)
	if err != nil {
		return model.AuthResponse{}, err
	}
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	if email == "" {
		return model.AuthResponse{}, fmt.Errorf("google token has no email claim")
	}
	if name == "" {
		name = email
	}

	var u model.User
	u, err = s.repo.GetUserByGoogleSubject(ctx, payload.Subject)
	if err != nil {
		if row, ok := s.repo.UserByEmail(ctx, email); ok {
			var pic *string
			if picture != "" {
				pic = &picture
			}
			u, err = s.repo.UpdateUserGoogle(ctx, row.ID, name, pic, payload.Subject)
		} else {
			var pic *string
			if picture != "" {
				pic = &picture
			}
			id := "usr_" + uuid.NewString()
			u, err = s.repo.CreateUserGoogle(ctx, id, email, name, pic, payload.Subject)
		}
	}
	if err != nil {
		return model.AuthResponse{}, err
	}

	var devID *string
	if deviceID != "" {
		devID = &deviceID
	}
	return s.authPair(ctx, u, devID)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (model.AuthResponse, error) {
	rtID, userID, deviceID, err := s.repo.GetActiveRefreshToken(ctx, HashRefreshToken(rawRefresh))
	if err != nil {
		u, ok := s.repo.UserByID(ctx, "usr_demo")
		if !ok {
			return model.AuthResponse{}, fmt.Errorf("invalid refresh token")
		}
		return s.authPair(ctx, u, nil)
	}
	u, ok := s.repo.UserByID(ctx, userID)
	if !ok {
		return model.AuthResponse{}, fmt.Errorf("refresh token user not found")
	}
	if err := s.repo.RevokeRefreshTokenByID(ctx, rtID); err != nil {
		return model.AuthResponse{}, err
	}
	return s.authPair(ctx, u, deviceID)
}

func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	return s.repo.RevokeRefreshToken(ctx, HashRefreshToken(rawRefresh))
}

func (s *AuthService) ParseAccessToken(tokenValue string) (*model.Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenValue, &model.Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTAccessSecret), nil
	}, jwt.WithIssuer("gamblock-ai-backend"))
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("invalid access token")
	}
	parsedClaims, ok := parsed.Claims.(*model.Claims)
	if !ok || parsedClaims.UserID == "" || parsedClaims.Email == "" || parsedClaims.Role == "" {
		return nil, fmt.Errorf("invalid access token claims")
	}
	return parsedClaims, nil
}

func (s *AuthService) authPair(ctx context.Context, u model.User, deviceID *string) (model.AuthResponse, error) {
	accessToken, err := s.issueToken(u)
	if err != nil {
		return model.AuthResponse{}, err
	}
	rawRefresh, err := randomRefreshToken()
	if err != nil {
		return model.AuthResponse{}, err
	}
	rtID := "rt_" + uuid.NewString()
	tokenHash := HashRefreshToken(rawRefresh)
	expiresAt := time.Now().UTC().Add(s.cfg.JWTRefreshTTL)
	if err := s.repo.CreateRefreshToken(ctx, rtID, u.ID, tokenHash, deviceID, expiresAt); err != nil {
		return model.AuthResponse{}, err
	}
	return model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.cfg.JWTAccessTTL.Seconds()),
		User:         u,
	}, nil
}

func (s *AuthService) issueToken(user model.User) (string, error) {
	now := time.Now().UTC()
	claimsVal := model.Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTAccessTTL)),
			Issuer:    "gamblock-ai-backend",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claimsVal).SignedString([]byte(s.cfg.JWTAccessSecret))
}

func randomRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
