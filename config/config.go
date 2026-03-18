package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL            string
	JWTSecret              string
	Port                   string
	SupabaseURL            string
	SupabaseServiceRoleKey string
	SupabaseStorageBucket  string
}

var App Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading env from system")
	}

	App = Config{
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://localhost:5432/bayarin?sslmode=disable"),
		JWTSecret:              getEnv("JWT_SECRET", "change-me-in-production"),
		Port:                   getEnv("PORT", "8080"),
		SupabaseURL:            getEnv("SUPABASE_URL", ""),
		SupabaseServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		SupabaseStorageBucket:  getEnv("SUPABASE_STORAGE_BUCKET", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
