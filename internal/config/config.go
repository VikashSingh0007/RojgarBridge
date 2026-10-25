package config

import (
	"os"
)

type Config struct {
	Port         string
	DBPath       string
	GeminiAPIKey string
	Environment  string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "rozgarhub.db"
	}

	apiKey := os.Getenv("GEMINI_API_KEY")

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	return &Config{
		Port:         port,
		DBPath:       dbPath,
		GeminiAPIKey: apiKey,
		Environment:  env,
	}
}
