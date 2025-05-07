package main

import (
	"database/sql"
	"errors"
	"go-progira/internal/application/scrapper"
	repository "go-progira/internal/repository/sql_database"
	"go-progira/pkg"
	"go-progira/pkg/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func migrate(migrationsPath, connString string) error {
	conn, err := sql.Open("postgres", connString)
	if err != nil {
		slog.Error("Failed to connect to database",
			slog.String("error", err.Error()))
		return err
	}

	defer func() {
		errClose := conn.Close()
		if errClose != nil {
			slog.Error("Failed to close database",
				slog.String("error", errClose.Error()))

			if err != nil {
				err = errors.Join(errClose, err)
			} else {
				err = errClose
			}
		}
	}()

	migrator := repository.Migrator{MigrationsPath: migrationsPath}

	err = migrator.ApplyMigrations(conn)
	if err != nil {
		slog.Error("Failed to apply migrations",
			slog.String("error", err.Error()))

		return err
	}

	return err
}

func main() {
	pkg.SetNewStdoutLogger()

	appConfig, errLoadEnv := config.LoadConfig(".env")
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

	err = migrate(appConfig.MigrationsPath, appConfig.DatabaseURL)
	if err != nil {
		return
	}

	slog.Info("Migrations successfully applied")

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
