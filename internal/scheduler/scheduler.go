package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/robfig/cron/v3"

	"school-attendance/internal/bot"
	"school-attendance/internal/config"
	"school-attendance/internal/db"
)

type Scheduler struct {
	cron *cron.Cron
	db   *db.DB
	bot  *bot.Bot
	cfg  *config.Config
}

func New(cfg *config.Config, database *db.DB, tgBot *bot.Bot) *Scheduler {
	c := cron.New(cron.WithLocation(cfg.Location))
	return &Scheduler{
		cron: c,
		db:   database,
		bot:  tgBot,
		cfg:  cfg,
	}
}

// AddMinutesToTimeStr parses "15:04" and adds minutes, returning a formatted "15:04" string.
func AddMinutesToTimeStr(timeStr string, minutes int) string {
	t, err := time.Parse("15:04", strings.TrimSpace(timeStr))
	if err != nil {
		return "09:30"
	}
	return t.Add(time.Duration(minutes) * time.Minute).Format("15:04")
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.cron == nil {
		return
	}

	// Run check every minute in Asia/Tashkent timezone
	_, err := s.cron.AddFunc("* * * * *", func() {
		s.RunChecks(ctx)
	})
	if err != nil {
		log.Printf("Scheduler xatosi: %v", err)
		return
	}

	s.cron.Start()
	log.Printf("Avtomatlashtirilgan cron scheduler ishga tushdi (Vaqt zonasi: %s)...", s.cfg.Timezone)
}

func (s *Scheduler) Stop() {
	if s != nil && s.cron != nil {
		s.cron.Stop()
	}
}

func (s *Scheduler) RunChecks(ctx context.Context) {
	now := time.Now().In(s.cfg.Location)
	currentHM := now.Format("15:04")

	school, err := s.db.Queries.GetFirstSchool(ctx)
	if err != nil {
		return
	}

	settings, err := s.db.Queries.GetSettingsBySchoolID(ctx, school.ID)
	deadlineTime := "09:00"
	var threshold int32 = 3
	if err == nil {
		if settings.DeadlineTime != "" {
			deadlineTime = settings.DeadlineTime
		}
		if settings.AbsenceThreshold > 0 {
			threshold = settings.AbsenceThreshold
		}
	}

	deadlinePlus30 := AddMinutesToTimeStr(deadlineTime, 30)

	// 1. Deadline time check: remind teachers of unmarked classes
	if currentHM == deadlineTime {
		s.RemindTeachersUnmarkedClasses(ctx, school.ID, now)
	}

	// 2. Deadline + 30 min check: notify director of still unmarked classes
	if currentHM == deadlinePlus30 {
		s.NotifyDirectorUnmarkedClasses(ctx, school.ID, now, deadlineTime)
	}

	// 3. Periodic threshold alerts check
	s.CheckThresholdAlerts(ctx, school.ID, now, threshold)
}

func (s *Scheduler) RemindTeachersUnmarkedClasses(ctx context.Context, schoolID int64, now time.Time) {
	if s.bot == nil {
		return
	}

	pgDate := pgtype.Date{Time: now, Valid: true}
	classes, err := s.db.Queries.ListClassesWithAttendanceStatusToday(ctx, db.ListClassesWithAttendanceStatusTodayParams{
		SchoolID: schoolID,
		Date:     pgDate,
	})
	if err != nil {
		return
	}

	for _, c := range classes {
		// If attendance has not been taken and class has students
		if c.MarkedStudents == 0 && c.TotalStudents > 0 {
			targetTelegramID := c.TeacherTelegramID
			targetName := c.TeacherName

			// Check effective teacher (substitute takes priority over primary)
			eff, effErr := s.db.Queries.GetEffectiveTeacherForClass(ctx, db.GetEffectiveTeacherForClassParams{
				ClassID: c.ClassID,
				Column2: pgDate,
			})
			if effErr == nil && eff.TelegramID > 0 {
				targetTelegramID = eff.TelegramID
				targetName = eff.TeacherName
			}

			if targetTelegramID > 0 {
				msg := fmt.Sprintf("⏰ **ESLATMA: Davomat topshirish vaqti keldi!**\n\n"+
					"Hurmatli *%s*, sizning **%s** sinfingiz uchun bugungi (%s) davomat hali olinmagan.\n\n"+
					"Iltimos, dars vaqtida o'quvchilar davomatini oling va saqlang.",
					targetName, c.ClassName, now.Format("02.01.2006"))

				markup := &tgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]tgbot.InlineKeyboardButton{
						{
							{
								Text:   "📋 Davomat Olish",
								WebApp: &tgbot.WebAppInfo{URL: s.cfg.WebAppURL},
							},
						},
					},
				}

				_ = s.bot.SendMessage(ctx, targetTelegramID, msg, markup)
			}
		}
	}
}

