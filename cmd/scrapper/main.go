package main

import (
	"database/sql"
	"errors"
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

	storage, err := repository.NewSQLStorage(connString)
	if errors.Is(err, repository.ErrPoolCreate) {
		slog.Error(err.Error())

		return
	}

	conn, err := sql.Open("postgres", connString)
	if err != nil {
		slog.Error("Error opening connection",
			slog.String("err", err.Error()))

		return
	}

	defer func(conn *sql.DB) {
		err := conn.Close()
		if err != nil {
			slog.Error("Error closing file",
				slog.String("err", err.Error()))
		}
	}(conn)

	migrator := repository.MustGetNewMigrator()

	err = migrator.ApplyMigrations(conn)
	if err != nil {
		slog.Error("Migrations error %v", err.Error(),
			slog.String("err", err.Error()))

		return
	}

	slog.Info("Migrations applied!!")

	botClient := scrapper.NewBotClient("http", "bot:8090", "/updates")
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
