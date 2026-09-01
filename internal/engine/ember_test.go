package engine

import (
	"strings"
	"testing"

	"github.com/pjhwa/yeomyeong/internal/skill"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

func TestEmberTalkPathCafeHand(t *testing.T) {
	l, out := startDalbitgol(t)
	enterFlags(t, l, out, map[string]int{
		yworld.ExaminedFlag("gangpo-pack"): 1,
		yworld.FirstMarketSaleFlag:         1,
	})
	walkTo(t, l, "dalbitgol:cafe-baekya", "north", "north", "north", "north", "north", "east", "east", "south", "south")
	drain(out)
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	card := findRoom(drain(out))
	if card == nil || !hasString(card.NPCs, "점원") {
		t.Fatalf("cafe npcs: %+v", card)
	}
	l.Submit(Talk{ConnID: "a", NPC: "점원"})
	snap := mustSnapshot(t, l)
	talk := drain(out)
	if snap.Players[0].Flags[yworld.EmberFlag] != 1 {
		t.Fatalf("ember after cafe-hand: %+v", snap.Players[0].Flags)
	}
	if !hasTextContains(talk, "보부상") && !hasTextContains(talk, "강포") {
		t.Fatalf("listening/peddler vibe: %#v", talk)
	}
	if !hasTextContains(talk, "듣") && !hasTextContains(talk, "끄덕") {
		t.Fatalf("acknowledgment: %#v", talk)
	}
	if hasTextContains(talk, "퀘스트") || hasTextContains(talk, "완료") {
		t.Fatalf("mission tone: %#v", talk)
	}
	assertNoSecrets(t, talk)
	if snap.Players[0].Flags[yworld.TalkFlag("cafe-hand")] != 1 {
		t.Fatal("cafe-hand_talked")
	}

	l.Submit(Talk{ConnID: "a", NPC: "손님"})
	_ = mustSnapshot(t, l)
	again := drain(out)
	if !hasTextContains(again, "우물길") {
		t.Fatalf("ember soft-hook on second cafe talk: %#v", again)
	}
	assertNoSecrets(t, again)
}

func TestEmberSkillPathDropLeaflet(t *testing.T) {
	l, out := startDalbitgol(t)
	enterFlagsBag(t, l, out, map[string]int{
		yworld.ExaminedFlag("gangpo-pack"): 1,
		yworld.FirstMarketSaleFlag:         1,
	}, []yworld.Stack{{ID: "leaflet", Qty: 1}})
	walkTo(t, l, "dalbitgol:packing-shed", "north", "north", "north", "north", "north", "east", "east", "east", "east", "south")
	drain(out)
	l.Submit(DropItem{ConnID: "a", ItemID: "전단"})
	snap := mustSnapshot(t, l)
	evs := drain(out)
	if snap.Players[0].Flags[yworld.EmberFlag] != 1 {
		t.Fatalf("ember after drop: %+v", snap.Players[0].Flags)
	}
	if qtyOf(snap.Players[0].Bag, "leaflet") != 0 {
		t.Fatalf("leaflet should stay on ground: bag=%v", snap.Players[0].Bag)
	}
	if qtyOf(snap.Ground["dalbitgol:packing-shed"], "leaflet") != 1 {
		t.Fatalf("ground leaflet: %v", snap.Ground["dalbitgol:packing-shed"])
	}
	if !hasTextContains(evs, "우물길") || !hasTextContains(evs, "다방") {
		t.Fatalf("drop ambient: %#v", evs)
	}
	assertNoSecrets(t, evs)

	l.Submit(DropItem{ConnID: "a", ItemID: "전단"})
	_ = mustSnapshot(t, l)
	second := drain(out)
	if countTextContains(second, "우물길") != 0 {
		t.Fatalf("ember ambient once: %#v", second)
	}
}

func TestEmberSkillPathDropLeafletAtCafe(t *testing.T) {
	l, out := startDalbitgol(t)
	enterFlagsBag(t, l, out, map[string]int{
		yworld.ExaminedFlag("gangpo-pack"): 1,
		yworld.DawnScentFlag:               1,
	}, []yworld.Stack{{ID: "leaflet", Qty: 1}})
	walkTo(t, l, "dalbitgol:cafe-baekya", "north", "north", "north", "north", "north", "east", "east", "south", "south")
	drain(out)
	l.Submit(DropItem{ConnID: "a", ItemID: "leaflet"})
	snap := mustSnapshot(t, l)
	evs := drain(out)
	if snap.Players[0].Flags[yworld.EmberFlag] != 1 {
		t.Fatalf("ember after cafe drop: %+v", snap.Players[0].Flags)
	}
	if !hasTextContains(evs, "전단") {
		t.Fatalf("drop ok: %#v", evs)
	}
	assertNoSecrets(t, evs)
}

func TestEmberRiskPathCheongramTalk(t *testing.T) {
	l, out := startDalbitgol(t)
	enterFlags(t, l, out, map[string]int{
		yworld.ExaminedFlag("gangpo-pack"):  1,
		yworld.DawnScentFlag:                1,
		yworld.SmuggleSuccessCountFlag:      1,
	})
	walkTo(t, l, "dalbitgol:school", "north", "north", "north", "north", "north", "east", "east", "south", "south", "south", "east")
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	snap := mustSnapshot(t, l)
	talk := drain(out)
	if snap.Players[0].Flags[yworld.EmberFlag] != 1 {
		t.Fatalf("ember after cheongram risk: %+v", snap.Players[0].Flags)
	}
	if !hasTextContains(talk, "이제 그쪽에서 자네를 부른다") {
		t.Fatalf("risk quote: %#v", talk)
	}
	if hasTextContains(talk, "잉크 냄새가 옷에") {
		t.Fatalf("dawn_scent line won over risk: %#v", talk)
	}
	assertNoSecrets(t, talk)

	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	_ = mustSnapshot(t, l)
	hook := drain(out)
	if !hasTextContains(hook, "우물길") || !hasTextContains(hook, "다방 문") {
		t.Fatalf("ember soft-hook after risk: %#v", hook)
	}
	if hasTextContains(hook, "이제 그쪽에서 자네를 부른다") {
		t.Fatalf("risk line repeated: %#v", hook)
	}
	assertNoSecrets(t, hook)
}

func TestEmberRiskPathHideLeaflet(t *testing.T) {
	l, out := startDalbitgolWith(t, skill.Always)
	enterFlagsBag(t, l, out, map[string]int{
		yworld.ExaminedFlag("gangpo-pack"): 1,
	}, []yworld.Stack{{ID: "leaflet", Qty: 1}})
	walkTo(t, l, "dalbitgol:station", "north", "north", "north", "north", "north", "east", "east", "south", "south", "east", "east", "east")
	drain(out)
	l.Submit(Hide{ConnID: "a", Query: "전단"})
	snap := mustSnapshot(t, l)
	evs := drain(out)
	p := snap.Players[0]
	if p.Flags[yworld.DawnScentFlag] != 1 || p.Flags[yworld.SmuggleSuccessCountFlag] != 1 {
		t.Fatalf("hide flags %+v", p.Flags)
	}
	if p.Flags[yworld.EmberFlag] != 1 {
		t.Fatalf("ember after hide: %+v", p.Flags)
	}
	if qtyOf(p.Bag, "leaflet") != 1 {
		t.Fatalf("keep leaflet: %v", p.Bag)
	}
	if !hasTextContains(evs, "우물길") {
		t.Fatalf("hide ember ambient: %#v", evs)
	}
	assertNoSecrets(t, evs)
}

func TestEmberRequiresPrereq(t *testing.T) {
	l, out := startDalbitgol(t)
	enterFlagsBag(t, l, out, map[string]int{
		yworld.FirstMarketSaleFlag:    1,
		yworld.DawnScentFlag:          1,
		yworld.SmuggleSuccessCountFlag: 1,
	}, []yworld.Stack{{ID: "leaflet", Qty: 2}})
	walkTo(t, l, "dalbitgol:cafe-baekya", "north", "north", "north", "north", "north", "east", "east", "south", "south")
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "점원"})
	snap := mustSnapshot(t, l)
	talk := drain(out)
	if snap.Players[0].Flags[yworld.EmberFlag] != 0 {
		t.Fatal("no ember without pack examine")
	}
	if hasTextContains(talk, "보부상") || hasTextContains(talk, "강포") {
		t.Fatalf("bland cafe chatter leaked when: %#v", talk)
	}
	if !hasTextContains(talk, "보리차") {
		t.Fatalf("cafe first: %#v", talk)
	}
	assertNoSecrets(t, talk)

	l.Submit(DropItem{ConnID: "a", ItemID: "leaflet"})
	snap = mustSnapshot(t, l)
	drop := drain(out)
	if snap.Players[0].Flags[yworld.EmberFlag] != 0 {
		t.Fatal("drop without pack")
	}
	if hasTextContains(drop, "우물길") {
		t.Fatalf("drop ambient without prereq: %#v", drop)
	}

	walkTo(t, l, "dalbitgol:school", "south", "east")
	drain(out)
	l.Submit(Talk{ConnID: "a", NPC: "청람"})
	snap = mustSnapshot(t, l)
	risk := drain(out)
	if snap.Players[0].Flags[yworld.EmberFlag] != 0 {
		t.Fatal("cheongram risk without pack")
	}
	if hasTextContains(risk, "이제 그쪽에서 자네를 부른다") {
		t.Fatalf("risk without prereq: %#v", risk)
	}
	assertNoSecrets(t, talk, drop, risk)
}

