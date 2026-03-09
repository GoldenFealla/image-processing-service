package presentation

import (
	"encoding/json"
	"net/http"

	"github.com/GoldenFealla/image-processing-service/internal/application"
)

type AuthHandler struct {
	auth *application.AuthService
}

func NewAuthHandler(auth *application.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /refresh", h.refresh)
	mux.HandleFunc("POST /logout", h.logout)

	// mux.HandleFunc("GET /google", h.googleRedirect)
	// mux.HandleFunc("GET /google/callback", h.googleCallback)

	return mux
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	tokens, err := h.auth.Register(r.Context(), body.Email, body.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeTokens(w, tokens)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	tokens, err := h.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	writeTokens(w, tokens)
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	tokens, err := h.auth.Refresh(r.Context(), body.RefreshToken)
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
// func (h *AuthHandler) googleRedirect(w http.ResponseWriter, r *http.Request) {
// 	url := h.auth.GoogleAuthURL()
// 	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
// }

// Google OAuth — handle callback from Google
// func (h *AuthHandler) googleCallback(w http.ResponseWriter, r *http.Request) {
// 	code := r.URL.Query().Get("code")
// 	if code == "" {
// 		http.Error(w, "missing code", http.StatusBadRequest)
// 		return
// 	}

// 	tokens, err := h.auth.HandleGoogleCallback(r.Context(), code)
// 	if err != nil {
// 		http.Error(w, "google auth failed", http.StatusUnauthorized)
// 		return
// 	}

// 	writeTokens(w, tokens)
// }

func writeTokens(w http.ResponseWriter, tokens *application.TokenPair) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}
