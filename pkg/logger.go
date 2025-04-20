package pkg

import (
	"log/slog"
	"os"
)

type Logger struct {
	logger *slog.Logger
}

func SetNewStdoutLogger() {
	logger := NewLogger(os.Stdout)

	slog.SetDefault(logger.logger)
}

func NewLogger(out *os.File) *Logger {
	opts := &slog.HandlerOptions{
		AddSource: true,
	}

	return &Logger{
		logger: slog.New(slog.NewJSONHandler(out, opts)),
	}
}
