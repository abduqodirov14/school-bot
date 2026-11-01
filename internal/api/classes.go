package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/config"
	"school-attendance/internal/db"
	"school-attendance/internal/importer"
)

type ClassHandler struct {
	db  *db.DB
	cfg *config.Config
}

func NewClassHandler(database *db.DB, cfg *config.Config) *ClassHandler {
	return &ClassHandler{db: database, cfg: cfg}
}

type CreateClassRequest struct {
	Name      string `json:"name"`
	TeacherID *int64 `json:"teacher_id"`
}

type UpdateClassRequest struct {
	Name      string `json:"name"`
	TeacherID *int64 `json:"teacher_id"`
}

type BulkCreateClassesRequest struct {
	StartGrade int      `json:"start_grade"`
	EndGrade   int      `json:"end_grade"`
	Parallels  []string `json:"parallels"`
}

type BulkCreateClassesResponse struct {
	CreatedCount int        `json:"created_count"`
	SkippedCount int        `json:"skipped_count"`
	Classes      []db.Class `json:"classes"`
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

	// Teacher role: list active classes where user is primary or substitute today
	now := time.Now().In(h.cfg.Location)
	todayPg := pgtype.Date{Time: now, Valid: true}
	teacherClasses, err := h.db.Queries.ListActiveClassesForTeacher(r.Context(), db.ListActiveClassesForTeacherParams{
		UserID:  user.ID,
		Column2: todayPg,
	})
	if err != nil {
		// Fallback to legacy ListClassesByTeacherID if class_teachers has no entries
		classes, _ := h.db.Queries.ListClassesByTeacherID(r.Context(), pgtype.Int8{Int64: user.ID, Valid: true})
		if classes == nil {
			classes = make([]db.Class, 0)
		}
		JSONResponse(w, classes, http.StatusOK)
		return
	}

	if teacherClasses == nil {
		teacherClasses = make([]db.ListActiveClassesForTeacherRow, 0)
	}
	JSONResponse(w, teacherClasses, http.StatusOK)
}

func (h *ClassHandler) CreateClass(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori sinf yarata oladi", http.StatusForbidden)
		return
	}

	var req CreateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar formati", http.StatusBadRequest)
		return
	}

	rawName := strings.TrimSpace(req.Name)
	if rawName == "" {
		JSONError(w, "Sinf nomi kiritilishi shart", http.StatusBadRequest)
		return
	}

	normName, _ := importer.NormalizeClassName(rawName)
	if normName == "" {
		normName = rawName
	}

	var teacherID pgtype.Int8
	if req.TeacherID != nil && *req.TeacherID > 0 {
		teacherID = pgtype.Int8{Int64: *req.TeacherID, Valid: true}
	}

	created, err := h.db.Queries.CreateClass(r.Context(), db.CreateClassParams{
		SchoolID:  user.SchoolID.Int64,
		TeacherID: teacherID,
		Name:      normName,
	})
	if err != nil {
		JSONError(w, "Sinfni yaratishda xatolik (balki bunday sinf allaqachon mavjuddir): "+err.Error(), http.StatusInternalServerError)
		return
	}

	if teacherID.Valid {
		_ = h.db.Queries.TerminateActivePrimaryTeacher(r.Context(), created.ID)
		_, _ = h.db.Queries.AssignPrimaryTeacher(r.Context(), db.AssignPrimaryTeacherParams{
			ClassID:    created.ID,
			UserID:     teacherID.Int64,
			AssignedBy: pgtype.Int8{Int64: user.ID, Valid: true},
		})
	}

	JSONResponse(w, created, http.StatusCreated)
}

