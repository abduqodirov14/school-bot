package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/service"
)

func (b *Bot) registerTeacherAttendanceHandlers() {
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "/davomat", tgbot.MatchTypeExact, b.handleTeacherDavomat)
	b.b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "bot_att_class:", tgbot.MatchTypePrefix, b.handleSelectClass)
	b.b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "att_toggle:", tgbot.MatchTypePrefix, b.handleToggleStudent)
	b.b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "att_all_pres:", tgbot.MatchTypePrefix, b.handleAllPresent)
	b.b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "att_save:", tgbot.MatchTypePrefix, b.handleSaveAttendance)
}

func (b *Bot) handleTeacherDavomat(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	fromID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	user, err := b.db.Queries.GetUserByTelegramID(ctx, fromID)
	if err != nil || user.Status != "active" {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Siz faol o'qituvchi emassiz yoki hisobingiz tasdiqlanmagan.",
		})
		return
	}

	classes, err := b.db.Queries.ListClassesByTeacherID(ctx, pgtype.Int8{Int64: user.ID, Valid: true})
	if err != nil || len(classes) == 0 {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "Sizga hali hech qanday sinf biriktirilmagan. Iltimos, direktor bilan bog'laning.",
		})
		return
	}

	if len(classes) == 1 {
		b.sendClassAttendanceKeyboard(ctx, botInstance, chatID, 0, classes[0].ID, classes[0].Name)
		return
	}

	var rows [][]models.InlineKeyboardButton
	for _, c := range classes {
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         "🏫 " + c.Name + " sinfi",
				CallbackData: fmt.Sprintf("bot_att_class:%d", c.ID),
			},
		})
	}

	_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   "Davomat olish uchun sinfni tanlang:",
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		},
	})
}

func (b *Bot) handleSelectClass(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	cb := update.CallbackQuery
	classIDStr := strings.TrimPrefix(cb.Data, "bot_att_class:")
	classID, err := strconv.ParseInt(classIDStr, 10, 64)
	if err != nil {
		return
	}

	class, err := b.db.Queries.GetClassByID(ctx, classID)
	if err != nil {
		return
	}

	_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
	})

	if cb.Message.Message != nil {
		b.sendClassAttendanceKeyboard(ctx, botInstance, cb.Message.Message.Chat.ID, cb.Message.Message.ID, class.ID, class.Name)
	}
}

func (b *Bot) sendClassAttendanceKeyboard(ctx context.Context, botInstance *tgbot.Bot, chatID int64, messageID int, classID int64, className string) {
	today := time.Now().In(b.cfg.Location)
	attService := service.NewAttendanceService(b.db)
	resp, err := attService.GetClassAttendance(ctx, classID, today)
	if err != nil {
		log.Printf("Davomatni olishda xatolik: %v", err)
		return
	}

	var keyboardRows [][]models.InlineKeyboardButton

	for _, s := range resp.Students {
		emoji := service.StatusEmoji(s.Status)
		label := service.StatusLabel(s.Status)
		buttonText := fmt.Sprintf("%s: %s %s", s.FullName, emoji, label)

		keyboardRows = append(keyboardRows, []models.InlineKeyboardButton{
			{
				Text:         buttonText,
				CallbackData: fmt.Sprintf("att_toggle:%d:%d:%s", classID, s.StudentID, s.Status),
			},
		})
	}

	// Bottom action buttons
	keyboardRows = append(keyboardRows, []models.InlineKeyboardButton{
		{
			Text:         "🟢 Hammasi keldi",
			CallbackData: fmt.Sprintf("att_all_pres:%d", classID),
		},
	})
	keyboardRows = append(keyboardRows, []models.InlineKeyboardButton{
		{
			Text:         "💾 Saqlash",
			CallbackData: fmt.Sprintf("att_save:%d", classID),
		},
	})

	savedStatusText := "❌ Hali saqlanmagan"
	if resp.IsSaved {
		savedStatusText = "✅ Saqlangan"
	}

	msgText := fmt.Sprintf("📋 **%s sinfi davomati**\n📅 Sana: `%s` | Holat: %s\n\n"+
		"👥 Jami: %d | ✅ Keldi: %d | ❌ Kelmadi: %d\n\n"+
		"O'quvchi ustiga bosib uning holatini o'zgartiring, so'ngra **Saqlash** tugmasini bosing:",
		className, today.Format("02.01.2006"), savedStatusText,
		resp.Summary.Total, resp.Summary.Present, resp.Summary.Absent)

	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: keyboardRows,
	}

	if messageID > 0 {
		_, _ = botInstance.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        msgText,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: markup,
		})
	} else {
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      chatID,
			Text:        msgText,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: markup,
		})
	}
}