func TestEmberHideWithoutPackDoesNotGrant(t *testing.T) {
	l, out := startDalbitgolWith(t, skill.Always)
	enterFlagsBag(t, l, out, nil, []yworld.Stack{{ID: "leaflet", Qty: 1}})
	walkTo(t, l, "dalbitgol:station", "north", "north", "north", "north", "north", "east", "east", "south", "south", "east", "east", "east")
	drain(out)
	l.Submit(Hide{ConnID: "a", Query: "leaflet"})
	snap := mustSnapshot(t, l)
	if snap.Players[0].Flags[yworld.DawnScentFlag] != 1 {
		t.Fatal("dawn_scent still grants")
	}
	if snap.Players[0].Flags[yworld.EmberFlag] != 0 {
		t.Fatalf("ember without pack: %+v", snap.Players[0].Flags)
	}
}

func TestEmberSoftHookRoomAndObjects(t *testing.T) {
	l, out := startDalbitgol(t)
	enterFlags(t, l, out, map[string]int{yworld.EmberFlag: 1})
	walkTo(t, l, "dalbitgol:market", "north", "north", "north", "north", "north", "east", "east")
	drain(out)
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	market := findRoom(drain(out))
	if market == nil || !strings.Contains(market.Description, "우물길") || !strings.Contains(market.Description, "다방 문") {
		t.Fatalf("market ember desc: %+v", market)
	}

	walkTo(t, l, "dalbitgol:cafe-baekya", "south", "south")
	drain(out)
	l.Submit(Look{ConnID: "a"})
	_ = mustSnapshot(t, l)
	cafe := findRoom(drain(out))
	if cafe == nil || !strings.Contains(cafe.Description, "우물길") {
		t.Fatalf("cafe ember desc: %+v", cafe)
	}
	l.Submit(Look{ConnID: "a", Target: "잔받침"})
	_ = mustSnapshot(t, l)
	saucer := drain(out)
	if !hasTextContains(saucer, "우물길") {
		t.Fatalf("saucer ember: %#v", saucer)
	}
	assertNoSecrets(t, saucer)

	walkTo(t, l, "dalbitgol:packing-shed", "east", "east", "north")
	drain(out)
	l.Submit(Look{ConnID: "a", Target: "빈 수레"})
	_ = mustSnapshot(t, l)
	cart := drain(out)
	if !hasTextContains(cart, "우물길") || !hasTextContains(cart, "다방 문") {
		t.Fatalf("empty-cart ember: %#v", cart)
	}
	assertNoSecrets(t, cart)
}

