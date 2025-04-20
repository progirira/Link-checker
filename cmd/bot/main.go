package main

import (
	"errors"
	"fmt"
	"go-progira/internal/application/bot/clients"
	"go-progira/internal/application/bot/processing"
	"go-progira/pkg"
	"go-progira/pkg/config"
	"go-progira/pkg/e"
	"log/slog"
)

func main() {
	pkg.SetNewStdoutLogger()

	envData, errLoadEnv := config.Set(".env")
	if errLoadEnv != nil {
		return
	}

	token, err := envData.GetByKeyFromEnv("TELEGRAM_BOT_API_TOKEN")
	if errors.Is(err, e.ErrNoValInEnv) {
		fmt.Println(err.Error())
		return
	}

	tgHost, err := envData.GetByKeyFromEnv("TELEGRAM_BOT_HOST")
	if errors.Is(err, e.ErrNoValInEnv) {
		fmt.Println(err.Error())

		return
	}

	tgClient := clients.NewTelegramClient("https", tgHost, token)
	slog.Info("Telegram client created",
		slog.String("host", tgHost))

	scrapperHost, err := envData.GetByKeyFromEnv("SCRAPPER_HOST")
	if errors.Is(err, e.ErrNoValInEnv) {
		fmt.Println(err.Error())
		return
	}

	scrapClient := clients.NewScrapperClient("http", scrapperHost)
	slog.Info("Scrapper client created",
		slog.String("host", scrapperHost))

	server := processing.NewServer(&tgClient)

	slog.Info("Bot server created")

	server.Start()

	manager := processing.NewManager(&tgClient, &scrapClient)

	slog.Info("Manager created")

	manager.Start()
}
