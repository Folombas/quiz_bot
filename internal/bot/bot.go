package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"quiz_bot/internal/config"
	"quiz_bot/internal/logger"
	"quiz_bot/internal/models"
	"quiz_bot/internal/ratelimit"
	"quiz_bot/internal/storage"
)

// Version бота
const Version = "0.3.0"

// Bot — структура телеграм-бота
type Bot struct {
	api         *tgbotapi.BotAPI
	config      *config.Config
	logger      *logger.Logger
	storage     *storage.Storage
	userRepo    *storage.UserRepository
	rateLimiter *ratelimit.RateLimiter
	questions   []models.Question
	interviewQuestions []models.Question
	mu          sync.RWMutex
}

// New создаёт нового бота
func New(cfg *config.Config) (*Bot, error) {
	l, err := logger.New(cfg.Log)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &Bot{
		config: cfg,
		logger: l,
	}, nil
}

// Run запускает бота
func (b *Bot) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Загружаем переменные из .env файла
	if err := godotenv.Load(); err != nil {
		b.logger.Info(".env file not found, using environment variables")
	}

	// Получаем токен
	botToken := b.config.Bot.Token
	if botToken == "" {
		botToken = os.Getenv(b.config.Bot.TokenEnvVar)
	}
	if botToken == "" {
		return errors.New("TELEGRAM_BOT_TOKEN is not set")
	}

	// Создаём бота
	var err error
	b.api, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}
	b.api.Debug = false
	b.logger.Info("Bot authorized", 
		slog.String("username", b.api.Self.UserName),
		slog.String("version", Version))

	// Инициализируем хранилище
	b.storage, err = storage.NewStorage(b.config.Database.SQLite.Path)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	defer b.storage.Close()
	b.logger.Info("Database initialized", slog.String("path", b.config.Database.SQLite.Path))

	b.userRepo = storage.NewUserRepository(b.storage.DB())

	// Инициализируем rate limiter
	if b.config.RateLimit.Enabled {
		b.rateLimiter = ratelimit.NewRateLimiter(
			b.config.RateLimit.RequestsPerMin,
			b.config.RateLimit.BurstSize,
		)
		b.logger.Info("Rate limiting enabled",
			slog.Int("requests_per_min", b.config.RateLimit.RequestsPerMin),
			slog.Int("burst_size", b.config.RateLimit.BurstSize))
	}

	// Загружаем вопросы
	if err := b.loadQuestions(); err != nil {
		return fmt.Errorf("failed to load questions: %w", err)
	}
	b.logger.Info("Questions loaded", slog.Int("count", len(b.questions)))

	// Загружаем вопросы собеседования
	if err := b.loadInterviewQuestions(); err != nil {
		b.logger.Warn("Interview questions not loaded", slog.String("error", err.Error()))
	} else {
		b.logger.Info("Interview questions loaded", slog.Int("count", len(b.interviewQuestions)))
	}

	// Канал обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = b.config.Bot.Timeout
	updates := b.api.GetUpdatesChan(u)

	// Таймер для автосохранения данных (каждые 5 минут)
	saveTicker := time.NewTicker(5 * time.Minute)
	defer saveTicker.Stop()

	b.logger.Info("Bot started")

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Shutting down bot...")
			return nil
		case <-saveTicker.C:
			b.logger.Debug("Auto-saving user data")
		case update := <-updates:
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Rate limiting
	if b.rateLimiter != nil {
		var chatID int64
		if update.Message != nil {
			chatID = update.Message.Chat.ID
		} else if update.CallbackQuery != nil {
			chatID = update.CallbackQuery.Message.Chat.ID
		}

		if chatID > 0 && !b.rateLimiter.Allow(chatID) {
			b.logger.Warn("Rate limit exceeded", slog.Int64("chat_id", chatID))
			return
		}
	}

	if update.Message != nil {
		b.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallback(update.CallbackQuery)
	}
}

