package engine

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pjhwa/yeomyeong/internal/content"
	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

func TestHideSuccessAtCheckpoint(t *testing.T) {
	l := startLivelihood(t, skill.Always)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "leaflet", Qty: 1}}}})
	l.Submit(Move{ConnID: "a", Dir: "east"}) // shop = checkpoint
	_ = mustSnapshot(t, l)
	_ = drain(out)

	l.Submit(Hide{ConnID: "a", Query: "전단"})
	snap := mustSnapshot(t, l)
	p := snap.Players[0]
	if qtyOf(p.Bag, "leaflet") != 1 {
		t.Fatalf("keep leaflet on success: %v", p.Bag)
	}
	if p.Flags[yworld.DawnScentFlag] != 1 || p.Flags[yworld.SmuggleSuccessCountFlag] != 1 || p.Flags[yworld.SmugglePassFlag] != 1 {
		t.Fatalf("flags %+v", p.Flags)
	}
	evs := drain(out)
	if !hasBodyContains(evs, "자루 밑에") {
		t.Fatalf("success line: %#v", evs)
	}
}

func TestHideFailConfiscatesContraband(t *testing.T) {
	l := startLivelihood(t, skill.Never)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "leaflet", Qty: 1}}}})
	l.Submit(Move{ConnID: "a", Dir: "east"})
	_ = mustSnapshot(t, l)
	_ = drain(out)

	l.Submit(Hide{ConnID: "a"})
	snap := mustSnapshot(t, l)
	if qtyOf(snap.Players[0].Bag, "leaflet") != 0 {
		t.Fatalf("confiscate want empty bag got %v", snap.Players[0].Bag)
	}
	if snap.Players[0].Flags[yworld.DawnScentFlag] != 0 {
		t.Fatal("no dawn_scent on fail")
	}
	if !hasBodyContains(drain(out), "빼앗아") {
		t.Fatal("fail confiscate line")
	}
}

func TestHidePassSkipsToll(t *testing.T) {
	l := startLivelihood(t, skill.Always)
	out := mustAttach(t, l, "a")
	// Start at shop (enter via move from start), hide, leave west, re-enter east with bulk.
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{
			Bag:  []yworld.Stack{{ID: "leaflet", Qty: 1}, {ID: "herb", Qty: 4}},
			Nyang: 10,
		}})
	l.Submit(Move{ConnID: "a", Dir: "east"})
	snap := mustSnapshot(t, l)
	// First enter may toll (Always forces hit); pay 2.
	if snap.Players[0].Nyang != 10-TollNyang {
		t.Fatalf("first enter toll nyang=%d bag=%v", snap.Players[0].Nyang, snap.Players[0].Bag)
	}
	_ = drain(out)
	l.Submit(Hide{ConnID: "a", Query: "leaflet"})
	_ = mustSnapshot(t, l)
	_ = drain(out)
	l.Submit(Move{ConnID: "a", Dir: "west"})
	_ = mustSnapshot(t, l)
	_ = drain(out)
	before := mustSnapshot(t, l).Players[0].Nyang
	l.Submit(Move{ConnID: "a", Dir: "east"})
	snap = mustSnapshot(t, l)
	if snap.Players[0].Nyang != before {
		t.Fatalf("smuggle_pass should skip toll: before=%d after=%d", before, snap.Players[0].Nyang)
	}
	if qtyOf(snap.Players[0].Bag, "herb") != 4 || qtyOf(snap.Players[0].Bag, "leaflet") != 1 {
		t.Fatalf("bag unchanged %+v", snap.Players[0].Bag)
	}
	if snap.Players[0].Flags[yworld.SmugglePassFlag] != 0 {
		t.Fatal("pass consumed")
	}
	evs := drain(out)
	if hasBodyContains(evs, "통행세") || hasBodyContains(evs, "놓고 가라고") {
		t.Fatalf("no toll line after pass: %#v", evs)
	}
}

func TestHideOutsideCheckpoint(t *testing.T) {
	l := startLivelihood(t, skill.Always)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "leaflet", Qty: 1}}}})
	l.Submit(Hide{ConnID: "a"})
	_ = mustSnapshot(t, l)
	if textCode(drain(out), text.CodeNoHide) == nil {
		t.Fatal("want no_hide at spawn")
	}
}

func TestLeafletDawnSellBonusOnce(t *testing.T) {
	l := startLivelihood(t, skill.Always)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{
			Bag:   []yworld.Stack{{ID: "leaflet", Qty: 2}},
			Flags: map[string]int{yworld.DawnScentFlag: 1},
		}})
	l.Submit(Move{ConnID: "a", Dir: "east"})
	l.Submit(Quote{ConnID: "a"})
	_ = mustSnapshot(t, l)
	unit := quotePrice(t, drain(out), "전단")
	l.Submit(Sell{ConnID: "a", Query: "전단", Qty: 1})
	snap := mustSnapshot(t, l)
	want := unit + LeafletDawnBonus
	if snap.Players[0].Nyang != want {
		t.Fatalf("dawn bonus nyang=%d want %d", snap.Players[0].Nyang, want)
	}
	if snap.Players[0].Flags[yworld.LeafletDawnBonusFlag] != 1 {
		t.Fatal("bonus flag")
	}
	_ = drain(out)
	l.Submit(Quote{ConnID: "a"})
	_ = mustSnapshot(t, l)
	unit2 := quotePrice(t, drain(out), "전단")
	l.Submit(Sell{ConnID: "a", Query: "전단", Qty: 1})
	snap = mustSnapshot(t, l)
	if snap.Players[0].Nyang != want+unit2 {
		t.Fatalf("second sell no bonus: nyang=%d want %d (unit2=%d)", snap.Players[0].Nyang, want+unit2, unit2)
	}
}

