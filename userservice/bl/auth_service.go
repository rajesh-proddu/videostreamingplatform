package bl

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/yourusername/videostreamingplatform/userservice/dl"
	"github.com/yourusername/videostreamingplatform/userservice/models"
	"github.com/yourusername/videostreamingplatform/utils/auth"
)

// TokenPair is an access + refresh token returned on login/refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthService handles registration, login, and token issuance. Access tokens
// carry the user's current entitlement so resource services can gate access
// locally; a refresh re-reads live subscription state.
type AuthService struct {
	store      dl.Store
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewAuthService constructs an AuthService.
func NewAuthService(store dl.Store, secret string, accessTTL, refreshTTL time.Duration) *AuthService {
	return &AuthService{store: store, secret: secret, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// Register creates a new user with a bcrypt-hashed password.
func (s *AuthService) Register(ctx context.Context, email, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{ID: uuid.NewString(), Email: email, PasswordHash: string(hash)}
	if err := s.store.CreateUser(ctx, user); err != nil {
		if dl.IsDuplicate(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return user, nil
}

// Login verifies credentials and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, dl.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, user.ID)
}

// Refresh validates a refresh token and mints a fresh token pair with
// re-evaluated entitlement.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := auth.Parse(s.secret, refreshToken)
	if err != nil || claims.TokenType != auth.TokenRefresh {
		return nil, ErrInvalidToken
	}
	if _, err := s.store.GetUserByID(ctx, claims.Subject); err != nil {
		return nil, ErrInvalidToken
	}
	return s.issueTokens(ctx, claims.Subject)
}

// issueTokens reads the user's current entitlement and signs access + refresh
// tokens.
func (s *AuthService) issueTokens(ctx context.Context, userID string) (*TokenPair, error) {
	plan, entitled, err := s.entitlement(ctx, userID)
	if err != nil {
		return nil, err
	}
	access, err := auth.Issue(s.secret, userID, plan, entitled, auth.TokenAccess, s.accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := auth.Issue(s.secret, userID, "", false, auth.TokenRefresh, s.refreshTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// entitlement returns the user's active plan name and whether they are entitled
// to paid content. Entitlement requires an active *paid* subscription — a free
// (zero-price) plan grants a plan name but no paid entitlement, so it cannot be
// used to slip past the paywall.
func (s *AuthService) entitlement(ctx context.Context, userID string) (plan string, entitled bool, err error) {
	sub, err := s.store.GetActiveSubscription(ctx, userID)
	if errors.Is(err, dl.ErrSubscriptionNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	p, err := s.store.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return "", false, err
	}
	return p.Name, p.AmountMinor > 0, nil
}