// --- Загрузка данных ---

func (b *Bot) loadQuestions() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, err := os.ReadFile("configs/questions.json")
	if err != nil {
		return err
	}

	var questions []models.Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return err
	}

	b.questions = questions
	return nil
}

func (b *Bot) loadInterviewQuestions() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, err := os.ReadFile("configs/interview_questions.json")
	if err != nil {
		return err
	}

	var questions []models.Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return err
	}

	b.interviewQuestions = questions
	return nil
}

// --- Обработчики ---

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if !msg.IsCommand() {
		return
	}

	chatID := msg.Chat.ID
	b.logger.Info("Command received",
		slog.Int64("chat_id", chatID),
		slog.String("command", msg.Command()))

	switch msg.Command() {
	case "start":
		b.sendStartMenu(chatID)
	case "help":
		b.sendHelp(chatID)
	case "quiz":
		b.handleQuizCommand(chatID)
	case "score":
		b.handleScoreCommand(chatID)
	case "leaderboard":
		b.sendLeaderboard(chatID)
	case "interview":
		b.handleInterviewCommand(chatID)
	case "reset":
		b.handleResetCommand(chatID)
	default:
		b.sendText(chatID, "Неизвестная команда. Напиши /help")
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID

	// Обработка команд меню
	switch callback.Data {
	case "cmd_quiz":
		b.handleQuizCommand(chatID)
		b.answerCallback(callback.ID, "")
		return
	case "cmd_score":
		b.handleScoreCommand(chatID)
		b.answerCallback(callback.ID, "")
		return
	case "cmd_leaderboard":
		b.sendLeaderboard(chatID)
		b.answerCallback(callback.ID, "")
		return
	case "cmd_reset":
		b.handleResetCommand(chatID)
		b.answerCallback(callback.ID, "")
		return
	case "cmd_help":
		b.sendText(chatID, getHelpText())
		b.answerCallback(callback.ID, "")
		return
	case "cmd_interview":
		b.handleInterviewCommand(chatID)
		b.answerCallback(callback.ID, "")
		return
	}

	// Обработка ответов на вопросы
	b.handleAnswer(callback)
}

func (b *Bot) handleAnswer(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	var qid, opt int
	var isInterview bool

	if _, err := fmt.Sscanf(callback.Data, "answer_%d_%d", &qid, &opt); err == nil {
		isInterview = false
	} else if _, err := fmt.Sscanf(callback.Data, "interview_%d_%d", &qid, &opt); err == nil {
		isInterview = true
	} else {
		b.logger.Warn("Failed to parse callback data", slog.String("data", callback.Data))
		b.answerCallback(callback.ID, "Ошибка")
		return
	}

	// Получаем пользователя
	user, err := b.userRepo.GetOrCreate(chatID)
	if err != nil {
		b.logger.Error("Failed to get user", slog.Int64("chat_id", chatID), slog.Any("error", err))
		return
	}

	// Находим вопрос
	questions := b.questions
	if isInterview {
		questions = b.interviewQuestions
	}

	var q *models.Question
	for i := range questions {
		if questions[i].ID == qid {
			q = &questions[i]
			break
		}
	}
	if q == nil {
		b.answerCallback(callback.ID, "Вопрос не найден")
		return
	}

	// Проверяем правильность
	var resultText string
	if opt == q.Correct {
		user.TotalEXP += q.Exp
		user.CorrectAnswers++
		resultText = fmt.Sprintf("✅ Правильно! +%d EXP", q.Exp)
	} else {
		user.WrongAnswers++
		correctOption := q.Options[q.Correct]
		resultText = fmt.Sprintf("❌ Неправильно. Правильный ответ: %s", correctOption)
	}

	// Обновляем уровень
	newLevel := int(float64(user.TotalEXP)/100) + 1
	if newLevel > user.Level {
		user.Level = newLevel
	}

	// Сохраняем прогресс
	if err := b.userRepo.Save(chatID, user); err != nil {
		b.logger.Error("Failed to save user", slog.Int64("chat_id", chatID), slog.Any("error", err))
	}

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	b.send(edit)

	// Отправляем результат
	b.sendText(chatID, resultText)
	b.answerCallback(callback.ID, "")
}