func (b *Bot) handleToggleStudent(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	cb := update.CallbackQuery

	// Format: att_toggle:<classID>:<studentID>:<currentStatus>
	parts := strings.Split(strings.TrimPrefix(cb.Data, "att_toggle:"), ":")
	if len(parts) < 3 {
		return
	}

	classID, _ := strconv.ParseInt(parts[0], 10, 64)
	studentID, _ := strconv.ParseInt(parts[1], 10, 64)
	currentStatus := parts[2]
	nextStatus := service.NextStatus(currentStatus)

	callerUser, _ := b.db.Queries.GetUserByTelegramID(ctx, cb.From.ID)

	today := time.Now().In(b.cfg.Location)
	attService := service.NewAttendanceService(b.db)

	_ = attService.SaveClassAttendance(ctx, classID, today, []service.AttendanceItemInput{
		{
			StudentID: studentID,
			Status:    nextStatus,
		},
	}, callerUser.ID)

	_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            fmt.Sprintf("%s -> %s", service.StatusEmoji(nextStatus), service.StatusLabel(nextStatus)),
	})

	class, _ := b.db.Queries.GetClassByID(ctx, classID)
	if cb.Message.Message != nil {
		b.sendClassAttendanceKeyboard(ctx, botInstance, cb.Message.Message.Chat.ID, cb.Message.Message.ID, classID, class.Name)
	}
}

func (b *Bot) handleAllPresent(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	cb := update.CallbackQuery
	classIDStr := strings.TrimPrefix(cb.Data, "att_all_pres:")
	classID, _ := strconv.ParseInt(classIDStr, 10, 64)

	callerUser, _ := b.db.Queries.GetUserByTelegramID(ctx, cb.From.ID)
	today := time.Now().In(b.cfg.Location)

	students, err := b.db.Queries.ListStudentsByClass(ctx, classID)
	if err == nil {
		var items []service.AttendanceItemInput
		for _, s := range students {
			items = append(items, service.AttendanceItemInput{
				StudentID: s.ID,
				Status:    service.StatusPresent,
			})
		}
		attService := service.NewAttendanceService(b.db)
		_ = attService.SaveClassAttendance(ctx, classID, today, items, callerUser.ID)
	}

	_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "🟢 Barcha o'quvchilar 'Keldi' qilindi!",
	})

	class, _ := b.db.Queries.GetClassByID(ctx, classID)
	if cb.Message.Message != nil {
		b.sendClassAttendanceKeyboard(ctx, botInstance, cb.Message.Message.Chat.ID, cb.Message.Message.ID, classID, class.Name)
	}
}

func (b *Bot) handleSaveAttendance(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	cb := update.CallbackQuery
	classIDStr := strings.TrimPrefix(cb.Data, "att_save:")
	classID, _ := strconv.ParseInt(classIDStr, 10, 64)

	class, err := b.db.Queries.GetClassByID(ctx, classID)
	if err != nil {
		return
	}

	_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "✅ Davomat muvaffaqiyatli saqlandi!",
		ShowAlert:       true,
	})

	today := time.Now().In(b.cfg.Location)
	attService := service.NewAttendanceService(b.db)
	resp, _ := attService.GetClassAttendance(ctx, classID, today)

	summaryText := ""
	if resp != nil {
		summaryText = fmt.Sprintf("\n📊 **Hisobot:**\n✅ Keldi: %d ta\n❌ Kelmadi: %d ta\n⏳ Kechikdi: %d ta\n📄 Sababli: %d ta",
			resp.Summary.Present, resp.Summary.Absent, resp.Summary.Late, resp.Summary.Excused)
	}

	if cb.Message.Message != nil {
		_, _ = botInstance.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:    cb.Message.Message.Chat.ID,
			MessageID: cb.Message.Message.ID,
			Text: fmt.Sprintf("✅ **%s sinfining bugungi (%s) davomati saqlandi!**%s\n\n"+
				"Qayta tahrirlash uchun /davomat buyrug'ini yuboring.",
				class.Name, today.Format("02.01.2006"), summaryText),
			ParseMode: models.ParseModeMarkdown,
		})
	}
}

