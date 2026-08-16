package net

import (
	"errors"
	"syscall"
	"testing"
)

func TestNextTCPAddr(t *testing.T) {
	got, err := nextTCPAddr(":4001")
	if err != nil || got != ":4002" {
		t.Fatalf("next :4001 -> %q %v", got, err)
	}
	got, err = nextTCPAddr("127.0.0.1:4001")
	if err != nil || got != "127.0.0.1:4002" {
		t.Fatalf("next host -> %q %v", got, err)
	}
	if _, err := nextTCPAddr(":65535"); err == nil {
		t.Fatal("65535 must not increment")
	}
}

func TestListenTCPFallsBackWhenBusy(t *testing.T) {
	held, err := listenTCP("127.0.0.1:0", nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Close() }()
	busy := held.Addr().String()
	ln, err := listenTCP(busy, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	if ln.Addr().String() == busy {
		t.Fatal("fallback bound the busy address")
	}
}

func TestIsAddrInUse(t *testing.T) {
	if !isAddrInUse(syscall.EADDRINUSE) {
		t.Fatal("EADDRINUSE")
	}
	if isAddrInUse(errors.New("no such host")) {
		t.Fatal("unrelated error")
	}
}
