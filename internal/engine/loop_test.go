package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentSubmitRosterOnlyChangesOnLoop(t *testing.T) {
	l := startLoop(t)
	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			id := ConnID(fmt.Sprintf("c%02d", i))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if _, err := l.Attach(ctx, id); err != nil {
				t.Errorf("attach %s: %v", id, err)
				return
			}
			ok := l.Submit(EnterWorld{ConnID: id, AccountID: AccountID(id), Username: "u" + string(id), Session: "s"})
			ok = l.Submit(Say{ConnID: id, Text: "hi"}) && ok
			ok = l.Submit(LeaveWorld{ConnID: id}) && ok
			if !ok {
				t.Errorf("dropped command for %s", id)
			}
		}(i)
	}
	wg.Wait()
	if snap := mustSnapshot(t, l); len(snap.Players) != 0 {
		t.Fatalf("want empty roster, got %+v", snap.Players)
	}
}

func TestConcurrentEnterVisibleOnlyViaSnapshot(t *testing.T) {
	l := startLoop(t)
	const n = 16
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			id := ConnID(fmt.Sprintf("p%02d", i))
			if !l.Submit(EnterWorld{ConnID: id, AccountID: AccountID(id), Username: fmt.Sprintf("이름%d", i), Session: "tok"}) {
				t.Errorf("enter %s dropped", id)
			}
		}(i)
	}
	wg.Wait()
	snap := mustSnapshot(t, l)
	if len(snap.Players) != n {
		t.Fatalf("want %d players, got %d", n, len(snap.Players))
	}
	snap.Players[0].Username = "mutated"
	if again := mustSnapshot(t, l); again.Players[0].Username == "mutated" {
		t.Fatal("snapshot must copy roster entries")
	}
}

func TestSayBroadcastsTextToEveryRosterConn(t *testing.T) {
	l := startLoop(t)
	outs := map[ConnID]<-chan Event{
		"a": mustAttach(t, l, "a"),
		"b": mustAttach(t, l, "b"),
		"c": mustAttach(t, l, "c"),
	}
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s1"})
	l.Submit(EnterWorld{ConnID: "b", AccountID: "2", Username: "을", Session: "s2"})
	l.Submit(EnterWorld{ConnID: "c", AccountID: "3", Username: "병", Session: "s3"})
	l.Submit(Say{ConnID: "a", Text: "안녕"})
	_ = mustSnapshot(t, l)
	for id, out := range outs {
		if !hasText(drain(out), ChannelSay, "갑", "안녕") {
			t.Errorf("%s: missing say broadcast", id)
		}
	}
}

func TestLeaveRemovesPlayerAndNotifiesRemaining(t *testing.T) {
	l := startLoop(t)
	outA := mustAttach(t, l, "a")
	outB := mustAttach(t, l, "b")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	l.Submit(EnterWorld{ConnID: "b", AccountID: "2", Username: "을", Session: "s"})
	l.Submit(LeaveWorld{ConnID: "a"})
	snap := mustSnapshot(t, l)
	if len(snap.Players) != 1 || snap.Players[0].ConnID != "b" {
		t.Fatalf("want only 을 remaining, got %+v", snap.Players)
	}
	if hasText(drain(outA), ChannelSys, "", "갑 님이 자리를 떴습니다.") {
		t.Fatal("leaver must not receive their own leave sys line")
	}
	if !hasText(drain(outB), ChannelSys, "", "갑 님이 자리를 떴습니다.") {
		t.Fatal("remaining player missing leave sys line")
	}
}

func TestNoopsAndReenter(t *testing.T) {
	l := startLoop(t)
	out := mustAttach(t, l, "a")
	l.Submit(Say{ConnID: "ghost", Text: "??"})
	l.Submit(LeaveWorld{ConnID: "ghost"})
	l.Submit(EnterWorld{Username: "nobody"})
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s1"})
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s2"})
	snap := mustSnapshot(t, l)
	if len(snap.Players) != 1 || snap.Players[0].Session != "s1" {
		t.Fatalf("re-enter must not replace, got %+v", snap.Players)
	}
	n := 0
	for _, ev := range drain(out) {
		if tx, ok := ev.(Text); ok && tx.Channel == ChannelSys && strings.Contains(tx.Body, "앉았습니다") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want one seated line, got %d", n)
	}
}

