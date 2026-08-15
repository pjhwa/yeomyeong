package net

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"log/slog"
	stdnet "net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/engine"
	"github.com/pjhwa/yeomyeong/internal/persist"
	"github.com/pjhwa/yeomyeong/internal/world"
)

func TestTwoClientsExchangeSay(t *testing.T) {
	addr, loop := startServer(t)

	a := dial(t, addr)
	b := dial(t, addr)
	loginNew(t, a, "갑을", "password1")
	loginNew(t, b, "병정", "password2")
	waitRoster(t, loop, 2)

	a.send(t, "say 안녕")
	want := "[말] 갑을: 안녕"
	if !strings.Contains(a.readUntil(t, want), want) {
		t.Fatal("speaker missed own say")
	}
	if !strings.Contains(b.readUntil(t, want), want) {
		t.Fatal("peer missed say")
	}

	b.sendCRLF(t, "말 반갑소")
	want = "[말] 병정: 반갑소"
	if !strings.Contains(a.readUntil(t, want), want) {
		t.Fatal("peer missed korean say")
	}
	if !strings.Contains(b.readUntil(t, want), want) {
		t.Fatal("speaker missed own korean say")
	}

	a.send(t, "quit")
	if !strings.Contains(b.readUntil(t, "갑을 님이 자리를 떴습니다."), "갑을 님이 자리를 떴습니다.") {
		t.Fatal("remaining player missed leave line")
	}
	waitRoster(t, loop, 1)
}

func TestUnknownUserCreatePrompt(t *testing.T) {
	addr, loop := startServer(t)
	c := dial(t, addr)
	c.readUntil(t, "여명 · YEOMYEONG")
	c.readUntil(t, "계정 이름:")
	c.send(t, "새유저")
	c.readUntil(t, "암호:")
	c.send(t, "password1")
	c.readUntil(t, "그 이름은 장부에 없습니다. 새로 만드시겠습니까? (y/n)")
	c.send(t, "n")
	c.readUntil(t, "계정 이름:")
	c.send(t, "새유저")
	c.readUntil(t, "암호:")
	c.send(t, "password1")
	c.readUntil(t, "새로 만드시겠습니까?")
	c.send(t, "y")
	c.readUntil(t, "암호:")
	c.send(t, "password1")
	c.readUntil(t, "새유저 님이 자리에 앉았습니다.")
	c.readUntil(t, ">")
	waitRoster(t, loop, 1)
}

func TestBadPassword(t *testing.T) {
	addr, _, store := startServerStore(t)
	ctx := context.Background()
	if _, err := store.Create(ctx, "갑을", "password1"); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.readUntil(t, "계정 이름:")
	c.send(t, "갑을")
	c.readUntil(t, "암호:")
	c.send(t, "wrongpass")
	got := c.readUntil(t, "계정 이름:")
	if !strings.Contains(got, "이름이나 암호가 맞지 않습니다.") {
		t.Fatalf("want bad-creds line, got %q", got)
	}
	if strings.Contains(got, "새로 만드시겠습니까") {
		t.Fatal("existing user must not see create prompt")
	}
	c.send(t, "갑을")
	c.readUntil(t, "암호:")
	c.send(t, "password1")
	c.readUntil(t, "갑을 님이 자리에 앉았습니다.")
}

func TestUnknownCommandEmptyAndRateLimit(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	loginNew(t, c, "갑을", "password1")

	c.send(t, "")
	c.send(t, "xyzzy")
	if !strings.Contains(c.readUntil(t, "모르는 말입니다. say / quit"), "모르는 말입니다. say / quit") {
		t.Fatal("missing help line")
	}

	for i := 0; i < 25; i++ {
		c.send(t, "say x")
	}
	if !strings.Contains(c.readUntil(t, "rate_limited"), "rate_limited") {
		t.Fatal("want rate_limited")
	}
}

func TestDisconnectRemovesFromRoster(t *testing.T) {
	addr, loop := startServer(t)
	c := dial(t, addr)
	loginNew(t, c, "갑을", "password1")
	waitRoster(t, loop, 1)
	_ = c.conn.Close()
	waitRoster(t, loop, 0)
}

func TestExistingLoginAndIAC(t *testing.T) {
	addr, _, store := startServerStore(t)
	if _, err := store.Create(context.Background(), "갑을", "password1"); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.readUntil(t, "계정 이름:")
	// IAC WILL ECHO then the name — option bytes must not pollute the username.
	if _, err := c.conn.Write([]byte{iac, iacWill, 1}); err != nil {
		t.Fatal(err)
	}
	c.send(t, "갑을")
	c.readUntil(t, "암호:")
	c.send(t, "password1")
	c.readUntil(t, "갑을 님이 자리에 앉았습니다.")
}

