package config

import (
	"errors"
	"fmt"
	"go-progira/pkg/e"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	TgAPIToken          string
	StackoverflowAPIKey string
	GithubAPIKey        string
	TgBotHost           string
	BotHost             string
	ScrapperHost        string
	DatabaseURL         string
	LinkService         string
	Batch               int
	Workers             int
}

func LoadConfig() (Config, error) {
	var errs []string

	if errLoadEnv := godotenv.Load(); errLoadEnv != nil {
		slog.Error(errLoadEnv.Error(),
			slog.String("error", errLoadEnv.Error()))

		return Config{}, errLoadEnv
	}

	get := func(key string) string {
		val := os.Getenv(key)
		if val == "" {
			errs = append(errs, fmt.Sprintf("missing env: %s", key))
		}

		return val
	}

	batchStr := os.Getenv("BATCH")
	if batchStr == "" {
		errs = append(errs, "missing env: BATCH")
	}

	workersStr := os.Getenv("NUMBER_OF_WORKERS")
	if workersStr == "" {
		errs = append(errs, "missing env: NUMBER_OF_WORKERS")
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("config errors:\n%s", strings.Join(errs, "\n"))
	}

	batch, err := strconv.Atoi(batchStr)
	if errors.Is(err, e.ErrNoValInEnv) {
		slog.Error(err.Error())

		return Config{}, fmt.Errorf("cannot convert string BATCH to int")
	}

	numOfWorkers, err := strconv.Atoi(workersStr)
	if errors.Is(err, e.ErrNoValInEnv) {
		slog.Error(err.Error())

		return Config{}, fmt.Errorf("cannot convert string NUMBER_OF_WORKERS to int")
	}

	config := Config{
		TgAPIToken:          get("TELEGRAM_BOT_API_TOKEN"),
		StackoverflowAPIKey: get("STACKOVERFLOW_API_KEY"),
		GithubAPIKey:        get("GITHUB_API_KEY"),
		TgBotHost:           get("TELEGRAM_BOT_HOST"),
		BotHost:             get("BOT_HOST"),
		ScrapperHost:        get("SCRAPPER_HOST"),
		DatabaseURL:         get("DATABASE_URL"),
		LinkService:         get("LINK_SERVICE"),
		Batch:               batch,
		Workers:             numOfWorkers,
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("config errors:\n%s", strings.Join(errs, "\n"))
	}

	return config, nil
}
