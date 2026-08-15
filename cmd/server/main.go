// Command server boots the YEOMYEONG process: env config, game loop,
// SIGINT/SIGTERM shutdown. Net listeners are not started in this PR.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pjhwa/yeomyeong/internal/config"
	"github.com/pjhwa/yeomyeong/internal/engine"
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
	log.Info("yeomyeong starting",
		"telnet", cfg.TelnetAddr,
		"ws", cfg.WSAddr,
		"database", cfg.DatabaseURL != "",
		"log_level", cfg.LogLevel.String(),
	)
	// internal/net is issue #4 / #12. The process must still boot and idle.
	log.Info("net listeners not started")

	loop := engine.New(log)
	loop.Run(ctx)
	log.Info("yeomyeong stopped")
	return nil
}
