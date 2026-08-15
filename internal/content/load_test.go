package content

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pjhwa/yeomyeong/internal/world"
)

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
	if !ok || shop.HeatModifier != 0 {
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
}

func TestNoHardcodedRooms(t *testing.T) {
	src, err := os.ReadFile("load.go")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(src, []byte("dalbitgol:market")) {
		t.Fatal("do not hardcode village rooms")
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
