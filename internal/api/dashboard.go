package api

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/config"
	"school-attendance/internal/db"
)

type DashboardHandler struct {
	db  *db.DB
	cfg *config.Config
}

func NewDashboardHandler(database *db.DB, cfg *config.Config) *DashboardHandler {
	return &DashboardHandler{db: database, cfg: cfg}
}

type GroupedStudent struct {
	StudentID   int64  `json:"student_id"`
	StudentName string `json:"student_name"`
	Status      string `json:"status"`
	Note        string `json:"note"`
}

type ClassGroupedStudents struct {
	ClassID   int64            `json:"class_id"`
	ClassName string           `json:"class_name"`
	Students  []GroupedStudent `json:"students"`
}

type ClassAttendanceStatusItem struct {
	ClassID        int64  `json:"class_id"`
	ClassName      string `json:"class_name"`
	TeacherID      *int64 `json:"teacher_id,omitempty"`
	TeacherName    string `json:"teacher_name"`
	TotalStudents  int64  `json:"total_students"`
	MarkedStudents int64  `json:"marked_students"`
	IsTaken        bool   `json:"is_taken"`
}

type TodayDashboardResponse struct {
	Date                 string                      `json:"date"`
	TotalStudents        int64                       `json:"total_students"`
	PresentCount         int64                       `json:"present_count"`
	AbsentCount          int64                       `json:"absent_count"`
	LateCount            int64                       `json:"late_count"`
	ExcusedCount         int64                       `json:"excused_count"`
	AttendancePercentage float64                     `json:"attendance_percentage"`
	AbsentList           []ClassGroupedStudents      `json:"absent_list"`
	LateList             []ClassGroupedStudents      `json:"late_list"`
	ClassesStatus        []ClassAttendanceStatusItem `json:"classes_status"`
	UnmarkedClassesCount int                         `json:"unmarked_classes_count"`
}

func (h *DashboardHandler) GetTodayDashboard(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab ma'lumotlari topilmadi", http.StatusBadRequest)
		return
	}

	dateStr := r.URL.Query().Get("date")
	targetDate := time.Now().In(h.cfg.Location)
	if dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			targetDate = parsedDate
		}
	}
	pgDate := pgtype.Date{Time: targetDate, Valid: true}

	// 1. Overall stats
	stats, err := h.db.Queries.GetTodaySchoolStats(r.Context(), db.GetTodaySchoolStatsParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})
	if err != nil {
		JSONError(w, "Statistikani yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var percentage float64
	if stats.TotalStudents > 0 {
		percentage = math.Round((float64(stats.PresentCount)/float64(stats.TotalStudents))*1000) / 10
	}

	// 2. Absent students grouped by class
	absentRows, err := h.db.Queries.ListAbsentStudentsToday(r.Context(), db.ListAbsentStudentsTodayParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})
	if err != nil {
		absentRows = nil
	}

	absentGroupMap := make(map[int64]*ClassGroupedStudents)
	var absentGroups []ClassGroupedStudents
	for _, row := range absentRows {
		group, exists := absentGroupMap[row.ClassID]
		if !exists {
			group = &ClassGroupedStudents{
				ClassID:   row.ClassID,
				ClassName: row.ClassName,
				Students:  make([]GroupedStudent, 0),
			}
			absentGroupMap[row.ClassID] = group
			absentGroups = append(absentGroups, *group)
		}
		// append to group
		for i := range absentGroups {
			if absentGroups[i].ClassID == row.ClassID {
				absentGroups[i].Students = append(absentGroups[i].Students, GroupedStudent{
					StudentID:   row.StudentID,
					StudentName: row.StudentName,
					Status:      row.Status,
					Note:        row.Note,
				})
				break
			}
		}
	}

	// 3. Late students grouped by class
	lateRows, err := h.db.Queries.ListLateStudentsToday(r.Context(), db.ListLateStudentsTodayParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})
	if err != nil {
		lateRows = nil
	}

	var lateGroups []ClassGroupedStudents
	for _, row := range lateRows {
		found := false
		for i := range lateGroups {
			if lateGroups[i].ClassID == row.ClassID {
				lateGroups[i].Students = append(lateGroups[i].Students, GroupedStudent{
					StudentID:   row.StudentID,
					StudentName: row.StudentName,
					Status:      row.Status,
					Note:        row.Note,
				})
				found = true
				break
			}
		}
		if !found {
			lateGroups = append(lateGroups, ClassGroupedStudents{
				ClassID:   row.ClassID,
				ClassName: row.ClassName,
				Students: []GroupedStudent{
					{
						StudentID:   row.StudentID,
						StudentName: row.StudentName,
						Status:      row.Status,
						Note:        row.Note,
					},
				},
			})
		}
	}

	// 4. Class attendance status
	classesRows, err := h.db.Queries.ListClassesWithAttendanceStatusToday(r.Context(), db.ListClassesWithAttendanceStatusTodayParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})
	if err != nil {
		classesRows = nil
	}

	var classStatusList []ClassAttendanceStatusItem
	unmarkedCount := 0
	for _, c := range classesRows {
		isTaken := c.MarkedStudents > 0
		if !isTaken && c.TotalStudents > 0 {
			unmarkedCount++
		}

		var teacherID *int64
		if c.TeacherID.Valid {
			teacherID = &c.TeacherID.Int64
		}

		classStatusList = append(classStatusList, ClassAttendanceStatusItem{
			ClassID:        c.ClassID,
			ClassName:      c.ClassName,
			TeacherID:      teacherID,
			TeacherName:    c.TeacherName,
			TotalStudents:  c.TotalStudents,
			MarkedStudents: c.MarkedStudents,
			IsTaken:        isTaken,
		})
	}

	resp := TodayDashboardResponse{
		Date:                 targetDate.Format("2006-01-02"),
		TotalStudents:        stats.TotalStudents,
		PresentCount:         stats.PresentCount,
		AbsentCount:          stats.AbsentCount,
		LateCount:            stats.LateCount,
		ExcusedCount:         stats.ExcusedCount,
		AttendancePercentage: percentage,
		AbsentList:           absentGroups,
		LateList:             lateGroups,
		ClassesStatus:        classStatusList,
		UnmarkedClassesCount: unmarkedCount,
	}

	JSONResponse(w, resp, http.StatusOK)
}

