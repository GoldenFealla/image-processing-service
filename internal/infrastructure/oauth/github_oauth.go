package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"golang.org/x/oauth2"
)

type GithubOAuthRepository struct {
	config *oauth2.Config
}

func NewGithubOAuthRepository(config *oauth2.Config) *GithubOAuthRepository {
	return &GithubOAuthRepository{
		config: config,
	}
}

func (g *GithubOAuthRepository) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state)
}

func (g *GithubOAuthRepository) ExchangeCode(ctx context.Context, code string) (*domain.OAuthUserInfo, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	client := g.config.Client(ctx, token)

	// Fetch profile
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	var raw struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	email := raw.Email

	// Email can be private, fetch from emails endpoint if missing
	if email == "" {
		email, err = g.fetchPrimaryEmail(client)
		if err != nil {
			return nil, err
		}
	}

	return &domain.OAuthUserInfo{
		ProviderID: strconv.Itoa(raw.ID),
		Email:      email,
		Name:       raw.Name,
		Provider:   "github",
	}, nil
}

func (g *GithubOAuthRepository) fetchPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", fmt.Errorf("failed to get user emails: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read emails response: %w", err)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", fmt.Errorf("failed to parse emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified primary email found")
}
