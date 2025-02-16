# Этап сборки
FROM golang:1.23.6-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git

WORKDIR /app

# Копируем только файлы, необходимые для сборки зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# Финальный этап
FROM alpine:3.18

WORKDIR /app

# Копируем собранное приложение
COPY --from=builder /app/main .
COPY app.config.env .
COPY migrations ./migrations

# Создаем непривилегированного пользователя
RUN adduser -D appuser
USER appuser

EXPOSE 8080

CMD ["./main"]