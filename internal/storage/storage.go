package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

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

	// Получаем список применённых миграций
	rows, err := s.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return err
		}
		applied[version] = true
	}

	// Получаем список файлов миграций
	files, err := migrationFiles.ReadDir(".")
	if err != nil {
		return err
	}

	// Сортируем и применяем миграции
	var versions []int
	for _, f := range files {
		if len(f.Name()) < 4 {
			continue
		}
		var v int
		fmt.Sscanf(f.Name(), "%d", &v)
		if v > 0 && !applied[v] {
			versions = append(versions, v)
		}
	}
	sort.Ints(versions)

	// Применяем каждую миграцию
	for _, v := range versions {
		filename := fmt.Sprintf("%d_", v)
		for _, f := range files {
			if len(f.Name()) >= len(filename) && f.Name()[:len(filename)] == filename {
				data, err := migrationFiles.ReadFile(f.Name())
				if err != nil {
					return err
				}

				// Выполняем миграцию в транзакции
				tx, err := s.db.Begin()
				if err != nil {
					return err
				}

				if _, err := tx.Exec(string(data)); err != nil {
					tx.Rollback()
					return err
				}

				if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", v); err != nil {
					tx.Rollback()
					return err
				}

				if err := tx.Commit(); err != nil {
					return err
				}
			}
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
