package engine

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/content"
	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

func TestGatherCraftSellPriceAndToll(t *testing.T) {
	l := startLivelihood(t, skill.Always)

	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	l.Submit(Gather{ConnID: "a", Skill: "캐다"})
	l.Submit(Gather{ConnID: "a", Query: "쑥"})
	l.Submit(Gather{ConnID: "a"})
	l.Submit(Gather{ConnID: "a"})
	snap := mustSnapshot(t, l)
	if qtyOf(snap.Players[0].Bag, "herb") != 3 {
		t.Fatalf("gather bag=%v", snap.Players[0].Bag)
	}
	if snap.Players[0].Skills["forage"] < 1 {
		t.Fatal("forage should grow")
	}
	evs := drain(out)
	if !hasBody(evs, text.T(text.Default, text.GatherOK, "쑥", "을")) {
		t.Fatalf("gather ok: %#v", evs)
	}
	if !hasBody(evs, text.T(text.Default, text.GatherEmpty)) {
		t.Fatalf("gather empty: %#v", evs)
	}

	l.Submit(Move{ConnID: "a", Dir: "north"})
	l.Submit(Gather{ConnID: "a", Skill: "캐다"})
	_ = mustSnapshot(t, l)
	if !hasBody(drain(out), text.T(text.Default, text.GatherNone)) {
		t.Fatal("yard has no node")
	}

	l.Submit(Quote{ConnID: "a"})
	_ = mustSnapshot(t, l)
	yardQuote := drain(out)
	if !hasBodyPrefix(yardQuote, "안마당 장터 시세") {
		t.Fatalf("yard quote: %#v", yardQuote)
	}
	yardHerb := quotePrice(t, yardQuote, "쑥")

	l.Submit(Sell{ConnID: "a", Query: "쑥", Qty: 2})
	snap = mustSnapshot(t, l)
	if qtyOf(snap.Players[0].Bag, "herb") != 1 || snap.Players[0].Nyang != yardHerb*2 {
		t.Fatalf("sell yard nyang=%d bag=%v want %d", snap.Players[0].Nyang, snap.Players[0].Bag, yardHerb*2)
	}
	sellEvs := drain(out)
	if !hasBody(sellEvs, text.T(text.Default, text.SellOK, "쑥", "을", yardHerb*2)) {
		t.Fatal("sell line")
	}
	if !hasBodyContains(sellEvs, "게시판 앞에서") || !hasBodyContains(sellEvs, "화물마당") {
		t.Fatalf("first sale ambient: %#v", sellEvs)
	}
	if snap.Players[0].Flags[yworld.FirstMarketSaleFlag] != 1 {
		t.Fatal("first_market_sale flag after sell")
	}
	l.Submit(Quote{ConnID: "a"})
	_ = mustSnapshot(t, l)
	after := quotePrice(t, drain(out), "쑥")
	if after >= yardHerb {
		t.Fatalf("sell must drop price %d → %d", yardHerb, after)
	}

	// Second market is dearer at boot; leftover herb sells higher at the shop.
	l.Submit(Move{ConnID: "a", Dir: "south"})
	l.Submit(Move{ConnID: "a", Dir: "east"})
	l.Submit(Quote{ConnID: "a"})
	_ = mustSnapshot(t, l)
	shopHerb := quotePrice(t, drain(out), "쑥")
	if shopHerb <= yardHerb {
		t.Fatalf("regional prices: yard=%d shop=%d", yardHerb, shopHerb)
	}
	l.Submit(Sell{ConnID: "a", Query: "herb", Qty: 1})
	snap = mustSnapshot(t, l)
	if qtyOf(snap.Players[0].Bag, "herb") != 0 || snap.Players[0].Nyang != yardHerb*2+shopHerb {
		t.Fatalf("shop sell %+v", snap.Players[0])
	}
	secondSell := drain(out)
	if hasBodyContains(secondSell, "게시판 앞에서") {
		t.Fatalf("second sell re-emitted ambient: %#v", secondSell)
	}

	// Craft nails at the forge shop.
	l.Submit(EnterWorld{ConnID: "b", AccountID: "2", Username: "을", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "ore", Qty: 2}}}})
	mustAttach(t, l, "b")
	l.Submit(Move{ConnID: "b", Dir: "east"})
	l.Submit(Craft{ConnID: "b", Query: "쇠못"})
	snap = mustSnapshot(t, l)
	var smith Player
	for _, p := range snap.Players {
		if p.ConnID == "b" {
			smith = p
		}
	}
	if qtyOf(smith.Bag, "nail") != 1 || qtyOf(smith.Bag, "ore") != 0 {
		t.Fatalf("craft bag=%v", smith.Bag)
	}
	if smith.Skills["smith"] < 1 {
		t.Fatal("craft should raise smith")
	}

	outC := mustAttach(t, l, "c")
	l.Submit(EnterWorld{ConnID: "c", AccountID: "3", Username: "병", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "ore", Qty: 1}}}})
	l.Submit(Craft{ConnID: "c", Query: "nail"})
	_ = mustSnapshot(t, l)
	if !hasBodyContains(drain(outC), "대장간에서") {
		t.Fatal("want forge gate at spawn")
	}
	l.Submit(Move{ConnID: "c", Dir: "east"})
	l.Submit(Craft{ConnID: "c", Query: "nail"})
	_ = mustSnapshot(t, l)
	if textCode(drain(outC), text.CodeNeedMat) == nil {
		t.Fatal("still short ore")
	}

	// Toll: bulk into checkpoint (shop), empty purse confiscates a good.
	outD := mustAttach(t, l, "d")
	l.Submit(EnterWorld{ConnID: "d", AccountID: "4", Username: "정", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "herb", Qty: 4}}}})
	l.Submit(Move{ConnID: "d", Dir: "east"})
	snap = mustSnapshot(t, l)
	var carrier Player
	for _, p := range snap.Players {
		if p.ConnID == "d" {
			carrier = p
		}
	}
	if qtyOf(carrier.Bag, "herb") != 3 {
		t.Fatalf("toll confiscate bag=%v", carrier.Bag)
	}
	if !hasBodyContains(drain(outD), "놓고 가라고") {
		t.Fatal("toll line")
	}

	outE := mustAttach(t, l, "e")
	l.Submit(EnterWorld{ConnID: "e", AccountID: "5", Username: "무", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: "herb", Qty: 4}}, Nyang: 10}})
	l.Submit(Move{ConnID: "e", Dir: "east"})
	snap = mustSnapshot(t, l)
	for _, p := range snap.Players {
		if p.ConnID == "e" {
			if p.Nyang != 10-TollNyang || qtyOf(p.Bag, "herb") != 4 {
				t.Fatalf("toll pay %+v", p)
			}
		}
	}
	if !hasBodyContains(drain(outE), "통행세") {
		t.Fatal("toll pay line")
	}

	l.Submit(Sheet{ConnID: "a"})
	_ = mustSnapshot(t, l)
	if !hasBodyContains(drain(out), "주머니:") {
		t.Fatal("wallet on sheet")
	}
}

