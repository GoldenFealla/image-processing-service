package application

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
)

type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

type UserIdentityRepository interface {
	Create(ctx context.Context, userID uuid.UUID, Provider string, ProviderID string) error
	FindByProvider(ctx context.Context, Provider string, ProviderID string) (*domain.UserIdentity, error)
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
}

type AuthUseCase interface {
	Login(ctx context.Context, form *domain.LoginForm) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Register(ctx context.Context, form *domain.RegisterForm) (*TokenPair, error)
	ValidateAccessToken(tokenStr string) (uuid.UUID, error)

	GetGoogleAuthURL(state string) string
	HandleGoogleCallback(ctx context.Context, code, state string) (*TokenPair, error)
}

type AuthService struct {
	users      UserRepository
	identities UserIdentityRepository
	session    SessionRepository
	config     AuthConfig
	oauth      domain.OAuthRepository
}

func NewAuthService(
	users UserRepository,
	identities UserIdentityRepository,
	session SessionRepository,
	config AuthConfig,
	oauth domain.OAuthRepository,
) *AuthService {
	return &AuthService{
		users:      users,
		identities: identities,
		session:    session,
		config:     config,
		oauth:      oauth,
	}
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
func (s *AuthService) Login(ctx context.Context, form *domain.LoginForm) (*TokenPair, error) {
	user, err := s.users.FindByUsernameOrEmail(ctx, form.UsernameOrEmail)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(form.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *AuthService) Register(ctx context.Context, form *domain.RegisterForm) (*TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Username:     form.Username,
		Email:        form.Email,
		PasswordHash: string(hash),
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
func (s *AuthService) GetGoogleAuthURL(state string) string {
	return s.oauth.GetAuthURL(state)
}

func (s *AuthService) HandleGoogleCallback(ctx context.Context, code, state string) (*TokenPair, error) {
	info, err := s.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	identity, err := s.identities.FindByProvider(ctx, info.Provider, info.ProviderID)
	if err != nil && !errors.Is(err, domain.ErrIdentityNotFound) {
		return nil, fmt.Errorf("failed to find identity: %w", err)
	}

	var user *domain.User
	if identity == nil {
		user, err = s.users.FindByEmail(ctx, info.Email)
		if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
			return nil, fmt.Errorf("failed to find or create user: %w", err)
		}

		user.Email = info.Email
		user.Username = info.Name

		if err = s.users.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		if err = s.identities.Create(ctx, user.ID, info.Provider, info.ProviderID); err != nil {
			return nil, fmt.Errorf("failed to create identity: %w", err)
		}
	} else {
		user, err = s.users.FindByID(ctx, identity.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to find user: %w", err)
		}
	}

	tokenPairs, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenPairs, nil
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
