# Этап сборки
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o gault ./cmd/server/

# Этап создания обараза
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gault .
COPY server_config.yml .
COPY db/schema.sql db/schema.sql
COPY db/migrations db/migrations
COPY certs/ca.crt certs/ca.crt
COPY certs/ca.key certs/ca.key
COPY certs/cert.conf certs/cert.conf
COPY certs/server.crt certs/server.crt
COPY certs/server.key certs/server.key

RUN chmod +x gault

CMD ["./gault"]
