FROM golang:1.23 AS builder

WORKDIR /app

COPY . .

RUN go build -o bot ./cmd/bot/main.go

CMD ["./bot"]
