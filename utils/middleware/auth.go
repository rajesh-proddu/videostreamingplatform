package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/yourusername/videostreamingplatform/utils/auth"
)

type ctxKey int

const claimsContextKey ctxKey = iota

// JWTAuth returns middleware that requires a valid HS256 access token. The
// verified claims are injected into the request context (see ClaimsFromContext).
// Apply it per-route (wrap individual handlers), not over an entire mux.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			claims, err := auth.Parse(secret, token)
			if err != nil || claims.TokenType != auth.TokenAccess {
				writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireEntitlement returns middleware that allows the request only if the
// token carries an active paid entitlement, otherwise 402 Payment Required. Must
// be applied inside JWTAuth.
func RequireEntitlement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			writeAuthError(w, http.StatusUnauthorized, "missing authentication")
			return
		}
		if !claims.Entitled {
			writeAuthError(w, http.StatusPaymentRequired, "active subscription required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClaimsFromContext returns the verified claims injected by JWTAuth.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims, ok
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
