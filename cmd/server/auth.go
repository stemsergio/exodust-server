package main

import (
	"encoding/base64"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// loadSecret reads the key shared with exodustWeb (Base64 in GAME_TOKEN_SECRET).
func loadSecret() ([]byte, error) {
	b64 := os.Getenv("GAME_TOKEN_SECRET")
	if b64 == "" {
		return nil, errors.New("GAME_TOKEN_SECRET is not set")
	}
	return base64.StdEncoding.DecodeString(b64)
}

// verifyToken returns the user id (sub) if the token is valid.
func verifyToken(tokenString string, secret []byte) (string, error) {
	token, err := jwt.Parse(tokenString,
		func(t *jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer("exodust-web"),
		jwt.WithAudience("exodust-game"),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return "", err
	}
	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", errors.New("token has no subject")
	}
	return sub, nil
}