func (s *Scheduler) NotifyDirectorUnmarkedClasses(ctx context.Context, schoolID int64, now time.Time, deadlineTime string) {
	if s.bot == nil {
		return
	}

	pgDate := pgtype.Date{Time: now, Valid: true}
	classes, err := s.db.Queries.ListClassesWithAttendanceStatusToday(ctx, db.ListClassesWithAttendanceStatusTodayParams{
		SchoolID: schoolID,
		Date:     pgDate,
	})
	if err != nil {
		return
	}

	var unmarkedList []string
	for _, c := range classes {
		if c.MarkedStudents == 0 && c.TotalStudents > 0 {
			teacherText := c.TeacherName
			if teacherText == "" {
				teacherText = "O'qituvchi biriktirilmagan"
			}
			unmarkedList = append(unmarkedList, fmt.Sprintf("🏫 **%s** — O'qituvchi: %s", c.ClassName, teacherText))
		}
	}

	if len(unmarkedList) == 0 {
		return
	}

	directorTGID := s.cfg.DirectorTelegramID
	if directorTGID == 0 {
		dirUser, err := s.db.Queries.GetDirectorBySchool(ctx, pgtype.Int8{Int64: schoolID, Valid: true})
		if err == nil {
			directorTGID = dirUser.TelegramID
		}
	}

	if directorTGID == 0 {
		return
	}

	msg := fmt.Sprintf("⚠️ **DIQQAT: Davomat olinmagan sinflar!**\n\n"+
		"Bugun (%s) quyidagi sinflar belgilangan muddatdan (%s) 30 daqiqa o'tsa ham hali davomat topshirmadi:\n\n%s\n\n"+
		"Iltimos, ushbu o'qituvchilarni ogohlantiring.",
		now.Format("02.01.2006"), deadlineTime, strings.Join(unmarkedList, "\n"))

	markup := &tgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbot.InlineKeyboardButton{
			{
				{
					Text:   "📱 Boshqaruv Panelini Ochish",
					WebApp: &tgbot.WebAppInfo{URL: s.cfg.WebAppURL},
				},
			},
		},
	}

	_ = s.bot.SendMessage(ctx, directorTGID, msg, markup)
}

