.PHONY: build run test clean docker-build docker-up docker-down

# Переменные
BINARY_NAME=quiz-bot
CMD_PATH=cmd/quiz-bot/main.go
BUILD_DIR=.

# Сборка бинарника
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

# Запуск бота
run:
	go run $(CMD_PATH)

# Тесты
test:
	go test -v ./...

# Очистка
clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	go clean

# Сборка Docker-образа
docker-build:
	docker build -f deploy/Dockerfile -t $(BINARY_NAME) .

# Запуск Docker
docker-up:
	docker-compose -f deploy/docker-compose.yml up -d

# Остановка Docker
docker-down:
	docker-compose -f deploy/docker-compose.yml down

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  make build       - Сборка бинарника"
	@echo "  make run         - Запуск бота"
	@echo "  make test        - Запуск тестов"
	@echo "  make clean       - Очистка"
	@echo "  make docker-build - Сборка Docker-образа"
	@echo "  make docker-up   - Запуск Docker"
	@echo "  make docker-down - Остановка Docker"
