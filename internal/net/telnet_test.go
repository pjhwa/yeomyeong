package net

import (
	"bufio"
	"bytes"
	"context"
	"errors"
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
	"github.com/pjhwa/yeomyeong/internal/skill"
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
	if !strings.Contains(b.readUntil(t, "갑을 님이 나갔어요."), "갑을 님이 나갔어요.") {
		t.Fatal("remaining player missed leave line")
	}
	waitRoster(t, loop, 1)
}

func TestUnknownUserCreatePrompt(t *testing.T) {
	addr, loop := startServer(t)
	c := dial(t, addr)
	c.readUntil(t, "여명 · YEOMYEONG")
	c.readUntil(t, "이름:")
	c.send(t, "새유저")
	c.readUntil(t, "비밀번호:")
	c.send(t, "password1")
	c.readUntil(t, "없는 이름이에요. 새로 만들까요? (y/n)")
	c.send(t, "n")
	c.readUntil(t, "이름:")
	c.send(t, "새유저")
	c.readUntil(t, "비밀번호:")
	c.send(t, "password1")
	c.readUntil(t, "새로 만들까요?")
	c.send(t, "y")
	c.readUntil(t, "비밀번호:")
	c.send(t, "password1")
	c.readUntil(t, "새유저 님이 들어왔어요.")
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
	c.readUntil(t, "이름:")
	c.send(t, "갑을")
	c.readUntil(t, "비밀번호:")
	c.send(t, "wrongpass")
	got := c.readUntil(t, "이름:")
	if !strings.Contains(got, "이름이나 비밀번호가 안 맞아요.") {
		t.Fatalf("want bad-creds line, got %q", got)
	}
	if strings.Contains(got, "새로 만들까요") {
		t.Fatal("existing user must not see create prompt")
	}
	c.send(t, "갑을")
	c.readUntil(t, "비밀번호:")
	c.send(t, "password1")
	c.readUntil(t, "갑을 님이 들어왔어요.")
}

