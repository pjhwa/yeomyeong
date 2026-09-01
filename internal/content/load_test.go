package content

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/world"
)

func TestLoadItemsAndSpawns(t *testing.T) {
	w, err := LoadWorld(fixture("valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	pebble, ok := w.Items.Get("pebble")
	if !ok || pebble.Name.KO != "조약돌" || pebble.Weight != 1 {
		t.Fatalf("pebble: %+v items=%d", pebble, w.Items.Len())
	}
	start, _ := w.Rooms.Room("test:start")
	shop, _ := w.Rooms.Room("test:shop")
	if !hasFlag(start.Flags, "yard") || !hasFlag(shop.Flags, "forge") {
		t.Fatalf("flags start=%v shop=%v", start.Flags, shop.Flags)
	}
	if stacksOf(w.Ground["test:start"], "pebble") != 1 || stacksOf(w.Ground["test:shop"], "cloak") != 1 {
		t.Fatalf("ground=%v", w.Ground)
	}
	if _, ok := w.Ground["test:yard"]; ok {
		t.Fatal("yard must not share start ground")
	}
	root := writeZone(t, minRoom(""))
	empty, err := LoadWorld(root, "test:start")
	if err != nil || empty.Items.Len() != 0 || len(empty.Ground) != 0 {
		t.Fatalf("missing items/spawns: %v %#v", err, empty)
	}
	writeSpawns(t, root, "- room: test:start\n  items: [ghost]\n")
	if _, err := LoadWorld(root, "test:start"); !errors.Is(err, ErrUnknownItem) {
		t.Fatalf("unknown item: %v", err)
	}
	root = writeZone(t, minRoom("  flags: [forge, kitchen, press, clinic, yard]\n"))
	if _, err := Load(root, "test:start"); err != nil {
		t.Fatal(err)
	}
	root = writeZone(t, minRoom(""))
	writeItems(t, root, minItem("hat", "head", 1))
	if _, err := LoadWorld(root, "test:start"); !errors.Is(err, ErrUnknownSlot) {
		t.Fatalf("slot: %v", err)
	}
}

func TestLoadLivelihood(t *testing.T) {
	w, err := LoadWorld(fixture("valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	sk, err := skill.Load(filepath.Join("..", "..", "content", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	liv, err := LoadLivelihood(fixture("valid"), w.Rooms, w.Items, sk)
	if err != nil {
		t.Fatal(err)
	}
	if n := liv.Craft.NodesIn("test:start"); len(n) != 1 || n[0].Item != "herb" {
		t.Fatalf("nodes %+v", n)
	}
	if _, ok := liv.Craft.LookupRecipe("nail"); !ok {
		t.Fatal("nail recipe")
	}
	ids := liv.Markets.IDs()
	if len(ids) != 2 || !liv.Markets.HasMarket("test") || !liv.Markets.HasMarket("test-east") {
		t.Fatalf("markets %v", ids)
	}
	p1, _ := liv.Markets.Quote("test", "herb")
	p2, _ := liv.Markets.Quote("test-east", "herb")
	if p1 >= p2 {
		t.Fatalf("want regional spread %d %d", p1, p2)
	}
}

func TestLoadValidGraph(t *testing.T) {
	cat, err := Load(fixture("valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	if cat.Spawn() != "test:start" || cat.Len() != 5 {
		t.Fatalf("spawn=%s len=%d", cat.Spawn(), cat.Len())
	}
	start, ok := cat.Room("test:start")
	if !ok || start.Name.KO != "시작 마당" || start.Name.EN != "" || start.Name.Text("en") != "시작 마당" {
		t.Fatalf("bare i18n: %+v", start)
	}
	if start.HeatModifier != 1 || len(start.Ambient) != 2 || start.Ambient[1].EN == "" {
		t.Fatalf("start extras: %+v", start)
	}
	yard, ok := cat.Room("test:yard")
	if !ok || yard.Name.KO != "안마당" || yard.Name.EN != "Inner Yard" || yard.Name.Text("en") != "Inner Yard" {
		t.Fatalf("map i18n: %+v", yard)
	}
	if yard.HeatModifier != 0.8 || yard.Market != "test" || len(yard.Foreshadow) != 1 || yard.Foreshadow[0] != "FS-014" {
		t.Fatalf("yard extras: %+v", yard)
	}
	shop, ok := cat.Room("test:shop")
	if !ok || shop.HeatModifier != 0 || shop.Market != "test-east" {
		t.Fatalf("explicit 0 heat: %+v", shop)
	}
	if dest, ok := cat.Exit("test:shop", "south"); !ok || dest != "test:cliff" {
		t.Fatalf("one-way: %q %v", dest, ok)
	}
	if n := len(Reachable(cat, "test:start")); n != 5 {
		t.Fatalf("reachable=%d", n)
	}
}

func TestLoadErrorFixtures(t *testing.T) {
	cases := []struct {
		dir, spawn string
		want       error
	}{
		{"missing-exit", "test:start", ErrUnknownExit},
		{"missing-spawn", "test:start", ErrSpawnMissing},
		{"empty", "test:start", ErrSpawnMissing},
		{"unknown-fs", "test:start", ErrUnknownForeshadow},
		{"unknown-flag", "test:start", ErrUnknownFlag},
	}
	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			if _, err := Load(fixture(tc.dir), tc.spawn); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func TestLoadEmptySpawnID(t *testing.T) {
	if _, err := Load(fixture("valid"), ""); !errors.Is(err, ErrSpawnMissing) {
		t.Fatalf("got %v", err)
	}
}

func TestLoadMissingZonesDir(t *testing.T) {
	if _, err := Load(t.TempDir(), "test:start"); err == nil {
		t.Fatal("want error")
	}
}

func TestLoadSchemaErrors(t *testing.T) {
	cases := []struct{ name, yaml, sub string }{
		{"bad-yaml", ": [\n", ""},
		{"missing-name", "- id: test:start\n  description: \"설명이다.\"\n", "name.ko"},
		{"missing-description", "- id: test:start\n  name: \"방\"\n", "description.ko"},
		{"missing-id", "- name: \"방\"\n  description: \"설명이다.\"\n", "missing id"},
		{"bad-id", "- id: NotASlug\n  name: \"방\"\n  description: \"설명이다.\"\n", "zone:slug"},
		{"zone-mismatch", "- id: other:start\n  name: \"방\"\n  description: \"설명이다.\"\n", "does not match"},
		{"unknown-dir", minRoom("  exits:\n    northeast: test:start\n"), "unknown exit direction"},
		{"empty-exit", minRoom("  exits:\n    north: \"\"\n"), "empty exit"},
		{"empty-ambient", minRoom("  ambient:\n    - \"\"\n"), "ambient"},
		{"bad-loc-type", "- id: test:start\n  name: [\"방\"]\n  description: \"설명이다.\"\n", "localized string"},
		{"name-en-only", "- id: test:start\n  name:\n    en: Only\n  description: \"설명이다.\"\n", "name.ko"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(writeZone(t, tc.yaml), "test:start")
			if err == nil || (tc.sub != "" && !strings.Contains(err.Error(), tc.sub)) {
				t.Fatalf("got %v want %q", err, tc.sub)
			}
		})
	}
}

func TestLoadDuplicateAndUnreachable(t *testing.T) {
	if _, err := Load(writeZone(t, minRoom("")+minRoom("")), "test:start"); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("dup: %v", err)
	}
	island := minRoom("") + "- id: test:island\n  name: \"섬\"\n  description: \"끊긴 방.\"\n"
	if _, err := Load(writeZone(t, island), "test:start"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("island: %v", err)
	}
}

func TestLoadSkipsZoneWithoutRooms(t *testing.T) {
	root := writeZone(t, minRoom(""))
	if err := os.MkdirAll(filepath.Join(root, "zones", "emptyzone"), 0o755); err != nil {
		t.Fatal(err)
	}
	cat, err := Load(root, "test:start")
	if err != nil || cat.Len() != 1 {
		t.Fatalf("err=%v len=%v", err, cat)
	}
}

func TestLoadLedgerMissingWhenForeshadowUsed(t *testing.T) {
	_, err := Load(writeZone(t, minRoom("  foreshadow: [FS-001]\n")), "test:start")
	if err == nil || !strings.Contains(err.Error(), "FORESHADOW.md") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadRoomsYamlIsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "zones", "test", "rooms.yaml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, "test:start"); err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("got %v", err)
	}
}

func TestReachableHelper(t *testing.T) {
	cat, err := Load(fixture("valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	if seen := Reachable(cat, "test:start"); !seen["test:cliff"] || !seen["test:loft"] {
		t.Fatalf("from start: %v", seen)
	}
	if seen := Reachable(cat, "test:cliff"); len(seen) != 1 || !seen["test:cliff"] {
		t.Fatalf("one-way cliff: %v", seen)
	}
	if seen := Reachable(cat, "test:yard"); !seen["test:start"] || !seen["test:cliff"] {
		t.Fatalf("cycle yard->start: %v", seen)
	}
	if seen := Reachable(cat, "test:missing"); len(seen) != 0 {
		t.Fatalf("unknown start: %v", seen)
	}
	if seen := Reachable(nil, "test:start"); len(seen) != 0 {
		t.Fatalf("nil: %v", seen)
	}
	alone, err := world.NewCatalog([]world.Room{{
		ID: "solo", Name: world.Localized{KO: "혼자"}, Description: world.Localized{KO: "방"},
	}}, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if seen := Reachable(alone, "solo"); len(seen) != 1 || !seen["solo"] {
		t.Fatalf("no exits: %v", seen)
	}
}

func TestLoadRealTreeIfPresent(t *testing.T) {
	root := filepath.Join("..", "..", "content")
	if _, err := os.Stat(filepath.Join(root, "zones", "dalbitgol", "rooms.yaml")); err != nil {
		t.Skip("real tree not present")
	}
	cat, err := Load(root, world.SpawnID)
	if err != nil {
		t.Fatal(err)
	}
	if cat.Spawn() != world.SpawnID || cat.Len() == 0 {
		t.Fatalf("spawn=%s len=%d", cat.Spawn(), cat.Len())
	}
	w, err := LoadWorld(root, world.SpawnID)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := skill.Load(filepath.Join(root, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	liv, err := LoadLivelihood(root, w.Rooms, w.Items, sk)
	if err != nil {
		t.Fatal(err)
	}
	if len(liv.Craft.Nodes()) == 0 || len(liv.Craft.Recipes()) != 4 || len(liv.Markets.IDs()) != 2 {
		t.Fatalf("livelihood nodes=%d recipes=%d markets=%v", len(liv.Craft.Nodes()), len(liv.Craft.Recipes()), liv.Markets.IDs())
	}
	if _, ok := w.Rooms.Room("solgol:market"); !ok {
		t.Fatal("solgol:market missing")
	}
	if it, ok := w.Items.Get("leaflet"); !ok || !it.Contraband {
		t.Fatalf("leaflet contraband: %+v %v", it, ok)
	}
	if n, ok := w.NPCs.Find("청람"); !ok {
		t.Fatal("cheongram npc missing")
	} else if len(n.TalkWhen) < 4 || n.TalkWhen[0].Flag != "ember" || n.TalkWhen[1].Flag != "dawn_scent" || n.TalkWhen[2].Flag != "first_market_sale" || n.TalkWhen[3].Flag != "examined:gangpo-pack" {
		t.Fatalf("cheongram talk.when: %+v", n.TalkWhen)
	}
	if n, ok := w.NPCs.FindInRoom("dalbitgol:packing-shed", "오씨"); !ok || n.ID != "clerk-oh" {
		t.Fatalf("clerk-oh: %+v %v", n, ok)
	}
	hand, ok := w.NPCs.FindInRoom("dalbitgol:cafe-baekya", "점원")
	if !ok || hand.ID != "cafe-hand" {
		t.Fatalf("cafe-hand: %+v %v", hand, ok)
	}
	if len(hand.TalkWhen) < 3 || hand.TalkWhen[0].Flag != "ember" || hand.TalkWhen[1].Flag != "dawn_scent" || hand.TalkWhen[2].Flag != "first_market_sale" {
		t.Fatalf("cafe-hand talk.when: %+v", hand.TalkWhen)
	}
	if _, ok := w.Objects.FindInRoom("dalbitgol:market", "신문"); !ok {
		t.Fatal("market newspaper object missing")
	}
	pack, ok := w.Objects.FindInRoom("dalbitgol:warehouse", "짐")
	if !ok || pack.ID != "gangpo-pack" {
		t.Fatalf("warehouse pack: %+v %v", pack, ok)
	}
	if !strings.Contains(pack.Description.KO, "한벽일보") || strings.TrimSpace(pack.AfterExamine.KO) == "" {
		t.Fatalf("warehouse pack clue: %+v", pack)
	}
	if !strings.Contains(pack.AfterExamine.KO, "시세") || !strings.Contains(pack.AfterExamine.KO, "쑥") {
		t.Fatalf("warehouse pack livelihood bridge: %+v", pack.AfterExamine)
	}
	ruts, ok := w.Objects.FindInRoom("dalbitgol:warehouse-lane", "자국")
	if !ok || ruts.ID != "cart-ruts" || !strings.Contains(ruts.Description.KO, "흐릿") {
		t.Fatalf("cart-ruts bland: %+v %v", ruts, ok)
	}
	if len(ruts.DescWhen) == 0 || ruts.DescWhen[0].Flag != "first_market_sale" || !strings.Contains(ruts.DescWhen[0].Line.KO, "화물마당") {
		t.Fatalf("cart-ruts when: %+v", ruts.DescWhen)
	}
	chit, ok := w.Objects.FindInRoom("dalbitgol:packing-shed", "꼬리표")
	if !ok || chit.ID != "cargo-chit" || !strings.Contains(chit.Description.KO, "잘 읽히지") {
		t.Fatalf("cargo-chit bland: %+v %v", chit, ok)
	}
	if len(chit.DescWhen) == 0 || chit.DescWhen[0].Flag != "first_market_sale" || !strings.Contains(chit.DescWhen[0].Line.KO, "만석상회") {
		t.Fatalf("cargo-chit when: %+v", chit.DescWhen)
	}
	cart, ok := w.Objects.FindInRoom("dalbitgol:packing-shed", "빈 수레")
	if !ok || cart.ID != "empty-cart" {
		t.Fatalf("empty-cart: %+v %v", cart, ok)
	}
	if len(cart.DescWhen) == 0 || cart.DescWhen[0].Flag != "ember" {
		t.Fatalf("empty-cart when: %+v", cart.DescWhen)
	}
	saucer, ok := w.Objects.FindInRoom("dalbitgol:cafe-baekya", "잔받침")
	if !ok || saucer.ID != "clean-saucer" {
		t.Fatalf("clean-saucer: %+v %v", saucer, ok)
	}
	if len(saucer.DescWhen) == 0 || saucer.DescWhen[0].Flag != "ember" {
		t.Fatalf("clean-saucer when: %+v", saucer.DescWhen)
	}
	cafe, ok := w.Rooms.Room("dalbitgol:cafe-baekya")
	if !ok || len(cafe.DescWhen) == 0 || cafe.DescWhen[0].Flag != "ember" {
		t.Fatalf("cafe-baekya when: %+v %v", cafe, ok)
	}
	market, ok := w.Rooms.Room("dalbitgol:market")
	if !ok || len(market.DescWhen) == 0 || market.DescWhen[0].Flag != "ember" {
		t.Fatalf("market ember when first: %+v %v", market.DescWhen, ok)
	}
}

func TestNoHardcodedRooms(t *testing.T) {
	for _, name := range []string{"load.go", "story.go", "items.go"} {
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(src, []byte("dalbitgol:market")) {
			t.Fatalf("%s: do not hardcode village rooms", name)
		}
	}
}

func TestLoadStoryYAML(t *testing.T) {
	w, err := LoadWorld(fixture("valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	if w.NPCs.Len() != 1 {
		t.Fatalf("npcs=%d", w.NPCs.Len())
	}
	n, ok := w.NPCs.FindInRoom("test:start", "훈장")
	if !ok || n.ID != "tutor" || !strings.Contains(n.TalkFirst.KO, "처음") {
		t.Fatalf("tutor: %+v %v", n, ok)
	}
	if len(n.TalkWhen) != 2 || n.TalkWhen[0].Flag != "examined:test-paper" || !strings.Contains(n.TalkWhen[0].Line.KO, "신문을 봤군") {
		t.Fatalf("tutor when: %+v", n.TalkWhen)
	}
	if n.TalkWhen[1].Flag != "other-flag" {
		t.Fatalf("tutor when[1]: %+v", n.TalkWhen)
	}
	if w.Objects.Len() != 1 {
		t.Fatalf("objects=%d", w.Objects.Len())
	}
	o, ok := w.Objects.FindInRoom("test:yard", "신문")
	if !ok || !strings.Contains(o.Description.KO, "활자") || !strings.Contains(o.Description.KO, "한벽일보") {
		t.Fatalf("paper: %+v %v", o, ok)
	}
	if !strings.Contains(o.AfterExamine.KO, "사다리") {
		t.Fatalf("after_examine: %+v", o)
	}
}

func TestLoadStoryErrors(t *testing.T) {
	root := writeZone(t, minRoom(""))
	writeStoryFile(t, root, "objects.yaml", "- id: ghost-paper\n  room: test:missing\n  name: 신문\n  aliases: [신문]\n  description: 설명이다.\n")
	if _, err := LoadWorld(root, "test:start"); err == nil || !strings.Contains(err.Error(), "unknown room") {
		t.Fatalf("unknown object room: %v", err)
	}
	root = writeZone(t, minRoom(""))
	writeStoryFile(t, root, "npcs.yaml", "- id: ghost\n  room: test:start\n  name: 사람\n  aliases: [사람]\n  look: 코트.\n  talk:\n    first: 안녕.\n")
	if _, err := LoadWorld(root, "test:start"); err == nil || !strings.Contains(err.Error(), "talk.second") {
		t.Fatalf("missing second talk: %v", err)
	}
	root = writeZone(t, minRoom(""))
	writeStoryFile(t, root, "npcs.yaml", "- id: ghost\n  room: test:start\n  name: 사람\n  aliases: [사람]\n  look: 코트.\n  talk:\n    first: 안녕.\n    second: 또.\n    when:\n      - flag: \"\"\n        ko: 단서.\n")
	if _, err := LoadWorld(root, "test:start"); err == nil || !strings.Contains(err.Error(), "empty flag") {
		t.Fatalf("empty when.flag: %v", err)
	}
	root = writeZone(t, minRoom(""))
	writeStoryFile(t, root, "npcs.yaml", "- id: ghost\n  room: test:start\n  name: 사람\n  aliases: [사람]\n  look: 코트.\n  talk:\n    first: 안녕.\n    second: 또.\n    when:\n      - ko: 단서.\n")
	if _, err := LoadWorld(root, "test:start"); err == nil || !strings.Contains(err.Error(), "empty flag") {
		t.Fatalf("missing when.flag: %v", err)
	}
	root = writeZone(t, minRoom(""))
	writeStoryFile(t, root, "npcs.yaml", "- id: ghost\n  room: test:start\n  name: 사람\n  aliases: [사람]\n  look: 코트.\n  talk:\n    first: 안녕.\n    second: 또.\n    when:\n      - flag: examined:ghost\n        ko: \"\"\n")
	if _, err := LoadWorld(root, "test:start"); err == nil || !strings.Contains(err.Error(), "ko is required") {
		t.Fatalf("empty when.ko: %v", err)
	}
	root = writeZone(t, minRoom(""))
	writeStoryFile(t, root, "objects.yaml", "- id: bad-fs\n  room: test:start\n  name: 신문\n  aliases: [신문]\n  description: 설명이다.\n  foreshadow: [FS-999]\n")
	_, err := LoadWorld(root, "test:start")
	if err == nil || (!errors.Is(err, ErrUnknownForeshadow) && !strings.Contains(err.Error(), "FORESHADOW.md")) {
		t.Fatalf("unknown fs: %v", err)
	}
}

func writeStoryFile(t *testing.T, root, name, yaml string) {
	t.Helper()
	path := filepath.Join(root, "zones", "test", name)
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixture(name string) string { return filepath.Join("testdata", name) }

func writeZone(t *testing.T, rooms string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "zones", "test")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rooms.yaml"), []byte(rooms), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func minRoom(extra string) string {
	return "- id: test:start\n  name: \"방\"\n  description: \"설명이다.\"\n" + extra
}

func minItem(id, slot string, weight int) string {
	return fmt.Sprintf("- id: %s\n  name: \"물건\"\n  description: \"설명이다.\"\n  slot: %s\n  weight: %d\n", id, slot, weight)
}

func writeItems(t *testing.T, root, yaml string) {
	t.Helper()
	dir := filepath.Join(root, "items")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSpawns(t *testing.T, root, yaml string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "zones", "test", "spawns.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stacksOf(piles []world.Stack, id string) int {
	for _, s := range piles {
		if s.ID == id {
			return s.Qty
		}
	}
	return 0
}
