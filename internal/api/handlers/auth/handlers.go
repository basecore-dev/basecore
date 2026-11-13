// Package auth contains HTTP handlers for authentication endpoints.
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/basecore-dev/basecore/internal/api/middlewares"
	"github.com/basecore-dev/basecore/internal/api/models"
	"github.com/basecore-dev/basecore/internal/config"
	"github.com/basecore-dev/basecore/internal/repository"
	"github.com/basecore-dev/basecore/internal/store"
	"github.com/golang-jwt/jwt/v5"
)

// RegisterHandler creates a user account from a JSON username and password.
func RegisterHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "method_not_allowed",
		})

		return
	}

	if store.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "database_not_connected",
		})

		return
	}

	var req models.RegisterReq
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_json",
		})

		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_json",
		})

		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "username_required",
		})

		return
	}
	if req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "password_required",
		})

		return
	}
	if len([]byte(req.Password)) > 72 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "password_too_long",
		})

		return
	}

	if err := repository.CreateUser(req.Username, req.Password); err != nil {
		switch {
		case errors.Is(err, repository.ErrUsernameAlreadyExists):
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "username_already_exists",
			})
		case errors.Is(err, repository.ErrDatabaseNotConnected):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "database_not_connected",
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "registration_failed",
			})
		}

		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "user_registered",
	})
}

// LoginHandler authenticates a user and returns a signed JWT.
func LoginHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "method_not_allowed",
		})
		return
	}

	if store.DB == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "database_not_connected",
		})
		return
	}

	var req models.LoginReq
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_json",
		})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_json",
		})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "username_required",
		})
		return
	}
	if req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "password_required",
		})
		return
	}
	if len([]byte(req.Password)) > 72 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "password_too_long",
		})
		return
	}

	userID, err := repository.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid_credentials",
			})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "authentication_failed",
		})
		return
	}

	secret, err := config.GetJWTSecret()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "jwt_secret_unavailable",
		})
		return
	}

	now := time.Now()
	expiresAt := now.Add(time.Hour)
	claims := models.Claims{
		UserID:   int(userID),
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "token_creation_failed",
		})
		return
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "refresh_token_creation_failed",
		})
		return
	}

	refreshTokenHash := sha256.Sum256([]byte(refreshToken))
	if err := repository.SaveRefreshToken(
		userID,
		hex.EncodeToString(refreshTokenHash[:]),
		now.Add(refreshTokenTTL),
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "refresh_token_storage_failed",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"token":        tokenString,
		"refreshToken": refreshToken,
		"token_type":   "Bearer",
		"expires_in":   int(time.Hour.Seconds()),
	})
}

// GetMe returns the identity from the authenticated request's JWT claims.
func GetMe(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	claims, ok := middlewares.GetClaims(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"user_id":  claims.UserID,
		"username": claims.Username,
	})
}
