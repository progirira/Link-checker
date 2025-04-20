package config

import (
	"bufio"
	"errors"
	"go-progira/pkg/e"
	"log/slog"
	"os"
	"strings"
)

type Env struct{}

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
