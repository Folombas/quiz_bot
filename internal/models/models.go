package models

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

// UsersMap — карта пользователей по chatID
type UsersMap map[int64]*UserData
