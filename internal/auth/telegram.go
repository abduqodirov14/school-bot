package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidInitData = errors.New("yaroqsiz initData formati")
	ErrMissingHash     = errors.New("initData ichida hash mavjud emas")
	ErrHashMismatch    = errors.New("initData imzosi mos kelmadi")
	ErrExpiredData     = errors.New("initData muddati o'tgan")
	ErrMissingUser     = errors.New("initData ichida foydalanuvchi ma'lumotlari topilmadi")
)

type TelegramUser struct {
	ID              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name,omitempty"`
	Username        string `json:"username,omitempty"`
	LanguageCode    string `json:"language_code,omitempty"`
	AllowsWriteToPm bool   `json:"allows_write_to_pm,omitempty"`
}

func (u *TelegramUser) FullName() string {
	if u.LastName != "" {
		return strings.TrimSpace(u.FirstName + " " + u.LastName)
	}
	return strings.TrimSpace(u.FirstName)
}

// ValidateInitData verifies Telegram Mini App initData using HMAC-SHA256 with the bot token.
// maxAge specifies the maximum allowed age of the auth_date (set 0 to disable expiration check).
func ValidateInitData(rawInitData string, botToken string, maxAge time.Duration) (*TelegramUser, error) {
	if rawInitData == "" {
		return nil, ErrInvalidInitData
	}

	values, err := url.ParseQuery(rawInitData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInitData, err)
	}

	hash := values.Get("hash")
	if hash == "" {
		return nil, ErrMissingHash
	}

	// Build data-check-string from all keys sorted alphabetically, excluding 'hash'
	var keys []string
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, values.Get(k)))
	}
	dataCheckString := strings.Join(pairs, "\n")

	// secret_key = HMAC_SHA256("WebAppData", bot_token)
	macSecret := hmac.New(sha256.New, []byte("WebAppData"))
	macSecret.Write([]byte(botToken))
	secretKey := macSecret.Sum(nil)

	// hash = HMAC_SHA256(secret_key, data_check_string)
	macData := hmac.New(sha256.New, secretKey)
	macData.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(macData.Sum(nil))

	if !strings.EqualFold(hash, expectedHash) {
		return nil, ErrHashMismatch
	}

	// Validate auth_date if maxAge > 0
	if maxAge > 0 {
		authDateStr := values.Get("auth_date")
		if authDateStr != "" {
			authTimestamp, err := strconv.ParseInt(authDateStr, 10, 64)
			if err == nil {
				authTime := time.Unix(authTimestamp, 0)
				if time.Since(authTime) > maxAge {
					return nil, ErrExpiredData
				}
			}
		}
	}

	userStr := values.Get("user")
	if userStr == "" {
		return nil, ErrMissingUser
	}

	var tgUser TelegramUser
	if err := json.Unmarshal([]byte(userStr), &tgUser); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMissingUser, err)
	}

	return &tgUser, nil
}

