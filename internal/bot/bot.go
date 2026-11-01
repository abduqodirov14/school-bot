package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v5/pgtype"

	"school-attendance/internal/config"
	"school-attendance/internal/db"
)

type Bot struct {
	b   *tgbot.Bot
	db  *db.DB
	cfg *config.Config
}

func New(cfg *config.Config, database *db.DB) (*Bot, error) {
	if cfg.BotToken == "" {
		log.Println("OGOHLANTIRISH: BOT_TOKEN ko'rsatilmagan. Bot ishga tushmaydi.")
		return nil, nil
	}

	bInstance := &Bot{
		db:  database,
		cfg: cfg,
	}

	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(bInstance.defaultHandler),
	}

	botClient, err := tgbot.New(cfg.BotToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("telegram botni yaratishda xatolik: %w", err)
	}

	bInstance.b = botClient
	bInstance.registerHandlers()

	return bInstance, nil
}

func (b *Bot) Start(ctx context.Context) {
	if b == nil || b.b == nil {
		return
	}
	log.Println("Telegram bot ishga tushdi va xabarlarni qabul qilmoqda...")
	b.b.Start(ctx)
}

func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string, markup models.ReplyMarkup) error {
	if b == nil || b.b == nil {
		return nil
	}
	_, err := b.b.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: markup,
	})
	return err
}

func (b *Bot) registerHandlers() {
	b.b.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, b.handleStart)
	b.b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "approve_teacher:", tgbot.MatchTypePrefix, b.handleApproveTeacher)
	b.b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "reject_teacher:", tgbot.MatchTypePrefix, b.handleRejectTeacher)
	b.registerTeacherAttendanceHandlers()
	b.registerDirectorCommands()
}

func (b *Bot) ensureSchool(ctx context.Context) (int64, error) {
	school, err := b.db.Queries.GetFirstSchool(ctx)
	if err == nil {
		return school.ID, nil
	}

	// Create initial school
	newSchool, err := b.db.Queries.CreateSchool(ctx, "1-sonli umumta'lim maktabi")
	if err != nil {
		return 0, fmt.Errorf("maktabni yaratishda xatolik: %w", err)
	}

	// Create default settings
	_, _ = b.db.Queries.UpsertSettings(ctx, db.UpsertSettingsParams{
		SchoolID:         newSchool.ID,
		AbsenceThreshold: 3,
		DeadlineTime:     "09:00",
	})

	return newSchool.ID, nil
}

