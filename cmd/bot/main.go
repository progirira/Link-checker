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
	"time"
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

func token() (string, error) {
	token, exists := os.LookupEnv("TELEGRAM_BOT_API_TOKEN")
	if !exists {
		log.Print("No TELEGRAM_BOT_API_TOKEN in .env file found")
		return token, ErrNoToken
	}

	return token, nil
}

func botHost() (string, error) {
	host, exists := os.LookupEnv("BOT_HOST")
	if !exists {
		log.Print("No BOT_HOST in .env file found")
		return host, ErrNoBotHost
	}
	return host, nil
}

func main() {

	err := loadEnv(".env")
	if err != nil {
		log.Print("No .env file found")
		return
	}

	token, err := token()
	if errors.Is(err, ErrNoToken) {
		fmt.Println(err.Error())
		return
	}

	host, err := botHost()
	if errors.Is(err, ErrNoToken) {
		fmt.Println(err.Error())
		return
	}

	tgClient := clients.NewTelegramClient(host, token)
	baseURL := ""
	scrapClient := clients.NewScrapperClient(3*time.Second, baseURL)

	botServer := bot.Server{
		TgClient: tgClient,
	}
	botServer.Start()

	manager := processing.NewManager(tgClient, scrapClient)
	manager.Start()
}
