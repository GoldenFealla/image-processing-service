package domain

import "context"

type OAuthUserInfo struct {
	ProviderID string
	Email      string
	Name       string
	Provider   string // "google", "github"
}

type OAuthRepository interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthUserInfo, error)
}
