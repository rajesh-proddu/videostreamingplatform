// Package auth provides HS256 JWT issuance and verification shared between the
// user service (which issues tokens) and other services (which verify them).
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when a token is missing, malformed, expired, or
// signed with the wrong key.
var ErrInvalidToken = errors.New("invalid token")

// Token types carried in the "typ" claim.
const (
	TokenAccess  = "access"
	TokenRefresh = "refresh"
)

// Claims is the JWT payload. Subject (RegisteredClaims) holds the user id. The
// entitlement claims (Plan, Entitled) let resource services gate access locally
// without a network call to the user service.
type Claims struct {
	Plan      string `json:"plan,omitempty"`
	Entitled  bool   `json:"entitled"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

// Issue signs an HS256 token for userID with the given entitlement and TTL.
func Issue(secret, userID, plan string, entitled bool, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Plan:      plan,
		Entitled:  entitled,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Parse verifies the signature and expiry and returns the claims.
func Parse(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
