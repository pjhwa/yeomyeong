package engine

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/content"
	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

func startStory(t *testing.T) *Loop {
	t.Helper()
	return startStorySink(t, nil)
}

func startStorySink(t *testing.T, sink SheetSink) *Loop {
	t.Helper()
	w, err := content.LoadWorld(filepath.Join("..", "content", "testdata", "valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	l := NewWithWorld(discardLog(), w.Rooms, w.Items, w.Ground, sink).WithNPCs(w.NPCs).WithObjects(w.Objects)
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

func TestTalkTwiceRemembersAndFlagsArePerPlayer(t *testing.T) {
	l := startStory(t)
	outA := mustAttach(t, l, "a")
	outB := mustAttach(t, l, "b")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s1"})
	l.Submit(EnterWorld{ConnID: "b", AccountID: "2", Username: "을", Session: "s2"})
	_ = mustSnapshot(t, l)
	drain(outA)
	drain(outB)

	l.Submit(Talk{ConnID: "a", NPC: "훈장"})
	_ = mustSnapshot(t, l)
	first := drain(outA)
	if !hasTextContains(first, "처음 보는 얼굴이군.") {
		t.Fatalf("first talk: %#v", first)
	}

	l.Submit(Talk{ConnID: "a", NPC: "선생"})
	_ = mustSnapshot(t, l)
	second := drain(outA)
	if !hasTextContains(second, "또 왔군, 자네.") {
		t.Fatalf("second talk: %#v", second)
	}
	if hasTextContains(second, "처음 보는 얼굴이군.") {
		t.Fatal("second talk reused first line")
	}

	l.Submit(Talk{ConnID: "b", NPC: "훈장"})
	snap := mustSnapshot(t, l)
	if !hasTextContains(drain(outB), "처음 보는 얼굴이군.") {
		t.Fatal("second player must get first line")
	}
	flagA, flagB := -1, -1
	for _, p := range snap.Players {
		switch p.Username {
		case "갑":
			flagA = p.Flags[yworld.TalkFlag("tutor")]
		case "을":
			flagB = p.Flags[yworld.TalkFlag("tutor")]
		}
	}
	if flagA != 1 || flagB != 1 {
		t.Fatalf("flags a=%d b=%d snap=%+v", flagA, flagB, snap.Players)
	}
	snap.Players[0].Flags["tutor_talked"] = 99
	again := mustSnapshot(t, l)
	if again.Players[0].Flags["tutor_talked"] == 99 {
		t.Fatal("snapshot must copy Flags")
	}
}

func TestTalkMissingNPC(t *testing.T) {
	l := startStory(t)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	evs := drain(out)
	if !hasTextContains(evs, "여기는 그 사람이 없어요.") {
		t.Fatalf("missing npc: %#v", evs)
	}
	if textCode(evs, text.CodeNotFound) == nil {
		t.Fatal("want not_found")
	}
}

func TestLookShowsNPCAndExamine(t *testing.T) {
	l := startStory(t)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	_ = mustSnapshot(t, l)
	card := findRoom(drain(out))
	if card == nil || len(card.NPCs) != 1 || card.NPCs[0] != "훈장" {
		t.Fatalf("spawn npcs: %+v", card)
	}

	l.Submit(Look{ConnID: "a", Target: "훈장"})
	_ = mustSnapshot(t, l)
	if !hasTextContains(drain(out), "낡은 코트") {
		t.Fatal("examine npc")
	}

	l.Submit(Move{ConnID: "a", Dir: "north"})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Look{ConnID: "a", Target: "신문"})
	_ = mustSnapshot(t, l)
	got := drain(out)
	if !hasTextContains(got, "한벽일보") || !hasTextContains(got, "활자") || !hasTextContains(got, "두 번") {
		t.Fatalf("newspaper: %#v", got)
	}

	l.Submit(Look{ConnID: "a", Target: "훈장"})
	_ = mustSnapshot(t, l)
	if !hasTextContains(drain(out), "여기는 그 사람이 없어요.") {
		t.Fatal("npc in other room")
	}
	l.Submit(Look{ConnID: "a", Target: "없는것"})
	_ = mustSnapshot(t, l)
	if !hasTextContains(drain(out), "여기엔 그런 게 없어요.") {
		t.Fatal("missing object")
	}
}

func TestTalkFlagPersistsOnLeave(t *testing.T) {
	store := newMemSheets()
	l := startStorySink(t, store)
	mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "acc-1", Username: "갑", Session: "s1"})
	l.Submit(Talk{ConnID: "a", NPC: "훈장"})
	_ = mustSnapshot(t, l)
	l.Submit(LeaveWorld{ConnID: "a"})
	_ = mustSnapshot(t, l)
	saved := store.load("acc-1")
	if saved.Flags[yworld.TalkFlag("tutor")] != 1 {
		t.Fatalf("saved flags %+v", saved.Flags)
	}
	out := mustAttach(t, l, "b")
	l.Submit(EnterWorld{ConnID: "b", AccountID: "acc-1", Username: "갑", Session: "s2", Sheet: saved})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Talk{ConnID: "b", NPC: "훈장"})
	_ = mustSnapshot(t, l)
	if !hasTextContains(drain(out), "또 왔군, 자네.") {
		t.Fatal("reconnect should remember")
	}
}

func TestDalbitgolNewspaperAndCheongram(t *testing.T) {
	w, err := content.LoadWorld(filepath.Join("..", "..", "content"), yworld.SpawnID)
	if err != nil {
		t.Fatal(err)
	}
	l := NewWithWorld(discardLog(), w.Rooms, w.Items, w.Ground, nil).WithNPCs(w.NPCs).WithObjects(w.Objects)
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
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	_ = mustSnapshot(t, l)
	drain(out)
	for _, dir := range []string{"north", "north", "north", "north", "north", "east", "east"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap := mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:market" {
		t.Fatalf("want market, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Look{ConnID: "a", Target: "신문"})
	_ = mustSnapshot(t, l)
	paper := drain(out)
	if !hasTextContains(paper, "한벽일보") || !hasTextContains(paper, "활자") || !hasTextContains(paper, "두 번") {
		t.Fatalf("market newspaper: %#v", paper)
	}
	for _, dir := range []string{"south", "south", "south", "east"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:school" {
		t.Fatalf("want school, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	card := findRoom(drain(out))
	if card == nil || len(card.NPCs) == 0 || card.NPCs[0] != "청람 선생" {
		t.Fatalf("school npcs: %+v", card)
	}
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	first := drain(out)
	if !hasTextContains(first, "처음 보는 얼굴이군") || !hasTextContains(first, "자네") {
		t.Fatalf("cheongram first: %#v", first)
	}
	if hasTextContains(first, "월송") || hasTextContains(first, "쇠말뚝") || hasTextContains(first, "새벽회") {
		t.Fatalf("leaked secret: %#v", first)
	}
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	mem := drain(out)
	if !hasTextContains(mem, "또 왔군") {
		t.Fatalf("cheongram second: %#v", mem)
	}
	if mustSnapshot(t, l).Players[0].Flags[yworld.TalkFlag("cheongram")] != 1 {
		t.Fatal("cheongram_talked flag")
	}
}

func hasTextContains(evs []Event, sub string) bool {
	for _, ev := range evs {
		if tx, ok := ev.(Text); ok && strings.Contains(tx.Body, sub) {
			return true
		}
	}
	return false
}
