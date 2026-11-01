package bot

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/db"
	"school-attendance/internal/export"
)

func (b *Bot) registerDirectorCommands() {
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "/bugun", tgbot.MatchTypeExact, b.handleBugun)
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "/kelmaganlar", tgbot.MatchTypeExact, b.handleKelmaganlar)
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "/sinflar", tgbot.MatchTypeExact, b.handleSinflar)
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "/hisobot", tgbot.MatchTypePrefix, b.handleHisobot)
}

func (b *Bot) isDirector(ctx context.Context, telegramID int64) (bool, *db.User) {
	if b.cfg.DirectorTelegramID != 0 && telegramID == b.cfg.DirectorTelegramID {
		user, _ := b.db.Queries.GetUserByTelegramID(ctx, telegramID)
		return true, &user
	}

	user, err := b.db.Queries.GetUserByTelegramID(ctx, telegramID)
	if err == nil && user.Role == "director" && user.Status == "active" {
		return true, &user
	}

	return false, nil
}

func (b *Bot) handleBugun(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	fromID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	isDir, user := b.isDirector(ctx, fromID)
	if !isDir || user == nil || !user.SchoolID.Valid {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Bu buyruq faqat maktab direktori uchun mo'ljallangan.",
		})
		return
	}

	today := time.Now().In(b.cfg.Location)
	pgDate := pgtype.Date{Time: today, Valid: true}

	stats, err := b.db.Queries.GetTodaySchoolStats(ctx, db.GetTodaySchoolStatsParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})
	if err != nil {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Statistikani olishda xatolik yuz berdi.",
		})
		return
	}

	classes, _ := b.db.Queries.ListClassesWithAttendanceStatusToday(ctx, db.ListClassesWithAttendanceStatusTodayParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})

	totalClasses := len(classes)
	takenClasses := 0
	var unmarkedClassNames []string

	for _, c := range classes {
		if c.MarkedStudents > 0 {
			takenClasses++
		} else if c.TotalStudents > 0 {
			unmarkedClassNames = append(unmarkedClassNames, c.ClassName)
		}
	}

	var percentage float64 = 0
	if stats.TotalStudents > 0 {
		percentage = math.Round((float64(stats.PresentCount)/float64(stats.TotalStudents))*1000) / 10
	}

	unmarkedText := "Barcha sinflar davomat topshirgan! 🎉"
	if len(unmarkedClassNames) > 0 {
		unmarkedText = strings.Join(unmarkedClassNames, ", ")
	}

	msg := fmt.Sprintf("📊 **Bugungi Maktab Davomati (%s):**\n\n"+
		"👥 **Jami o'quvchilar:** %d nafar\n"+
		"✅ **Darsda:** %d nafar (%.1f%%)\n"+
		"❌ **Kelmaganlar:** %d nafar\n"+
		"⏳ **Kechikkanlar:** %d nafar\n"+
		"📄 **Sababli:** %d nafar\n\n"+
		"🏫 **Sinflar holati:** %d/%d ta sinf topshirdi\n"+
		"⚠️ **Topshirmagan sinflar:** %s\n\n"+
		"Batafsil ma'lumot uchun Mini App panelini oching.",
		today.Format("02.01.2006"), stats.TotalStudents,
		stats.PresentCount, percentage, stats.AbsentCount,
		stats.LateCount, stats.ExcusedCount,
		takenClasses, totalClasses, unmarkedText)

	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:   "📱 Boshqaruv Panelini Ochish",
					WebApp: &models.WebAppInfo{URL: b.cfg.WebAppURL},
				},
			},
		},
	}

	_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        msg,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: markup,
	})
}

func (b *Bot) handleKelmaganlar(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	fromID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	isDir, user := b.isDirector(ctx, fromID)
	if !isDir || user == nil || !user.SchoolID.Valid {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Bu buyruq faqat maktab direktori uchun mo'ljallangan.",
		})
		return
	}

	today := time.Now().In(b.cfg.Location)
	pgDate := pgtype.Date{Time: today, Valid: true}

	absentStudents, err := b.db.Queries.ListAbsentStudentsToday(ctx, db.ListAbsentStudentsTodayParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})
	if err != nil || len(absentStudents) == 0 {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("🎉 **Bugun (%s) barcha o'quvchilar darsda!**\nHech kim sababsiz dars qoldirmagan.", today.Format("02.01.2006")),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	// Group by class
	grouped := make(map[string][]string)
	var order []string
	for _, row := range absentStudents {
		info := row.StudentName
		if row.Note != "" {
			info += fmt.Sprintf(" _(Izoh: %s)_", row.Note)
		}
		if _, exists := grouped[row.ClassName]; !exists {
			order = append(order, row.ClassName)
		}
		grouped[row.ClassName] = append(grouped[row.ClassName], info)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("❌ **Bugun (%s) darsga kelmaganlar ro'yxati:**\n\n", today.Format("02.01.2006")))

	for _, className := range order {
		sb.WriteString(fmt.Sprintf("🏫 **%s sinfi:**\n", className))
		for idx, studentStr := range grouped[className] {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", idx+1, studentStr))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Jami: **%d nafar** o'quvchi darsga kelmadi.", len(absentStudents)))

	_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      sb.String(),
		ParseMode: models.ParseModeMarkdown,
	})
}


