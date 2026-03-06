FROM golang:1.21-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY cmd/ ./cmd/
COPY configs/ ./configs/

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o quiz-bot cmd/quiz-bot/main.go

# Финальный образ
FROM alpine:latest

WORKDIR /app

# Устанавливаем CA-сертификаты для HTTPS
RUN apk --no-cache add ca-certificates

# Копируем бинарник и конфиги
COPY --from=builder /app/quiz-bot .
COPY --from=builder /app/configs ./configs

# Копируем .env.example как шаблон
COPY .env.example .env.example

EXPOSE 8080

CMD ["./quiz-bot"]