func TestSubmitNilAndFullQueue(t *testing.T) {
	l := New(discardLog())
	if l.Submit(nil) {
		t.Fatal("nil command must be rejected")
	}
	for i := 0; i < CommandQueueSize; i++ {
		if !l.Submit(Say{ConnID: "x", Text: "n"}) {
			t.Fatalf("queue filled early at %d", i)
		}
	}
	if l.Submit(Say{ConnID: "x", Text: "overflow"}) {
		t.Fatal("full queue must return false")
	}
}

func TestOutboundOverflowDropsOldest(t *testing.T) {
	var buf bytes.Buffer
	l := New(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go l.Run(ctx)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	const extra = 5
	for i := 0; i < OutboundSize+extra; i++ {
		l.Submit(Say{ConnID: "a", Text: fmt.Sprintf("m%03d", i)})
	}
	_ = mustSnapshot(t, l)
	evs := drain(out)
	if len(evs) != OutboundSize {
		t.Fatalf("want %d buffered events, got %d", OutboundSize, len(evs))
	}
	last, ok := evs[len(evs)-1].(Text)
	if !ok || last.Body != fmt.Sprintf("m%03d", OutboundSize+extra-1) {
		t.Fatalf("want newest say kept, last=%#v", evs[len(evs)-1])
	}
	if !strings.Contains(buf.String(), "outbound overflow") {
		t.Fatalf("want overflow log, got %q", buf.String())
	}
}

func TestAttachDetachAndUnknownCommand(t *testing.T) {
	var buf bytes.Buffer
	l := New(slog.New(slog.NewTextHandler(&buf, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go l.Run(ctx)

	a := mustAttach(t, l, "a")
	if b := mustAttach(t, l, "a"); a != b {
		t.Fatal("second Attach must return the existing buffer")
	}
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	req, cancelReq := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelReq()
	if err := l.Detach(req, "a"); err != nil {
		t.Fatal(err)
	}
	if err := l.Detach(req, "missing"); err != nil {
		t.Fatal(err)
	}
	for range a {
	}
	l.Submit(Say{ConnID: "a", Text: "gone"})
	l.Submit(bogusCmd{})
	_ = mustSnapshot(t, l)
	if !strings.Contains(buf.String(), "unknown command") {
		t.Fatalf("want unknown-command log, got %q", buf.String())
	}
}

func TestEventTargets(t *testing.T) {
	for i, ev := range []Event{
		Text{ConnID: "c", Channel: ChannelSay, From: "u", Body: "x"},
		Drop{ConnID: "c"},
	} {
		if ev.Target() != "c" {
			t.Fatalf("%d: Target()=%q", i, ev.Target())
		}
	}
}

func TestNewNilLoggerAndShutdown(t *testing.T) {
	l := New(nil)
	if l.log == nil {
		t.Fatal("New(nil) must install a default logger")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.Run(ctx)
	}()
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestControlRequestsCancelled(t *testing.T) {
	l := New(discardLog())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := l.Snapshot(ctx); err == nil {
		t.Fatal("want ctx error")
	}
	if _, err := l.Attach(ctx, "a"); err == nil {
		t.Fatal("want ctx error")
	}
	if err := l.Detach(ctx, "a"); err == nil {
		t.Fatal("want ctx error")
	}
}

func TestWorldTypesHaveNoMutex(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte("sync.Mutex")) || bytes.Contains(b, []byte("sync.RWMutex")) {
			t.Errorf("%s: engine must not put a mutex on world state", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

type bogusCmd struct{}

func (bogusCmd) command() {}

func startLoop(t *testing.T) *Loop {
	t.Helper()
	l := New(discardLog())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("loop did not stop")
		}
	})
	return l
}

func mustAttach(t *testing.T, l *Loop, id ConnID) <-chan Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := l.Attach(ctx, id)
	if err != nil {
		t.Fatalf("attach %s: %v", id, err)
	}
	return ch
}

func mustSnapshot(t *testing.T, l *Loop) Snapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	snap, err := l.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return snap
}

func drain(ch <-chan Event) []Event {
	var out []Event
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, ev)
		default:
			return out
		}
	}
}

func hasText(evs []Event, channel, from, body string) bool {
	for _, ev := range evs {
		tx, ok := ev.(Text)
		if ok && tx.Channel == channel && tx.From == from && tx.Body == body {
			return true
		}
	}
	return false
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
