package net

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pjhwa/yeomyeong/internal/engine"
	"github.com/pjhwa/yeomyeong/internal/persist"
)

func TestMixedTelnetAndWSExchangeSay(t *testing.T) {
	telnetAddr, loop, store := startServerStore(t)
	wsAddr := startWS(t, loop, store)

	tn := dial(t, telnetAddr)
	loginNew(t, tn, "갑을", "password1")

	ws := dialWS(t, wsAddr)
	ws.send(t, typeAuthCreate, "c1", map[string]string{"username": "병정", "password": "password2"})
	ok := ws.readType(t, typeAuthOK)
	if got := payloadString(t, ok, "username"); got != "병정" {
		t.Fatalf("auth.ok username=%q", got)
	}
	if payloadString(t, ok, "session") == "" {
		t.Fatal("auth.ok missing session")
	}
	waitRoster(t, loop, 2)

	ws.send(t, typeCmdSay, "s1", map[string]string{"text": "안녕"})
	want := "[말] 병정: 안녕"
	if !strings.Contains(tn.readUntil(t, want), want) {
		t.Fatal("telnet missed websocket say")
	}
	ws.waitText(t, engine.ChannelSay, "병정", "안녕")

	tn.send(t, "say 반갑소")
	ws.waitText(t, engine.ChannelSay, "갑을", "반갑소")
	if !strings.Contains(tn.readUntil(t, "[말] 갑을: 반갑소"), "[말] 갑을: 반갑소") {
		t.Fatal("telnet speaker missed own say")
	}

	ws.send(t, typeCmdQuit, "q1", map[string]any{})
	if !strings.Contains(tn.readUntil(t, "병정 님이 자리를 떴습니다."), "병정 님이 자리를 떴습니다.") {
		t.Fatal("telnet missed websocket leave")
	}
	waitRoster(t, loop, 1)
}

func TestWSBadFrameKeepsConnection(t *testing.T) {
	_, loop, store := startServerStore(t)
	ws := dialWS(t, startWS(t, loop, store))

	_ = ws.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if err := ws.conn.WriteMessage(websocket.TextMessage, []byte("{")); err != nil {
		t.Fatal(err)
	}
	got := ws.readType(t, typeSys)
	if payloadString(t, got, "code") != codeBadFrame {
		t.Fatalf("invalid json: %+v", got)
	}

	ws.sendRaw(t, `{"v":2,"type":"cmd.say","id":"v","payload":{"text":"x"}}`)
	got = ws.readType(t, typeSys)
	if payloadString(t, got, "code") != codeBadFrame || got.ID != "v" {
		t.Fatalf("bad v: %+v", got)
	}

	ws.send(t, "cmd.look", "u", map[string]any{})
	got = ws.readType(t, typeSys)
	if payloadString(t, got, "code") != codeBadFrame || got.ID != "u" {
		t.Fatalf("unknown type: %+v", got)
	}

	if err := ws.conn.WriteMessage(websocket.BinaryMessage, []byte("nope")); err != nil {
		t.Fatal(err)
	}
	got = ws.readType(t, typeSys)
	if payloadString(t, got, "code") != codeBadFrame {
		t.Fatalf("binary: %+v", got)
	}

	ws.send(t, typeAuthCreate, "ok", map[string]string{"username": "갑을", "password": "password1"})
	if ws.readType(t, typeAuthOK).Type != typeAuthOK {
		t.Fatal("connection should stay up after bad frames")
	}
	waitRoster(t, loop, 1)
}

func TestWSAuthErrorsAndLogin(t *testing.T) {
	_, loop, store := startServerStore(t)
	addr := startWS(t, loop, store)

	a := dialWS(t, addr)
	a.send(t, typeAuthCreate, "badu", map[string]string{"username": "x", "password": "password1"})
	got := a.readType(t, typeAuthErr)
	if payloadString(t, got, "code") != persist.ErrBadUsername.Error() {
		t.Fatalf("bad username: %+v", got)
	}

	a.send(t, typeAuthCreate, "badp", map[string]string{"username": "갑을", "password": "short"})
	got = a.readType(t, typeAuthErr)
	if payloadString(t, got, "code") != persist.ErrBadPassword.Error() {
		t.Fatalf("bad password: %+v", got)
	}

	a.send(t, typeAuthCreate, "c1", map[string]string{"username": "갑을", "password": "password1"})
	if a.readType(t, typeAuthOK).ID != "c1" {
		t.Fatal("create should succeed")
	}

	a.send(t, typeAuthCreate, "c2", map[string]string{"username": "병정", "password": "password2"})
	if a.readType(t, typeAuthOK).ID != "c2" {
		t.Fatal("second auth on seated conn should echo auth.ok")
	}

	b := dialWS(t, addr)
	b.send(t, typeAuthCreate, "dup", map[string]string{"username": "갑을", "password": "password2"})
	got = b.readType(t, typeAuthErr)
	if payloadString(t, got, "code") != persist.ErrNameTaken.Error() {
		t.Fatalf("name taken: %+v", got)
	}

	b.send(t, typeAuthLogin, "bad", map[string]string{"username": "갑을", "password": "wrongpass"})
	got = b.readType(t, typeAuthErr)
	if payloadString(t, got, "code") != persist.ErrBadCredentials.Error() {
		t.Fatalf("bad creds: %+v", got)
	}

	b.send(t, typeAuthLogin, "ok", map[string]string{"username": "갑을", "password": "password1"})
	if payloadString(t, b.readType(t, typeAuthOK), "username") != "갑을" {
		t.Fatal("login failed")
	}
	waitRoster(t, loop, 2)
}

