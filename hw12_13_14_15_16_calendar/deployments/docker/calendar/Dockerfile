# Этап сборки (используем свежий Alpine для Go)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Аргументы для версионирования
ARG RELEASE=dev
ARG BUILD_DATE
ARG GIT_HASH

# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version.Release=${RELEASE} \
    -X github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version.BuildDate=${BUILD_DATE} \
    -X github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version.GitHash=${GIT_HASH}" \
    -o bin/calendar ./cmd/calendar

# Финальный минимальный образ
FROM alpine:3.20

# Установка корневых сертификатов и таймзоны
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник и конфиг из builder-этапа
COPY --from=builder /app/bin/calendar .
COPY configs/calendar.yaml ./config.yaml

EXPOSE 8888 50051

ENTRYPOINT ["./calendar", "--config=./config.yaml"]