func (b *Bot) handleStart(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	from := update.Message.From
	chatID := update.Message.Chat.ID
	fullName := strings.TrimSpace(from.FirstName + " " + from.LastName)
	if fullName == "" {
		fullName = from.Username
		if fullName == "" {
			fullName = fmt.Sprintf("Foydalanuvchi %d", from.ID)
		}
	}

	schoolID, err := b.ensureSchool(ctx)
	if err != nil {
		log.Printf("Xatolik maktabni tekshirishda: %v", err)
	}

	sID := pgtype.Int8{Int64: schoolID, Valid: schoolID != 0}
	usernameText := pgtype.Text{String: from.Username, Valid: from.Username != ""}

	// 1. Is this the configured Director?
	if b.cfg.DirectorTelegramID != 0 && from.ID == b.cfg.DirectorTelegramID {
		user, err := b.db.Queries.GetUserByTelegramID(ctx, from.ID)
		if err != nil {
			// Register as active director
			_, _ = b.db.Queries.CreateUser(ctx, db.CreateUserParams{
				SchoolID:   sID,
				TelegramID: from.ID,
				Username:   usernameText,
				FullName:   fullName,
				Role:       "director",
				Status:     "active",
			})
		} else {
			// Ensure role and active status
			_, _ = b.db.Queries.UpdateUserStatus(ctx, db.UpdateUserStatusParams{
				TelegramID: from.ID,
				Status:     "active",
				Role:       "director",
			})
			_ = user
		}

		replyMarkup := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:   "📱 Boshqaruv Panelini Ochish",
						WebApp: &models.WebAppInfo{URL: b.cfg.WebAppURL},
					},
				},
			},
		}

		msg := fmt.Sprintf("🎓 **Assalomu alaykum, Hurmatli Direktor (%s)!**\n\n"+
			"Maktab Davomati boshqaruv tizimiga xush kelibsiz.\n\n"+
			"Quyidagi tugma orqali maktab boshqaruv panelini (Mini App) ochishingiz mumkin.\n\n"+
			"Shuningdek, quyidagi buyruqlardan foydalanishingiz mumkin:\n"+
			"📊 /bugun — Bugungi umumiy davomat ko'rsatkichi\n"+
			"❌ /kelmaganlar — Bugun darsga kelmaganlar ro'yxati\n"+
			"🏫 /sinflar — Barcha sinflar holati", fullName)

		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:      chatID,
			Text:        msg,
			ParseMode:   models.ParseModeMarkdown,
			ReplyMarkup: replyMarkup,
		})
		return
	}

	// 2. Regular user / teacher check
	user, err := b.db.Queries.GetUserByTelegramID(ctx, from.ID)
	if err != nil {
		// New user -> register as pending teacher
		createdUser, createErr := b.db.Queries.CreateUser(ctx, db.CreateUserParams{
			SchoolID:   sID,
			TelegramID: from.ID,
			Username:   usernameText,
			FullName:   fullName,
			Role:       "teacher",
			Status:     "pending",
		})
		if createErr != nil {
			log.Printf("Yangi foydalanuvchini yaratishda xatolik: %v", createErr)
			_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
				ChatID: chatID,
				Text:   "Kechirasiz, ro'yxatdan o'tishda xatolik yuz berdi. Iltimos, qayta urinib ko'ring.",
			})
			return
		}

		// Notify user
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text: fmt.Sprintf("Assalomu alaykum, *%s*!\n\n"+
				"Maktab Davomati tizimiga xush kelibsiz. Sizning o'qituvchi sifatida ro'yxatdan o'tish so'rovingiz direktorga yuborildi.\n\n"+
				"⏳ Iltimos, direktor tasdiqlashini kuting.", fullName),
			ParseMode: models.ParseModeMarkdown,
		})

		// Notify Director
		b.notifyDirectorNewUser(ctx, botInstance, createdUser)
		return
	}

	// Existing user handling
	switch user.Status {
	case "pending":
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "⏳ Sizning arizangiz maktab direktori tomonidan ko'rib chiqilmoqda. Iltimos, kuting.",
		})
	case "rejected":
		_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Kechirasiz, sizning so'rovingiz maktab direktori tomonidan rad etilgan.",
		})
	case "active":
		if user.Role == "director" {
			replyMarkup := &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:   "📱 Boshqaruv Paneli",
							WebApp: &models.WebAppInfo{URL: b.cfg.WebAppURL},
						},
					},
				},
			}
			_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
				ChatID:      chatID,
				Text:        fmt.Sprintf("🎓 **Assalomu alaykum, Hurmatli Direktor (%s)!**\n\nBoshqaruv panelini ochish uchun pastdagi tugmani bosing.", user.FullName),
				ParseMode:   models.ParseModeMarkdown,
				ReplyMarkup: replyMarkup,
			})
		} else {
			replyMarkup := &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{
							Text:   "📋 Davomat Olish (Mini App)",
							WebApp: &models.WebAppInfo{URL: b.cfg.WebAppURL},
						},
					},
				},
			}
			_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
				ChatID: chatID,
				Text: fmt.Sprintf("Assalomu alaykum, *%s*!\n\n"+
					"Davomat olish uchun pastdagi Mini App tugmasini bosing:", user.FullName),
				ParseMode:   models.ParseModeMarkdown,
				ReplyMarkup: replyMarkup,
			})
		}
	}
}

func (b *Bot) notifyDirectorNewUser(ctx context.Context, botInstance *tgbot.Bot, user db.User) {
	directorTGID := b.cfg.DirectorTelegramID
	if directorTGID == 0 && user.SchoolID.Valid {
		dirUser, err := b.db.Queries.GetDirectorBySchool(ctx, user.SchoolID)
		if err == nil {
			directorTGID = dirUser.TelegramID
		}
	}

	if directorTGID == 0 {
		log.Println("Direktor Telegram ID topilmadi, xabarnoma yuborilmadi.")
		return
	}

	usernameDisplay := "mavjud emas"
	if user.Username.Valid && user.Username.String != "" {
		usernameDisplay = "@" + user.Username.String
	}

	text := fmt.Sprintf("🔔 **Yangi o'qituvchi ro'yxatdan o'tmoqchi:**\n\n"+
		"👤 **F.I.SH:** %s\n"+
		"🆔 **Telegram ID:** `%d`\n"+
		"🌐 **Username:** %s\n\n"+
		"Ushbu foydalanuvchini o'qituvchi sifatida tasdiqlaysizmi?",
		user.FullName, user.TelegramID, usernameDisplay)

	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "✅ Tasdiqlash (O'qituvchi)",
					CallbackData: fmt.Sprintf("approve_teacher:%d", user.ID),
				},
				{
					Text:         "❌ Rad etish",
					CallbackData: fmt.Sprintf("reject_teacher:%d", user.ID),
				},
			},
		},
	}

	_, err := botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      directorTGID,
		Text:        text,
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: markup,
	})
	if err != nil {
		log.Printf("Direktorga xabarnoma yuborishda xatolik: %v", err)
	}
}

