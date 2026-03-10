package application

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByProviderID(ctx context.Context, provider, providerID string) (*domain.User, error)

	Create(ctx context.Context, user *domain.User) error
}

type SessionRepository interface {
	IsRefreshTokenValid(ctx context.Context, token string) (uuid.UUID, error)
	SaveRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiry time.Time) error
	RevokeRefreshToken(ctx context.Context, token string) error
}

type AuthConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration // 15 min
	RefreshTokenTTL time.Duration // 7 days

	PublicKey  *ecdsa.PublicKey
	PrivateKey *ecdsa.PrivateKey

	GoogleClientID     string
	GoogleClientSecret string
}

type AuthService struct {
	users   UserRepository
	session SessionRepository
	config  AuthConfig
}

func NewAuthService(users UserRepository, session SessionRepository, config AuthConfig) *AuthService {
	return &AuthService{users: users, session: session, config: config}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrTokenExpired            = errors.New("expired token")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidClaims           = errors.New("invalid claims")
	ErrUnexpectedSigningMethod = errors.New("unexpected signing method")
	ErrInvalidSubClaim         = errors.New("invalid sub claim")
	ErrInvalidUserID           = errors.New("invalid user id in token")
)

// === Local ===
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Provider:     "local",
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := s.session.IsRefreshTokenValid(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if err := s.session.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, userID)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.session.RevokeRefreshToken(ctx, refreshToken)
}

func (s *AuthService) ValidateAccessToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, ErrUnexpectedSigningMethod
		}
		return s.config.PublicKey, nil
	})
	if err != nil || !token.Valid {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrTokenExpired
		}
		return uuid.Nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, ErrInvalidClaims
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, ErrInvalidSubClaim
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, ErrInvalidUserID
	}

	return userID, nil
}

// === Google OAuth ===
func (s *AuthService) LoginWithGoogle(ctx context.Context, googleUserID, email string) (*TokenPair, error) {
	user, err := s.users.FindByProviderID(ctx, "google", googleUserID)
	if err != nil {
		// first time, create user
		user = &domain.User{
			Email:      email,
			Provider:   "google",
			ProviderID: googleUserID,
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, err
		}
	}

	return s.issueTokens(ctx, user.ID)
}

// === Internal helpers ===
func (s *AuthService) issueTokens(ctx context.Context, userID uuid.UUID) (*TokenPair, error) {
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	expiry := time.Now().Add(s.config.RefreshTokenTTL)
	if err := s.session.SaveRefreshToken(ctx, userID, refreshToken, expiry); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthService) generateAccessToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(s.config.AccessTokenTTL).Unix(),
		"iat": time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(s.config.PrivateKey)
}

func (s *AuthService) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
