// Package config initializes BaseCore's local configuration path and manages
// secrets stored in the system keyring.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// ErrJWTSecretNotFound is returned by CheckJWTSecret when the JWT secret is
// missing from the system keyring.
var ErrJWTSecretNotFound = errors.New("config: JWT secret not found in keyring")

// BaseCorePath is the path to BaseCore's local configuration directory.
var BaseCorePath string

var (
	// ServiceName identifies BaseCore in the system keyring.
	ServiceName string = "basecore"
	// JWTSecretKey identifies the JWT secret entry in the system keyring.
	JWTSecretKey string = "jwt_secret_key"
)

// Init sets BaseCorePath to the user's .basecore directory and creates it.
func Init() error {
	home, err := os.UserHomeDir()

	if err != nil {
		return fmt.Errorf("config: get user home dir: %w", err)
	}

	BaseCorePath = filepath.Join(home, ".basecore")

	err = os.MkdirAll(BaseCorePath, 0755)

	if err != nil {
		return fmt.Errorf("config: create .basecore folder: %w", err)
	}

	return nil
}

// GetBaseCorePath returns the path initialized by Init.
func GetBaseCorePath() string {
	return BaseCorePath
}

// SetJWTSecret stores jwtSecret in the system keyring.
func SetJWTSecret(jwtSecret string) error {
	if err := keyring.Set(ServiceName, JWTSecretKey, jwtSecret); err != nil {
		return fmt.Errorf("config: store JWT secret in keyring: %w", err)
	}

	return nil
}

// GetJWTSecret returns the JWT secret stored in the system keyring.
// It returns ErrJWTSecretNotFound when no secret is stored and wraps other
// keyring errors.
func GetJWTSecret() (string, error) {
	secret, err := keyring.Get(ServiceName, JWTSecretKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrJWTSecretNotFound
		}
		return "", fmt.Errorf("config: get JWT secret from keyring: %w", err)
	}

	return secret, nil
}

// CheckJWTSecret reports whether the JWT secret exists in the system keyring.
// It returns ErrJWTSecretNotFound when no secret is stored and wraps other
// keyring errors.
func CheckJWTSecret() error {
	_, err := GetJWTSecret()
	return err
}
