package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenManager struct {
	secret     []byte
	expiration time.Duration
}

func NewTokenManager(secret string, expiration time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), expiration: expiration}
}

func (m *TokenManager) Create(userID uuid.UUID, now time.Time) (string, int64, error) {
	expires := now.Add(m.expiration)
	claims := jwt.RegisteredClaims{Subject: userID.String(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires)}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	value, err := token.SignedString(m.secret)
	return value, int64(m.expiration.Seconds()), err
}

func (m *TokenManager) Parse(tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return uuid.Nil, errors.New("invalid claims")
	}
	return uuid.Parse(claims.Subject)
}
