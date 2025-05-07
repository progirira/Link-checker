package config

import (
	"bufio"
	"errors"
	"fmt"
	"go-progira/pkg/e"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TgAPIToken          string
	StackoverflowAPIKey string
	TgBotHost           string
	BotHost             string
	ScrapperHost        string
	DatabaseURL         string
	LinkService         string
	MigrationsPath      string
	Batch               int
}

type Env struct{}

func LoadConfig(filename string) (Config, error) {
	var errs []string

	envData, errLoadEnv := Set(filename)
	if errLoadEnv != nil {
		return Config{}, errLoadEnv
	}

	get := func(key string) string {
		val, _ := envData.GetByKeyFromEnv(key)
		if val == "" {
			errs = append(errs, fmt.Sprintf("missing env: %s", key))
		}

		return val
	}

	batchStr, errLoad := envData.GetByKeyFromEnv("BATCH")
	if errLoad != nil {
		errs = append(errs, "missing env: BATCH")
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("config errors:\n%s", strings.Join(errs, "\n"))
	}

	batch, err := strconv.Atoi(batchStr)
	if errors.Is(err, e.ErrNoValInEnv) {
		slog.Error(err.Error())

		return Config{}, fmt.Errorf("cannot convert string BATCH to int")
	}

	return Config{
		TgAPIToken:          get("TELEGRAM_BOT_API_TOKEN"),
		StackoverflowAPIKey: get("STACKOVERFLOW_API_KEY"),
		TgBotHost:           get("TELEGRAM_BOT_HOST"),
		BotHost:             get("BOT_HOST"),
		ScrapperHost:        get("SCRAPPER_HOST"),
		DatabaseURL:         get("DATABASE_URL"),
		LinkService:         get("LINK_SERVICE"),
		MigrationsPath:      get("MIGRATIONS_PATH"),
		Batch:               batch,
	}, nil
}

func Set(filename string) (*Env, error) {
	errLoad := loadEnv(filename)
	if errLoad != nil {
		return nil, errors.New("error loading data")
	}

	return &Env{}, nil
}

func (env *Env) GetByKeyFromEnv(key string) (string, error) {
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
