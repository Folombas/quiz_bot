# Quiz Bot — Telegram Go Quiz Bot

[![Go Report Card](https://goreportcard.com/badge/github.com/Folombas/quiz_bot)](https://goreportcard.com/report/github.com/Folombas/quiz_bot)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Телеграм-бот для викторины по языку Go с вопросами для подготовки к собеседованиям.

## 🚀 Особенности

- ✅ **120+ вопросов** для подготовки к собеседованиям на позицию Gopher
- ✅ **70+ вопросов** базовой викторины по Go
- ✅ **SQLite база данных** с миграциями
- ✅ **Rate limiting** для защиты от спама
- ✅ **Структурированное логирование** (slog)
- ✅ **YAML конфигурация** для разных окружений
- ✅ **Unit-тесты** ключевых компонентов
- ✅ **Docker** поддержка

## 📁 Структура проекта

```
quiz_bot/
├── cmd/
│   └── quiz-bot/
│       └── main.go              # Точка входа
├── internal/
│   ├── bot/                     # Логика бота
│   ├── config/                  # Конфигурация
│   ├── logger/                  # Логирование
│   ├── models/                  # Модели данных
│   ├── ratelimit/               # Rate limiting
│   └── storage/                 # База данных и репозитории
├── configs/
│   ├── config.dev.yaml          # Dev конфигурация
│   ├── config.prod.yaml         # Prod конфигурация
│   ├── questions.json           # Вопросы викторины
│   └── interview_questions.json # Вопросы собеседования
├── data/                        # SQLite база данных
├── migrations/                  # SQL миграции
├── deploy/
│   ├── Dockerfile
│   └── docker-compose.yml
├── tests/                       # Интеграционные тесты
├── go.mod
├── Makefile
└── README.md
```

## 🛠 Требования

- Go 1.21+
- SQLite 3
- Telegram Bot Token

## 🚀 Быстрый старт

### 1. Клонирование репозитория

```bash
git clone git@github.com:Folombas/quiz_bot.git
cd quiz_bot
```

### 2. Настройка переменных окружения

```bash
cp .env.example .env
# Отредактируйте .env и добавьте ваш TELEGRAM_BOT_TOKEN
```

### 3. Настройка конфигурации

```bash
# Для разработки
cp configs/config.dev.yaml configs/config.yaml

# Для продакшена
cp configs/config.prod.yaml configs/config.yaml
```

### 4. Запуск

```bash
# Через Make
make run

# Или напрямую
go run cmd/quiz-bot/main.go

# С кастомным конфигом
go run cmd/quiz-bot/main.go -config configs/config.prod.yaml
```

### 5. Запуск через Docker

```bash
# Сборка и запуск
make docker-build
make docker-up

# Или через docker-compose
docker-compose -f deploy/docker-compose.yml up -d
```

## 📋 Команды бота

| Команда | Описание |
|---------|----------|
| `/start` | Главное меню |
| `/quiz` | Обычный вопрос викторины |
| `/interview` | Вопрос собеседования (Gopher/Go Offer) |
| `/score` | Статистика игрока |
| `/leaderboard` | Таблица лидеров |
| `/reset` | Сброс прогресса |
| `/help` | Справка |

## ⚙️ Конфигурация

### config.dev.yaml (разработка)

```yaml
bot:
  token_env_var: TELEGRAM_BOT_TOKEN
  max_connections: 100
  timeout: 60

database:
  type: sqlite
  sqlite:
    path: data/quiz_bot.db

log:
  level: debug
  format: text
  output: stdout

rate_limit:
  enabled: true
  requests_per_min: 30
  burst_size: 10
```

### config.prod.yaml (продакшен)

```yaml
bot:
  token_env_var: TELEGRAM_BOT_TOKEN
  max_connections: 200
  timeout: 120

database:
  type: sqlite
  sqlite:
    path: data/quiz_bot.db

log:
  level: info
  format: json
  output: stdout

rate_limit:
  enabled: true
  requests_per_min: 20
  burst_size: 5
```

## 🧪 Тесты

```bash
# Запустить все тесты
make test

# Запустить с покрытием
go test -cover ./...

# Запустить конкретный пакет
go test ./internal/config/... -v
```

## 📦 Make команды

```bash
make build          # Сборка бинарника
make run            # Запуск бота
make test           # Запуск тестов
make clean          # Очистка
make docker-build   # Сборка Docker-образа
make docker-up      # Запуск Docker
make docker-down    # Остановка Docker
```

## 🏗 Архитектура

### Компоненты

1. **Bot** (`internal/bot`) — основная логика, обработка команд
2. **Storage** (`internal/storage`) — работа с SQLite, миграции, репозитории
3. **Config** (`internal/config`) — загрузка YAML конфигурации
4. **Logger** (`internal/logger`) — структурированное логирование
5. **RateLimiter** (`internal/ratelimit`) — token bucket algorithm

### База данных

```sql
users
├── chat_id (PRIMARY KEY)
├── total_exp
├── correct_answers
├── wrong_answers
├── level
└── created_at, updated_at

user_quiz_progress
├── id
├── chat_id (FK)
├── question_id
└── answered_at

user_interview_progress
├── id
├── chat_id (FK)
├── question_id
└── answered_at
```

## 📊 Метрики

- Total Users — общее количество пользователей
- Active Users — активные за последний час
- Total Quiz Answers — всего ответов на вопросы викторины
- Total Interview Answers — всего ответов на вопросы собеседования

## 🔐 Безопасность

- Токен бота хранится в переменной окружения
- Rate limiting защищает от DDoS
- SQLite ACID транзакции обеспечивают целостность данных

## 📝 Лицензия

MIT

## 👥 Авторы

- Folombas

## 🙏 Благодарности

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [SQLite](https://www.sqlite.org/)
- [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml)
