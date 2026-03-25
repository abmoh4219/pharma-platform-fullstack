package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	Username    string `json:"username"`
	Role        string `json:"role"`
	ScopeID     int64  `json:"scope_id"`
	Institution string `json:"institution"`
	Department  string `json:"department"`
	Team        string `json:"team"`
	jwt.RegisteredClaims
}

type TokenInput struct {
	UserID      int64
	Username    string
	Role        string
	ScopeID     int64
	Institution string
	Department  string
	Team        string
	ExpiryHours int
}

func IssueToken(secret string, in TokenInput) (string, string, time.Time, error) {
	jti, err := randHex(16)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate jti: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(in.ExpiryHours) * time.Hour)

	claims := Claims{
		Username:    in.Username,
		Role:        in.Role,
		ScopeID:     in.ScopeID,
		Institution: in.Institution,
		Department:  in.Department,
		Team:        in.Team,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(in.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signed, jti, expiresAt, nil
}

func ParseToken(secret string, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithLeeway(0),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.ExpiresAt == nil || time.Now().UTC().After(claims.ExpiresAt.Time) {
		return nil, ErrExpiredToken
	}
	if claims.ID == "" || claims.Subject == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func randHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