func (b *Bot) handleSinflar(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	fromID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	isDir, user := b.isDirector(ctx, fromID)
	if !isDir || user == nil || !user.SchoolID.Valid {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Bu buyruq faqat maktab direktori uchun mo'ljallangan.",
		})
		return
	}

	today := time.Now().In(b.cfg.Location)
	pgDate := pgtype.Date{Time: today, Valid: true}

	classes, err := b.db.Queries.ListClassesWithAttendanceStatusToday(ctx, db.ListClassesWithAttendanceStatusTodayParams{
		SchoolID: user.SchoolID.Int64,
		Date:     pgDate,
	})
	if err != nil || len(classes) == 0 {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Maktabda sinflar topilmadi.",
		})
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🏫 **Sinflar davomat holati (%s):**\n\n", today.Format("02.01.2006")))

	for _, c := range classes {
		teacherText := c.TeacherName
		if teacherText == "" {
			teacherText = "O'qituvchi biriktirilmagan"
		}

		if c.MarkedStudents > 0 {
			sb.WriteString(fmt.Sprintf("✅ **%s** — Olingan (%d/%d) | 👤 %s\n",
				c.ClassName, c.MarkedStudents, c.TotalStudents, teacherText))
		} else {
			sb.WriteString(fmt.Sprintf("⚠️ **%s** — **Olinmagan!** (0/%d) | 👤 %s\n",
				c.ClassName, c.TotalStudents, teacherText))
		}
	}

	_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      sb.String(),
		ParseMode: models.ParseModeMarkdown,
	})
}


func (b *Bot) handleHisobot(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	fromID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	isDir, user := b.isDirector(ctx, fromID)
	if !isDir || user == nil || !user.SchoolID.Valid {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Bu buyruq faqat maktab direktori uchun mo'ljallangan.",
		})
		return
	}

	now := time.Now().In(b.cfg.Location)
	year := now.Year()
	month := now.Month()

	textParts := strings.Fields(update.Message.Text)
	if len(textParts) > 1 {
		monthParts := strings.Split(textParts[1], "-")
		if len(monthParts) == 2 {
			y, yErr := strconv.Atoi(monthParts[0])
			m, mErr := strconv.Atoi(monthParts[1])
			if yErr == nil && mErr == nil && m >= 1 && m <= 12 {
				year = y
				month = time.Month(m)
			}
		}
	}

	_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   fmt.Sprintf("⏳ %04d-%02d oyi uchun Excel hisoboti tayyorlanmoqda...", year, month),
	})

	exporter := export.NewExcelExporter(b.db)
	buf, err := exporter.GenerateMonthlyReport(ctx, user.SchoolID.Int64, year, month)
	if err != nil {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Hisobotni yaratishda xatolik yuz berdi: " + err.Error(),
		})
		return
	}

	filename := fmt.Sprintf("Davomat_Hisobot_%04d-%02d.xlsx", year, month)
	caption := fmt.Sprintf("📄 **%04d-%02d oyi uchun to'liq maktab davomati hisoboti**\n\n"+
		"• 'Umumiy' sahifa: maktab bo'yicha qatnashish foizlari\n"+
		"• Har bir sinf uchun alohida sahifalar (kunlar kesimida: +, -, K, S)\n"+
		"• Me'yordan oshgan o'quvchilar qizil rangda ajratilgan.", year, month)

	_, _ = botInstance.SendDocument(ctx, &tgbot.SendDocumentParams{
		ChatID: chatID,
		Document: &models.InputFileUpload{
			Filename: filename,
			Data:     buf,
		},
		Caption:   caption,
		ParseMode: models.ParseModeMarkdown,
	})
}