func (h *DashboardHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab ma'lumotlari topilmadi", http.StatusBadRequest)
		return
	}

	settings, err := h.db.Queries.GetSettingsBySchoolID(r.Context(), user.SchoolID.Int64)
	if err != nil {
		// Return default settings
		JSONResponse(w, map[string]any{
			"school_id":         user.SchoolID.Int64,
			"absence_threshold": 3,
			"deadline_time":     "09:00",
		}, http.StatusOK)
		return
	}

	JSONResponse(w, settings, http.StatusOK)
}

type UpdateSettingsRequest struct {
	AbsenceThreshold int32  `json:"absence_threshold"`
	DeadlineTime     string `json:"deadline_time"`
}

func (h *DashboardHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab ma'lumotlari topilmadi", http.StatusBadRequest)
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONError(w, "Yaroqsiz ma'lumotlar", http.StatusBadRequest)
		return
	}

	if req.AbsenceThreshold <= 0 {
		req.AbsenceThreshold = 3
	}
	req.DeadlineTime = strings.TrimSpace(req.DeadlineTime)
	if req.DeadlineTime == "" {
		req.DeadlineTime = "09:00"
	}

	saved, err := h.db.Queries.UpsertSettings(r.Context(), db.UpsertSettingsParams{
		SchoolID:         user.SchoolID.Int64,
		AbsenceThreshold: req.AbsenceThreshold,
		DeadlineTime:     req.DeadlineTime,
	})
	if err != nil {
		JSONError(w, "Sozlamalarni saqlashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	JSONResponse(w, saved, http.StatusOK)
}

type StudentHistoryDayItem struct {
	Date   string `json:"date"`
	Day    int    `json:"day"`
	Status string `json:"status"`
	Note   string `json:"note"`
}

type StudentHistoryResponse struct {
	StudentID            int64                   `json:"student_id"`
	StudentName          string                  `json:"student_name"`
	ClassName            string                  `json:"class_name"`
	YearMonth            string                  `json:"year_month"`
	Threshold            int32                   `json:"threshold"`
	TotalDaysMarked      int                     `json:"total_days_marked"`
	PresentCount         int                     `json:"present_count"`
	AbsentCount          int                     `json:"absent_count"`
	LateCount            int                     `json:"late_count"`
	ExcusedCount         int                     `json:"excused_count"`
	ExceededThreshold    bool                    `json:"exceeded_threshold"`
	Days                 []StudentHistoryDayItem `json:"days"`
}

func (h *DashboardHandler) GetStudentHistory(w http.ResponseWriter, r *http.Request) {
	user, ok := GetUser(r.Context())
	if !ok || !user.SchoolID.Valid {
		JSONError(w, "Maktab ma'lumotlari topilmadi", http.StatusBadRequest)
		return
	}

	studentIDStr := chi.URLParam(r, "id")
	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		JSONError(w, "Yaroqsiz o'quvchi identifikatori", http.StatusBadRequest)
		return
	}

	student, err := h.db.Queries.GetStudentByID(r.Context(), studentID)
	if err != nil {
		JSONError(w, "O'quvchi topilmadi", http.StatusNotFound)
		return
	}

	// If caller is teacher, ensure this student belongs to their class
	if user.Role == "teacher" {
		class, err := h.db.Queries.GetClassByID(r.Context(), student.ClassID)
		if err != nil || !class.TeacherID.Valid || class.TeacherID.Int64 != user.ID {
			JSONError(w, "Ushbu o'quvchi davomat tarixini ko'rishga ruxsat yo'q", http.StatusForbidden)
			return
		}
	}

	now := time.Now().In(h.cfg.Location)
	monthStr := r.URL.Query().Get("month") // YYYY-MM
	var year int = now.Year()
	var month time.Month = now.Month()
	if monthStr != "" {
		parts := strings.Split(monthStr, "-")
		if len(parts) == 2 {
			y, yErr := strconv.Atoi(parts[0])
			m, mErr := strconv.Atoi(parts[1])
			if yErr == nil && mErr == nil && m >= 1 && m <= 12 {
				year = y
				month = time.Month(m)
			}
		}
	}

	startDate := time.Date(year, month, 1, 0, 0, 0, 0, h.cfg.Location)
	endDate := startDate.AddDate(0, 1, -1)

	records, err := h.db.Queries.GetStudentAttendanceHistory(r.Context(), db.GetStudentAttendanceHistoryParams{
		StudentID: studentID,
		Date:      pgtype.Date{Time: startDate, Valid: true},
		Date_2:    pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		JSONError(w, "Tarixni yuklashda xatolik: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get school settings for threshold
	var threshold int32 = 3
	settings, err := h.db.Queries.GetSettingsBySchoolID(r.Context(), user.SchoolID.Int64)
	if err == nil && settings.AbsenceThreshold > 0 {
		threshold = settings.AbsenceThreshold
	}

	resp := StudentHistoryResponse{
		StudentID:   student.ID,
		StudentName: student.FullName,
		ClassName:   student.ClassName,
		YearMonth:   startDate.Format("2006-01"),
		Threshold:   threshold,
		Days:        make([]StudentHistoryDayItem, 0),
	}

	for _, rec := range records {
		if !rec.Date.Valid {
			continue
		}
		recDate := rec.Date.Time
		noteStr := ""
		if rec.Note.Valid {
			noteStr = rec.Note.String
		}

		resp.Days = append(resp.Days, StudentHistoryDayItem{
			Date:   recDate.Format("2006-01-02"),
			Day:    recDate.Day(),
			Status: rec.Status,
			Note:   noteStr,
		})

		resp.TotalDaysMarked++
		switch rec.Status {
		case "present":
			resp.PresentCount++
		case "absent":
			resp.AbsentCount++
		case "late":
			resp.LateCount++
		case "excused":
			resp.ExcusedCount++
		}
	}

	if int32(resp.AbsentCount) >= threshold {
		resp.ExceededThreshold = true
	}

	JSONResponse(w, resp, http.StatusOK)
}

