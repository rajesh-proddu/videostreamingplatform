package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/yourusername/videostreamingplatform/userservice/bl"
)

// AuthHandler serves registration, login, and token refresh.
type AuthHandler struct {
	svc *bl.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(svc *bl.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register creates a new account.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Email == "" || c.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	user, err := h.svc.Register(r.Context(), c.Email, c.Password)
	if errors.Is(err, bl.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not register")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": user.ID, "email": user.Email})
}

// Login authenticates and returns a token pair.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil || c.Email == "" || c.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	pair, err := h.svc.Login(r.Context(), c.Email, c.Password)
	if errors.Is(err, bl.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not log in")
		return
	}
	writeJSON(w, http.StatusOK, pair)
}

// Refresh exchanges a refresh token for a fresh token pair.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}
	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if errors.Is(err, bl.ErrInvalidToken) {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not refresh")
		return
	}
	writeJSON(w, http.StatusOK, pair)
}
