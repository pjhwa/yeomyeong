// Command server boots the YEOMYEONG process: env config, game loop,
// Telnet + WebSocket listeners, SIGINT/SIGTERM shutdown.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pjhwa/yeomyeong/internal/config"
	"github.com/pjhwa/yeomyeong/internal/content"
	"github.com/pjhwa/yeomyeong/internal/engine"
	ynet "github.com/pjhwa/yeomyeong/internal/net"
	"github.com/pjhwa/yeomyeong/internal/persist"
	"github.com/pjhwa/yeomyeong/internal/world"
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

const contentRoot = "content"

func run(ctx context.Context, log *slog.Logger, cfg config.Config) error {
	store, err := persist.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	if c, ok := store.(io.Closer); ok {
		defer func() { _ = c.Close() }()
	}

	if _, err := loadWorld(log, contentRoot); err != nil {
		return err
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

	err = serveListeners(ctx, log, cfg, loop, store)
	cancel()
	<-loopDone
	if err != nil {
		return err
	}
	log.Info("yeomyeong stopped")
	return nil
}

// serveListeners runs Telnet and WebSocket until ctx is cancelled.
// A bind failure on either side cancels the other. Both enqueue the
// same engine.Command values; neither mutates the roster.
func serveListeners(ctx context.Context, log *slog.Logger, cfg config.Config, loop *engine.Loop, store persist.AccountStore) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	telnet := ynet.NewServer(cfg.TelnetAddr, loop, store, log)
	ws := ynet.NewWS(cfg.WSAddr, loop, store, log)
	errCh := make(chan error, 2)
	go func() { errCh <- telnet.Serve(ctx) }()
	go func() { errCh <- ws.Serve(ctx) }()

	var first error
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil && first == nil {
			first = err
			cancel()
		}
	}
	return first
}

// loadWorld loads content/zones if that directory exists. Missing zones is
// not a boot failure (M1 rooms land later). Invalid zones are fatal.
// Real-tree spawn is dalbitgol:gate (D-028).
func loadWorld(log *slog.Logger, root string) (*world.Catalog, error) {
	zones := filepath.Join(root, "zones")
	st, err := os.Stat(zones)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("no content")
			return nil, nil
		}
		return nil, fmt.Errorf("stat content zones: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("content zones %s is not a directory", zones)
	}
	cat, err := content.Load(root, world.SpawnID)
	if err != nil {
		return nil, fmt.Errorf("load content: %w", err)
	}
	log.Info("content loaded", "rooms", cat.Len(), "spawn", cat.Spawn())
	return cat, nil
}
