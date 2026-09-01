package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/config"
	"github.com/pjhwa/yeomyeong/internal/content"
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
	if !bytes.Contains([]byte(got), []byte("no content")) {
		t.Fatalf("want no content log, got %q", got)
	}
}

func TestRunFailsOnInvalidZones(t *testing.T) {
	dir := t.TempDir()
	zone := filepath.Join(dir, "content", "zones", "dalbitgol")
	if err := os.MkdirAll(zone, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(zone, "rooms.yaml"), []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv(config.EnvDatabase, "")
	t.Setenv(config.EnvTelnetAddr, "127.0.0.1:0")
	t.Setenv(config.EnvWSAddr, "127.0.0.1:0")
	err := run(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), config.Load())
	if !errors.Is(err, content.ErrSpawnMissing) {
		t.Fatalf("got %v", err)
	}
}

func TestLoadWorldZonesNotDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "zones"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := loadWorld(log, dir); err == nil {
		t.Fatal("want error when zones is a file")
	}
}

func TestRunFailsOnInvalidSkills(t *testing.T) {
	dir := t.TempDir()
	skills := filepath.Join(dir, "content", "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skills, "m2.yaml"), []byte("skills: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv(config.EnvDatabase, "")
	t.Setenv(config.EnvTelnetAddr, "127.0.0.1:0")
	t.Setenv(config.EnvWSAddr, "127.0.0.1:0")
	err := run(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), config.Load())
	if err == nil {
		t.Fatal("want skill load error")
	}
}

func TestLoadSkillsNotDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "skills"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := loadSkills(log, dir); err == nil {
		t.Fatal("want error when skills is a file")
	}
	if cat, err := loadSkills(log, t.TempDir()); err != nil || cat != nil {
		t.Fatalf("missing: %v %v", cat, err)
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
