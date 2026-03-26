package presentation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/google/uuid"
)

type AuthHandler struct {
	auth application.AuthUseCase

	oauthState  domain.OAuthStateStore
	RedirectURL string
}

func NewAuthHandler(auth application.AuthUseCase, oauthState domain.OAuthStateStore, RedirectURL string) *AuthHandler {
	return &AuthHandler{auth: auth, oauthState: oauthState, RedirectURL: RedirectURL}
}

func (h *AuthHandler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /check", h.check)

	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /refresh", h.refresh)
	mux.HandleFunc("POST /logout", h.logout)

	mux.HandleFunc("GET /google", h.googleRedirect)
	mux.HandleFunc("GET /google/callback", h.googleCallback)

	mux.HandleFunc("GET /github", h.githubRedirect)
	mux.HandleFunc("GET /github/callback", h.githubCallback)

	return mux
}

func (h *AuthHandler) check(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	tokenStr := authHeader[7:] // strip "Bearer "

	_, err := h.auth.ValidateAccessToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var body *domain.RegisterForm
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	tokens, err := h.auth.Register(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeTokens(w, r, tokens)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var body *domain.LoginForm
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	tokens, err := h.auth.Login(r.Context(), body)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	writeTokens(w, r, tokens)
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	tokens, err := h.auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	writeTokens(w, r, tokens)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	if err := h.auth.Logout(r.Context(), cookie.Value); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}

	clearRefreshToken(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// Google OAuth — redirect user to Google
func (h *AuthHandler) googleRedirect(w http.ResponseWriter, r *http.Request) {
	state := uuid.New().String()
	h.oauthState.SaveState(r.Context(), state)
	http.Redirect(w, r, h.auth.GetGoogleAuthURL(state), http.StatusFound)
}

// Google OAuth — handle callback from Google
func (h *AuthHandler) googleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	fmt.Println(state)
	isValid, err := h.oauthState.ValidateState(r.Context(), state)
	if err != nil {
		http.Error(w, fmt.Sprintf("auth failed: %v", err), http.StatusInternalServerError)
		return
	}
	if !isValid {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	tokenPairs, err := h.auth.HandleGoogleCallback(r.Context(), r.URL.Query().Get("code"), state)
	if err != nil {
		http.Error(w, fmt.Sprintf("auth failed: %v", err), http.StatusInternalServerError)
		return
	}

	setRefeshToken(w, r, tokenPairs)
	http.Redirect(w, r, h.RedirectURL, http.StatusFound)
}

// Google OAuth — redirect user to Google
func (h *AuthHandler) githubRedirect(w http.ResponseWriter, r *http.Request) {
	state := uuid.New().String()
	h.oauthState.SaveState(r.Context(), state)
	http.Redirect(w, r, h.auth.GetGithubAuthURL(state), http.StatusFound)
}

// Google OAuth — handle callback from Google
func (h *AuthHandler) githubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	fmt.Println(state)
	isValid, err := h.oauthState.ValidateState(r.Context(), state)
	if err != nil {
		http.Error(w, fmt.Sprintf("auth failed: %v", err), http.StatusInternalServerError)
		return
	}
	if !isValid {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	tokenPairs, err := h.auth.HandleGithubCallback(r.Context(), r.URL.Query().Get("code"), state)
	if err != nil {
		http.Error(w, fmt.Sprintf("auth failed: %v", err), http.StatusInternalServerError)
		return
	}

	setRefeshToken(w, r, tokenPairs)
	http.Redirect(w, r, h.RedirectURL, http.StatusFound)
}

func isHttps(r *http.Request) bool {
	// Go is behind nginx
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

func clearRefreshToken(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   isHttps(r),
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func setRefeshToken(w http.ResponseWriter, r *http.Request, tokens *application.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		HttpOnly: true,
		Secure:   isHttps(r),
		SameSite: http.SameSiteLaxMode,
		Path:     "/",              // only sent to auth endpoint
		MaxAge:   7 * 24 * 60 * 60, // 7 days in seconds
	})
}

func writeTokens(w http.ResponseWriter, r *http.Request, tokens *application.TokenPair) {
	setRefeshToken(w, r, tokens)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": tokens.AccessToken,
	})
}