// --- Команды ---

func (b *Bot) handleQuizCommand(chatID int64) {
	user, err := b.userRepo.GetOrCreate(chatID)
	if err != nil {
		b.logger.Error("Failed to get user", slog.Int64("chat_id", chatID), slog.Any("error", err))
		return
	}
	b.sendRandomQuestion(chatID, user)
}

func (b *Bot) handleScoreCommand(chatID int64) {
	user, err := b.userRepo.GetOrCreate(chatID)
	if err != nil {
		b.logger.Error("Failed to get user", slog.Int64("chat_id", chatID), slog.Any("error", err))
		return
	}
	b.sendScore(chatID, user)
}

func (b *Bot) handleInterviewCommand(chatID int64) {
	user, err := b.userRepo.GetOrCreate(chatID)
	if err != nil {
		b.logger.Error("Failed to get user", slog.Int64("chat_id", chatID), slog.Any("error", err))
		return
	}
	b.sendRandomInterviewQuestion(chatID, user)
}

func (b *Bot) handleResetCommand(chatID int64) {
	if err := b.userRepo.ResetProgress(chatID); err != nil {
		b.logger.Error("Failed to reset progress", slog.Int64("chat_id", chatID), slog.Any("error", err))
		b.sendText(chatID, "Ошибка при сбросе прогресса")
		return
	}
	b.sendText(chatID, "🔄 Прогресс сброшен. Все вопросы снова доступны!")
}

// --- Отправка сообщений ---

func (b *Bot) sendText(chatID int64, text string) {
	b.send(tgbotapi.NewMessage(chatID, text))
}

func (b *Bot) sendMarkdown(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.send(msg)
}

func (b *Bot) send(c tgbotapi.Chattable) {
	if _, err := b.api.Send(c); err != nil {
		b.logger.Error("Failed to send message", slog.Any("error", err))
	}
}

func (b *Bot) answerCallback(id string, text string) {
	callback := tgbotapi.NewCallback(id, text)
	b.api.Request(callback)
}

// --- Меню и справка ---

func (b *Bot) sendStartMenu(chatID int64) {
	text := "Привет! Я Go-викторина 🧠\n\n" +
		"Проверь свои знания языка Go. Отвечай на вопросы и получай EXP."

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 Начать викторину", "cmd_quiz"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💼 Вопросы к собеседованию - Gopher, Go Offer!", "cmd_interview"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Моя статистика", "cmd_score"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏆 Таблица лидеров", "cmd_leaderboard"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сбросить прогресс", "cmd_reset"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Помощь", "cmd_help"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	b.send(msg)
}

func (b *Bot) sendHelp(chatID int64) {
	text := "📋 *Справка по командам:*\n\n" + getHelpText()

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 Начать викторину", "cmd_quiz"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Моя статистика", "cmd_score"),
			tgbotapi.NewInlineKeyboardButtonData("🏆 Лидеры", "cmd_leaderboard"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сбросить прогресс", "cmd_reset"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.send(msg)
}

func (b *Bot) sendScore(chatID int64, user *models.UserData) {
	totalAnswers := user.CorrectAnswers + user.WrongAnswers
	text := fmt.Sprintf("📊 *Твоя статистика*\n\n"+
		"Уровень: %d\n"+
		"Всего EXP: %d\n"+
		"Правильных ответов: %d\n"+
		"Неправильных: %d\n"+
		"Всего ответов: %d",
		user.Level, user.TotalEXP, user.CorrectAnswers, user.WrongAnswers, totalAnswers)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 Начать викторину", "cmd_quiz"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏆 Таблица лидеров", "cmd_leaderboard"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Сбросить прогресс", "cmd_reset"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Помощь", "cmd_help"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.send(msg)
}

