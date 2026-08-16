package net

import (
	"errors"
	"fmt"
	"log/slog"
	stdnet "net"
	"strconv"
	"strings"
	"syscall"
)

const bindRetries = 10

// listenTCP binds addr. If that port is already in use it tries the next
// ports (up to bindRetries) so a local OrbStack/Docker occupant of :4000
// does not prevent `go run ./cmd/server`.
func listenTCP(addr string, log *slog.Logger, kind string) (stdnet.Listener, error) {
	if log == nil {
		log = slog.Default()
	}
	cur := addr
	var last error
	for i := 0; i < bindRetries; i++ {
		ln, err := stdnet.Listen("tcp", cur)
		if err == nil {
			if cur != addr {
				log.Warn(kind+" requested address in use, bound the next free port",
					"wanted", addr, "bound", ln.Addr().String())
			}
			return ln, nil
		}
		last = err
		if !isAddrInUse(err) {
			return nil, err
		}
		next, err := nextTCPAddr(cur)
		if err != nil {
			return nil, fmt.Errorf("%w (cannot increment %q)", last, cur)
		}
		cur = next
	}
	return nil, fmt.Errorf("listen %s %s: %w", kind, addr, last)
}

func isAddrInUse(err error) bool {
	var op *stdnet.OpError
	if errors.As(err, &op) {
		err = op.Err
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	return strings.Contains(err.Error(), "address already in use")
}

func nextTCPAddr(addr string) (string, error) {
	host, port, err := stdnet.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return "", err
	}
	if n <= 0 || n >= 65535 {
		return "", fmt.Errorf("port %d has no successor", n)
	}
	return stdnet.JoinHostPort(host, strconv.Itoa(n+1)), nil
}
