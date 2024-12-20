package yamll

import (
	"log/slog"
	"os"
	"strings"
)

// SetLogger sets yamll logger with desired log level.
func (cfg *Config) SetLogger() {
	loggerOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     cfg.setLogLevel(),
	}

	stdLogger := slog.NewTextHandler(os.Stderr, loggerOpts)
	cfg.log = slog.New(stdLogger)
}

func (cfg *Config) setLogLevel() slog.Level {
	switch strings.ToLower(cfg.LogLevel) {
	case strings.ToLower(slog.LevelWarn.String()):
		return slog.LevelWarn
	case strings.ToLower(slog.LevelDebug.String()):
		return slog.LevelDebug
	case strings.ToLower(slog.LevelError.String()):
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLogger gets the initialised logger.
func (cfg *Config) GetLogger() *slog.Logger {
	return cfg.log
}
