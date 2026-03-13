package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
}

var App Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading env from system")
	}

	App = Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/bayarin?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
