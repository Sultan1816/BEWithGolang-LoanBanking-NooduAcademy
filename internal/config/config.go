package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	HTTPAddress string
	JWTSecret   []byte
	JWTIssuer   string
	JWTAudience string
	JWTTTL      time.Duration
	CORSOrigin  string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return Config{}, errors.New("JWT_SECRET harus memiliki minimal 32 karakter")
	}

	ttl := 15 * time.Minute
	if value := os.Getenv("JWT_TTL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed < time.Minute || parsed > 24*time.Hour {
			return Config{}, errors.New("JWT_TTL harus berdurasi antara 1 menit dan 24 jam")
		}
		ttl = parsed
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL wajib diisi")
	}

	issuer := strings.TrimSpace(os.Getenv("JWT_ISSUER"))
	if issuer == "" {
		issuer = "noodu-login-service"
	}
	audience := strings.TrimSpace(os.Getenv("JWT_AUDIENCE"))
	if audience == "" {
		audience = "noodu-client"
	}

	address := strings.TrimSpace(os.Getenv("HTTP_ADDRESS"))
	if address == "" {
		address = "127.0.0.1:8080"
	}

	return Config{
		DatabaseURL: databaseURL,
		HTTPAddress: address,
		JWTSecret:   []byte(secret),
		JWTIssuer:   issuer,
		JWTAudience: audience,
		JWTTTL:      ttl,
		CORSOrigin:  strings.TrimSpace(os.Getenv("CORS_ORIGIN")),
	}, nil
}
