# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src

# Кэшируем зависимости (docker layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Сборка статического бинарника
# -ldflags:
#   -s -w          — удаляем таблицу символов и DWARF-отладочную информацию
#   -extldflags "-static" — полностью статическая линковка (для scratch)
# CGO_ENABLED=1   — требуется для sqlite3 (матн)
RUN CGO_ENABLED=1 \
    GOOS=linux \
    go build -ldflags="-s -w -extldflags '-static'" \
    -o /dnso ./main.go

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM scratch

# CA-сертификаты для возможных HTTPS-вызовов
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Бинарник
COPY --from=builder /dnso /dnso

# Порт DNS (UDP) и веб-интерфейса (TCP)
EXPOSE 53/udp
EXPOSE 9000/tcp

# Точка входа
ENTRYPOINT ["/dnso"]
CMD ["serve"]