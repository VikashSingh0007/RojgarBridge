package config

import (
	"os"
	"strings"
)

type Config struct {
	Port             string
	DBPath           string
	DatabaseURL      string
	GeminiAPIKey     string
	Environment      string
	TelegramBotToken string
	TelegramChatID   string
	GoogleClientID   string
	GoogleClientSecret string
	GoogleRedirectURL string
	FrontendURL      string
}

func loadDotEnv() {
	paths := []string{".env", "../.env"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
			}
		}
		break
	}
}

func Load() *Config {
	loadDotEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "rozgarhub.db"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	apiKey := os.Getenv("GEMINI_API_KEY")

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	tgChatID := os.Getenv("TELEGRAM_CHAT_ID")

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if googleRedirectURL == "" {
		googleRedirectURL = "http://localhost:" + port + "/api/auth/google/callback"
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	return &Config{
		Port:               port,
		DBPath:             dbPath,
		DatabaseURL:        databaseURL,
		GeminiAPIKey:       apiKey,
		Environment:        env,
		TelegramBotToken:   tgToken,
		TelegramChatID:     tgChatID,
		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
		GoogleRedirectURL:  googleRedirectURL,
		FrontendURL:        frontendURL,
	}
}
