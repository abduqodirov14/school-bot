package config

import (
	"os"
	"strconv"
	"time"
	_ "time/tzdata"
)

type Config struct {
	Port               string
	BotToken           string
	BotUsername        string
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

	botUsername := os.Getenv("BOT_USERNAME")
	if botUsername == "" {
		botUsername = "HozirDavomatBot"
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
		// Fallback to UTC+5 for Tashkent
		loc = time.FixedZone("Asia/Tashkent", 5*60*60)
	}

	return &Config{
		Port:               port,
		BotToken:           botToken,
		BotUsername:        botUsername,
		DatabaseURL:        databaseURL,
		WebAppURL:          webAppURL,
		DirectorTelegramID: directorTGID,
		Timezone:           tzStr,
		Location:           loc,
	}, nil
}