func TestReadLineCRLFAndIAC(t *testing.T) {
	a, b := stdnet.Pipe()
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	go func() {
		_, _ = a.Write([]byte{iac, iacWill, 1})
		_, _ = a.Write([]byte("hello\r\n"))
		_, _ = a.Write([]byte("world\n"))
		_, _ = a.Write([]byte{'x', iac, iac, 'y', '\n'})
		_, _ = a.Write([]byte{iac, iacSb, 1, 2, iac, iacSe})
		_, _ = a.Write([]byte("z\n"))
	}()
	r := bufio.NewReader(b)
	got, err := readLine(r)
	if err != nil || got != "hello" {
		t.Fatalf("crlf: %q %v", got, err)
	}
	got, err = readLine(r)
	if err != nil || got != "world" {
		t.Fatalf("lf: %q %v", got, err)
	}
	got, err = readLine(r)
	if err != nil || got != "xy" {
		t.Fatalf("dropped iac: %q %v", got, err)
	}
	got, err = readLine(r)
	if err != nil || got != "z" {
		t.Fatalf("subneg: %q %v", got, err)
	}
}

func TestLookMoveAndNoExit(t *testing.T) {
	addr, loop := startServer(t)

	a := dial(t, addr)
	b := dial(t, addr)
	loginNew(t, a, "갑을", "password1")
	loginNew(t, b, "병정", "password2")
	waitRoster(t, loop, 2)

	a.send(t, "look")
	got := a.readUntil(t, "여기: 병정")
	if !strings.Contains(got, "시작 마당") || !strings.Contains(got, "흙마당이 넓다.") {
		t.Fatalf("look missing name/desc: %q", got)
	}
	if !strings.Contains(got, "출구: 북쪽(안마당)") {
		t.Fatalf("look missing exits: %q", got)
	}

	a.send(t, "s")
	if !strings.Contains(a.readUntil(t, "그쪽으로는 갈 수 없습니다."), "그쪽으로는 갈 수 없습니다.") {
		t.Fatal("missing no_exit")
	}

	a.send(t, "n")
	if !strings.Contains(a.readUntil(t, "우물이 가운데 있다."), "우물이 가운데 있다.") {
		t.Fatal("n did not move")
	}
	b.send(t, "보다")
	got = b.readUntil(t, "출구: 북쪽(안마당)")
	if strings.Contains(got, "여기:") || strings.Contains(b.buf, "여기:") {
		t.Fatal("yard walker must not appear in start who")
	}

	a.send(t, "가다 남")
	got = a.readUntil(t, "여기: 병정")
	if !strings.Contains(got, "시작 마당") {
		t.Fatalf("return missing name: %q", got)
	}

	a.send(t, "go north")
	if !strings.Contains(a.readUntil(t, "우물이 가운데 있다."), "우물이 가운데 있다.") {
		t.Fatal("go north failed")
	}
	a.send(t, "남")
	if !strings.Contains(a.readUntil(t, "흙마당이 넓다."), "흙마당이 넓다.") {
		t.Fatal("남 failed")
	}

	snap := waitRoster(t, loop, 2)
	for _, p := range snap.Players {
		if p.Username == "갑을" && p.RoomID != "test:start" {
			t.Fatalf("adapter must not write RoomID; 갑을=%+v", p)
		}
	}
}

func TestSplitCmdAndFormat(t *testing.T) {
	v, r := splitCmd("  말  안녕 세계  ")
	if v != "말" || r != "안녕 세계" {
		t.Fatalf("split: %q %q", v, r)
	}
	v, r = splitCmd("quit")
	if v != "quit" || r != "" {
		t.Fatalf("quit: %q %q", v, r)
	}
	say := formatText(engine.Text{Channel: engine.ChannelSay, From: "갑", Body: "안녕"})
	if say != "[말] 갑: 안녕" {
		t.Fatalf("say format: %q", say)
	}
	sys := formatText(engine.Text{Channel: engine.ChannelSys, Body: "자리에 앉았습니다."})
	if sys != "자리에 앉았습니다." {
		t.Fatalf("sys format: %q", sys)
	}
	lines := formatRoom(engine.Room{
		Name:        "시작 마당",
		Description: "흙마당이 넓다.",
		Exits:       map[string]string{"north": "안마당", "east": "가게"},
		Who:         []string{"병정"},
	})
	if len(lines) != 4 || lines[2] != "출구: 북쪽(안마당), 동쪽(가게)" || lines[3] != "여기: 병정" {
		t.Fatalf("room format: %#v", lines)
	}
	if n := formatRoom(engine.Room{Name: "벼랑", Description: "끝."}); len(n) != 2 {
		t.Fatalf("omit empty chrome: %#v", n)
	}
	if d, ok := parseTelnetDir("북"); !ok || d != "north" {
		t.Fatalf("parse 북: %q %v", d, ok)
	}
	if !isLook("look", "look") || !isLook("l", "l") || !isLook("보다", "보다") || !isLook("살펴", "살펴") {
		t.Fatal("look verbs")
	}
	for _, in := range []string{"s", "south", "e", "east", "w", "west", "u", "up", "d", "down", "아래"} {
		if _, ok := parseTelnetDir(in); !ok {
			t.Fatalf("parse %q", in)
		}
	}
}

