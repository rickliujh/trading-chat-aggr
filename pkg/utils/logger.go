package utils

import (
	"log/slog"
	"os"

	"github.com/go-logr/logr"
)

func NewLogger(level int) *logr.Logger {
	logger := logr.FromSlogHandler(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			AddSource: true,
			Level: slog.Level(0-level),
		},
	))
	return &logger
}
