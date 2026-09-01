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

func startDalbitgol(t *testing.T) (*Loop, <-chan Event) {
	t.Helper()
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
	return l, out
}

func TestDalbitgolNewspaperAndCheongram(t *testing.T) {
	l, out := startDalbitgol(t)
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
	assertNoSecrets(t, first)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	mem := drain(out)
	if !hasTextContains(mem, "또 왔군") {
		t.Fatalf("cheongram second: %#v", mem)
	}
	if !hasTextContains(mem, "만석상회 창고") || !hasTextContains(mem, "짐이 남아") {
		t.Fatalf("warehouse nudge: %#v", mem)
	}
	if mustSnapshot(t, l).Players[0].Flags[yworld.TalkFlag("cheongram")] != 1 {
		t.Fatal("cheongram_talked flag")
	}

	for _, dir := range []string{"west", "north", "north", "north"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:market" {
		t.Fatalf("back to market, at %s", snap.Players[0].RoomID)
	}
	for _, dir := range []string{"east", "east", "east"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:warehouse" {
		t.Fatalf("want warehouse, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	wh := findRoom(drain(out))
	if wh == nil || !hasString(wh.Objects, "보부상 봇짐") {
		t.Fatalf("warehouse objects: %+v", wh)
	}
	l.Submit(Look{ConnID: "a", Target: "짐"})
	_ = mustSnapshot(t, l)
	pack := drain(out)
	if !hasTextContains(pack, "거짓 바닥") || !hasTextContains(pack, "찻잔") || !hasTextContains(pack, "강포") {
		t.Fatalf("pack clue: %#v", pack)
	}
	if !hasTextContains(pack, "한벽일보") || !hasTextContains(pack, "활자") || !hasTextContains(pack, "무허가") {
		t.Fatalf("pack newspaper echo: %#v", pack)
	}
	if !hasTextContains(pack, "수레 축") {
		t.Fatalf("pack reaction: %#v", pack)
	}
	if !hasTextContains(pack, "쑥") || !hasTextContains(pack, "시세") {
		t.Fatalf("pack livelihood bridge: %#v", pack)
	}
	assertNoSecrets(t, pack)
	if mustSnapshot(t, l).Players[0].Flags[yworld.ExaminedFlag("gangpo-pack")] != 1 {
		t.Fatal("examined:gangpo-pack flag")
	}
	l.Submit(Look{ConnID: "a", Target: "보부상"})
	_ = mustSnapshot(t, l)
	again := drain(out)
	if !hasTextContains(again, "거짓 바닥") {
		t.Fatalf("second examine missing description: %#v", again)
	}
	if hasTextContains(again, "수레 축") {
		t.Fatalf("second examine repeated reaction: %#v", again)
	}

	l.Submit(Move{ConnID: "a", Dir: "west"})
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:warehouse-lane" {
		t.Fatalf("want warehouse-lane, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	lane := findRoom(drain(out))
	if lane == nil || !hasString(lane.Objects, "수레 자국") {
		t.Fatalf("lane objects: %+v", lane)
	}
	l.Submit(Look{ConnID: "a", Target: "자국"})
	_ = mustSnapshot(t, l)
	ruts := drain(out)
	if !hasTextContains(ruts, "흐릿") {
		t.Fatalf("cart-ruts bland before sale: %#v", ruts)
	}
	if hasTextContains(ruts, "화물마당") {
		t.Fatalf("cart-ruts concrete without sale: %#v", ruts)
	}
	if !hasTextContains(ruts, "덧문") {
		t.Fatalf("cart-ruts reaction: %#v", ruts)
	}
	assertNoSecrets(t, ruts)
	if mustSnapshot(t, l).Players[0].Flags[yworld.ExaminedFlag("cart-ruts")] != 1 {
		t.Fatal("examined:cart-ruts flag")
	}

	l.Submit(Move{ConnID: "a", Dir: "south"})
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:packing-shed" {
		t.Fatalf("want packing-shed, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	shed := findRoom(drain(out))
	if shed == nil || !hasString(shed.Objects, "화물 꼬리표") || !hasString(shed.NPCs, "오씨 점원") {
		t.Fatalf("shed card: %+v", shed)
	}
	l.Submit(Look{ConnID: "a", Target: "꼬리표"})
	_ = mustSnapshot(t, l)
	chit := drain(out)
	if !hasTextContains(chit, "잘 읽히지") || !hasTextContains(chit, "꼬리표") {
		t.Fatalf("cargo-chit bland before sale: %#v", chit)
	}
	if hasTextContains(chit, "확인하라고") || hasTextContains(chit, "만석상회 도장") {
		t.Fatalf("cargo-chit concrete without sale: %#v", chit)
	}
	assertNoSecrets(t, chit)

	l.Submit(Talk{ConnID: "a", NPC: "오씨"})
	_ = mustSnapshot(t, l)
	clerk := drain(out)
	if !hasTextContains(clerk, "시세") || !hasTextContains(clerk, "쑥") {
		t.Fatalf("clerk pack-when (시세 nudge): %#v", clerk)
	}
	if hasTextContains(clerk, "새벽이 오기 전") || hasTextContains(clerk, "다방 백야 쪽에서도") {
		t.Fatalf("clerk concrete without sale: %#v", clerk)
	}
	assertNoSecrets(t, clerk)
	if mustSnapshot(t, l).Players[0].Flags[yworld.TalkFlag("clerk-oh")] != 1 {
		t.Fatal("clerk-oh_talked flag")
	}

	for _, dir := range []string{"south", "south", "west"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:school" {
		t.Fatalf("want school for gated 청람, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	gated := drain(out)
	if !hasTextContains(gated, "시세") || !hasTextContains(gated, "쑥") {
		t.Fatalf("cheongram pack-when (시세 nudge): %#v", gated)
	}
	if hasTextContains(gated, "다방에도 물어보게") || hasTextContains(gated, "화물마당") {
		t.Fatalf("cheongram concrete without sale: %#v", gated)
	}
	if hasTextContains(gated, "또 왔군") {
		t.Fatalf("cheongram when reused second: %#v", gated)
	}
	assertNoSecrets(t, gated)
}

func TestCourierTrailOpensAfterSale(t *testing.T) {
	l, out := startDalbitgol(t)
	l.Submit(EnterWorld{
		ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Flags: map[string]int{
			yworld.ExaminedFlag("gangpo-pack"): 1,
			yworld.FirstMarketSaleFlag:         1,
		}},
	})
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
	if !hasTextContains(paper, "정거장 화물마당") || !hasTextContains(paper, "수레 자국") {
		t.Fatalf("newspaper after sale: %#v", paper)
	}
	assertNoSecrets(t, paper)

	for _, dir := range []string{"east", "east"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:warehouse-lane" {
		t.Fatalf("want warehouse-lane, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Look{ConnID: "a", Target: "자국"})
	_ = mustSnapshot(t, l)
	ruts := drain(out)
	if !hasTextContains(ruts, "짚") || !hasTextContains(ruts, "기름") || !hasTextContains(ruts, "화물마당") {
		t.Fatalf("cart-ruts after sale: %#v", ruts)
	}
	assertNoSecrets(t, ruts)

	l.Submit(Move{ConnID: "a", Dir: "south"})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Look{ConnID: "a", Target: "꼬리표"})
	_ = mustSnapshot(t, l)
	chit := drain(out)
	if !hasTextContains(chit, "만석상회") || !hasTextContains(chit, "확인하라고") || !hasTextContains(chit, "정거장") {
		t.Fatalf("cargo-chit after sale: %#v", chit)
	}
	assertNoSecrets(t, chit)

	l.Submit(Talk{ConnID: "a", NPC: "오씨"})
	_ = mustSnapshot(t, l)
	clerk := drain(out)
	if !hasTextContains(clerk, "새벽이 오기 전") || !hasTextContains(clerk, "화물마당") || !hasTextContains(clerk, "다방 백야") {
		t.Fatalf("clerk concrete after sale: %#v", clerk)
	}
	assertNoSecrets(t, clerk)

	for _, dir := range []string{"south", "south", "west"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:school" {
		t.Fatalf("want school, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	gated := drain(out)
	if !hasTextContains(gated, "수레 자국") || !hasTextContains(gated, "다방에도 물어보게") || !hasTextContains(gated, "화물마당") {
		t.Fatalf("cheongram concrete after sale: %#v", gated)
	}
	assertNoSecrets(t, gated)
}

func TestTalkWhenFromTestdata(t *testing.T) {
	l := startStory(t)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	_ = mustSnapshot(t, l)
	drain(out)

	l.Submit(Talk{ConnID: "a", NPC: "훈장"})
	_ = mustSnapshot(t, l)
	if !hasTextContains(drain(out), "처음 보는 얼굴이군.") {
		t.Fatal("ungated first")
	}

	l.Submit(Move{ConnID: "a", Dir: "north"})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Look{ConnID: "a", Target: "신문"})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Move{ConnID: "a", Dir: "south"})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "훈장"})
	_ = mustSnapshot(t, l)
	got := drain(out)
	if !hasTextContains(got, "신문을 봤군, 자네.") {
		t.Fatalf("when after examine: %#v", got)
	}
	if hasTextContains(got, "또 왔군") {
		t.Fatalf("when reused second: %#v", got)
	}
}

func TestTalkWhenFirstMatchAndSetsTalked(t *testing.T) {
	l := startStory(t)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{
		ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Flags: map[string]int{"other-flag": 1, "examined:test-paper": 1}},
	})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "훈장"})
	_ = mustSnapshot(t, l)
	got := drain(out)
	if !hasTextContains(got, "신문을 봤군, 자네.") {
		t.Fatalf("first when should win: %#v", got)
	}
	if hasTextContains(got, "다른 단서를 봤군") {
		t.Fatalf("later when fired: %#v", got)
	}
	if mustSnapshot(t, l).Players[0].Flags[yworld.TalkFlag("tutor")] != 1 {
		t.Fatal("when-line must still set talked")
	}

	outB := mustAttach(t, l, "b")
	l.Submit(EnterWorld{
		ConnID: "b", AccountID: "2", Username: "을", Session: "s2",
		Sheet: yworld.Sheet{Flags: map[string]int{"other-flag": 1}},
	})
	_ = mustSnapshot(t, l)
	drain(outB)
	l.Submit(Talk{ConnID: "b", NPC: "훈장"})
	_ = mustSnapshot(t, l)
	secondWhen := drain(outB)
	if !hasTextContains(secondWhen, "다른 단서를 봤군") {
		t.Fatalf("second when: %#v", secondWhen)
	}
	if hasTextContains(secondWhen, "신문을 봤군") {
		t.Fatalf("first when without its flag: %#v", secondWhen)
	}
}

