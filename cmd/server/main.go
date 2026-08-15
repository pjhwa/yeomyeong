// Command server boots the YEOMYEONG process: env config, game loop,
// SIGINT/SIGTERM shutdown. Net listeners are not started in this PR.
package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pjhwa/yeomyeong/internal/config"
	"github.com/pjhwa/yeomyeong/internal/engine"
	"github.com/pjhwa/yeomyeong/internal/persist"
)

func main() {
	cfg := config.Load()
	log := cfg.Logger()
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, log, cfg); err != nil {
		log.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger, cfg config.Config) error {
	store, err := persist.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	if c, ok := store.(io.Closer); ok {
		defer func() { _ = c.Close() }()
	}

	driver := "memory"
	if _, ok := store.(*persist.Postgres); ok {
		driver = "postgres"
	}
	log.Info("yeomyeong starting",
		"telnet", cfg.TelnetAddr,
		"ws", cfg.WSAddr,
		"database", cfg.DatabaseURL != "",
		"account_store", driver,
		"log_level", cfg.LogLevel.String(),
	)
	// internal/net is issue #4 / #12. The process must still boot and idle.
	// Auth hashing stays in persist; the loop never sees a password (D-012).
	log.Info("net listeners not started")

	loop := engine.New(log)
	loop.Run(ctx)
	log.Info("yeomyeong stopped")
	return nil
}