func TestWSSayBeforeAuthRateLimitQuit(t *testing.T) {
	_, loop, store := startServerStore(t)
	addr := startWS(t, loop, store)

	c := dialWS(t, addr)
	c.send(t, typeCmdSay, "s0", map[string]string{"text": "hello"})
	got := c.readType(t, typeSys)
	if payloadString(t, got, "code") != codeNotAuth || got.ID != "s0" {
		t.Fatalf("not_authenticated: %+v", got)
	}

	c.send(t, typeAuthCreate, "c1", map[string]string{"username": "갑을", "password": "password1"})
	_ = c.readType(t, typeAuthOK)
	waitRoster(t, loop, 1)

	c.send(t, typeCmdSay, "empty", map[string]string{"text": ""})
	got = c.readType(t, typeSys)
	if payloadString(t, got, "code") != codeBadFrame {
		t.Fatalf("empty say: %+v", got)
	}

	for i := 0; i < 25; i++ {
		c.send(t, typeCmdSay, "r", map[string]string{"text": "x"})
	}
	if payloadString(t, c.readType(t, typeSys), "code") != codeRateLimited {
		t.Fatal("want rate_limited")
	}

	d := dialWS(t, addr)
	d.send(t, typeCmdQuit, "q", map[string]any{})
	waitRoster(t, loop, 1)
}

func TestWSDisconnectRemovesFromRoster(t *testing.T) {
	_, loop, store := startServerStore(t)
	c := dialWS(t, startWS(t, loop, store))
	c.send(t, typeAuthCreate, "c1", map[string]string{"username": "갑을", "password": "password1"})
	_ = c.readType(t, typeAuthOK)
	waitRoster(t, loop, 1)
	_ = c.conn.Close()
	waitRoster(t, loop, 0)
}

func TestWSOnlyGetWS(t *testing.T) {
	_, loop, store := startServerStore(t)
	addr := startWS(t, loop, store)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET / status=%d", resp.StatusCode)
	}
	resp, err = client.Get("http://" + addr + wsPath)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		t.Fatal("GET /ws should exist")
	}
}

func TestParseFrameAndPayload(t *testing.T) {
	f, err := parseFrame([]byte(`{"v":1,"type":"cmd.say","id":"a","payload":{"text":"hi"}}`))
	if err != nil || f.V != 1 || f.Type != typeCmdSay || f.ID != "a" {
		t.Fatalf("parse: %+v %v", f, err)
	}
	var p struct {
		Text string `json:"text"`
	}
	if err := decodePayload(f.Payload, &p); err != nil || p.Text != "hi" {
		t.Fatalf("payload: %q %v", p.Text, err)
	}
	if _, err := parseFrame([]byte(`{`)); err == nil {
		t.Fatal("want invalid json")
	}
	if err := decodePayload(json.RawMessage(strings.Repeat("x", maxPayload+1)), &p); err == nil {
		t.Fatal("want oversize payload")
	}
	if err := decodePayload(nil, &p); err != nil {
		t.Fatalf("nil payload: %v", err)
	}
	code, msg := mapAuthErr(persist.ErrNameTaken)
	if code != "name_taken" || msg != "name_taken" {
		t.Fatalf("mapAuthErr: %s %s", code, msg)
	}
	code, msg = mapAuthErr(io.EOF)
	if code != codeInternal || msg != codeInternal {
		t.Fatalf("mapAuthErr unknown: %s %s", code, msg)
	}
}

func startWS(t *testing.T, loop *engine.Loop, store persist.AccountStore) string {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())
	srv := NewWS("127.0.0.1:0", loop, store, log)
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ctx) }()
	select {
	case <-srv.ready:
	case err := <-errCh:
		cancel()
		t.Fatalf("ws serve: %v", err)
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("ws listen timeout")
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
			t.Error("ws serve did not stop")
		}
	})
	return srv.BoundAddr()
}

type wsClient struct {
	conn *websocket.Conn
}

func dialWS(t *testing.T, addr string) *wsClient {
	t.Helper()
	u := url.URL{Scheme: "ws", Host: addr, Path: wsPath}
	d := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, _, err := d.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &wsClient{conn: conn}
}

func (c *wsClient) send(t *testing.T, typ, id string, payload any) {
	t.Helper()
	if payload == nil {
		payload = map[string]any{}
	}
	c.sendRaw(t, mustJSON(t, outFrame{V: protoV, Type: typ, ID: id, Payload: payload}))
}

func (c *wsClient) sendRaw(t *testing.T, raw string) {
	t.Helper()
	_ = c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if err := c.conn.WriteMessage(websocket.TextMessage, []byte(raw)); err != nil {
		t.Fatal(err)
	}
}

func (c *wsClient) mustRead(t *testing.T) inFrame {
	t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	f, err := parseFrame(data)
	if err != nil {
		t.Fatalf("client frame: %s %v", data, err)
	}
	return f
}

func (c *wsClient) readType(t *testing.T, typ string) inFrame {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		f := c.mustRead(t)
		if f.Type == typ {
			return f
		}
	}
	t.Fatalf("timeout waiting for type %s", typ)
	return inFrame{}
}

func (c *wsClient) waitText(t *testing.T, channel, from, text string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		f := c.mustRead(t)
		if f.Type != typeText {
			continue
		}
		var p struct {
			Channel string `json:"channel"`
			From    string `json:"from"`
			Text    string `json:"text"`
		}
		if err := decodePayload(f.Payload, &p); err != nil {
			t.Fatal(err)
		}
		if p.Channel == channel && (from == "" || p.From == from) && strings.Contains(p.Text, text) {
			return
		}
	}
	t.Fatalf("timeout waiting for text %s %s %q", channel, from, text)
}

func payloadString(t *testing.T, f inFrame, key string) string {
	t.Helper()
	var m map[string]string
	if err := decodePayload(f.Payload, &m); err != nil {
		t.Fatal(err)
	}
	return m[key]
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
