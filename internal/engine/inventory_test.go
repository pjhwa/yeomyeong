package engine

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/content"
	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

func TestGetDropEquip(t *testing.T) {
	l, _ := startInv(t, newMemSheets())
	outH := mustAttach(t, l, "h")
	l.Submit(EnterWorld{ConnID: "h", AccountID: "h", Username: "을", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "pebble", Qty: 18}}}})
	l.Submit(Get{ConnID: "h", ItemID: "rod"})
	_ = mustSnapshot(t, l)
	if textCode(drain(outH), text.CodeTooHeavy) == nil {
		t.Fatal("want too_heavy")
	}
	l.Submit(LeaveWorld{ConnID: "h"})

	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "acc", Username: "갑", Session: "s"})
	_ = mustSnapshot(t, l)
	card := findRoom(drain(out))
	if card == nil || qtyName(card.Ground, "조약돌") == 0 || qtyName(card.Ground, "나무막대") == 0 {
		t.Fatalf("spawn ground: %+v", card)
	}
	l.Submit(Get{ConnID: "a", ItemID: "pebble"})
	l.Submit(Equip{ConnID: "a", ItemID: "pebble"})
	l.Submit(Get{ConnID: "a", ItemID: "rod"})
	l.Submit(Equip{ConnID: "a", ItemID: "rod"})
	l.Submit(Unequip{ConnID: "a", Slot: yworld.SlotMainHand})
	l.Submit(DropItem{ConnID: "a", ItemID: "pebble"})
	snap := mustSnapshot(t, l)
	p := snap.Players[0]
	if qtyOf(p.Bag, "pebble") != 0 || qtyOf(p.Bag, "rod") != 1 || p.Equip.MainHand != "" {
		t.Fatalf("bag=%v equip=%v", p.Bag, p.Equip)
	}
	if qtyOf(snap.Ground["test:start"], "pebble") != 1 {
		t.Fatalf("ground: %v", snap.Ground["test:start"])
	}
	evs := drain(out)
	if textCode(evs, text.CodeNotWearable) == nil {
		t.Fatal("pebble not wearable")
	}
}

func TestGroundIsPerRoom(t *testing.T) {
	l, _ := startInv(t, nil)
	mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	l.Submit(Move{ConnID: "a", Dir: "north"})
	l.Submit(Get{ConnID: "a", ItemID: "pebble"})
	l.Submit(EnterWorld{ConnID: "b", AccountID: "2", Username: "을", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "cloak", Qty: 1}}}})
	l.Submit(Move{ConnID: "b", Dir: "north"})
	l.Submit(DropItem{ConnID: "b", ItemID: "cloak"})
	snap := mustSnapshot(t, l)
	if qtyOf(snap.Ground["test:start"], "pebble") != 1 || qtyOf(snap.Ground["test:yard"], "pebble") != 0 {
		t.Fatalf("pebble leaked: %v", snap.Ground)
	}
	if qtyOf(snap.Ground["test:yard"], "cloak") != 1 || qtyOf(snap.Ground["test:start"], "cloak") != 0 {
		t.Fatalf("cloak leaked: %v", snap.Ground)
	}
}

func TestMemoryReconnectRestoresBag(t *testing.T) {
	store := newMemSheets()
	l, _ := startInv(t, store)
	mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "acc-1", Username: "갑", Session: "s1"})
	l.Submit(Get{ConnID: "a", ItemID: "pebble"})
	l.Submit(Get{ConnID: "a", ItemID: "rod"})
	l.Submit(Equip{ConnID: "a", ItemID: "rod"})
	_ = mustSnapshot(t, l)
	l.Submit(LeaveWorld{ConnID: "a"})
	if n := len(mustSnapshot(t, l).Players); n != 0 {
		t.Fatalf("left %d", n)
	}
	saved := store.load("acc-1")
	mustAttach(t, l, "b")
	l.Submit(EnterWorld{ConnID: "b", AccountID: "acc-1", Username: "갑", Session: "s2", Sheet: saved})
	snap := mustSnapshot(t, l)
	if len(snap.Players) != 1 || qtyOf(snap.Players[0].Bag, "pebble") != 1 || snap.Players[0].Equip.MainHand != "rod" {
		t.Fatalf("reconnect %+v saved %+v", snap.Players, saved)
	}
}

func startInv(t *testing.T, sink SheetSink) (*Loop, *content.World) {
	t.Helper()
	w, err := content.LoadWorld(filepath.Join("..", "content", "testdata", "valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	l := NewWithWorld(discardLog(), w.Rooms, w.Items, w.Ground, sink)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); l.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("loop did not stop")
		}
	})
	return l, w
}

type memSheets struct {
	mu sync.Mutex
	m  map[string]yworld.Sheet
}

func newMemSheets() *memSheets { return &memSheets{m: map[string]yworld.Sheet{}} }

func (m *memSheets) SaveAsync(id string, sheet yworld.Sheet) {
	m.mu.Lock()
	m.m[id] = yworld.CloneSheet(sheet)
	m.mu.Unlock()
}

func (m *memSheets) load(id string) yworld.Sheet {
	m.mu.Lock()
	defer m.mu.Unlock()
	return yworld.CloneSheet(m.m[id])
}

func qtyOf(piles []yworld.Stack, id string) int {
	for _, s := range piles {
		if s.ID == id {
			return s.Qty
		}
	}
	return 0
}

func qtyName(names []string, want string) int {
	n := 0
	for _, s := range names {
		if s == want {
			n++
		}
	}
	return n
}
