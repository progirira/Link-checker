package main

import (
	"go-progira/internal/application/scrapper"
	repository "go-progira/internal/repository/sql_database"
	"go-progira/pkg"
	"go-progira/pkg/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	pkg.SetNewStdoutLogger()

	appConfig, errLoadEnv := config.LoadConfig()
	if errLoadEnv != nil {
		slog.Error(errLoadEnv.Error(),
			slog.String("error", errLoadEnv.Error()))
		return
	}

	storage, err := repository.NewLinkService(appConfig.LinkService, appConfig.DatabaseURL)
	if err != nil {
		slog.Error(err.Error())

		return
	}

	botClient := scrapper.NewBotClient("http", appConfig.BotHost, "/updates")
	scr := scrapper.NewServer(storage, botClient)

	slog.Info("Going to start scrapper server",
		slog.Int("Batch", appConfig.Batch))
	scr.Start(&appConfig)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	slog.Info("Shutting down...")
}