func TestBuyAndCraftStation(t *testing.T) {
	l := startLivelihood(t, skill.Always)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Nyang: 80}})
	l.Submit(Quote{ConnID: "a"})
	_ = mustSnapshot(t, l)
	if textCode(drain(out), text.CodeNoMarket) == nil {
		t.Fatal("spawn is not a market")
	}
	l.Submit(Move{ConnID: "a", Dir: "east"})
	l.Submit(Quote{ConnID: "a"})
	_ = mustSnapshot(t, l)
	unit := quotePrice(t, drain(out), "쑥")
	l.Submit(Buy{ConnID: "a", Query: "쑥", Qty: 1})
	snap := mustSnapshot(t, l)
	if qtyOf(snap.Players[0].Bag, "herb") != 1 || snap.Players[0].Nyang != 80-unit {
		t.Fatalf("buy %+v unit=%d", snap.Players[0], unit)
	}

	l.Submit(Craft{ConnID: "a", Query: ""})
	_ = mustSnapshot(t, l)
	if !hasBodyContains(drain(out), "쇠못") {
		t.Fatal("list recipes at forge")
	}
	l.Submit(Move{ConnID: "a", Dir: "west"})
	l.Submit(Craft{ConnID: "a", Query: "쇠못"})
	_ = mustSnapshot(t, l)
	if !hasBodyContains(drain(out), "대장간에서") {
		t.Fatal("need forge")
	}
}

func TestParseNameQty(t *testing.T) {
	name, n := ParseNameQty("쑥 3")
	if name != "쑥" || n != 3 {
		t.Fatalf("%q %d", name, n)
	}
	name, n = ParseNameQty("쇠못")
	if name != "쇠못" || n != 1 {
		t.Fatalf("%q %d", name, n)
	}
	name, n = ParseNameQty("")
	if name != "" || n != 1 {
		t.Fatalf("%q %d", name, n)
	}
}

func startLivelihood(t *testing.T, rng func() float64) *Loop {
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
	l := NewWithWorld(discardLog(), w.Rooms, w.Items, w.Ground, nil).
		WithSkills(cat).WithCraft(liv.Craft).WithMarkets(liv.Markets).WithRand(rng)
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
	return l
}

func hasBodyPrefix(evs []Event, prefix string) bool {
	for _, ev := range evs {
		if tx, ok := ev.(Text); ok && strings.HasPrefix(tx.Body, prefix) {
			return true
		}
	}
	return false
}

func hasBodyContains(evs []Event, sub string) bool {
	for _, ev := range evs {
		if tx, ok := ev.(Text); ok && strings.Contains(tx.Body, sub) {
			return true
		}
	}
	return false
}

func quotePrice(t *testing.T, evs []Event, name string) int {
	t.Helper()
	for _, ev := range evs {
		tx, ok := ev.(Text)
		if !ok {
			continue
		}
		if !strings.HasPrefix(tx.Body, name+" ") {
			continue
		}
		var n int
		if _, err := parseQuoteNyang(tx.Body, &n); err != nil || n < 1 {
			t.Fatalf("parse %q", tx.Body)
		}
		return n
	}
	t.Fatalf("no quote for %s in %#v", name, evs)
	return 0
}

func parseQuoteNyang(line string, n *int) (int, error) {
	// "쑥 2냥 — 오늘은 흔하다"
	rest := line
	if i := strings.Index(rest, " "); i >= 0 {
		rest = rest[i+1:]
	}
	if i := strings.Index(rest, "냥"); i >= 0 {
		rest = rest[:i]
	}
	return 0, scanInt(rest, n)
}

func scanInt(s string, n *int) error {
	s = strings.TrimSpace(s)
	var v int
	for _, r := range s {
		if r < '0' || r > '9' {
			return errNotInt
		}
		v = v*10 + int(r-'0')
	}
	if s == "" {
		return errNotInt
	}
	*n = v
	return nil
}

type errNotIntT struct{}

func (errNotIntT) Error() string { return "not int" }

var errNotInt errNotIntT