func TestEmberDropWrongRoom(t *testing.T) {
	l, out := startDalbitgol(t)
	enterFlagsBag(t, l, out, map[string]int{
		yworld.ExaminedFlag("gangpo-pack"): 1,
		yworld.FirstMarketSaleFlag:         1,
	}, []yworld.Stack{{ID: "leaflet", Qty: 1}})
	walkTo(t, l, "dalbitgol:market", "north", "north", "north", "north", "north", "east", "east")
	drain(out)
	l.Submit(DropItem{ConnID: "a", ItemID: "leaflet"})
	snap := mustSnapshot(t, l)
	if snap.Players[0].Flags[yworld.EmberFlag] != 0 {
		t.Fatal("drop leaflet at market must not grant ember")
	}
}

func enterFlags(t *testing.T, l *Loop, out <-chan Event, flags map[string]int) {
	t.Helper()
	enterFlagsBag(t, l, out, flags, nil)
}

func enterFlagsBag(t *testing.T, l *Loop, out <-chan Event, flags map[string]int, bag []yworld.Stack) {
	t.Helper()
	l.Submit(EnterWorld{
		ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Flags: flags, Bag: bag},
	})
	_ = mustSnapshot(t, l)
	drain(out)
}

func walkTo(t *testing.T, l *Loop, want string, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		l.Submit(Move{ConnID: "a", Dir: dir})
	}
	snap := mustSnapshot(t, l)
	if snap.Players[0].RoomID != want {
		t.Fatalf("want %s, at %s", want, snap.Players[0].RoomID)
	}
}
