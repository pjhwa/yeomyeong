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
	"github.com/pjhwa/yeomyeong/internal/skill"
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

	cat, items, ground, err := loadWorld(log, contentRoot)
	if err != nil {
		return err
	}
	skills, err := loadSkills(log, contentRoot)
	if err != nil {
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

	saver := persist.NewAsyncSaver(store, log)
	defer saver.Close()

	loop := engine.NewWithWorld(log, cat, items, ground, saver).WithSkills(skills)
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

// loadWorld loads rooms, optional items, and optional spawns. Missing zones is
// not a boot failure. Invalid content is fatal. Real-tree spawn is
// dalbitgol:gate (D-028). Skills load separately (loadSkills).
func loadWorld(log *slog.Logger, root string) (*world.Catalog, *world.Items, map[string][]world.Stack, error) {
	zones := filepath.Join(root, "zones")
	st, err := os.Stat(zones)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("no content")
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("stat content zones: %w", err)
	}
	if !st.IsDir() {
		return nil, nil, nil, fmt.Errorf("content zones %s is not a directory", zones)
	}
	w, err := content.LoadWorld(root, world.SpawnID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load content: %w", err)
	}
	log.Info("content loaded", "rooms", w.Rooms.Len(), "items", w.Items.Len(), "spawn", w.Rooms.Spawn())
	return w.Rooms, w.Items, w.Ground, nil
}

var _ engine.SheetSink = (*persist.AsyncSaver)(nil)

// loadSkills loads content/skills if that directory exists. Missing skills is
// not a boot failure. Invalid YAML is fatal. The catalog is attached to the loop.
func loadSkills(log *slog.Logger, root string) (*skill.Catalog, error) {
	dir := filepath.Join(root, "skills")
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat content skills: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("content skills %s is not a directory", dir)
	}
	cat, err := skill.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}
	log.Info("skills loaded", "skills", cat.Len())
	return cat, nil
}
