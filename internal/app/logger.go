package app

import (
	"log/slog"

	"github.com/urfave/cli/v2"
)

const LoggerMetadataKey = "logger"

type (
	LogFormat string //
	LogLevel  string //
)

// LoggerConfig holds logging configuration.
type LoggerConfig struct {
	LogLevel  LogLevel
	LogFormat LogFormat
}

type GetLoggerFunc func(*cli.Context) *slog.Logger

// GetLogger retrieves the logger from the CLI context metadata.
func GetLogger(c *cli.Context) *slog.Logger {
	if c.App != nil && c.App.Metadata != nil {
		if logger, ok := c.App.Metadata[LoggerMetadataKey].(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

var _ GetLoggerFunc = GetLogger
