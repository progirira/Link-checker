package main

import (
	"database/sql"
	"errors"
	"fmt"
	"go-progira/internal/application/scrapper"
	repository "go-progira/internal/repository/sql_database"
	"go-progira/pkg"
	"go-progira/pkg/config"
	"go-progira/pkg/e"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
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

	envData, errLoadEnv := config.Set(".env")
	if errLoadEnv != nil {
		return
	}

	connString, err := envData.GetByKeyFromEnv("DATABASE_URL")
	if errors.Is(err, e.ErrNoValInEnv) {
		slog.Error(err.Error())

		return
	}

	linkServiceType, err := envData.GetByKeyFromEnv("LINK_SERVICE")
	if errors.Is(err, e.ErrNoValInEnv) {
		slog.Error(err.Error())

		return
	}

	storage, err := repository.NewLinkService(linkServiceType, connString)
	if err != nil {
		slog.Error(err.Error())

		return
	}

	migrationsPath, err := envData.GetByKeyFromEnv("MIGRATIONS_PATH")
	if errors.Is(err, e.ErrNoValInEnv) {
		slog.Error(err.Error())

		return
	}

	err = migrate(migrationsPath, connString)
	if err != nil {
		return
	}

	slog.Info("Migrations successfully applied")

	botHost, err := envData.GetByKeyFromEnv("BOT_HOST")
	if errors.Is(err, e.ErrNoValInEnv) {
		fmt.Println(err.Error())
		return
	}

	botClient := scrapper.NewBotClient("http", botHost, "/updates")
	scr := scrapper.NewServer(storage, botClient)

	batchStr, errLoad := envData.GetByKeyFromEnv("BATCH")
	if errLoad != nil {
		return
	}

	batch, _ := strconv.Atoi(batchStr)

	slog.Info("Going to start scrapper server",
		slog.Int("Batch", batch))
	scr.Start(batch)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	slog.Info("Shutting down...")
}