func TestLimiterWindow(t *testing.T) {
	var lim limiter
	now := time.Unix(1_700_000_000, 0)
	for i := 0; i < cmdRate; i++ {
		if !lim.allow(now) {
			t.Fatalf("allowed %d dropped", i)
		}
	}
	if lim.allow(now) {
		t.Fatal("21st in window must drop")
	}
	if !lim.allow(now.Add(cmdWindow + time.Millisecond)) {
		t.Fatal("after window must allow")
	}
}

func startServer(t *testing.T) (addr string, loop *engine.Loop) {
	t.Helper()
	addr, loop, _ = startServerStore(t)
	return addr, loop
}

func startServerStore(t *testing.T) (string, *engine.Loop, *persist.Memory) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	loop := engine.NewWithCatalog(log, testCatalog(t))
	store := persist.NewMemory()
	ctx, cancel := context.WithCancel(context.Background())
	srv := NewServer("127.0.0.1:0", loop, store, log)

	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		loop.Run(ctx)
	}()
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ctx) }()

	select {
	case <-srv.ready:
	case err := <-errCh:
		cancel()
		t.Fatalf("serve: %v", err)
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("listen timeout")
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
			t.Error("serve did not stop")
		}
		select {
		case <-loopDone:
		case <-time.After(2 * time.Second):
			t.Error("loop did not stop")
		}
	})
	return srv.BoundAddr(), loop, store
}

type testConn struct {
	conn stdnet.Conn
	buf  string
}

func dial(t *testing.T, addr string) *testConn {
	t.Helper()
	c, err := stdnet.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return &testConn{conn: c}
}

func (c *testConn) send(t *testing.T, line string) {
	t.Helper()
	_ = c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(c.conn, line+"\n"); err != nil {
		t.Fatal(err)
	}
}

func (c *testConn) sendCRLF(t *testing.T, line string) {
	t.Helper()
	_ = c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(c.conn, line+"\r\n"); err != nil {
		t.Fatal(err)
	}
}

func (c *testConn) readUntil(t *testing.T, needle string) string {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	_ = c.conn.SetReadDeadline(deadline)
	tmp := make([]byte, 256)
	var collected strings.Builder
	for {
		if i := strings.Index(c.buf, needle); i >= 0 {
			end := i + len(needle)
			collected.WriteString(c.buf[:end])
			c.buf = c.buf[end:]
			return collected.String()
		}
		collected.WriteString(c.buf)
		c.buf = ""
		n, err := c.conn.Read(tmp)
		if n > 0 {
			c.buf += string(tmp[:n])
			continue
		}
		if err != nil {
			t.Fatalf("readUntil %q: %v\ngot:\n%s", needle, err, collected.String())
		}
	}
}

func loginNew(t *testing.T, c *testConn, name, pass string) {
	t.Helper()
	c.readUntil(t, "계정 이름:")
	c.send(t, name)
	c.readUntil(t, "암호:")
	c.send(t, pass)
	c.readUntil(t, "새로 만드시겠습니까?")
	c.send(t, "y")
	c.readUntil(t, "암호:")
	c.send(t, pass)
	c.readUntil(t, name+" 님이 자리에 앉았습니다.")
	c.readUntil(t, ">")
}

func waitRoster(t *testing.T, loop *engine.Loop, n int) engine.Snapshot {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	var last engine.Snapshot
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		snap, err := loop.Snapshot(ctx)
		cancel()
		if err == nil && len(snap.Players) == n {
			return snap
		}
		if err == nil {
			last = snap
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("roster want %d got %d (%+v)", n, len(last.Players), last.Players)
	return last
}

func testCatalog(t *testing.T) *world.Catalog {
	t.Helper()
	cat, err := world.NewCatalog([]world.Room{
		{
			ID: "test:start", Name: world.Localized{KO: "시작 마당"},
			Description: world.Localized{KO: "흙마당이 넓다."},
			Exits:       map[string]string{"north": "test:yard"},
		},
		{
			ID: "test:yard", Name: world.Localized{KO: "안마당"},
			Description: world.Localized{KO: "우물이 가운데 있다."},
			Exits:       map[string]string{"south": "test:start"},
		},
	}, "test:start")
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestAdaptersNeverWriteRoomID(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte("RoomID")) {
			t.Errorf("%s: adapters must not touch Player.RoomID", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFormatNoSayPrefixOnSys(t *testing.T) {
	if strings.Contains(formatText(engine.Text{Channel: engine.ChannelSys, Body: "x"}), "[말]") {
		t.Fatal("sys line must not carry [말]")
	}
}

func TestMaxLineIgnored(t *testing.T) {
	a, b := stdnet.Pipe()
	defer func() { _ = a.Close() }()
	defer func() { _ = b.Close() }()
	go func() {
		_, _ = a.Write(bytes.Repeat([]byte("a"), maxLine+8))
		_, _ = a.Write([]byte("\nok\n"))
	}()
	r := bufio.NewReader(b)
	got, err := readLine(r)
	if err != nil || got != "" {
		t.Fatalf("oversize: %q %v", got, err)
	}
	got, err = readLine(r)
	if err != nil || got != "ok" {
		t.Fatalf("after oversize: %q %v", got, err)
	}
}
