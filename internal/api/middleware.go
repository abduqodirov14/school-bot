package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"school-attendance/internal/auth"
	"school-attendance/internal/db"
)

type contextKey string

const userContextKey contextKey = "auth_user"

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

func JSONResponse(w http.ResponseWriter, data any, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// AuthMiddleware validates Telegram initData HMAC signature on every request
// and sets the authenticated active user in the request context.
func AuthMiddleware(botToken string, database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawInitData := r.Header.Get("X-Telegram-Init-Data")
			if rawInitData == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "tma ") {
					rawInitData = strings.TrimPrefix(authHeader, "tma ")
				} else if strings.HasPrefix(authHeader, "Bearer ") {
					rawInitData = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}
			if rawInitData == "" {
				rawInitData = r.URL.Query().Get("initData")
			}

			if rawInitData == "" {
				JSONError(w, "Telegram initData topilmadi. Avtorizatsiya talab qilinadi.", http.StatusUnauthorized)
				return
			}

			tgUser, err := auth.ValidateInitData(rawInitData, botToken, 24*time.Hour)
			if err != nil {
				JSONError(w, "Avtorizatsiya xatosi: "+err.Error(), http.StatusUnauthorized)
				return
			}

			user, err := database.Queries.GetUserByTelegramID(r.Context(), tgUser.ID)
			if err != nil {
				JSONError(w, "Foydalanuvchi tizimda topilmadi. Avval Telegram botida /start bosing.", http.StatusUnauthorized)
				return
			}

			if user.Status == "pending" {
				JSONError(w, "Sizning hisobingiz hali tasdiqlanmagan. Iltimos, direktor tasdiqlashini kuting.", http.StatusForbidden)
				return
			}

			if user.Status == "rejected" {
				JSONError(w, "Sizning hisobingiz direktor tomonidan rad etilgan.", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser retrieves the authenticated user from the context.
func GetUser(ctx context.Context) (*db.User, bool) {
	u, ok := ctx.Value(userContextKey).(*db.User)
	return u, ok
}

// RequireRole ensures the authenticated user has one of the specified roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok {
				JSONError(w, "Avtorizatsiya talab qilinadi", http.StatusUnauthorized)
				return
			}

			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				JSONError(w, "Ushbu amalni bajarish uchun ruxsat yo'q", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

