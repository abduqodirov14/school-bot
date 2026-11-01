package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"school-attendance/internal/config"
	"school-attendance/internal/db"
	"school-attendance/internal/service"
)

type AttendanceHandler struct {
	db                *db.DB
	cfg               *config.Config
	service           *service.AttendanceService
	onAttendanceSaved func(studentID int64, schoolID int64, date time.Time)
}

func NewAttendanceHandler(database *db.DB, cfg *config.Config) *AttendanceHandler {
	return &AttendanceHandler{
		db:      database,
		cfg:     cfg,
		service: service.NewAttendanceService(database),
	}
}

func (h *AttendanceHandler) SetOnAttendanceSaved(fn func(studentID int64, schoolID int64, date time.Time)) {
	h.onAttendanceSaved = fn
}

type SaveAttendanceRequest struct {
	ClassID int64                         `json:"class_id"`
	Date    string                        `json:"date"`
	Items   []service.AttendanceItemInput `json:"items"`
}

func (h *AttendanceHandler) GetAttendance(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab ma'lumotlari topilmadi", http.StatusBadRequest)
		return
	}

	classIDStr := r.URL.Query().Get("class_id")
	if classIDStr == "" {
		JSONError(w, "Sinf tanlanishi shart ('class_id')", http.StatusBadRequest)
		return
	}

	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
		return
	}

	// Verify teacher authorization
	if user.Role == "teacher" {
		class, err := h.db.Queries.GetClassByID(r.Context(), classID)
		if err != nil || !class.TeacherID.Valid || class.TeacherID.Int64 != user.ID {
			JSONError(w, "Siz faqat o'zingizga biriktirilgan sinf davomatini ko'ra olasiz", http.StatusForbidden)
			return
		}
	}

	dateStr := r.URL.Query().Get("date")
	targetDate := time.Now().In(h.cfg.Location)
	if dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			JSONError(w, "Sana formati noto'g'ri (YYYY-MM-DD bo'lishi kerak)", http.StatusBadRequest)
			return
		}
		targetDate = parsedDate
	}

	resp, err := h.service.GetClassAttendance(r.Context(), classID, targetDate)
	if err != nil {
		JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, resp, http.StatusOK)
}

func (h *AttendanceHandler) SaveAttendance(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab ma'lumotlari topilmadi", http.StatusBadRequest)
		return
	}

	var req SaveAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz so'rov formati", http.StatusBadRequest)
		return
	}

	if req.ClassID <= 0 {
		JSONError(w, "Sinf tanlanishi shart", http.StatusBadRequest)
		return
	}

	if user.Role == "teacher" {
		class, err := h.db.Queries.GetClassByID(r.Context(), req.ClassID)
		if err != nil || !class.TeacherID.Valid || class.TeacherID.Int64 != user.ID {
			JSONError(w, "Siz faqat o'zingizga biriktirilgan sinf davomatini saqlay olasiz", http.StatusForbidden)
			return
		}
	}

	targetDate := time.Now().In(h.cfg.Location)
	if strings.TrimSpace(req.Date) != "" {
		parsedDate, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			JSONError(w, "Sana formati noto'g'ri (YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		targetDate = parsedDate
	}

	if err := h.service.SaveClassAttendance(r.Context(), req.ClassID, targetDate, req.Items, user.ID); err != nil {
		JSONError(w, "Davomatni saqlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if h.onAttendanceSaved != nil && user.SchoolID.Valid {
		for _, item := range req.Items {
			if strings.ToLower(strings.TrimSpace(item.Status)) == service.StatusAbsent {
				go h.onAttendanceSaved(item.StudentID, user.SchoolID.Int64, targetDate)
			}
		}
	}

	JSONResponse(w, map[string]any{
		"message":     "Davomat muvaffaqiyatli saqlandi",
		"saved_count": len(req.Items),
		"date":        targetDate.Format("2006-01-02"),
	}, http.StatusOK)
}

func (h *AttendanceHandler) GetAttendanceLogs(w http.ResponseWriter, r *http.Request) {
	studentIDStr := r.URL.Query().Get("student_id")
	if studentIDStr == "" {
		JSONError(w, "O'quvchi tanlanishi shart ('student_id')", http.StatusBadRequest)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz o'quvchi identifikatori", http.StatusBadRequest)
		return
	}

	logs, err := h.db.Queries.ListAttendanceLogsByStudent(r.Context(), studentID)
	if err != nil {
		JSONError(w, "Audit tarixini olishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, logs, http.StatusOK)
}

