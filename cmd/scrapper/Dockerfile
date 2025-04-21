FROM golang:1.23 AS builder

WORKDIR /app

COPY . .

COPY migrations /app/migrations

RUN go build -o scrapper ./cmd/scrapper/main.go

CMD ["./scrapper"]
