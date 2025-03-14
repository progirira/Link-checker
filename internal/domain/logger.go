package domain

import (
	"fmt"
	"go-progira/lib/e"
	"log/slog"
	"os"
)

func SetNewLogger(filename string) (err error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("log file %w: %v", e.ErrOpenFile, filename)
	}

	logger := NewLogger(file)

	slog.SetDefault(logger.logger)

	return nil
}

type Logger struct {
	logger *slog.Logger
}

func NewLogger(out *os.File) *Logger {
	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	return &Logger{
		logger: slog.New(slog.NewJSONHandler(out, opts)),
	}
}
