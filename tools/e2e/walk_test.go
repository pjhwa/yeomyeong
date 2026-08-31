package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pjhwa/yeomyeong/internal/content"
	"github.com/pjhwa/yeomyeong/internal/world"
)

// Walk from dalbitgol:gate. Exits and fragments are taken from
// content/zones/dalbitgol/rooms.yaml (issue #30).
var dalbitgolWalk = []struct {
	cmd      string
	room     string
	fragment string
}{
	{"북", "dalbitgol:gate-road", "수레바퀴 자국"},
	{"w", "dalbitgol:ditch", "함석 바가지"},
	{"n", "dalbitgol:creek-bridge", "넓적한 돌 네 장"},
	{"n", "dalbitgol:creek", "손을 담그면 찬물"},
	{"e", "dalbitgol:printshop", "닳은 활자"},
	{"e", "dalbitgol:type-yard", "인쇄기가 젖은 원지"},
	{"e", "dalbitgol:cafe-baekya", "볶은 보리차"},
	{"s", "dalbitgol:school-lane", "석판이 기대어"},
}

func TestNewUserWalksDalbitgol(t *testing.T) {
	cat := loadDalbitgol(t)
	requireUniqueFragments(t, cat, "dalbitgol:gate", "녹가루")
	seen := map[string]struct{}{"dalbitgol:gate": {}}
	here := "dalbitgol:gate"
	for _, step := range dalbitgolWalk {
		dest, ok := cat.Exit(here, dirOf(step.cmd))
		if !ok || dest != step.room {
			t.Fatalf("path %s %q: catalog exit=%q ok=%v, want %s", here, step.cmd, dest, ok, step.room)
		}
		if _, dup := seen[step.room]; dup {
			t.Fatalf("path revisits %s", step.room)
		}
		seen[step.room] = struct{}{}
		requireUniqueFragments(t, cat, step.room, step.fragment)
		here = step.room
	}
	if n := len(seen); n < 8 {
		t.Fatalf("path has %d distinct rooms, want ≥8", n)
	}

	h := startHarnessWithCatalog(t, cat)
	c := h.dialTelnet(t, "walker")
	spawn := c.createUserUntilPrompt(t, "나그네", "password1")
	if !strings.Contains(spawn, "녹가루") {
		c.h.failf(t, "spawn card missing 녹가루 (dalbitgol:gate)\nlast recv: %q", spawn)
	}

	for _, step := range dalbitgolWalk {
		c.send(t, step.cmd)
		c.expectLine(t, step.fragment)
	}
}

func loadDalbitgol(t *testing.T) *world.Catalog {
	t.Helper()
	root := repoContentRoot(t)
	cat, err := content.Load(root, world.SpawnID)
	if err != nil {
		t.Fatalf("content.Load(%s, %s): %v", root, world.SpawnID, err)
	}
	if cat.Spawn() != world.SpawnID {
		t.Fatalf("spawn=%s want %s", cat.Spawn(), world.SpawnID)
	}
	if cat.Len() < 40 {
		t.Fatalf("real dalbitgol catalog: %d rooms, want ≥40 (fixture?)", cat.Len())
	}
	if _, ok := cat.Room("dalbitgol:gate"); !ok {
		t.Fatal("dalbitgol:gate missing after Load")
	}
	return cat
}

func repoContentRoot(t *testing.T) string {
	t.Helper()
	start, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := start
	for {
		cand := filepath.Join(dir, "content")
		if _, err := os.Stat(filepath.Join(cand, "zones", "dalbitgol", "rooms.yaml")); err == nil {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("content/zones/dalbitgol/rooms.yaml not found from %s", start)
		}
		dir = parent
	}
}

func requireUniqueFragments(t *testing.T, cat *world.Catalog, roomID, frag string) {
	t.Helper()
	r, ok := cat.Room(roomID)
	if !ok {
		t.Fatalf("invented room %s", roomID)
	}
	if !strings.Contains(r.Description.KO, frag) {
		t.Fatalf("fragment %q not in %s YAML description", frag, roomID)
	}
	for _, id := range cat.IDs() {
		if id == roomID {
			continue
		}
		other, _ := cat.Room(id)
		if strings.Contains(other.Description.KO, frag) {
			t.Fatalf("fragment %q is not unique to %s (also %s)", frag, roomID, id)
		}
	}
}

func dirOf(cmd string) string {
	switch cmd {
	case "n", "north", "북":
		return "north"
	case "s", "south", "남":
		return "south"
	case "e", "east", "동":
		return "east"
	case "w", "west", "서":
		return "west"
	default:
		return cmd
	}
}
