package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func signInitData(params map[string]string, botToken string) string {
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, params[k]))
	}
	dataCheckString := strings.Join(pairs, "\n")

	macSecret := hmac.New(sha256.New, []byte("WebAppData"))
	macSecret.Write([]byte(botToken))
	secretKey := macSecret.Sum(nil)

	macData := hmac.New(sha256.New, secretKey)
	macData.Write([]byte(dataCheckString))
	hash := hex.EncodeToString(macData.Sum(nil))

	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("hash", hash)

	return values.Encode()
}

func TestValidateInitData_Success(t *testing.T) {
	botToken := "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
	now := time.Now().Unix()
	userJSON := `{"id":987654321,"first_name":"Ali","last_name":"Valiyev","username":"alivaliyev"}`

	params := map[string]string{
		"auth_date": fmt.Sprintf("%d", now),
		"query_id":  "AAG_AABBCCDDEE",
		"user":      userJSON,
	}

	initData := signInitData(params, botToken)

	user, err := ValidateInitData(initData, botToken, 24*time.Hour)
	if err != nil {
		t.Fatalf("expected valid initData, got error: %v", err)
	}

	if user.ID != 987654321 {
		t.Errorf("expected user ID 987654321, got %d", user.ID)
	}
	if user.FullName() != "Ali Valiyev" {
		t.Errorf("expected full name 'Ali Valiyev', got %s", user.FullName())
	}
	if user.Username != "alivaliyev" {
		t.Errorf("expected username 'alivaliyev', got %s", user.Username)
	}
}

func TestValidateInitData_TamperedData(t *testing.T) {
	botToken := "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
	now := time.Now().Unix()
	userJSON := `{"id":987654321,"first_name":"Ali","last_name":"Valiyev"}`

	params := map[string]string{
		"auth_date": fmt.Sprintf("%d", now),
		"user":      userJSON,
	}

	initData := signInitData(params, botToken)
	// Tamper initData by changing query string
	tampered := initData + "extra=1"

	_, err := ValidateInitData(tampered, botToken, 24*time.Hour)
	if err == nil {
		t.Fatal("expected error for tampered data, got nil")
	}
}

func TestValidateInitData_Expired(t *testing.T) {
	botToken := "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
	past := time.Now().Add(-48 * time.Hour).Unix()
	userJSON := `{"id":112233,"first_name":"Eski","last_name":"User"}`

	params := map[string]string{
		"auth_date": fmt.Sprintf("%d", past),
		"user":      userJSON,
	}

	initData := signInitData(params, botToken)

	_, err := ValidateInitData(initData, botToken, 24*time.Hour)
	if err != ErrExpiredData {
		t.Fatalf("expected ErrExpiredData, got: %v", err)
	}
}

func TestValidateInitData_MissingHash(t *testing.T) {
	_, err := ValidateInitData("user=something", "token", 0)
	if err != ErrMissingHash {
		t.Fatalf("expected ErrMissingHash, got: %v", err)
	}
}

