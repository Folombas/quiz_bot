# Quiz Bot - Telegram Go Quiz Bot

Телеграм-бот для викторины по языку Go с вопросами для подготовки к собеседованиям.

## Структура проекта (Standard Go Project Layout)

```
quiz_bot/
├── cmd/
│   └── quiz-bot/          # Точка входа приложения
│       └── main.go
├── internal/               # Приватный код приложения
│   ├── bot/               # Логика бота
│   └── models/            # Модели данных
├── configs/                # Конфигурационные файлы
│   ├── questions.json     # Вопросы викторины
│   ├── interview_questions.json  # Вопросы собеседования
│   └── users.json         # Данные пользователей (не коммитится)
├── deploy/                 # Файлы для развёртывания
│   ├── Dockerfile
│   └── docker-compose.yml
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Быстрый старт

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

### 3. Запуск

```bash
go run cmd/quiz-bot/main.go
```

Или соберите бинарник:

```bash
go build -o quiz-bot cmd/quiz-bot/main.go
./quiz-bot
```

### 4. Запуск через Docker

```bash
docker-compose up -d
```

## Команды бота

| Команда | Описание |
|---------|----------|
| `/start` | Главное меню |
| `/quiz` | Обычный вопрос викторины |
| `/interview` | Вопрос собеседования (Gopher/Go Offer) |
| `/score` | Статистика игрока |
| `/leaderboard` | Таблица лидеров |
| `/reset` | Сброс прогресса |
| `/help` | Справка |

## Версии

- **v0.2.0** — Добавлены вопросы собеседования (120 вопросов)
- **v0.1.0** — Базовая викторина (70 вопросов)

## Лицензия

MIT
