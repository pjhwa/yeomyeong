// Package config loads process configuration from the environment (D-015).
package config

import (
	"log/slog"
	"os"
	"strings"
)

const (
	DefaultTelnetAddr = ":4000"
	DefaultWSAddr     = ":8080"
	DefaultLogLevel   = "info"

	EnvTelnetAddr = "YEOMYEONG_TELNET_ADDR"
	EnvWSAddr     = "YEOMYEONG_WS_ADDR"
	EnvDatabase   = "DATABASE_URL"
	EnvLogLevel   = "YEOMYEONG_LOG_LEVEL"
)

// Config is the process configuration. DATABASE_URL is stored but unused
// until persist lands; empty means the in-memory AccountStore (D-014).
type Config struct {
	TelnetAddr  string
	WSAddr      string
	DatabaseURL string
	LogLevel    slog.Level
}

// Load reads configuration from the environment, applying D-015 defaults.
func Load() Config {
	return Config{
		TelnetAddr:  envOr(EnvTelnetAddr, DefaultTelnetAddr),
		WSAddr:      envOr(EnvWSAddr, DefaultWSAddr),
		DatabaseURL: os.Getenv(EnvDatabase),
		LogLevel:    parseLevel(envOr(EnvLogLevel, DefaultLogLevel)),
	}
}

// Logger returns a JSON slog logger at the configured level.
func (c Config) Logger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: c.LogLevel}))
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
