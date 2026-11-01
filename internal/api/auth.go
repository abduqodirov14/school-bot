package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"school-attendance/internal/db"
)

type AuthHandler struct {
	db *db.DB
}

func NewAuthHandler(database *db.DB) *AuthHandler {
	return &AuthHandler{db: database}
}

type UserProfileResponse struct {
	User     db.User     `json:"user"`
	School   *db.School  `json:"school,omitempty"`
	Settings *db.Setting `json:"settings,omitempty"`
	Classes  []db.Class  `json:"classes,omitempty"`
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok {
		JSONError(w, "Avtorizatsiyadan o'tilmagan", http.StatusUnauthorized)
		return
	}

	resp := UserProfileResponse{
		User: *user,
	}

	if user.SchoolID.Valid {
		school, err := h.db.Queries.GetSchoolByID(r.Context(), user.SchoolID.Int64)
		if err == nil {
			resp.School = &school
		}

		if user.Role == "director" {
			settings, err := h.db.Queries.GetSettingsBySchoolID(r.Context(), user.SchoolID.Int64)
			if err == nil {
				resp.Settings = &settings
			}
		} else if user.Role == "teacher" {
			classes, err := h.db.Queries.ListClassesByTeacherID(r.Context(), pgtype.Int8{Int64: user.ID, Valid: true})
			if err == nil {
				if classes == nil {
					classes = make([]db.Class, 0)
				}
				resp.Classes = classes
			} else {
				resp.Classes = make([]db.Class, 0)
			}
		}
	}

	JSONResponse(w, resp, http.StatusOK)
}
