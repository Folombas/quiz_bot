package main

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
)

// Version бота
const Version = "0.2.0"

// Question представляет вопрос викторины
type Question struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Correct  int      `json:"correct"` // индекс правильного ответа (0-3)
	Exp      int      `json:"exp"`     // очки за правильный ответ
}

// UserData хранит статистику игрока
type UserData struct {
	TotalEXP       int   `json:"total_exp"`
	CorrectAnswers int   `json:"correct_answers"`
	WrongAnswers   int   `json:"wrong_answers"`
	Level          int   `json:"level"`
	AskedQuestions []int `json:"asked_questions"` // ID вопросов, уже заданных пользователю
	InterviewAsked []int `json:"interview_asked"` // ID вопросов собеседования, уже заданных
}

// глобальные переменные
var (
	questions        []Question
	interviewQuestions []Question
	users            = make(map[int64]*UserData)
	bot              *tgbotapi.BotAPI
	questionsFile    = "questions.json"
	interviewQuestionsFile = "../qwen_test/questions.json" // вопросы собеседования
	dataFile         = "users.json"                        // файл для сохранения данных
)

func main() {
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

	// Создаём бота с токеном из переменной
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Ошибка создания бота:", err)
	}
	bot.Debug = false
	log.Printf("Бот авторизован: %s (v%s)", bot.Self.UserName, Version)

	// Загружаем вопросы
	if err := loadQuestions(); err != nil {
		log.Fatal("Ошибка загрузки вопросов:", err)
	}
	log.Printf("Загружено %d вопросов", len(questions))

	// Загружаем вопросы собеседования
	if err := loadInterviewQuestions(); err != nil {
		log.Println("Предупреждение: вопросы собеседования не загружены:", err)
	} else {
		log.Printf("Загружено %d вопросов собеседования", len(interviewQuestions))
	}

	// Загружаем сохранённые данные пользователей
	loadUserData()

	// Канал для graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Канал обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Таймер для автосохранения данных (каждые 5 минут)
	saveTicker := time.NewTicker(5 * time.Minute)
	defer saveTicker.Stop()

	for {
		select {
		case <-stop:
			log.Println("Остановка бота, сохраняем данные...")
			saveUserData()
			return
		case <-saveTicker.C:
			saveUserData()
		case update := <-updates:
			if update.Message != nil {
				handleMessage(update.Message)
			} else if update.CallbackQuery != nil {
				handleCallback(update.CallbackQuery)
			}
		}
	}
}

// --- Загрузка/сохранение данных ---

func loadQuestions() error {
	data, err := ioutil.ReadFile(questionsFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &questions)
}

func loadInterviewQuestions() error {
	data, err := ioutil.ReadFile(interviewQuestionsFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &interviewQuestions)
}

func loadUserData() {
	data, err := ioutil.ReadFile(dataFile)
	if err != nil {
		log.Println("Файл пользователей не найден, начинаем с пустой статистикой")
		return
	}
	err = json.Unmarshal(data, &users)
	if err != nil {
		log.Println("Ошибка парсинга users.json, используем пустую статистику")
	}
}

func saveUserData() {
	data, _ := json.MarshalIndent(users, "", "  ")
	if err := ioutil.WriteFile(dataFile, data, 0644); err != nil {
		log.Println("Ошибка сохранения данных пользователей:", err)
	} else {
		log.Println("Данные пользователей сохранены")
	}
}

// --- Работа с пользователями ---

func getUser(chatID int64) *UserData {
	if _, ok := users[chatID]; !ok {
		users[chatID] = &UserData{
			TotalEXP:       0,
			CorrectAnswers: 0,
			WrongAnswers:   0,
			Level:          1,
			AskedQuestions: []int{},
		}
	}
	return users[chatID]
}

func updateLevel(user *UserData) {
	newLevel := int(math.Floor(float64(user.TotalEXP)/100)) + 1
	if newLevel > user.Level {
		user.Level = newLevel
		// можно отправить сообщение о повышении уровня
	}
}

// --- Обработчики ---

func handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	user := getUser(chatID)

	if !msg.IsCommand() {
		return // игнорируем некоманды
	}

	switch msg.Command() {
	case "start":
		text := "Привет! Я Go-викторина 🧠\n\n" +
			"Проверь свои знания языка Go. Отвечай на вопросы и получай EXP."

		// Создаём клавиатуру с основными командами
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
		bot.Send(msg)

	case "help":
		text := "📋 *Справка по командам:*\n\n" + getHelpText()

		// Создаём клавиатуру с быстрыми действиями
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
		bot.Send(msg)

	case "quiz":
		sendRandomQuestion(chatID, user)

	case "score":
		totalAnswers := user.CorrectAnswers + user.WrongAnswers
		text := fmt.Sprintf("📊 *Твоя статистика*\n\n"+
			"Уровень: %d\n"+
			"Всего EXP: %d\n"+
			"Правильных ответов: %d\n"+
			"Неправильных: %d\n"+
			"Всего ответов: %d\n"+
			"Отвечено вопросов: %d из %d",
			user.Level, user.TotalEXP, user.CorrectAnswers, user.WrongAnswers,
			totalAnswers, len(user.AskedQuestions), len(questions))

		// Создаём клавиатуру с быстрыми действиями
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
		bot.Send(msg)

	case "leaderboard":
		sendLeaderboard(chatID)

	case "interview":
		sendRandomInterviewQuestion(chatID, user)

	case "reset":
		user.AskedQuestions = []int{}
		user.InterviewAsked = []int{}
		bot.Send(tgbotapi.NewMessage(chatID, "🔄 Прогресс сброшен. Все вопросы снова доступны!"))

	default:
		bot.Send(tgbotapi.NewMessage(chatID, "Неизвестная команда. Напиши /help"))
	}
}

