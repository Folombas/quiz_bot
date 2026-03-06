package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"quiz_bot/internal/models"
)

// Version бота
const Version = "0.2.0"

// Config — конфигурация бота
type Config struct {
	QuestionsFile          string
	InterviewQuestionsFile string
	UsersFile              string
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		QuestionsFile:          "configs/questions.json",
		InterviewQuestionsFile: "configs/interview_questions.json",
		UsersFile:              "configs/users.json",
	}
}

// Bot — структура телеграм-бота
type Bot struct {
	api                *tgbotapi.BotAPI
	config             Config
	questions          []models.Question
	interviewQuestions []models.Question
	users              models.UsersMap
}

// New создаёт нового бота
func New(cfg Config) *Bot {
	return &Bot{
		config: cfg,
		users:  make(models.UsersMap),
	}
}

// Run запускает бота
func (b *Bot) Run() error {
	// Загружаем переменные из .env файла
	err := godotenv.Load()
	if err != nil {
		log.Println("Файл .env не найден, используем системные переменные окружения")
	}

	// Читаем токен из переменной окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Не задан TELEGRAM_BOT_TOKEN в .env или окружении")
	}

	// Создаём бота
	b.api, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return fmt.Errorf("ошибка создания бота: %w", err)
	}
	b.api.Debug = false
	log.Printf("Бот авторизован: %s (v%s)", b.api.Self.UserName, Version)

	// Загружаем вопросы
	if err := b.loadQuestions(); err != nil {
		return fmt.Errorf("ошибка загрузки вопросов: %w", err)
	}
	log.Printf("Загружено %d вопросов", len(b.questions))

	// Загружаем вопросы собеседования
	if err := b.loadInterviewQuestions(); err != nil {
		log.Println("Предупреждение: вопросы собеседования не загружены:", err)
	} else {
		log.Printf("Загружено %d вопросов собеседования", len(b.interviewQuestions))
	}

	// Загружаем данные пользователей
	b.loadUserData()

	// Канал для graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Канал обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	// Таймер для автосохранения данных (каждые 5 минут)
	saveTicker := time.NewTicker(5 * time.Minute)
	defer saveTicker.Stop()

	for {
		select {
		case <-stop:
			log.Println("Остановка бота, сохраняем данные...")
			b.saveUserData()
			return nil
		case <-saveTicker.C:
			b.saveUserData()
		case update := <-updates:
			b.handleUpdate(update)
		}
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		b.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallback(update.CallbackQuery)
	}
}

// --- Загрузка/сохранение данных ---

func (b *Bot) loadQuestions() error {
	data, err := ioutil.ReadFile(b.config.QuestionsFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &b.questions)
}

func (b *Bot) loadInterviewQuestions() error {
	data, err := ioutil.ReadFile(b.config.InterviewQuestionsFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &b.interviewQuestions)
}

func (b *Bot) loadUserData() {
	data, err := ioutil.ReadFile(b.config.UsersFile)
	if err != nil {
		log.Println("Файл пользователей не найден, начинаем с пустой статистикой")
		return
	}
	err = json.Unmarshal(data, &b.users)
	if err != nil {
		log.Println("Ошибка парсинга users.json, используем пустую статистику")
	}
}

func (b *Bot) saveUserData() {
	data, _ := json.MarshalIndent(b.users, "", "  ")
	if err := ioutil.WriteFile(b.config.UsersFile, data, 0644); err != nil {
		log.Println("Ошибка сохранения данных пользователей:", err)
	} else {
		log.Println("Данные пользователей сохранены")
	}
}

// --- Работа с пользователями ---

func (b *Bot) getUser(chatID int64) *models.UserData {
	if _, ok := b.users[chatID]; !ok {
		b.users[chatID] = &models.UserData{
			TotalEXP:       0,
			CorrectAnswers: 0,
			WrongAnswers:   0,
			Level:          1,
			AskedQuestions: []int{},
			InterviewAsked: []int{},
		}
	}
	return b.users[chatID]
}

func (b *Bot) updateLevel(user *models.UserData) {
	newLevel := int(math.Floor(float64(user.TotalEXP)/100)) + 1
	if newLevel > user.Level {
		user.Level = newLevel
	}
}