func TestUnknownCommandEmptyAndRateLimit(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	loginNew(t, c, "갑을", "password1")

	c.send(t, "")
	c.send(t, "xyzzy")
	if !strings.Contains(c.readUntil(t, "무슨 말인지 모르겠어요. 보다, 종료"), "무슨 말인지 모르겠어요. 보다, 종료") {
		t.Fatal("missing help line")
	}

	for i := 0; i < 25; i++ {
		c.send(t, "say x")
	}
	if !strings.Contains(c.readUntil(t, "너무 빨리 입력했어요"), "너무 빨리 입력했어요") {
		t.Fatal("want 너무 빨리 입력했어요")
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

func TestNegotiateSendsWillEchoAndSGA(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	got := c.readN(t, 6)
	want := []byte{iac, iacWill, 1, iac, iacWill, 3}
	if !bytes.Equal(got, want) {
		t.Fatalf("negotiate prefix: %v want %v", got, want)
	}
	banner := c.readUntil(t, "이름:")
	if !strings.Contains(banner, "여명 · YEOMYEONG") {
		t.Fatalf("missing banner: %q", banner)
	}
	if !strings.Contains(banner, "한글이 깨지면: nc ") {
		t.Fatalf("missing hangul hint: %q", banner)
	}
}

func TestPasswordHiddenUsernameEchoed(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	c.readUntil(t, "이름:")
	c.send(t, "갑을")
	nameChunk := c.readUntil(t, "비밀번호:")
	if !strings.Contains(nameChunk, "갑을") {
		t.Fatalf("username must be echoed, got %q", nameChunk)
	}
	c.send(t, "s3cretPW")
	passChunk := c.readUntil(t, "새로 만들까요?")
	if strings.Contains(passChunk, "s3cretPW") {
		t.Fatalf("password must not be echoed, got %q", passChunk)
	}
	c.send(t, "y")
	c.readUntil(t, "비밀번호:")
	c.send(t, "s3cretPW")
	createChunk := c.readUntil(t, "갑을 님이 들어왔어요.")
	if strings.Contains(createChunk, "s3cretPW") {
		t.Fatalf("create password must not be echoed, got %q", createChunk)
	}
}

func TestBackspaceAndControlChars(t *testing.T) {
	addr, loop := startServer(t)
	c := dial(t, addr)
	c.readUntil(t, "이름:")
	// "갑x<BS>을" plus C0 junk (^E ^S) must become "갑을".
	raw := []byte("갑")
	raw = append(raw, 'x', 0x08)
	raw = append(raw, []byte("을")...)
	raw = append(raw, 0x05, 0x13, '\n')
	_ = c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := c.conn.Write(raw); err != nil {
		t.Fatal(err)
	}
	c.readUntil(t, "비밀번호:")
	c.send(t, "password1")
	c.readUntil(t, "새로 만들까요?")
	c.send(t, "y")
	c.readUntil(t, "비밀번호:")
	c.send(t, "password1")
	c.readUntil(t, "갑을 님이 들어왔어요.")
	waitRoster(t, loop, 1)
}

func TestExistingLoginAndIAC(t *testing.T) {
	addr, _, store := startServerStore(t)
	if _, err := store.Create(context.Background(), "갑을", "password1"); err != nil {
		t.Fatal(err)
	}
	c := dial(t, addr)
	c.readUntil(t, "이름:")
	// IAC WILL ECHO then the name — option bytes must not pollute the username.
	if _, err := c.conn.Write([]byte{iac, iacWill, 1}); err != nil {
		t.Fatal(err)
	}
	c.send(t, "갑을")
	c.readUntil(t, "비밀번호:")
	c.send(t, "password1")
	c.readUntil(t, "갑을 님이 들어왔어요.")
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
	got, err = readLineSoon(t, r, a, []byte("cronly\r"))
	if err != nil || got != "cronly" {
		t.Fatalf("bare cr: %q %v", got, err)
	}
	got, err = readLineSoon(t, r, a, []byte("crnul\r\x00"))
	if err != nil || got != "crnul" {
		t.Fatalf("cr nul: %q %v", got, err)
	}
}

func readLineSoon(t *testing.T, r *bufio.Reader, w stdnet.Conn, payload []byte) (string, error) {
	t.Helper()
	type res struct {
		s   string
		err error
	}
	ch := make(chan res, 1)
	go func() {
		s, err := readLine(r)
		ch <- res{s, err}
	}()
	if _, err := w.Write(payload); err != nil {
		return "", err
	}
	select {
	case got := <-ch:
		return got.s, got.err
	case <-time.After(2 * time.Second):
		return "", errors.New("timeout")
	}
}

func TestTalkAndExamine(t *testing.T) {
	addr, _, _ := startStoryServer(t)
	c := dial(t, addr)
	loginNew(t, c, "갑을", "password1")

	c.send(t, "보다")
	got := c.readUntil(t, "사람: 훈장")
	if !strings.Contains(got, "시작 마당") {
		t.Fatalf("look npcs: %q", got)
	}

	c.send(t, "대화 훈장")
	if !strings.Contains(c.readUntil(t, "처음 보는 얼굴이군."), "처음 보는 얼굴이군.") {
		t.Fatal("first talk")
	}
	c.send(t, "talk 선생")
	if !strings.Contains(c.readUntil(t, "또 왔군, 자네."), "또 왔군, 자네.") {
		t.Fatal("second talk")
	}

	c.send(t, "보다 훈장")
	if !strings.Contains(c.readUntil(t, "낡은 코트"), "낡은 코트") {
		t.Fatal("examine npc")
	}

	c.send(t, "대화 청람")
	if !strings.Contains(c.readUntil(t, "여기는 그 사람이 없어요."), "여기는 그 사람이 없어요.") {
		t.Fatal("missing npc")
	}

	c.send(t, "n")
	c.readUntil(t, "우물이 가운데 있다.")
	c.send(t, "보다 신문")
	got = c.readUntil(t, "두 번")
	if !strings.Contains(got, "한벽일보") || !strings.Contains(got, "활자") {
		t.Fatalf("newspaper: %q", got)
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
	if !strings.Contains(a.readUntil(t, "그쪽으로는 갈 수 없어요."), "그쪽으로는 갈 수 없어요.") {
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
	withNPC := formatRoom(engine.Room{
		Name: "서당", Description: "마루.", NPCs: []string{"청람 선생"},
	})
	if len(withNPC) != 3 || withNPC[2] != "사람: 청람 선생" {
		t.Fatalf("npc line: %#v", withNPC)
	}
	if !isTalk("talk", "talk") || !isTalk("대화", "대화") || !isTalk("말걸다", "말걸다") {
		t.Fatal("talk verbs")
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

func startStoryServer(t *testing.T) (string, *engine.Loop, *persist.Memory) {
	t.Helper()
	return startServerStoreOpts(t, true)
}

func startServerStore(t *testing.T) (string, *engine.Loop, *persist.Memory) {
	t.Helper()
	return startServerStoreOpts(t, false)
}

func startServerStoreOpts(t *testing.T, story bool) (string, *engine.Loop, *persist.Memory) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	loop := engine.NewWithCatalog(log, testCatalog(t))
	if skills, err := skill.Load(filepath.Join("..", "..", "content", "skills")); err != nil {
		t.Fatal(err)
	} else {
		loop = loop.WithSkills(skills)
	}
	if story {
		npcs, objs := testStoryCatalogs(t)
		loop = loop.WithNPCs(npcs).WithObjects(objs)
	}
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

func (c *testConn) readN(t *testing.T, n int) []byte {
	t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, n)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
		t.Fatalf("readN %d: %v", n, err)
	}
	return buf
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
	c.readUntil(t, "이름:")
	c.send(t, name)
	c.readUntil(t, "비밀번호:")
	c.send(t, pass)
	c.readUntil(t, "새로 만들까요?")
	c.send(t, "y")
	c.readUntil(t, "비밀번호:")
	c.send(t, pass)
	c.readUntil(t, name+" 님이 들어왔어요.")
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

func testStoryCatalogs(t *testing.T) (*world.NPCs, *world.Objects) {
	t.Helper()
	npcs, err := world.NewNPCs([]world.NPC{{
		ID: "tutor", Room: "test:start",
		Name: world.Localized{KO: "훈장"}, Aliases: []string{"훈장", "선생"},
		Look:       world.Localized{KO: "낡은 코트 소매에 잉크가 묻어 있다."},
		TalkFirst:  world.Localized{KO: "처음 보는 얼굴이군."},
		TalkSecond: world.Localized{KO: "또 왔군, 자네."},
	}})
	if err != nil {
		t.Fatal(err)
	}
	objs, err := world.NewObjects([]world.Object{{
		ID: "test-paper", Room: "test:yard",
		Name: world.Localized{KO: "한벽일보"}, Aliases: []string{"한벽일보", "신문", "게시판"},
		Description: world.Localized{KO: "한벽일보 한 줄에 같은 활자가 두 번 찍혀 어긋나 있다. 잉크가 손가락에 묻고, 활자 냄새가 난다."},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return npcs, objs
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

func TestPracticeSkillsInvVerbs(t *testing.T) {
	addr, _ := startServer(t)
	c := dial(t, addr)
	loginNew(t, c, "갑을", "password1")

	c.send(t, "skills")
	if !strings.Contains(c.readUntil(t, "가방:"), "가방:") {
		t.Fatal("skills")
	}
	c.send(t, "숙련")
	c.readUntil(t, "몸:")
	c.send(t, "inv")
	c.readUntil(t, "가방:")
	c.send(t, "소지")
	c.readUntil(t, "들고 있는 것:")

	c.send(t, "practice nope")
	if !strings.Contains(c.readUntil(t, "그런 기술은 없어요."), "그런 기술은 없어요.") {
		t.Fatal("practice unknown")
	}
	c.send(t, "익히다")
	if !strings.Contains(c.readUntil(t, "무슨 말인지 모르겠어요. 보다, 종료"), "무슨 말인지 모르겠어요. 보다, 종료") {
		t.Fatal("practice empty")
	}

	c.send(t, "두드리다")
	got := c.readUntil(t, "모루")
	if strings.Contains(got, "모르는 말입니다") || strings.Contains(got, "그런 기술은 없어요") {
		t.Fatalf("두드리다 should practice smith, got %q", got)
	}
	if strings.Contains(got, "숙련이 늘었습니다") || strings.Contains(got, "smith") {
		t.Fatalf("practice must not show a number bar: %q", got)
	}

	c.send(t, "get 없는물건")
	c.readUntil(t, "여기엔 그런 게 없어요.")
	c.send(t, "집다 x")
	c.readUntil(t, "여기엔 그런 게 없어요.")
	c.send(t, "drop 없는물건")
	c.readUntil(t, "그런 물건이 없어요.")
	c.send(t, "놓다 x")
	c.readUntil(t, "그런 물건이 없어요.")
	c.send(t, "equip 없는물건")
	c.readUntil(t, "그런 물건이 없어요.")
	c.send(t, "들다 x")
	c.readUntil(t, "그런 물건이 없어요.")
	c.send(t, "unequip main_hand")
	c.readUntil(t, "거기엔 아무것도 없어요.")
	c.send(t, "벗다 몸")
	c.readUntil(t, "거기엔 아무것도 없어요.")
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