func TestCourierTrailGatedTalkNeedsPack(t *testing.T) {
	l, out := startDalbitgol(t)
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	_ = mustSnapshot(t, l)
	drain(out)
	for _, dir := range []string{"north", "north", "north", "north", "north", "east", "east", "east", "east", "south"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap := mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:packing-shed" {
		t.Fatalf("want packing-shed, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "오씨"})
	_ = mustSnapshot(t, l)
	first := drain(out)
	if !hasTextContains(first, "포장간") {
		t.Fatalf("clerk first: %#v", first)
	}
	if hasTextContains(first, "강포") || hasTextContains(first, "새벽이 오기 전") || hasTextContains(first, "다방 백야 쪽에서도") {
		t.Fatalf("clerk when without pack: %#v", first)
	}
	l.Submit(Talk{ConnID: "a", NPC: "오씨"})
	_ = mustSnapshot(t, l)
	second := drain(out)
	if !hasTextContains(second, "또 오셨어요") {
		t.Fatalf("clerk second: %#v", second)
	}
	if hasTextContains(second, "강포") || hasTextContains(second, "다방 백야 쪽에서도") {
		t.Fatalf("clerk second leaked when: %#v", second)
	}

	for _, dir := range []string{"south", "south", "west"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:school" {
		t.Fatalf("want school, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	cfirst := drain(out)
	if !hasTextContains(cfirst, "처음 보는 얼굴이군") {
		t.Fatalf("cheongram first: %#v", cfirst)
	}
	if hasTextContains(cfirst, "다방에도 물어보게") || hasTextContains(cfirst, "창고 짐을 봤군") {
		t.Fatalf("cheongram when without pack: %#v", cfirst)
	}
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	csecond := drain(out)
	if !hasTextContains(csecond, "또 왔군") {
		t.Fatalf("cheongram second: %#v", csecond)
	}
	if hasTextContains(csecond, "다방에도 물어보게") || hasTextContains(csecond, "수레 자국") {
		t.Fatalf("cheongram second leaked when: %#v", csecond)
	}
	assertNoSecrets(t, first, second, cfirst, csecond)
}

func TestExamineReactionOnce(t *testing.T) {
	l := startStory(t)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	_ = mustSnapshot(t, l)
	drain(out)
	l.Submit(Move{ConnID: "a", Dir: "north"})
	_ = mustSnapshot(t, l)
	card := findRoom(drain(out))
	if card == nil || !hasString(card.Objects, "한벽일보") {
		t.Fatalf("yard objects: %+v", card)
	}

	l.Submit(Look{ConnID: "a", Target: "신문"})
	_ = mustSnapshot(t, l)
	first := drain(out)
	if !hasTextContains(first, "한벽일보") || !hasTextContains(first, "사다리") {
		t.Fatalf("first examine: %#v", first)
	}
	if countTextContains(first, "사다리") != 1 {
		t.Fatalf("reaction count: %#v", first)
	}
	if mustSnapshot(t, l).Players[0].Flags[yworld.ExaminedFlag("test-paper")] != 1 {
		t.Fatal("examined:test-paper flag")
	}

	l.Submit(Look{ConnID: "a", Target: "게시판"})
	_ = mustSnapshot(t, l)
	second := drain(out)
	if !hasTextContains(second, "한벽일보") {
		t.Fatalf("second examine missing description: %#v", second)
	}
	if hasTextContains(second, "사다리") {
		t.Fatalf("second examine repeated reaction: %#v", second)
	}
}

func hasTextContains(evs []Event, sub string) bool {
	return countTextContains(evs, sub) > 0
}

func countTextContains(evs []Event, sub string) int {
	n := 0
	for _, ev := range evs {
		if tx, ok := ev.(Text); ok && strings.Contains(tx.Body, sub) {
			n++
		}
	}
	return n
}

func hasString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

var storySecrets = []string{"월송", "쇠말뚝", "새벽회", "한도규", "서월향", "월향", "무쇠 심장", "분맥"}

func assertNoSecrets(t *testing.T, batches ...[]Event) {
	t.Helper()
	for _, evs := range batches {
		for _, secret := range storySecrets {
			if hasTextContains(evs, secret) {
				t.Fatalf("leaked secret %q: %#v", secret, evs)
			}
		}
	}
}
