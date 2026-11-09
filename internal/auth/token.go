package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

func NewTokenManager(secret []byte, issuer, audience string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: secret, issuer: issuer, audience: audience, ttl: ttl}
}

func (manager *TokenManager) Issue(userID int64) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(manager.ttl)
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    manager.issuer,
		Subject:   strconv.FormatInt(userID, 10),
		Audience:  jwt.ClaimStrings{manager.audience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(manager.secret)
	return signed, expiresAt, err
}

func (manager *TokenManager) Verify(value string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(value, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("algoritma token tidak diizinkan")
		}
		return manager.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(manager.issuer), jwt.WithAudience(manager.audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(30*time.Second))
	if err != nil || !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	if _, err := strconv.ParseInt(claims.Subject, 10, 64); err != nil {
		return nil, errors.New("subject token tidak valid")
	}
	return claims, nil
}
