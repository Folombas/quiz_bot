package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Storage — обёртка над SQL базой
type Storage struct {
	db *sql.DB
}

// NewStorage создаёт новое хранилище
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Включаем foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	s := &Storage{db: db}

	// Применяем миграции
	if err := s.Migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// Migrate применяет все миграции
func (s *Storage) Migrate() error {
	// Создаём таблицу миграций
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	// Проверяем, применена ли миграция 1
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Применяем миграцию 1
		migration1 := `
		CREATE TABLE IF NOT EXISTS users (
			chat_id INTEGER PRIMARY KEY,
			total_exp INTEGER NOT NULL DEFAULT 0,
			correct_answers INTEGER NOT NULL DEFAULT 0,
			wrong_answers INTEGER NOT NULL DEFAULT 0,
			level INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS user_quiz_progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER NOT NULL,
			question_id INTEGER NOT NULL,
			answered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chat_id) REFERENCES users(chat_id) ON DELETE CASCADE,
			UNIQUE(chat_id, question_id)
		);

		CREATE TABLE IF NOT EXISTS user_interview_progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER NOT NULL,
			question_id INTEGER NOT NULL,
			answered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (chat_id) REFERENCES users(chat_id) ON DELETE CASCADE,
			UNIQUE(chat_id, question_id)
		);

		CREATE INDEX IF NOT EXISTS idx_users_level ON users(level);
		CREATE INDEX IF NOT EXISTS idx_users_exp ON users(total_exp);
		CREATE INDEX IF NOT EXISTS idx_quiz_progress_chat ON user_quiz_progress(chat_id);
		CREATE INDEX IF NOT EXISTS idx_interview_progress_chat ON user_interview_progress(chat_id);

		INSERT INTO schema_migrations (version) VALUES (1);
		`

		if _, err := s.db.Exec(migration1); err != nil {
			return fmt.Errorf("migration 1 failed: %w", err)
		}
	}

	return nil
}

// Close закрывает соединение
func (s *Storage) Close() error {
	return s.db.Close()
}

// DB возвращает sql.DB
func (s *Storage) DB() *sql.DB {
	return s.db
}
