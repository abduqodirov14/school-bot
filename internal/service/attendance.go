package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/db"
)

const (
	StatusPresent  = "present"
	StatusAbsent   = "absent"
	StatusLate     = "late"
	StatusExcused  = "excused"
	StatusUnmarked = "unmarked"
)

// ValidStatuses contains all supported attendance statuses.
var ValidStatuses = map[string]bool{
	StatusPresent:  true,
	StatusAbsent:   true,
	StatusLate:     true,
	StatusExcused:  true,
	StatusUnmarked: true,
}

// NextStatus computes the next status in the cyclic toggle:
// unmarked -> present -> absent -> late -> excused -> present
func NextStatus(current string) string {
	switch strings.ToLower(strings.TrimSpace(current)) {
	case StatusUnmarked:
		return StatusPresent
	case StatusPresent:
		return StatusAbsent
	case StatusAbsent:
		return StatusLate
	case StatusLate:
		return StatusExcused
	case StatusExcused:
		return StatusPresent
	default:
		return StatusPresent
	}
}

// StatusLabel returns Uzbek Latin label for status.
func StatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusPresent:
		return "Keldi"
	case StatusAbsent:
		return "Kelmadi"
	case StatusLate:
		return "Kechikdi"
	case StatusExcused:
		return "Sababli"
	default:
		return "Belgilanmagan"
	}
}

// StatusCode returns single-letter code used in reporting (+, -, K, S).
func StatusCode(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusPresent:
		return "+"
	case StatusAbsent:
		return "-"
	case StatusLate:
		return "K"
	case StatusExcused:
		return "S"
	default:
		return ""
	}
}

// StatusEmoji returns emoji symbol for Telegram Bot representation.
func StatusEmoji(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case StatusPresent:
		return "✅"
	case StatusAbsent:
		return "❌"
	case StatusLate:
		return "⏳"
	case StatusExcused:
		return "📄"
	default:
		return "⚪"
	}
}

type AttendanceItemInput struct {
	StudentID int64  `json:"student_id"`
	Status    string `json:"status"`
	Note      string `json:"note"`
}

type ClassAttendanceResponse struct {
	ClassID   int64                            `json:"class_id"`
	Date      string                           `json:"date"`
	IsSaved   bool                             `json:"is_saved"`
	Students  []db.GetClassAttendanceByDateRow `json:"students"`
	Summary   AttendanceSummary                `json:"summary"`
}

type AttendanceSummary struct {
	Total    int `json:"total"`
	Present  int `json:"present"`
	Absent   int `json:"absent"`
	Late     int `json:"late"`
	Excused  int `json:"excused"`
	Unmarked int `json:"unmarked"`
}

type AttendanceService struct {
	db *db.DB
}

func NewAttendanceService(database *db.DB) *AttendanceService {
	return &AttendanceService{db: database}
}

func (s *AttendanceService) GetClassAttendance(ctx context.Context, classID int64, targetDate time.Time) (*ClassAttendanceResponse, error) {
	pgDate := pgtype.Date{Time: targetDate, Valid: true}

	rows, err := s.db.Queries.GetClassAttendanceByDate(ctx, db.GetClassAttendanceByDateParams{
		ClassID: classID,
		Date:    pgDate,
	})
	if err != nil {
		return nil, fmt.Errorf("davomatni olishda xatolik: %w", err)
	}

	summary := AttendanceSummary{
		Total: len(rows),
	}
	isSaved := false

	for _, row := range rows {
		switch row.Status {
		case StatusPresent:
			summary.Present++
			isSaved = true
		case StatusAbsent:
			summary.Absent++
			isSaved = true
		case StatusLate:
			summary.Late++
			isSaved = true
		case StatusExcused:
			summary.Excused++
			isSaved = true
		default:
			summary.Unmarked++
		}
	}

	return &ClassAttendanceResponse{
		ClassID:  classID,
		Date:     targetDate.Format("2006-01-02"),
		IsSaved:  isSaved,
		Students: rows,
		Summary:  summary,
	}, nil
}

func (s *AttendanceService) SaveClassAttendance(ctx context.Context, classID int64, targetDate time.Time, items []AttendanceItemInput, teacherUserID int64) error {
	pgDate := pgtype.Date{Time: targetDate, Valid: true}
	teacherID := pgtype.Int8{Int64: teacherUserID, Valid: teacherUserID > 0}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tranzaksiyani boshlashda xatolik: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.db.Queries.WithTx(tx)

	for _, item := range items {
		status := strings.ToLower(strings.TrimSpace(item.Status))
		if !ValidStatuses[status] {
			status = StatusUnmarked
		}

		// Check existing record
		existing, err := qtx.GetAttendanceRecord(ctx, db.GetAttendanceRecordParams{
			StudentID: item.StudentID,
			Date:      pgDate,
		})

		var oldStatus = StatusUnmarked
		var isNew = false
		if err != nil {
			// Record does not exist yet
			isNew = true
		} else {
			oldStatus = existing.Status
		}

		noteText := pgtype.Text{String: strings.TrimSpace(item.Note), Valid: strings.TrimSpace(item.Note) != ""}

		// Upsert attendance
		record, err := qtx.UpsertAttendance(ctx, db.UpsertAttendanceParams{
			StudentID: item.StudentID,
			Date:      pgDate,
			Status:    status,
			Note:      noteText,
			MarkedBy:  teacherID,
		})
		if err != nil {
			return fmt.Errorf("o'quvchi (ID %d) davomatini saqlashda xatolik: %w", item.StudentID, err)
		}

		// Log change if status or note changed or if newly marked
		if isNew || oldStatus != status || (existing.Note.Valid && existing.Note.String != item.Note) {
			_, err = qtx.CreateAttendanceLog(ctx, db.CreateAttendanceLogParams{
				AttendanceID: record.ID,
				StudentID:    item.StudentID,
				Date:         pgDate,
				ChangedBy:    teacherID,
				OldStatus:    oldStatus,
				NewStatus:    status,
				Note:         noteText,
			})
			if err != nil {
				return fmt.Errorf("davomat audit jurnalini yozishda xatolik: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

