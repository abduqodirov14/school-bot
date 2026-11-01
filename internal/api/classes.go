package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/db"
)

type ClassHandler struct {
	db *db.DB
}

func NewClassHandler(database *db.DB) *ClassHandler {
	return &ClassHandler{db: database}
}

type CreateClassRequest struct {
	Name      string `json:"name"`
	TeacherID *int64 `json:"teacher_id"`
}

type UpdateClassRequest struct {
	Name      string `json:"name"`
	TeacherID *int64 `json:"teacher_id"`
}

func (h *ClassHandler) ListClasses(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab topilmadi", http.StatusBadRequest)
		return
	}

	if user.Role == "director" {
		classes, err := h.db.Queries.ListClassesWithStudentCount(r.Context(), user.SchoolID.Int64)
		if err != nil {
			JSONError(w, "Sinflarni yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if classes == nil {
			classes = make([]db.ListClassesWithStudentCountRow, 0)
		}
		JSONResponse(w, classes, http.StatusOK)
		return
	}

	// Teacher role: only list classes assigned to this teacher
	classes, err := h.db.Queries.ListClassesByTeacherID(r.Context(), pgtype.Int8{Int64: user.ID, Valid: true})
	if err != nil {
		JSONError(w, "Sinflarni yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if classes == nil {
		classes = make([]db.Class, 0)
	}
	JSONResponse(w, classes, http.StatusOK)
}

func (h *ClassHandler) CreateClass(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab topilmadi", http.StatusBadRequest)
		return
	}

	var req CreateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz so'rov ma'lumotlari", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		JSONError(w, "Sinf nomi kiritilishi shart", http.StatusBadRequest)
		return
	}

	var teacherID pgtype.Int8
	if req.TeacherID != nil && *req.TeacherID > 0 {
		teacherID = pgtype.Int8{Int64: *req.TeacherID, Valid: true}
	}

	created, err := h.db.Queries.CreateClass(r.Context(), db.CreateClassParams{
		SchoolID:  user.SchoolID.Int64,
		TeacherID: teacherID,
		Name:      req.Name,
	})
	if err != nil {
		JSONError(w, "Sinf yaratishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, created, http.StatusCreated)
}

func (h *ClassHandler) UpdateClass(w http.ResponseWriter, r *http.Request) {
	classIDStr := chi.URLParam(r, "id")
	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
		return
	}

	var req UpdateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz so'rov ma'lumotlari", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		JSONError(w, "Sinf nomi kiritilishi shart", http.StatusBadRequest)
		return
	}

	var teacherID pgtype.Int8
	if req.TeacherID != nil && *req.TeacherID > 0 {
		teacherID = pgtype.Int8{Int64: *req.TeacherID, Valid: true}
	}

	updated, err := h.db.Queries.UpdateClass(r.Context(), db.UpdateClassParams{
		ID:        classID,
		Name:      req.Name,
		TeacherID: teacherID,
	})
	if err != nil {
		JSONError(w, "Sinfni yangilashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, updated, http.StatusOK)
}

func (h *ClassHandler) DeleteClass(w http.ResponseWriter, r *http.Request) {
	classIDStr := chi.URLParam(r, "id")
	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
		return
	}

	if err := h.db.Queries.DeleteClass(r.Context(), classID); err != nil {
		JSONError(w, "Sinfni o'chirishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]string{"message": "Sinf muvaffaqiyatli o'chirildi"}, http.StatusOK)
}

func (h *ClassHandler) ListTeachers(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab topilmadi", http.StatusBadRequest)
		return
	}

	teachers, err := h.db.Queries.ListTeachersBySchool(r.Context(), user.SchoolID)
	if err != nil {
		JSONError(w, "O'qituvchilarni yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if teachers == nil {
		teachers = make([]db.User, 0)
	}

	JSONResponse(w, teachers, http.StatusOK)
}

