package config

import (
	"log/slog"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv(EnvTelnetAddr, "")
	t.Setenv(EnvWSAddr, "")
	t.Setenv(EnvDatabase, "")
	t.Setenv(EnvLogLevel, "")
	cfg := Load()
	if cfg.TelnetAddr != DefaultTelnetAddr || cfg.WSAddr != DefaultWSAddr || cfg.DatabaseURL != "" || cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("defaults: %+v", cfg)
	}

	t.Setenv(EnvTelnetAddr, "127.0.0.1:4001")
	t.Setenv(EnvWSAddr, "127.0.0.1:8081")
	t.Setenv(EnvDatabase, "postgres://localhost/yeomyeong")
	t.Setenv(EnvLogLevel, "debug")
	cfg = Load()
	if cfg.TelnetAddr != "127.0.0.1:4001" || cfg.WSAddr != "127.0.0.1:8081" || cfg.DatabaseURL == "" || cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("overrides: %+v", cfg)
	}
	if cfg.Logger() == nil {
		t.Fatal("Logger() returned nil")
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":    slog.LevelDebug,
		"INFO":     slog.LevelInfo,
		"warn":     slog.LevelWarn,
		"warning":  slog.LevelWarn,
		"error":    slog.LevelError,
		"nope":     slog.LevelInfo,
		"  DEBUG ": slog.LevelDebug,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q)=%v want %v", in, got, want)
		}
	}
}
