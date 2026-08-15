package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	stdnet "net"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pjhwa/yeomyeong/internal/engine"
	ynet "github.com/pjhwa/yeomyeong/internal/net"
	"github.com/pjhwa/yeomyeong/internal/persist"
	"github.com/pjhwa/yeomyeong/internal/world"
)

const (
	wsPath     = "/ws"
	protoV     = 1
	typeCreate = "auth.create"
	typeOK     = "auth.ok"
	typeSay    = "cmd.say"
	typeText   = "text"
	channelSay = "say"
	readWait   = 15 * time.Second
	ioWait     = 2 * time.Second
)

// transcript is a send/recv log dumped on every assertion failure so the
// output is a reproduction, not just "not equal".
type transcript struct {
	mu    sync.Mutex
	lines []string
}

func (tr *transcript) add(format string, args ...any) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.lines = append(tr.lines, time.Now().Format("15:04:05.000")+" "+fmt.Sprintf(format, args...))
}

func (tr *transcript) String() string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.lines) == 0 {
		return "(empty — no bytes sent or received)"
	}
	return strings.Join(tr.lines, "\n")
}

func formatFail(tr *transcript, msg string) string {
	return msg + "\n--- reproduction transcript ---\n" + tr.String()
}

type harness struct {
	telnetAddr string
	wsAddr     string
	loop       *engine.Loop
	tr         *transcript
}

func (h *harness) failf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Fatal(formatFail(h.tr, fmt.Sprintf(format, args...)))
}

func startHarness(t *testing.T) *harness {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat, err := world.NewCatalog([]world.Room{{
		ID: world.SpawnID, Name: world.Localized{KO: "문"},
		Description: world.Localized{KO: "문."},
	}}, world.SpawnID)
	if err != nil {
		t.Fatal(err)
	}
	loop := engine.NewWithCatalog(log, cat)
	store := persist.NewMemory()
	ctx, cancel := context.WithCancel(context.Background())

	telnetAddr := reserveAddr(t)
	wsAddr := reserveAddr(t)
	telnet := ynet.NewServer(telnetAddr, loop, store, log)
	ws := ynet.NewWS(wsAddr, loop, store, log)

	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		loop.Run(ctx)
	}()
	telnetErr := make(chan error, 1)
	wsErr := make(chan error, 1)
	go func() { telnetErr <- telnet.Serve(ctx) }()
	go func() { wsErr <- ws.Serve(ctx) }()

	waitTCP(t, telnetAddr)
	waitTCP(t, wsAddr)
	select {
	case err := <-telnetErr:
		cancel()
		t.Fatalf("telnet serve: %v", err)
	case err := <-wsErr:
		cancel()
		t.Fatalf("ws serve: %v", err)
	default:
	}

	h := &harness{telnetAddr: telnetAddr, wsAddr: wsAddr, loop: loop, tr: &transcript{}}
	t.Cleanup(func() {
		cancel()
		select {
		case <-telnetErr:
		case <-time.After(ioWait):
			t.Error("telnet serve did not stop")
		}
		select {
		case <-wsErr:
		case <-time.After(ioWait):
			t.Error("ws serve did not stop")
		}
		select {
		case <-loopDone:
		case <-time.After(ioWait):
			t.Error("loop did not stop")
		}
	})
	return h
}

func reserveAddr(t *testing.T) string {
	t.Helper()
	ln, err := stdnet.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return addr
}

func waitTCP(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(ioWait)
	var last error
	for time.Now().Before(deadline) {
		c, err := stdnet.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return
		}
		last = err
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("listen %s: %v", addr, last)
}

func (h *harness) waitRoster(t *testing.T, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last engine.Snapshot
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		snap, err := h.loop.Snapshot(ctx)
		cancel()
		if err == nil && len(snap.Players) == n {
			h.tr.add("SYS roster=%d", n)
			return
		}
		if err == nil {
			last = snap
		}
		time.Sleep(20 * time.Millisecond)
	}
	h.failf(t, "roster want %d got %d (%+v)", n, len(last.Players), last.Players)
}

type telnetClient struct {
	name string
	conn stdnet.Conn
	buf  string
	h    *harness
}

func (h *harness) dialTelnet(t *testing.T, name string) *telnetClient {
	t.Helper()
	c, err := stdnet.DialTimeout("tcp", h.telnetAddr, ioWait)
	if err != nil {
		h.failf(t, "%s dial telnet %s: %v", name, h.telnetAddr, err)
	}
	t.Cleanup(func() { _ = c.Close() })
	h.tr.add("%s DIAL telnet %s", name, h.telnetAddr)
	return &telnetClient{name: name, conn: c, h: h}
}

func (c *telnetClient) send(t *testing.T, line string) {
	t.Helper()
	c.h.tr.add("%s SEND %q", c.name, line)
	_ = c.conn.SetWriteDeadline(time.Now().Add(ioWait))
	if _, err := io.WriteString(c.conn, line+"\n"); err != nil {
		c.h.failf(t, "%s send %q: %v", c.name, line, err)
	}
}