func (b *Bot) sendLeaderboard(chatID int64) {
	top, err := b.userRepo.GetTop(10)
	if err != nil {
		b.logger.Error("Failed to get leaderboard", slog.Any("error", err))
		b.sendText(chatID, "Ошибка при загрузке таблицы лидеров")
		return
	}

	text := "🏆 *Топ игроков*\n\n"
	for i, item := range top {
		text += fmt.Sprintf("%d. ID %d – Уровень %d, EXP %d\n", i+1, item.ChatID, item.User.Level, item.User.TotalEXP)
	}
	if len(top) == 0 {
		text = "Пока нет игроков. Будь первым!"
	}

	b.sendMarkdown(chatID, text)
}

// --- Вопросы ---

func (b *Bot) sendRandomQuestion(chatID int64, user *models.UserData) {
	b.mu.RLock()
	questions := b.questions
	b.mu.RUnlock()

	if len(questions) == 0 {
		b.sendText(chatID, "Вопросы пока не загружены. Попробуй позже.")
		return
	}

	available := b.getAvailableQuestions(questions, user.AskedQuestions)
	if len(available) == 0 {
		b.sendText(chatID, "Вы ответили на все вопросы! Отправьте /reset, чтобы начать заново.")
		return
	}

	q := available[time.Now().UnixNano()%int64(len(available))]
	b.sendQuestion(chatID, &q, "answer", true)
}

func (b *Bot) sendRandomInterviewQuestion(chatID int64, user *models.UserData) {
	b.mu.RLock()
	questions := b.interviewQuestions
	b.mu.RUnlock()

	if len(questions) == 0 {
		b.sendText(chatID, "Вопросы собеседования пока не загружены. Попробуй позже.")
		return
	}

	available := b.getAvailableQuestions(questions, user.InterviewAsked)
	if len(available) == 0 {
		b.sendText(chatID, "Вы ответили на все вопросы собеседования! Отправьте /reset, чтобы начать заново.")
		return
	}

	q := available[time.Now().UnixNano()%int64(len(available))]
	b.sendQuestion(chatID, &q, "interview", false)
}

func (b *Bot) getAvailableQuestions(all []models.Question, asked []int) []models.Question {
	askedMap := make(map[int]bool)
	for _, id := range asked {
		askedMap[id] = true
	}

	var available []models.Question
	for _, q := range all {
		if !askedMap[q.ID] {
			available = append(available, q)
		}
	}
	return available
}

func (b *Bot) sendQuestion(chatID int64, q *models.Question, prefix string, isQuiz bool) {
	text := fmt.Sprintf("❓ *Вопрос:*\n%s", q.Question)
	if prefix == "interview" {
		text = fmt.Sprintf("💼 *Вопрос собеседования (Gopher/Go Offer):*\n%s", q.Question)
	}

	keyboard := b.createQuestionKeyboard(q.ID, q.Options, prefix)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.send(msg)
}

func (b *Bot) createQuestionKeyboard(qid int, options []string, prefix string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, opt := range options {
		callbackData := fmt.Sprintf("%s_%d_%d", prefix, qid, i)
		btn := tgbotapi.NewInlineKeyboardButtonData(opt, callbackData)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// --- Вспомогательные функции ---

func getHelpText() string {
	return "📋 Доступные команды:\n" +
		"/quiz – начать викторину (новый вопрос)\n" +
		"/interview – вопрос собеседования (Gopher/Go Offer)\n" +
		"/score – показать твой прогресс\n" +
		"/leaderboard – топ-10 игроков\n" +
		"/reset – сбросить список отвеченных вопросов"
}

// LoadQuestionsFromJSON загружает вопросы из JSON файла
func (b *Bot) LoadQuestionsFromJSON(path string) ([]models.Question, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var questions []models.Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}
