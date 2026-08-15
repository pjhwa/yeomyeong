package main

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/config"
)

func TestRunIdlesAndStops(t *testing.T) {
	t.Setenv(config.EnvDatabase, "")
	t.Setenv(config.EnvTelnetAddr, "127.0.0.1:0")
	t.Setenv(config.EnvWSAddr, "127.0.0.1:0")
	var buf syncBuf
	log := slog.New(slog.NewTextHandler(&buf, nil))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, log, config.Load()) }()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got := buf.String()
		if bytes.Contains([]byte(got), []byte("telnet listening")) && bytes.Contains([]byte(got), []byte("ws listening")) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not return after cancel")
	}
	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("telnet listening")) {
		t.Fatalf("want telnet listening log, got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("ws listening")) {
		t.Fatalf("want ws listening log, got %q", got)
	}
}

// syncBuf is a bytes.Buffer safe for concurrent slog writes and test reads.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}
