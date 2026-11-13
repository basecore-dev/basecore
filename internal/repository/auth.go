// Package repository provides database operations for application data.
package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/basecore-dev/basecore/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrDatabaseNotConnected is returned when a repository operation needs a
	// database connection but none is available.
	ErrDatabaseNotConnected = errors.New("repository: database not connected")
	// ErrUsernameAlreadyExists is returned when a user already uses the username.
	ErrUsernameAlreadyExists = errors.New("repository: username already exists")
	// ErrInvalidCredentials is returned when a username and password do not match.
	ErrInvalidCredentials = errors.New("repository: invalid credentials")
)

// CreateUser hashes password with bcrypt and stores the user in the database.
// It returns ErrUsernameAlreadyExists when username is already registered.
func CreateUser(username, password string) error {
	if store.DB == nil {
		return ErrDatabaseNotConnected
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("repository: hash user password: %w", err)
	}

	if _, err := store.DB.Exec(
		"INSERT INTO users (username, password) VALUES (?, ?)",
		username,
		string(hashedPassword),
	); err != nil {
		if store.IsUniqueError(err) {
			return ErrUsernameAlreadyExists
		}
		return fmt.Errorf("repository: insert user: %w", err)
	}

	return nil
}

// AuthenticateUser verifies the user's password and returns the user's ID.
// It returns ErrInvalidCredentials when the username or password is incorrect.
func AuthenticateUser(username, password string) (int64, error) {
	if store.DB == nil {
		return 0, ErrDatabaseNotConnected
	}

	var userID int64
	var hashedPassword string
	err := store.DB.QueryRow(
		"SELECT id, password FROM users WHERE username = ?",
		username,
	).Scan(&userID, &hashedPassword)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidCredentials
	}
	if err != nil {
		return 0, fmt.Errorf("repository: find user for authentication: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return 0, ErrInvalidCredentials
	}

	return userID, nil
}

// SaveRefreshToken stores a SHA-256 refresh token hash for a user.
func SaveRefreshToken(userID int64, tokenHash string, expiresAt time.Time) error {
	if store.DB == nil {
		return ErrDatabaseNotConnected
	}

	if _, err := store.DB.Exec(
		"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES (?, ?, ?)",
		userID,
		tokenHash,
		expiresAt,
	); err != nil {
		return fmt.Errorf("repository: save refresh token: %w", err)
	}

	return nil
}
