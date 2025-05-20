package main

import (
	"go-progira/internal/application/bot/clients"
	"go-progira/internal/application/bot/processing"
	"go-progira/pkg"
	"go-progira/pkg/config"
	"log/slog"
)

func main() {
	pkg.SetNewStdoutLogger()

	appConfig, errLoadEnv := config.LoadConfig()
	if errLoadEnv != nil {
		slog.Error(errLoadEnv.Error(),
			slog.String("error", errLoadEnv.Error()))
		return
	}

	tgClient := clients.NewTelegramClient("https", appConfig.TgBotHost, appConfig.TgAPIToken)
	slog.Info("Telegram client created",
		slog.String("host", appConfig.TgBotHost))

	scrapClient := clients.NewScrapperClient("http", appConfig.ScrapperHost)
	slog.Info("Scrapper client created",
		slog.String("host", appConfig.ScrapperHost))

	server := processing.NewServer(&tgClient)

	slog.Info("Bot server created")

	server.Start(&appConfig)

	manager := processing.NewManager(&tgClient, &scrapClient)

	slog.Info("Manager created")

	manager.Start()
}
