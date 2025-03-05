package main

import (
	"bufio"
	"errors"
	"fmt"
	"go-progira/internal/application/bot"
	"go-progira/internal/application/bot/clients"
	"go-progira/internal/application/bot/processing"
	"log"
	"os"
	"strings"
)

func loadEnv(filename string) error {
	file, err := os.Open(filename)

	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)

		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		os.Setenv(key, value)
	}

	return scanner.Err()
}

func getByKeyFromEnv(key string) (string, error) {
	val, exists := os.LookupEnv(key)
	if !exists {
		log.Printf("No %s in .env file found", key)
		return val, ErrNoVal
	}

	return val, nil
}

func main() {
	err := loadEnv(".env")
	if err != nil {
		log.Print("No .env file found")
		return
	}

	token, err := getByKeyFromEnv("TELEGRAM_BOT_API_TOKEN")
	if errors.Is(err, ErrNoVal) {
		fmt.Println(err.Error())
		return
	}

	host, err := getByKeyFromEnv("BOT_HOST")
	if errors.Is(err, ErrNoVal) {
		fmt.Println(err.Error())
		return
	}

	tgClient := clients.NewTelegramClient("https", host, token)

	host = "localhost:8090"
	scrapClient := clients.NewScrapperClient("http", host)

	server := bot.NewServer(&tgClient)
	server.Start()

	manager := processing.NewManager(&tgClient, scrapClient)
	manager.Start()
}
