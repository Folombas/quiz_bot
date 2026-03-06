package storage

import (
	"database/sql"
	"time"

	"quiz_bot/internal/models"
)

// UserRepository — репозиторий для работы с пользователями
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository создаёт новый репозиторий
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetOrCreate получает пользователя или создаёт нового
func (r *UserRepository) GetOrCreate(chatID int64) (*models.UserData, error) {
	user := &models.UserData{
		Level: 1,
	}

	// Пробуем получить существующего
	row := r.db.QueryRow(`
		SELECT total_exp, correct_answers, wrong_answers, level
		FROM users WHERE chat_id = ?
	`, chatID)

	err := row.Scan(&user.TotalEXP, &user.CorrectAnswers, &user.WrongAnswers, &user.Level)
	if err == sql.ErrNoRows {
		// Создаём нового пользователя
		_, err = r.db.Exec(`
			INSERT INTO users (chat_id, total_exp, correct_answers, wrong_answers, level)
			VALUES (?, 0, 0, 0, 1)
		`, chatID)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	if err != nil {
		return nil, err
	}

	// Загружаем прогресс викторины
	quizRows, err := r.db.Query(`
		SELECT question_id FROM user_quiz_progress WHERE chat_id = ?
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer quizRows.Close()

	for quizRows.Next() {
		var qid int
		if err := quizRows.Scan(&qid); err != nil {
			return nil, err
		}
		user.AskedQuestions = append(user.AskedQuestions, qid)
	}

	// Загружаем прогресс собеседования
	interviewRows, err := r.db.Query(`
		SELECT question_id FROM user_interview_progress WHERE chat_id = ?
	`, chatID)
	if err != nil {
		return nil, err
	}
	defer interviewRows.Close()

	for interviewRows.Next() {
		var qid int
		if err := interviewRows.Scan(&qid); err != nil {
			return nil, err
		}
		user.InterviewAsked = append(user.InterviewAsked, qid)
	}

	return user, nil
}

// Save сохраняет данные пользователя
func (r *UserRepository) Save(chatID int64, user *models.UserData) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Обновляем пользователя
	_, err = tx.Exec(`
		UPDATE users SET 
			total_exp = ?, 
			correct_answers = ?, 
			wrong_answers = ?, 
			level = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE chat_id = ?
	`, user.TotalEXP, user.CorrectAnswers, user.WrongAnswers, user.Level, chatID)
	if err != nil {
		return err
	}

	// Сохраняем прогресс викторины
	if err := r.saveProgress(tx, chatID, user.AskedQuestions, "user_quiz_progress"); err != nil {
		return err
	}

	// Сохраняем прогресс собеседования
	if err := r.saveProgress(tx, chatID, user.InterviewAsked, "user_interview_progress"); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *UserRepository) saveProgress(tx *sql.Tx, chatID int64, questionIDs []int, tableName string) error {
	for _, qid := range questionIDs {
		_, err := tx.Exec(`
			INSERT OR IGNORE INTO `+tableName+` (chat_id, question_id)
			VALUES (?, ?)
		`, chatID, qid)
		if err != nil {
			return err
		}
	}
	return nil
}

// ResetProgress сбрасывает прогресс пользователя
func (r *UserRepository) ResetProgress(chatID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM user_quiz_progress WHERE chat_id = ?`, chatID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM user_interview_progress WHERE chat_id = ?`, chatID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetTop получает топ пользователей
func (r *UserRepository) GetTop(limit int) ([]struct {
	ChatID int64
	User   *models.UserData
}, error) {
	rows, err := r.db.Query(`
		SELECT chat_id, total_exp, correct_answers, wrong_answers, level
		FROM users
		ORDER BY total_exp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []struct {
		ChatID int64
		User   *models.UserData
	}

	for rows.Next() {
		var chatID int64
		user := &models.UserData{}
		if err := rows.Scan(&chatID, &user.TotalEXP, &user.CorrectAnswers, &user.WrongAnswers, &user.Level); err != nil {
			return nil, err
		}
		result = append(result, struct {
			ChatID int64
			User   *models.UserData
		}{ChatID: chatID, User: user})
	}

	return result, nil
}

// GetAllUsers получает всех пользователей
func (r *UserRepository) GetAllUsers() ([]int64, error) {
	rows, err := r.db.Query(`SELECT chat_id FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}
		chatIDs = append(chatIDs, chatID)
	}

	return chatIDs, nil
}

// RecordAnswer записывает ответ на вопрос
func (r *UserRepository) RecordAnswer(chatID int64, questionID int, isInterview bool) error {
	tableName := "user_quiz_progress"
	if isInterview {
		tableName = "user_interview_progress"
	}

	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO `+tableName+` (chat_id, question_id)
		VALUES (?, ?)
	`, chatID, questionID)
	return err
}

// HasAnswered проверяет, отвечал ли пользователь на вопрос
func (r *UserRepository) HasAnswered(chatID int64, questionID int, isInterview bool) (bool, error) {
	tableName := "user_quiz_progress"
	if isInterview {
		tableName = "user_interview_progress"
	}

	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM `+tableName+` WHERE chat_id = ? AND question_id = ?
	`, chatID, questionID).Scan(&count)

	return count > 0, err
}

// Stats возвращает статистику хранилища
type Stats struct {
	TotalUsers      int64 `json:"total_users"`
	TotalQuizAnswers int64 `json:"total_quiz_answers"`
	TotalInterviewAnswers int64 `json:"total_interview_answers"`
}

// GetStats возвращает статистику по всем пользователям
func (r *UserRepository) GetStats() (*Stats, error) {
	stats := &Stats{}

	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`SELECT COUNT(*) FROM user_quiz_progress`).Scan(&stats.TotalQuizAnswers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`SELECT COUNT(*) FROM user_interview_progress`).Scan(&stats.TotalInterviewAnswers)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// UpdateUserLevel обновляет уровень пользователя
func (r *UserRepository) UpdateUserLevel(chatID int64, level int) error {
	_, err := r.db.Exec(`
		UPDATE users SET level = ?, updated_at = CURRENT_TIMESTAMP
		WHERE chat_id = ?
	`, level, chatID)
	return err
}

// GetUserByChatID получает пользователя по chatID
func (r *UserRepository) GetUserByChatID(chatID int64) (*models.UserData, error) {
	return r.GetOrCreate(chatID)
}

// SaveUserWithTimestamp сохраняет пользователя с обновлением timestamp
func (r *UserRepository) SaveUserWithTimestamp(chatID int64, user *models.UserData) error {
	return r.Save(chatID, user)
}

// DeleteUser удаляет пользователя
func (r *UserRepository) DeleteUser(chatID int64) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE chat_id = ?`, chatID)
	return err
}

// GetUserCount возвращает количество пользователей
func (r *UserRepository) GetUserCount() (int64, error) {
	var count int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// GetActiveUsers возвращает активных пользователей за последний час
func (r *UserRepository) GetActiveUsers(duration time.Duration) (int64, error) {
	var count int64
	err := r.db.QueryRow(`
		SELECT COUNT(DISTINCT chat_id) FROM (
			SELECT chat_id FROM user_quiz_progress WHERE answered_at >= datetime('now', ?)
			UNION
			SELECT chat_id FROM user_interview_progress WHERE answered_at >= datetime('now', ?)
		)
	`, "-"+duration.String(), "-"+duration.String()).Scan(&count)
	return count, err
}
