package main

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/config"
)

func TestRunIdlesAndStops(t *testing.T) {
	t.Setenv(config.EnvDatabase, "")
	t.Setenv(config.EnvTelnetAddr, "127.0.0.1:0")
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, log, config.Load()) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not return after cancel")
	}
	if !bytes.Contains(buf.Bytes(), []byte("telnet listening")) {
		t.Fatalf("want telnet listening log, got %q", buf.String())
	}
}
