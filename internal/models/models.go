// Package models содержит модели данных
package models

// Question представляет вопрос викторины
type Question struct {
	ID       int      `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Correct  int      `json:"correct"`
	Exp      int      `json:"exp"`
}

// UserData хранит статистику игрока
type UserData struct {
	TotalEXP       int   `json:"total_exp"`
	CorrectAnswers int   `json:"correct_answers"`
	WrongAnswers   int   `json:"wrong_answers"`
	Level          int   `json:"level"`
	AskedQuestions []int `json:"asked_questions"`
	InterviewAsked []int `json:"interview_asked"`
}
