package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
	BcryptCost  int
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	bcryptCost := 12
	if v := os.Getenv("BCRYPT_COST"); v != "" {
		if c, err := strconv.Atoi(v); err == nil && c >= 4 && c <= 31 {
			bcryptCost = c
		}
	}

	return &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        port,
		BcryptCost:  bcryptCost,
	}
}
