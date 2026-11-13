// Package middlewares contains HTTP middleware used by the API router.
package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/basecore-dev/basecore/internal/api/models"
	"github.com/basecore-dev/basecore/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"

// AuthMiddleware validates a Bearer JWT and makes its claims available to the
// next handler through GetClaims.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := strings.Fields(r.Header.Get("Authorization"))
		if len(authorization) != 2 || !strings.EqualFold(authorization[0], "Bearer") {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "missing_or_invalid_authorization_header",
			})
			return
		}

		secret, err := config.GetJWTSecret()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "jwt_secret_unavailable",
			})
			return
		}

		claims := new(models.Claims)
		token, err := jwt.ParseWithClaims(
			authorization[1],
			claims,
			func(token *jwt.Token) (any, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuedAt(),
		)
		if err != nil || token == nil || !token.Valid || claims.UserID < 1 || claims.Username == "" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid_or_expired_token",
			})
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, *claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaims returns claims added to the request by AuthMiddleware.
// The boolean is false when the request has not passed through the middleware.
func GetClaims(r *http.Request) (models.Claims, bool) {
	claims, ok := r.Context().Value(claimsContextKey).(models.Claims)
	return claims, ok
}
