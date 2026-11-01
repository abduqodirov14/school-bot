package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	BotToken           string
	DatabaseURL        string
	WebAppURL          string
	DirectorTelegramID int64
	Timezone           string
	Location           *time.Location
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		// Log warning or keep empty for testing
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/school_attendance?sslmode=disable"
	}

	webAppURL := os.Getenv("WEB_APP_URL")
	if webAppURL == "" {
		webAppURL = "http://localhost:5173"
	}

	var directorTGID int64
	if idStr := os.Getenv("DIRECTOR_TELEGRAM_ID"); idStr != "" {
		parsed, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			directorTGID = parsed
		}
	}

	tzStr := os.Getenv("TIMEZONE")
	if tzStr == "" {
		tzStr = "Asia/Tashkent"
	}

	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", tzStr, err)
	}

	return &Config{
		Port:               port,
		BotToken:           botToken,
		DatabaseURL:        databaseURL,
		WebAppURL:          webAppURL,
		DirectorTelegramID: directorTGID,
		Timezone:           tzStr,
		Location:           loc,
	}, nil
}

