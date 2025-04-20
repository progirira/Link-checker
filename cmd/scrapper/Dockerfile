FROM golang:1.23 AS builder

WORKDIR /app

COPY . .

RUN go build -o scrapper ./cmd/scrapper/main.go

CMD ["./scrapper"]
