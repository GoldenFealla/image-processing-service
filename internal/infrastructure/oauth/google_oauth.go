// internal/infrastructure/oauth/google_oauth.go
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/oauth2"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
)

type GoogleOAuthRepository struct {
	config *oauth2.Config
}

func NewGoogleOAuthRepository(config *oauth2.Config) *GoogleOAuthRepository {
	return &GoogleOAuthRepository{
		config: config,
	}
}

func (g *GoogleOAuthRepository) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state)
}

func (g *GoogleOAuthRepository) ExchangeCode(ctx context.Context, code string) (*domain.OAuthUserInfo, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := g.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var raw struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &domain.OAuthUserInfo{
		ProviderID: raw.ID,
		Email:      raw.Email,
		Name:       raw.Name,
		Picture:    raw.Picture,
		Provider:   "google",
	}, nil
}