func (c *telnetClient) readUntil(t *testing.T, needle string) string {
	t.Helper()
	deadline := time.Now().Add(readWait)
	_ = c.conn.SetReadDeadline(deadline)
	tmp := make([]byte, 256)
	var collected strings.Builder
	for {
		if i := strings.Index(c.buf, needle); i >= 0 {
			end := i + len(needle)
			collected.WriteString(c.buf[:end])
			c.buf = c.buf[end:]
			got := collected.String()
			c.h.tr.add("%s RECV %q", c.name, got)
			return got
		}
		collected.WriteString(c.buf)
		c.buf = ""
		n, err := c.conn.Read(tmp)
		if n > 0 {
			c.buf += string(tmp[:n])
			continue
		}
		if err != nil {
			c.h.tr.add("%s RECV %q (error %v, waiting for %q)", c.name, collected.String(), err, needle)
			c.h.failf(t, "%s readUntil %q: %v", c.name, needle, err)
		}
	}
}

func (c *telnetClient) expectLine(t *testing.T, want string) {
	t.Helper()
	got := c.readUntil(t, want)
	if !strings.Contains(got, want) {
		c.h.failf(t, "%s did not receive %q\nlast recv: %q", c.name, want, got)
	}
}

func (c *telnetClient) createUser(t *testing.T, user, pass string) {
	t.Helper()
	c.readUntil(t, "계정 이름:")
	c.send(t, user)
	c.readUntil(t, "암호:")
	c.send(t, pass)
	c.readUntil(t, "새로 만드시겠습니까?")
	c.send(t, "y")
	c.readUntil(t, "암호:")
	c.send(t, pass)
	c.readUntil(t, user+" 님이 자리에 앉았습니다.")
	c.readUntil(t, ">")
}

type wsFrame struct {
	V       int             `json:"v"`
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type wsClient struct {
	name string
	conn *websocket.Conn
	h    *harness
}

func (h *harness) dialWS(t *testing.T, name string) *wsClient {
	t.Helper()
	u := url.URL{Scheme: "ws", Host: h.wsAddr, Path: wsPath}
	d := websocket.Dialer{HandshakeTimeout: ioWait}
	conn, _, err := d.Dial(u.String(), nil)
	if err != nil {
		h.failf(t, "%s dial ws %s: %v", name, u.String(), err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	h.tr.add("%s DIAL ws %s", name, u.String())
	return &wsClient{name: name, conn: conn, h: h}
}

func (c *wsClient) send(t *testing.T, typ, id string, payload any) {
	t.Helper()
	if payload == nil {
		payload = map[string]any{}
	}
	raw, err := json.Marshal(wsFrame{V: protoV, Type: typ, ID: id, Payload: mustRaw(t, payload)})
	if err != nil {
		c.h.failf(t, "%s marshal: %v", c.name, err)
	}
	c.h.tr.add("%s SEND %s", c.name, raw)
	_ = c.conn.SetWriteDeadline(time.Now().Add(ioWait))
	if err := c.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		c.h.failf(t, "%s send %s: %v", c.name, raw, err)
	}
}

func (c *wsClient) readFrame(t *testing.T) wsFrame {
	t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(readWait))
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		c.h.failf(t, "%s read frame: %v", c.name, err)
	}
	c.h.tr.add("%s RECV %s", c.name, data)
	var f wsFrame
	if err := json.Unmarshal(data, &f); err != nil {
		c.h.failf(t, "%s parse frame %s: %v", c.name, data, err)
	}
	return f
}

func (c *wsClient) readType(t *testing.T, typ string) wsFrame {
	t.Helper()
	deadline := time.Now().Add(readWait)
	for time.Now().Before(deadline) {
		f := c.readFrame(t)
		if f.Type == typ {
			return f
		}
	}
	c.h.failf(t, "%s timeout waiting for type %s", c.name, typ)
	return wsFrame{}
}

func (c *wsClient) expectSay(t *testing.T, from, text string) {
	t.Helper()
	deadline := time.Now().Add(readWait)
	for time.Now().Before(deadline) {
		f := c.readFrame(t)
		if f.Type != typeText {
			continue
		}
		var p struct {
			Channel string `json:"channel"`
			From    string `json:"from"`
			Text    string `json:"text"`
		}
		if err := json.Unmarshal(f.Payload, &p); err != nil {
			c.h.failf(t, "%s text payload: %v", c.name, err)
		}
		if p.Channel == channelSay && p.From == from && strings.Contains(p.Text, text) {
			return
		}
	}
	c.h.failf(t, "%s did not receive say from %s %q", c.name, from, text)
}

func (c *wsClient) createUser(t *testing.T, user, pass, id string) {
	t.Helper()
	c.send(t, typeCreate, id, map[string]string{"username": user, "password": pass})
	ok := c.readType(t, typeOK)
	var p struct {
		Username string `json:"username"`
		Session  string `json:"session"`
	}
	if err := json.Unmarshal(ok.Payload, &p); err != nil {
		c.h.failf(t, "%s auth.ok payload: %v", c.name, err)
	}
	if p.Username != user {
		c.h.failf(t, "%s auth.ok username=%q want %q", c.name, p.Username, user)
	}
	if p.Session == "" {
		c.h.failf(t, "%s auth.ok missing session", c.name)
	}
}

func mustRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