// BulkCreateClasses creates multiple classes in a single transaction (e.g. 5-11 grades, A, B, V parallels)
func (h *ClassHandler) BulkCreateClasses(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori sinflarni ommaviy yarata oladi", http.StatusForbidden)
		return
	}

	var req BulkCreateClassesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar formati", http.StatusBadRequest)
		return
	}

	if req.StartGrade < 1 || req.EndGrade > 12 || req.StartGrade > req.EndGrade {
		JSONError(w, "Sinf oraliqlari noto'g'ri (masalan: 5 dan 11 gacha bo'lishi kerak)", http.StatusBadRequest)
		return
	}

	if len(req.Parallels) == 0 {
		JSONError(w, "Kamida bitta parallel (harf) tanlanishi shart (masalan: A, B, V)", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		JSONError(w, "Tranzaksiyani boshlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	qtx := h.db.Queries.WithTx(tx)

	var createdClasses []db.Class
	createdCount := 0
	skippedCount := 0

	for grade := req.StartGrade; grade <= req.EndGrade; grade++ {
		for _, rawP := range req.Parallels {
			normP, _ := importer.NormalizeClassName(rawP)
			if normP == "" {
				normP = strings.ToUpper(strings.TrimSpace(rawP))
			}
			className := fmt.Sprintf("%d%s", grade, normP)

			created, err := qtx.CreateClassIfNotExists(ctx, db.CreateClassIfNotExistsParams{
				SchoolID: user.SchoolID.Int64,
				Name:     className,
			})
			if err != nil {
				// class already existed
				skippedCount++
				continue
			}
			if created.ID == 0 {
				skippedCount++
			} else {
				createdCount++
				createdClasses = append(createdClasses, created)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		JSONError(w, "Tranzaksiyani saqlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, BulkCreateClassesResponse{
		CreatedCount: createdCount,
		SkippedCount: skippedCount,
		Classes:      createdClasses,
	}, http.StatusOK)
}

func (h *ClassHandler) UpdateClass(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori sinfni yangilay oladi", http.StatusForbidden)
		return
	}

	classIDStr := chi.URLParam(r, "id")
	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
		return
	}

	var req UpdateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar formati", http.StatusBadRequest)
		return
	}

	rawName := strings.TrimSpace(req.Name)
	if rawName == "" {
		JSONError(w, "Sinf nomi kiritilishi shart", http.StatusBadRequest)
		return
	}

	normName, _ := importer.NormalizeClassName(rawName)
	if normName == "" {
		normName = rawName
	}

	var teacherID pgtype.Int8
	if req.TeacherID != nil && *req.TeacherID > 0 {
		teacherID = pgtype.Int8{Int64: *req.TeacherID, Valid: true}
	}

	updated, err := h.db.Queries.UpdateClass(r.Context(), db.UpdateClassParams{
		ID:        classID,
		Name:      normName,
		TeacherID: teacherID,
	})
	if err != nil {
		JSONError(w, "Sinfni yangilashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if teacherID.Valid {
		_ = h.db.Queries.TerminateActivePrimaryTeacher(r.Context(), classID)
		_, _ = h.db.Queries.AssignPrimaryTeacher(r.Context(), db.AssignPrimaryTeacherParams{
			ClassID:    classID,
			UserID:     teacherID.Int64,
			AssignedBy: pgtype.Int8{Int64: user.ID, Valid: true},
		})
	}

	JSONResponse(w, updated, http.StatusOK)
}

func (h *ClassHandler) DeleteClass(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori sinfni o'chira oladi", http.StatusForbidden)
		return
	}

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

type AssignTeacherRequest struct {
	TelegramID     int64  `json:"telegram_id"`
	FullName       string `json:"full_name"`
	GenerateInvite bool   `json:"generate_invite"`
}

type AssignTeacherResponse struct {
	ClassID     int64      `json:"class_id"`
	Teacher     *db.User   `json:"teacher,omitempty"`
	InviteToken string     `json:"invite_token,omitempty"`
	InviteURL   string     `json:"invite_url,omitempty"`
	ShareURL    string     `json:"share_url,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Message     string     `json:"message"`
}

// AssignTeacher handles POST /api/classes/{id}/teacher
func (h *ClassHandler) AssignTeacher(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori sinf rahbarini biriktira oladi", http.StatusForbidden)
		return
	}

	classIDStr := chi.URLParam(r, "id")
	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
		return
	}

	targetClass, err := h.db.Queries.GetClassByID(r.Context(), classID)
	if err != nil {
		JSONError(w, "Sinf topilmadi", http.StatusNotFound)
		return
	}

	var req AssignTeacherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar", http.StatusBadRequest)
		return
	}

	// 1. Generate Invite Link flow
	if req.GenerateInvite {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			JSONError(w, "Token generatsiya qilishda xatolik", http.StatusInternalServerError)
			return
		}
		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		expiresAt := time.Now().Add(7 * 24 * time.Hour)

		invite, err := h.db.Queries.CreateInvite(r.Context(), db.CreateInviteParams{
			Token:     token,
			SchoolID:  user.SchoolID.Int64,
			ClassID:   classID,
			Role:      "primary",
			CreatedBy: pgtype.Int8{Int64: user.ID, Valid: true},
			ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		})
		if err != nil {
			JSONError(w, "Taklif havolasini yaratishda xatolik: "+err.Error(), http.StatusInternalServerError)
			return
		}

		botUsername := h.cfg.BotUsername
		if botUsername == "" {
			botUsername = "HozirDavomatBot"
		}
		inviteURL := fmt.Sprintf("https://t.me/%s?start=inv_%s", botUsername, token)
		shareText := fmt.Sprintf("Assalomu alaykum! %s sinfi rahbari bo'lish uchun quyidagi havola orqali botga kiring:", targetClass.Name)
		shareURL := fmt.Sprintf("https://t.me/share/url?url=%s&text=%s", url.QueryEscape(inviteURL), url.QueryEscape(shareText))

		var expTime *time.Time
		if invite.ExpiresAt.Valid {
			expTime = &invite.ExpiresAt.Time
		}

		JSONResponse(w, AssignTeacherResponse{
			ClassID:     classID,
			InviteToken: token,
			InviteURL:   inviteURL,
			ShareURL:    shareURL,
			ExpiresAt:   expTime,
			Message:     "Taklif havolasi muvaffaqiyatli yaratildi",
		}, http.StatusOK)
		return
	}

	// 2. Direct assignment by Telegram ID
	if req.TelegramID <= 0 {
		JSONError(w, "Telegram ID yoki 'generate_invite: true' ko'rsatilishi shart", http.StatusBadRequest)
		return
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		fullName = fmt.Sprintf("O'qituvchi %d", req.TelegramID)
	}

	// In transaction: upsert user as invited, terminate current primary, assign new primary
	ctx := r.Context()
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		JSONError(w, "Tranzaksiyani boshlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	qtx := h.db.Queries.WithTx(tx)

	teacherUser, err := qtx.UpsertInvitedUser(ctx, db.UpsertInvitedUserParams{
		SchoolID:   user.SchoolID,
		TelegramID: req.TelegramID,
		FullName:   fullName,
	})
	if err != nil {
		JSONError(w, "Foydalanuvchini yaratishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = qtx.TerminateActivePrimaryTeacher(ctx, classID)

	_, err = qtx.AssignPrimaryTeacher(ctx, db.AssignPrimaryTeacherParams{
		ClassID:    classID,
		UserID:     teacherUser.ID,
		AssignedBy: pgtype.Int8{Int64: user.ID, Valid: true},
	})
	if err != nil {
		JSONError(w, "Sinf rahbarini biriktirishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, _ = qtx.UpdateClassTeacher(ctx, db.UpdateClassTeacherParams{
		ID:        classID,
		TeacherID: pgtype.Int8{Int64: teacherUser.ID, Valid: true},
	})

	if err := tx.Commit(ctx); err != nil {
		JSONError(w, "Tranzaksiyani saqlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, AssignTeacherResponse{
		ClassID: classID,
		Teacher: &teacherUser,
		Message: "Sinf rahbari muvaffaqiyatli biriktirildi",
	}, http.StatusOK)
}

// UnassignTeacher handles DELETE /api/classes/{id}/teacher
func (h *ClassHandler) UnassignTeacher(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori sinf rahbarini bekor qila oladi", http.StatusForbidden)
		return
	}

	classIDStr := chi.URLParam(r, "id")
	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tx, err := h.db.Pool.Begin(ctx)
	if err != nil {
		JSONError(w, "Tranzaksiyani boshlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	qtx := h.db.Queries.WithTx(tx)

	_ = qtx.TerminateActivePrimaryTeacher(ctx, classID)
	_, _ = qtx.UpdateClassTeacher(ctx, db.UpdateClassTeacherParams{
		ID:        classID,
		TeacherID: pgtype.Int8{Valid: false},
	})

	if err := tx.Commit(ctx); err != nil {
		JSONError(w, "Saqlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]string{
		"message": "Sinf rahbari vakolati darhol bekor qilindi",
	}, http.StatusOK)
}

type AssignSubstituteRequest struct {
	UserID    int64  `json:"user_id"`
	Date      string `json:"date,omitempty"` // YYYY-MM-DD
	ValidFrom string `json:"valid_from,omitempty"`
	ValidTo   string `json:"valid_until,omitempty"`
}

// AssignSubstitute handles POST /api/classes/{id}/substitute
func (h *ClassHandler) AssignSubstitute(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Faqat maktab direktori o'rinbosar biriktira oladi", http.StatusForbidden)
		return
	}

	classIDStr := chi.URLParam(r, "id")
	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz sinf identifikatori", http.StatusBadRequest)
		return
	}

	var req AssignSubstituteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar", http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 {
		JSONError(w, "O'qituvchi tanlanishi shart", http.StatusBadRequest)
		return
	}

	// Parse date range
	now := time.Now().In(h.cfg.Location)
	from := now
	to := now

	if req.Date != "" {
		if t, err := time.Parse("2006-01-02", req.Date); err == nil {
			from = t
			to = t
		}
	} else {
		if req.ValidFrom != "" {
			if t, err := time.Parse("2006-01-02", req.ValidFrom); err == nil {
				from = t
			}
		}
		if req.ValidTo != "" {
			if t, err := time.Parse("2006-01-02", req.ValidTo); err == nil {
				to = t
			}
		}
	}

	pgFrom := pgtype.Date{Time: from, Valid: true}
	pgTo := pgtype.Date{Time: to, Valid: true}

	sub, err := h.db.Queries.AssignSubstituteTeacher(r.Context(), db.AssignSubstituteTeacherParams{
		ClassID:    classID,
		UserID:     req.UserID,
		ValidFrom:  pgFrom,
		ValidUntil: pgTo,
		AssignedBy: pgtype.Int8{Int64: user.ID, Valid: true},
	})
	if err != nil {
		JSONError(w, "O'rinbosar o'qituvchini biriktirishda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, map[string]any{
		"message":    "O'rinbosar o'qituvchi muvaffaqiyatli biriktirildi",
		"substitute": sub,
	}, http.StatusOK)
}

// ListTeachersOverview handles GET /api/classes/teachers/overview
func (h *ClassHandler) ListTeachersOverview(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid || user.Role != "director" {
		JSONError(w, "Ruxsat etilmagan", http.StatusForbidden)
		return
	}

	now := time.Now().In(h.cfg.Location)
	todayPg := pgtype.Date{Time: now, Valid: true}

	overview, err := h.db.Queries.ListClassTeachersOverview(r.Context(), db.ListClassTeachersOverviewParams{
		SchoolID: user.SchoolID.Int64,
		Column2:  todayPg,
	})
	if err != nil {
		JSONError(w, "Sinf rahbarlari ro'yxatini yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if overview == nil {
		overview = make([]db.ListClassTeachersOverviewRow, 0)
	}

	botUsername := h.cfg.BotUsername
	if botUsername == "" {
		botUsername = "HozirDavomatBot"
	}

	type TeacherOverviewItem struct {
		ClassID            int64      `json:"class_id"`
		ClassName          string     `json:"class_name"`
		StudentCount       int64      `json:"student_count"`
		PrimaryUserID      *int64     `json:"primary_user_id,omitempty"`
		PrimaryTeacherName string     `json:"primary_teacher_name"`
		PrimaryTelegramID  *int64     `json:"primary_telegram_id,omitempty"`
		PrimaryStatus      string     `json:"primary_status"` // "active", "invited", "none"
		SubUserID          *int64     `json:"sub_user_id,omitempty"`
		SubTeacherName     string     `json:"sub_teacher_name,omitempty"`
		SubValidFrom       *string    `json:"sub_valid_from,omitempty"`
		SubValidUntil      *string    `json:"sub_valid_until,omitempty"`
		ActiveInviteToken  string     `json:"active_invite_token,omitempty"`
		InviteURL          string     `json:"invite_url,omitempty"`
		ShareURL           string     `json:"share_url,omitempty"`
		InviteExpiresAt    *time.Time `json:"invite_expires_at,omitempty"`
	}

	var items []TeacherOverviewItem
	for _, row := range overview {
		status := "none"
		if row.PrimaryTeacherName.Valid && row.PrimaryTeacherName.String != "" {
			if row.PrimaryUserStatus.Valid && row.PrimaryUserStatus.String != "" {
				status = row.PrimaryUserStatus.String
			} else {
				status = "active"
			}
		}

		var primID *int64
		if row.PrimaryUserID.Valid {
			primID = &row.PrimaryUserID.Int64
		}
		var primTG *int64
		if row.PrimaryTelegramID.Valid {
			primTG = &row.PrimaryTelegramID.Int64
		}

		var subID *int64
		if row.SubUserID.Valid {
			subID = &row.SubUserID.Int64
		}
		subName := ""
		if row.SubTeacherName.Valid {
			subName = row.SubTeacherName.String
		}

		var subFrom, subTo *string
		if row.SubValidFrom.Valid {
			str := row.SubValidFrom.Time.Format("2006-01-02")
			subFrom = &str
		}
		if row.SubValidUntil.Valid {
			str := row.SubValidUntil.Time.Format("2006-01-02")
			subTo = &str
		}

		token := ""
		inviteURL := ""
		shareURL := ""
		var expiresAt *time.Time
		if row.ActiveInviteToken.Valid && row.ActiveInviteToken.String != "" {
			token = row.ActiveInviteToken.String
			inviteURL = fmt.Sprintf("https://t.me/%s?start=inv_%s", botUsername, token)
			shareText := fmt.Sprintf("Assalomu alaykum! %s sinfi rahbari bo'lish uchun havola:", row.ClassName)
			shareURL = fmt.Sprintf("https://t.me/share/url?url=%s&text=%s", url.QueryEscape(inviteURL), url.QueryEscape(shareText))
			if row.ActiveInviteExpiresAt.Valid {
				expiresAt = &row.ActiveInviteExpiresAt.Time
			}
		}

		items = append(items, TeacherOverviewItem{
			ClassID:            row.ClassID,
			ClassName:          row.ClassName,
			StudentCount:       row.StudentCount,
			PrimaryUserID:      primID,
			PrimaryTeacherName: row.PrimaryTeacherName.String,
			PrimaryTelegramID:  primTG,
			PrimaryStatus:      status,
			SubUserID:          subID,
			SubTeacherName:     subName,
			SubValidFrom:       subFrom,
			SubValidUntil:      subTo,
			ActiveInviteToken:  token,
			InviteURL:          inviteURL,
			ShareURL:           shareURL,
			InviteExpiresAt:    expiresAt,
		})
	}

	JSONResponse(w, items, http.StatusOK)
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
