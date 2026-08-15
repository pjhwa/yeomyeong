// Command server boots the YEOMYEONG process: env config, game loop,
// Telnet listener, SIGINT/SIGTERM shutdown. WebSocket is issue #12.
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
	ynet "github.com/pjhwa/yeomyeong/internal/net"
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	loop := engine.New(log)
	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		loop.Run(ctx)
	}()

	srv := ynet.NewServer(cfg.TelnetAddr, loop, store, log)
	err = srv.Serve(ctx)
	cancel()
	<-loopDone
	if err != nil {
		return err
	}
	log.Info("yeomyeong stopped")
	return nil
}