func (b *Bot) handleApproveTeacher(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	cb := update.CallbackQuery
	callerID := cb.From.ID

	// Verify that caller is director
	if b.cfg.DirectorTelegramID != 0 && callerID != b.cfg.DirectorTelegramID {
		callerUser, err := b.db.Queries.GetUserByTelegramID(ctx, callerID)
		if err != nil || callerUser.Role != "director" {
			_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Faqat direktor tasdiqlashi mumkin!",
				ShowAlert:       true,
			})
			return
		}
	}

	idStr := strings.TrimPrefix(cb.Data, "approve_teacher:")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}

	_, err = b.db.Queries.GetUserByID(ctx, userID)
	if err != nil {
		_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Foydalanuvchi topilmadi!",
		})
		return
	}

	updatedUser, err := b.db.Queries.UpdateUserStatusByID(ctx, db.UpdateUserStatusByIDParams{
		ID:     userID,
		Status: "active",
		Role:   "teacher",
	})
	if err != nil {
		log.Printf("Foydalanuvchi statusini yangilashda xatolik: %v", err)
		_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Xatolik yuz berdi!",
		})
		return
	}

	_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "✅ O'qituvchi tasdiqlandi!",
	})

	// Update director message
	if cb.Message.Message != nil {
		_, _ = botInstance.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:    cb.Message.Message.Chat.ID,
			MessageID: cb.Message.Message.ID,
			Text: fmt.Sprintf("✅ **%s** muvaffaqiyatli o'qituvchi sifatida tasdiqlandi.",
				updatedUser.FullName),
			ParseMode: models.ParseModeMarkdown,
		})
	}

	// Notify teacher
	teacherMarkup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:   "📋 Davomat Olish (Mini App)",
					WebApp: &models.WebAppInfo{URL: b.cfg.WebAppURL},
				},
			},
		},
	}

	_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: updatedUser.TelegramID,
		Text: "🎉 **Tabriklaymiz!**\n\n" +
			"Direktor sizning o'qituvchi sifatida ro'yxatdan o'tish arizangizni tasdiqladi.\n" +
			"Endi siz tizimdan foydalanib davomat olishingiz mumkin.\n\n" +
			"Davomat olish uchun pastdagi tugmani bosing:",
		ParseMode:   models.ParseModeMarkdown,
		ReplyMarkup: teacherMarkup,
	})
}

func (b *Bot) handleRejectTeacher(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	cb := update.CallbackQuery
	callerID := cb.From.ID

	// Verify that caller is director
	if b.cfg.DirectorTelegramID != 0 && callerID != b.cfg.DirectorTelegramID {
		callerUser, err := b.db.Queries.GetUserByTelegramID(ctx, callerID)
		if err != nil || callerUser.Role != "director" {
			_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
				CallbackQueryID: cb.ID,
				Text:            "Faqat direktor rad etishi mumkin!",
				ShowAlert:       true,
			})
			return
		}
	}

	idStr := strings.TrimPrefix(cb.Data, "reject_teacher:")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return
	}

	_, err = b.db.Queries.GetUserByID(ctx, userID)
	if err != nil {
		_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Foydalanuvchi topilmadi!",
		})
		return
	}

	updatedUser, err := b.db.Queries.UpdateUserStatusByID(ctx, db.UpdateUserStatusByIDParams{
		ID:     userID,
		Status: "rejected",
		Role:   "teacher",
	})
	if err != nil {
		_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
			CallbackQueryID: cb.ID,
			Text:            "Xatolik yuz berdi!",
		})
		return
	}

	_, _ = botInstance.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cb.ID,
		Text:            "❌ Ariza rad etildi!",
	})

	// Update director message
	if cb.Message.Message != nil {
		_, _ = botInstance.EditMessageText(ctx, &tgbot.EditMessageTextParams{
			ChatID:    cb.Message.Message.Chat.ID,
			MessageID: cb.Message.Message.ID,
			Text: fmt.Sprintf("❌ **%s** arizasi rad etildi.",
				updatedUser.FullName),
			ParseMode: models.ParseModeMarkdown,
		})
	}

	// Notify user
	_, _ = botInstance.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: updatedUser.TelegramID,
		Text: "❌ Kechirasiz, sizning o'qituvchi sifatida ro'yxatdan o'tish arizangiz " +
			"maktab direktori tomonidan rad etildi.",
	})
}

func (b *Bot) defaultHandler(ctx context.Context, botInstance *tgbot.Bot, update *models.Update) {
	// Handled by specific route handlers or ignored
}