func (s *Scheduler) CheckThresholdAlerts(ctx context.Context, schoolID int64, now time.Time, threshold int32) {
	if s.bot == nil {
		return
	}

	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, s.cfg.Location)
	endDate := startDate.AddDate(0, 1, -1)
	yearMonth := now.Format("2006-01")

	// Find students exceeding threshold
	exceedingStudents, err := s.db.Queries.GetStudentsExceedingThreshold(ctx, db.GetStudentsExceedingThresholdParams{
		SchoolID: schoolID,
		Date:     pgtype.Date{Time: startDate, Valid: true},
		Date_2:   pgtype.Date{Time: endDate, Valid: true},
		MinAbsences: int64(threshold),
	})
	if err != nil || len(exceedingStudents) == 0 {
		return
	}

	directorTGID := s.cfg.DirectorTelegramID
	if directorTGID == 0 {
		dirUser, err := s.db.Queries.GetDirectorBySchool(ctx, pgtype.Int8{Int64: schoolID, Valid: true})
		if err == nil {
			directorTGID = dirUser.TelegramID
		}
	}

	if directorTGID == 0 {
		return
	}

	for _, st := range exceedingStudents {
		// Check if already notified this month
		sent, err := s.db.Queries.HasThresholdAlertBeenSent(ctx, db.HasThresholdAlertBeenSentParams{
			StudentID: st.StudentID,
			YearMonth: yearMonth,
		})
		if err == nil && !sent {
			// Record alert to prevent duplicates
			_ = s.db.Queries.CreateThresholdAlert(ctx, db.CreateThresholdAlertParams{
				StudentID: st.StudentID,
				YearMonth: yearMonth,
			})

			// Send alert to director
			msg := fmt.Sprintf("🚨 **OGOHLANTIRISH: Dars qoldirish normasi oshdi!**\n\n"+
				"👤 **O'quvchi:** %s\n"+
				"🏫 **Sinf:** %s\n"+
				"❌ **Shu oyda sababsiz dars qoldirgan:** %d kun\n"+
				"⚠️ **Belgilangan chegara:** %d kun\n\n"+
				"Ushbu o'quvchi bu oyda birinchi marta chegaradan oshdi. Iltimos, ota-onasi bilan bog'laning.",
				st.StudentName, st.ClassName, st.AbsentCount, threshold)

			_ = s.bot.SendMessage(ctx, directorTGID, msg, nil)
		}
	}
}

// CheckSingleStudentThreshold is triggered when attendance is saved in real time
func (s *Scheduler) CheckSingleStudentThreshold(ctx context.Context, studentID int64, schoolID int64, targetDate time.Time) {
	if s.bot == nil {
		return
	}

	settings, err := s.db.Queries.GetSettingsBySchoolID(ctx, schoolID)
	var threshold int32 = 3
	if err == nil && settings.AbsenceThreshold > 0 {
		threshold = settings.AbsenceThreshold
	}

	startDate := time.Date(targetDate.Year(), targetDate.Month(), 1, 0, 0, 0, 0, s.cfg.Location)
	endDate := startDate.AddDate(0, 1, -1)
	yearMonth := targetDate.Format("2006-01")

	// Count unexcused absences
	absentCount, err := s.db.Queries.CountUnexcusedAbsencesInMonth(ctx, db.CountUnexcusedAbsencesInMonthParams{
		StudentID: studentID,
		Date:      pgtype.Date{Time: startDate, Valid: true},
		Date_2:    pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil || absentCount < int64(threshold) {
		return
	}

	// Check if already notified this month
	sent, err := s.db.Queries.HasThresholdAlertBeenSent(ctx, db.HasThresholdAlertBeenSentParams{
		StudentID: studentID,
		YearMonth: yearMonth,
	})
	if err != nil || sent {
		return
	}

	// Mark notified
	_ = s.db.Queries.CreateThresholdAlert(ctx, db.CreateThresholdAlertParams{
		StudentID: studentID,
		YearMonth: yearMonth,
	})

	directorTGID := s.cfg.DirectorTelegramID
	if directorTGID == 0 {
		dirUser, err := s.db.Queries.GetDirectorBySchool(ctx, pgtype.Int8{Int64: schoolID, Valid: true})
		if err == nil {
			directorTGID = dirUser.TelegramID
		}
	}
	if directorTGID == 0 {
		return
	}

	student, err := s.db.Queries.GetStudentByID(ctx, studentID)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("🚨 **OGOHLANTIRISH: Dars qoldirish normasi oshdi!**\n\n"+
		"👤 **O'quvchi:** %s\n"+
		"🏫 **Sinf:** %s\n"+
		"❌ **Shu oyda sababsiz dars qoldirgan:** %d kun\n"+
		"⚠️ **Belgilangan chegara:** %d kun\n\n"+
		"Ushbu o'quvchi bu oyda birinchi marta me'yordan oshdi. Iltimos, ota-onasi bilan bog'laning.",
		student.FullName, student.ClassName, absentCount, threshold)

	_ = s.bot.SendMessage(ctx, directorTGID, msg, nil)
}

