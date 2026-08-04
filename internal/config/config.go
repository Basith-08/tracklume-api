package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv            string
	AppPort           string
	BaseURL           string
	DatabaseURL       string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	JWTSecret         string
	JWTExpiration     time.Duration
	CORSOrigins       []string
	RequestTimeout    time.Duration
	ShutdownTimeout   time.Duration
	BodyLimit         int64
	RateLimitRequests int
	RateLimitWindow   time.Duration
}

func Load() (Config, error) {
	c := Config{
		AppEnv:            env("APP_ENV", "development"),
		AppPort:           env("APP_PORT", "8080"),
		BaseURL:           env("APP_BASE_URL", "http://localhost:8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DBHost:            env("DB_HOST", "localhost"),
		DBPort:            env("DB_PORT", "5432"),
		DBUser:            env("POSTGRES_USER", "tracklume"),
		DBPassword:        os.Getenv("POSTGRES_PASSWORD"),
		DBName:            env("POSTGRES_DB", "tracklume"),
		DBSSLMode:         env("DB_SSLMODE", "disable"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		CORSOrigins:       split(os.Getenv("CORS_ALLOWED_ORIGINS")),
		BodyLimit:         int64(envInt("BODY_LIMIT_BYTES", 1<<20)),
		RateLimitRequests: envInt("AUTH_RATE_LIMIT_REQUESTS", 10),
	}
	var err error
	if c.JWTExpiration, err = time.ParseDuration(env("JWT_EXPIRATION", "1h")); err != nil || c.JWTExpiration <= 0 {
		return Config{}, fmt.Errorf("JWT_EXPIRATION must be a positive duration: %w", err)
	}
	if c.RequestTimeout, err = time.ParseDuration(env("REQUEST_TIMEOUT", "15s")); err != nil || c.RequestTimeout <= 0 {
		return Config{}, fmt.Errorf("REQUEST_TIMEOUT must be a positive duration: %w", err)
	}
	if c.ShutdownTimeout, err = time.ParseDuration(env("SHUTDOWN_TIMEOUT", "10s")); err != nil || c.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration: %w", err)
	}
	if c.RateLimitWindow, err = time.ParseDuration(env("AUTH_RATE_LIMIT_WINDOW", "1m")); err != nil || c.RateLimitWindow <= 0 {
		return Config{}, fmt.Errorf("AUTH_RATE_LIMIT_WINDOW must be a positive duration: %w", err)
	}
	if c.BodyLimit < 1024 {
		return Config{}, errors.New("BODY_LIMIT_BYTES must be at least 1024")
	}
	if c.RateLimitRequests < 1 {
		return Config{}, errors.New("AUTH_RATE_LIMIT_REQUESTS must be positive")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	if c.AppEnv == "production" && len(c.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must contain at least 32 characters in production")
	}
	if c.DatabaseURL == "" && (c.DBUser == "" || c.DBName == "") {
		return Config{}, errors.New("database configuration is incomplete")
	}
	return c, nil
}

func (c Config) DatabaseDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func split(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
