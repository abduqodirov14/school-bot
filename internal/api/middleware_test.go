package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"school-attendance/internal/db"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name         string
		user         *db.User
		allowedRoles []string
		expectedCode int
	}{
		{
			name:         "No user in context",
			user:         nil,
			allowedRoles: []string{"director"},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Teacher trying to access director route",
			user:         &db.User{ID: 1, Role: "teacher", Status: "active"},
			allowedRoles: []string{"director"},
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "Director accessing director route",
			user:         &db.User{ID: 2, Role: "director", Status: "active"},
			allowedRoles: []string{"director"},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Teacher accessing teacher or director route",
			user:         &db.User{ID: 3, Role: "teacher", Status: "active"},
			allowedRoles: []string{"teacher", "director"},
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := RequireRole(tc.allowedRoles...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.user != nil {
				ctx := context.WithValue(req.Context(), userContextKey, tc.user)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.expectedCode {
				t.Errorf("expected status code %d, got %d", tc.expectedCode, rec.Code)
			}
		})
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	middleware := AuthMiddleware("test_token", nil)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing initData, got %d", rec.Code)
	}
}

