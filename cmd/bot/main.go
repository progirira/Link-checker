package main

import (
	"bufio"
	"errors"
	"fmt"
	"go-progira/internal/application/bot/clients"
	"go-progira/internal/application/bot/processing"
	"go-progira/internal/domain"
	"go-progira/lib/e"
	"log/slog"
	"os"
	"strings"
)

func loadEnv(filename string) (err error) {
	file, openErr := os.Open(filename)
	if openErr != nil {
		slog.Error(
			e.ErrOpenFile.Error(),
			slog.String("error", openErr.Error()),
			slog.String("filename", filename),
		)

		return e.ErrOpenFile
	}

	defer func(file *os.File) {
		if closeErr := file.Close(); closeErr != nil {
			if err == nil { // если других ошибок не было
				slog.Error(
					e.ErrCloseFile.Error(),
					slog.String("error", closeErr.Error()),
					slog.String("filename", filename),
				)

				err = e.ErrCloseFile
			}
		}
	}(file)

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

		errSetEnv := os.Setenv(key, value)
		if errSetEnv != nil {
			slog.Error(
				e.ErrOsSetEnv.Error(),
				slog.String("error", errSetEnv.Error()),
				slog.String("filename", filename),
				slog.String("key", key),
				slog.String("value", value),
			)

			return e.ErrOsSetEnv
		}
	}

	if scanner.Err() != nil {
		slog.Error(
			e.ErrScanFile.Error(),
			slog.String("error", scanner.Err().Error()),
			slog.String("filename", filename),
		)

		return e.ErrScanFile
	}

	return err
}

func getByKeyFromEnv(key string) (string, error) {
	val, exists := os.LookupEnv(key)
	if !exists {
		slog.Error(
			e.ErrNoValInEnv.Error(),
			slog.String("key", key),
		)

		return "", e.ErrNoValInEnv
	}

	return val, nil
}

func main() {
	fileForLogs := "logs/bot_logs"

	loggerErr := domain.SetNewLogger(fileForLogs)
	if errors.Is(loggerErr, e.ErrOpenFile) {
		fmt.Println(loggerErr.Error())

		return
	}

	errLoadEnv := loadEnv(".env")
	if errLoadEnv != nil {
		return
	}

	token, err := getByKeyFromEnv("TELEGRAM_BOT_API_TOKEN")
	if errors.Is(err, e.ErrNoValInEnv) {
		fmt.Println(err.Error())

		return
	}

	host, err := getByKeyFromEnv("BOT_HOST")
	if errors.Is(err, e.ErrNoValInEnv) {
		fmt.Println(err.Error())

		return
	}

	tgClient := clients.NewTelegramClient("https", host, token)

	host = "localhost:8090"
	scrapClient := clients.NewScrapperClient("http", host)

	server := processing.NewServer(&tgClient)
	server.Start()

	manager := processing.NewManager(&tgClient, &scrapClient)
	manager.Start()
}
