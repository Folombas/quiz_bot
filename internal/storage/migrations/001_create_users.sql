-- Migration 001: Create users table

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
