// Package models defines API request and response types.
package models

import "github.com/golang-jwt/jwt/v5"

// RegisterReq contains the username and password submitted for registration.
type RegisterReq struct {
	// Username is the account name requested by the user.
	Username string `json:"username"`
	// Password is the plaintext password submitted for hashing before storage.
	Password string `json:"password"`
}

// LoginReq contains the username and password submitted for authentication.
type LoginReq struct {
	// Username is the account name to authenticate.
	Username string `json:"username"`
	// Password is the plaintext password to verify against the stored hash.
	Password string `json:"password"`
}

// Claims contains the user identity and registered claims carried by a JWT.
type Claims struct {
	// UserID is the identifier of the authenticated account.
	UserID int `json:"user_id"`
	// Username is the account name of the authenticated user.
	Username string `json:"username"`

	jwt.RegisteredClaims
}