func TestHideNonContrabandStillTolls(t *testing.T) {
	l := startLivelihood(t, skill.Always)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{
			Bag:  []yworld.Stack{{ID: "herb", Qty: 4}},
			Nyang: 10,
		}})
	l.Submit(Move{ConnID: "a", Dir: "east"})
	snap := mustSnapshot(t, l)
	if snap.Players[0].Nyang != 10-TollNyang {
		t.Fatalf("non-contraband bulk still tolls: nyang=%d bag=%v", snap.Players[0].Nyang, snap.Players[0].Bag)
	}
	if !hasBodyContains(drain(out), "통행세") {
		t.Fatal("toll pay line for ordinary goods")
	}
	l.Submit(Hide{ConnID: "a", Query: "쑥"})
	_ = mustSnapshot(t, l)
	evs := drain(out)
	if !hasBodyContains(evs, "민감한 짐") {
		t.Fatalf("hide rejects non-contraband: %#v", evs)
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].Flags[yworld.DawnScentFlag] != 0 || snap.Players[0].Flags[yworld.SmugglePassFlag] != 0 {
		t.Fatalf("no smuggle flags without contraband: %+v", snap.Players[0].Flags)
	}
}

func TestHideFailAllowsRetry(t *testing.T) {
	l := startLivelihood(t, skill.Never)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "leaflet", Qty: 1}}}})
	l.Submit(Move{ConnID: "a", Dir: "east"})
	_ = mustSnapshot(t, l)
	_ = drain(out)
	l.Submit(Hide{ConnID: "a"})
	snap := mustSnapshot(t, l)
	if qtyOf(snap.Players[0].Bag, "leaflet") != 0 {
		t.Fatal("confiscated")
	}
	_ = drain(out)
	// Soft-lock guard: player can still act; hide again reports need, not hang.
	l.Submit(Hide{ConnID: "a"})
	_ = mustSnapshot(t, l)
	if !hasBodyContains(drain(out), "민감한 짐") {
		t.Fatal("after confiscation hide should ask for contraband, not soft-lock")
	}
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	if len(drain(out)) == 0 {
		t.Fatal("look still works after fail")
	}
}


func TestDawnScentSoftHookCheongramAndMarket(t *testing.T) {
	l, out := startDalbitgol(t)
	l.Submit(EnterWorld{
		ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Flags: map[string]int{
			yworld.DawnScentFlag:       1,
			yworld.FirstMarketSaleFlag: 1,
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
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	marketEvs := drain(out)
	card := findRoom(marketEvs)
	if card == nil || !strings.Contains(card.Description, "잉크 묻은 손") || !strings.Contains(card.Description, "새벽 전에") {
		t.Fatalf("market dawn_scent description: %+v", card)
	}
	assertNoSecrets(t, marketEvs)

	for _, dir := range []string{"south", "south", "south", "east"} {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap = mustSnapshot(t, l)
	if snap.Players[0].RoomID != "dalbitgol:school" {
		t.Fatalf("want school, at %s", snap.Players[0].RoomID)
	}
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	talk := drain(out)
	if !hasTextContains(talk, "잉크 냄새가 옷에") || !hasTextContains(talk, "새벽이") {
		t.Fatalf("cheongram dawn_scent talk: %#v", talk)
	}
	if hasTextContains(talk, "처음 보는 얼굴이군") || hasTextContains(talk, "다방에도 물어보게") {
		t.Fatalf("wrong talk line under dawn_scent: %#v", talk)
	}
	assertNoSecrets(t, talk)
}

func TestHideFailFine(t *testing.T) {
	l := idleLivelihood(t)
	out := make(chan Event, 16)
	l.outbound["a"] = out
	p := Player{ConnID: "a", Username: "갑", RoomID: "test:shop", Nyang: SmuggleFineNyang + 1, Flags: map[string]int{}}
	l.applyHideFail(&p, "leaflet")
	if p.Nyang != 1 {
		t.Fatalf("fine nyang=%d want 1", p.Nyang)
	}
	if !hasBodyContains(drain(out), "벌금") {
		t.Fatal("fine line")
	}
}

func TestHideFailAmbientWhenBroke(t *testing.T) {
	l := idleLivelihood(t)
	out := make(chan Event, 16)
	l.outbound["a"] = out
	p := Player{ConnID: "a", Username: "갑", RoomID: "test:shop", Flags: map[string]int{}}
	l.applyHideFail(&p, "leaflet")
	if p.Nyang != 0 {
		t.Fatalf("nyang=%d", p.Nyang)
	}
	if !hasBodyContains(drain(out), "다행히") {
		t.Fatal("ambient when broke")
	}
}

func idleLivelihood(t *testing.T) *Loop {
	t.Helper()
	root := filepath.Join("..", "content", "testdata", "valid")
	w, err := content.LoadWorld(root, "test:start")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := skill.Load(filepath.Join("..", "..", "content", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	liv, err := content.LoadLivelihood(root, w.Rooms, w.Items, cat)
	if err != nil {
		t.Fatal(err)
	}
	return NewWithWorld(discardLog(), w.Rooms, w.Items, w.Ground, nil).
		WithSkills(cat).WithCraft(liv.Craft).WithMarkets(liv.Markets).WithRand(skill.Never)
}