// --- Обработчики ---

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	user := b.getUser(chatID)

	if !msg.IsCommand() {
		return
	}

	switch msg.Command() {
	case "start":
		b.sendStartMenu(chatID)
	case "help":
		b.sendHelp(chatID)
	case "quiz":
		b.sendRandomQuestion(chatID, user)
	case "score":
		b.sendScore(chatID, user)
	case "leaderboard":
		b.sendLeaderboard(chatID)
	case "interview":
		b.sendRandomInterviewQuestion(chatID, user)
	case "reset":
		user.AskedQuestions = []int{}
		user.InterviewAsked = []int{}
		b.sendText(chatID, "🔄 Прогресс сброшен. Все вопросы снова доступны!")
	default:
		b.sendText(chatID, "Неизвестная команда. Напиши /help")
	}
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	user := b.getUser(chatID)

	// Обработка команд меню
	switch callback.Data {
	case "cmd_quiz":
		b.sendRandomQuestion(chatID, user)
		b.answerCallback(callback.ID, "")
		return
	case "cmd_score":
		b.sendScore(chatID, user)
		b.answerCallback(callback.ID, "")
		return
	case "cmd_leaderboard":
		b.sendLeaderboard(chatID)
		b.answerCallback(callback.ID, "")
		return
	case "cmd_reset":
		user.AskedQuestions = []int{}
		user.InterviewAsked = []int{}
		b.sendText(chatID, "🔄 Прогресс сброшен. Все вопросы снова доступны!")
		b.answerCallback(callback.ID, "")
		return
	case "cmd_help":
		b.sendText(chatID, getHelpText())
		b.answerCallback(callback.ID, "")
		return
	case "cmd_interview":
		b.sendRandomInterviewQuestion(chatID, user)
		b.answerCallback(callback.ID, "")
		return
	}

	// Обработка ответов на вопросы
	var qid, opt int
	var isInterview bool

	if _, err := fmt.Sscanf(callback.Data, "answer_%d_%d", &qid, &opt); err == nil {
		isInterview = false
	} else if _, err := fmt.Sscanf(callback.Data, "interview_%d_%d", &qid, &opt); err == nil {
		isInterview = true
	} else {
		log.Println("Ошибка парсинга callback data:", callback.Data)
		b.answerCallback(callback.ID, "Ошибка")
		return
	}

	// Находим вопрос
	var q *models.Question
	if isInterview {
		for i := range b.interviewQuestions {
			if b.interviewQuestions[i].ID == qid {
				q = &b.interviewQuestions[i]
				break
			}
		}
	} else {
		for i := range b.questions {
			if b.questions[i].ID == qid {
				q = &b.questions[i]
				break
			}
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
	b.updateLevel(user)

	// Удаляем клавиатуру
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	b.send(edit)

	// Отправляем результат
	b.sendText(chatID, resultText)
	b.answerCallback(callback.ID, "")
}

func (b *Bot) answerCallback(id string, text string) {
	callback := tgbotapi.NewCallback(id, text)
	b.api.Request(callback)
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
	b.api.Send(c)
}

// --- Команды меню ---

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
		"Всего ответов: %d\n"+
		"Отвечено вопросов: %d из %d",
		user.Level, user.TotalEXP, user.CorrectAnswers, user.WrongAnswers,
		totalAnswers, len(user.AskedQuestions), len(b.questions))

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
	type kv struct {
		ChatID int64
		User   *models.UserData
	}
	var list []kv
	for id, u := range b.users {
		list = append(list, kv{id, u})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].User.TotalEXP > list[j].User.TotalEXP
	})

	limit := 10
	if len(list) < limit {
		limit = len(list)
	}
	top := list[:limit]

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
	if len(b.questions) == 0 {
		b.sendText(chatID, "Вопросы пока не загружены. Попробуй позже.")
		return
	}

	askedMap := make(map[int]bool)
	for _, id := range user.AskedQuestions {
		askedMap[id] = true
	}

	var available []models.Question
	for _, q := range b.questions {
		if !askedMap[q.ID] {
			available = append(available, q)
		}
	}

	if len(available) == 0 {
		b.sendText(chatID, "Вы ответили на все вопросы! Отправьте /reset, чтобы начать заново.")
		return
	}

	q := available[time.Now().UnixNano()%int64(len(available))]
	user.AskedQuestions = append(user.AskedQuestions, q.ID)

	text := fmt.Sprintf("❓ *Вопрос:*\n%s", q.Question)
	keyboard := b.createQuestionKeyboard(q.ID, q.Options, "answer")

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.send(msg)
}

func (b *Bot) sendRandomInterviewQuestion(chatID int64, user *models.UserData) {
	if len(b.interviewQuestions) == 0 {
		b.sendText(chatID, "Вопросы собеседования пока не загружены. Попробуй позже.")
		return
	}

	askedMap := make(map[int]bool)
	for _, id := range user.InterviewAsked {
		askedMap[id] = true
	}

	var available []models.Question
	for _, q := range b.interviewQuestions {
		if !askedMap[q.ID] {
			available = append(available, q)
		}
	}

	if len(available) == 0 {
		b.sendText(chatID, "Вы ответили на все вопросы собеседования! Отправьте /reset, чтобы начать заново.")
		return
	}

	q := available[time.Now().UnixNano()%int64(len(available))]
	user.InterviewAsked = append(user.InterviewAsked, q.ID)

	text := fmt.Sprintf("💼 *Вопрос собеседования (Gopher/Go Offer):*\n%s", q.Question)
	keyboard := b.createQuestionKeyboard(q.ID, q.Options, "interview")

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