func handleCallback(callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	user := getUser(chatID)

	// Обработка команд из кнопок меню
	if callback.Data == "cmd_quiz" {
		sendRandomQuestion(chatID, user)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	} else if callback.Data == "cmd_score" {
		totalAnswers := user.CorrectAnswers + user.WrongAnswers
		text := fmt.Sprintf("📊 *Твоя статистика*\n\n"+
			"Уровень: %d\n"+
			"Всего EXP: %d\n"+
			"Правильных ответов: %d\n"+
			"Неправильных: %d\n"+
			"Всего ответов: %d\n"+
			"Отвечено вопросов: %d из %d",
			user.Level, user.TotalEXP, user.CorrectAnswers, user.WrongAnswers,
			totalAnswers, len(user.AskedQuestions), len(questions))
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	} else if callback.Data == "cmd_leaderboard" {
		sendLeaderboard(chatID)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	} else if callback.Data == "cmd_reset" {
		user.AskedQuestions = []int{}
		user.InterviewAsked = []int{}
		bot.Send(tgbotapi.NewMessage(chatID, "🔄 Прогресс сброшен. Все вопросы снова доступны!"))
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	} else if callback.Data == "cmd_help" {
		bot.Send(tgbotapi.NewMessage(chatID, getHelpText()))
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	} else if callback.Data == "cmd_interview" {
		sendRandomInterviewQuestion(chatID, user)
		bot.Request(tgbotapi.NewCallback(callback.ID, ""))
		return
	}

	// Данные кнопки: "answer_<question_id>_<option_index>" или "interview_<question_id>_<option_index>"
	var qid, opt int
	var isInterview bool

	// Проверяем формат callback data
	if _, err := fmt.Sscanf(callback.Data, "answer_%d_%d", &qid, &opt); err == nil {
		isInterview = false
	} else if _, err := fmt.Sscanf(callback.Data, "interview_%d_%d", &qid, &opt); err == nil {
		isInterview = true
	} else {
		log.Println("Ошибка парсинга callback data:", callback.Data)
		bot.Request(tgbotapi.NewCallback(callback.ID, "Ошибка"))
		return
	}

	// Находим вопрос в соответствующем наборе
	var q *Question
	if isInterview {
		for i := range interviewQuestions {
			if interviewQuestions[i].ID == qid {
				q = &interviewQuestions[i]
				break
			}
		}
	} else {
		for i := range questions {
			if questions[i].ID == qid {
				q = &questions[i]
				break
			}
		}
	}
	if q == nil {
		bot.Request(tgbotapi.NewCallback(callback.ID, "Вопрос не найден"))
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
	updateLevel(user)

	// Удаляем инлайн-клавиатуру у сообщения с вопросом
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	bot.Send(edit)

	// Отправляем результат
	bot.Send(tgbotapi.NewMessage(chatID, resultText))

	// Закрываем колбэк (убираем "часики")
	bot.Request(tgbotapi.NewCallback(callback.ID, ""))
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

func sendRandomQuestion(chatID int64, user *UserData) {
	if len(questions) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Вопросы пока не загружены. Попробуй позже."))
		return
	}

	// Составляем список доступных вопросов (исключая уже заданные)
	askedMap := make(map[int]bool)
	for _, id := range user.AskedQuestions {
		askedMap[id] = true
	}

	var available []Question
	for _, q := range questions {
		if !askedMap[q.ID] {
			available = append(available, q)
		}
	}

	if len(available) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Вы ответили на все вопросы! Отправьте /reset, чтобы начать заново."))
		return
	}

	// Случайный выбор из доступных
	q := available[time.Now().UnixNano()%int64(len(available))]

	// Добавляем вопрос в список заданных
	user.AskedQuestions = append(user.AskedQuestions, q.ID)

	// Формируем текст вопроса
	text := fmt.Sprintf("❓ *Вопрос:*\n%s", q.Question)

	// Создаём инлайн-клавиатуру с вариантами
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, opt := range q.Options {
		callbackData := fmt.Sprintf("answer_%d_%d", q.ID, i)
		btn := tgbotapi.NewInlineKeyboardButtonData(opt, callbackData)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func sendRandomInterviewQuestion(chatID int64, user *UserData) {
	if len(interviewQuestions) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Вопросы собеседования пока не загружены. Попробуй позже."))
		return
	}

	// Составляем список доступных вопросов (исключая уже заданные)
	askedMap := make(map[int]bool)
	for _, id := range user.InterviewAsked {
		askedMap[id] = true
	}

	var available []Question
	for _, q := range interviewQuestions {
		if !askedMap[q.ID] {
			available = append(available, q)
		}
	}

	if len(available) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Вы ответили на все вопросы собеседования! Отправьте /reset, чтобы начать заново."))
		return
	}

	// Случайный выбор из доступных
	q := available[time.Now().UnixNano()%int64(len(available))]

	// Добавляем вопрос в список заданных
	user.InterviewAsked = append(user.InterviewAsked, q.ID)

	// Формируем текст вопроса
	text := fmt.Sprintf("💼 *Вопрос собеседования (Gopher/Go Offer):*\n%s", q.Question)

	// Создаём инлайн-клавиатуру с вариантами
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, opt := range q.Options {
		callbackData := fmt.Sprintf("interview_%d_%d", q.ID, i)
		btn := tgbotapi.NewInlineKeyboardButtonData(opt, callbackData)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

func sendLeaderboard(chatID int64) {
	type kv struct {
		ChatID int64
		User   *UserData
	}
	var list []kv
	for id, u := range users {
		list = append(list, kv{id, u})
	}
	// Сортировка по убыванию EXP
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

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
