package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/GoldenFealla/image-processing-service/internal/application"
	"github.com/GoldenFealla/image-processing-service/internal/domain"
	"github.com/google/uuid"
)

type AuthHandler struct {
	auth application.AuthUseCase
}

func NewAuthHandler(auth application.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
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

	writeTokens(w, tokens)
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

	writeTokens(w, tokens)
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

	writeTokens(w, tokens)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if err := h.auth.Logout(r.Context(), body.RefreshToken); err != nil {
		http.Error(w, "logout failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Google OAuth — redirect user to Google
func (h *AuthHandler) googleRedirect(w http.ResponseWriter, r *http.Request) {
	state := uuid.New().String()
	// store state in cookie or session
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Value: state})
	http.Redirect(w, r, h.auth.GetGoogleAuthURL(state), http.StatusTemporaryRedirect)
}

// Google OAuth — handle callback from Google
func (h *AuthHandler) googleCallback(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("oauth_state")
	if cookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	tokenPairs, err := h.auth.HandleGoogleCallback(r.Context(), r.URL.Query().Get("code"), cookie.Value)
	if err != nil {
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}

	// redirect frontend with token
	http.Redirect(w, r, "https://yourapp.com/auth/success?token="+tokenPairs.AccessToken, http.StatusTemporaryRedirect)
}

func writeTokens(w http.ResponseWriter, tokens *application.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		HttpOnly: true,
		Secure:   false, // Change later
		SameSite: http.SameSiteStrictMode,
		Path:     "/auth/refresh",  // only sent to refresh endpoint
		MaxAge:   7 * 24 * 60 * 60, // 7 days in seconds
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": tokens.AccessToken,
	})
}
