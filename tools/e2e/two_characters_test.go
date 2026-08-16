package e2e

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pjhwa/yeomyeong/internal/content"
	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/world"
)

// Two accounts, two stations (issue #45, PLAN.md §5 M2).
// Real content/ via LoadWorld + skill.Load. Production rng; loop practice
// until the YAML title appears (cap 200). No skill.TryGain helper, no
// ENGINE/GAMEPLAY/WORLD edits.

const (
	maxPractice = 200
	cmdPace     = 80 * time.Millisecond

	smithRoom   = "dalbitgol:smithy"
	speechRoom  = "dalbitgol:cafe-baekya"
	smithTitle  = "달빛골의 대장장이"
	speechTitle = "말을 부리는 자"

	rateLimited = "rate_limited"
)

func TestTwoCharactersPlayDifferently(t *testing.T) {
	w, skills := loadRealPracticeWorld(t)
	h := startHarnessWithWorld(t, w.Rooms, w.Items, w.Ground, skills)

	a := h.dialTelnet(t, "telnet-A")
	a.createUser(t, "갑을", "password1")
	walkTo(t, a, w.Rooms, world.SpawnID, smithRoom)
	pickAndEquip(t, a, "쇠망치", "hammer")
	aLoops := practiceUntilTitle(t, a, skills, "두드리다", smithTitle)
	aSheet := readSheet(t, a, "숙련")
	t.Logf("account A practice loops=%d sheet=%q", aLoops, compactSheet(aSheet))

	b := h.dialTelnet(t, "telnet-B")
	b.createUser(t, "병정", "password2")
	walkTo(t, b, w.Rooms, world.SpawnID, speechRoom)
	bLoops := practiceUntilTitle(t, b, skills, "이야기하다", speechTitle)
	bSheet := readSheet(t, b, "skills")
	t.Logf("account B practice loops=%d sheet=%q", bLoops, compactSheet(bSheet))

	if !strings.Contains(aSheet, smithTitle) {
		h.failf(t, "A title missing %q\nsheet: %q", smithTitle, aSheet)
	}
	if strings.Contains(aSheet, speechTitle) {
		h.failf(t, "A must not have speaker title\nsheet: %q", aSheet)
	}
	if !sheetHasSkill(aSheet, "대장", "smith") {
		h.failf(t, "A skills missing 대장/smith\nsheet: %q", aSheet)
	}

	if !strings.Contains(bSheet, speechTitle) {
		h.failf(t, "B title missing %q\nsheet: %q", speechTitle, bSheet)
	}
	if strings.Contains(bSheet, smithTitle) {
		h.failf(t, "B must not have smith title\nsheet: %q", bSheet)
	}
	if !sheetHasSkill(bSheet, "언변", "speech") {
		h.failf(t, "B skills missing 언변/speech\nsheet: %q", bSheet)
	}
	if aSheet == bSheet {
		h.failf(t, "titles/sheets must differ\nA: %q\nB: %q", aSheet, bSheet)
	}
}

func loadRealPracticeWorld(t *testing.T) (*content.World, *skill.Catalog) {
	t.Helper()
	root := repoContentRoot(t)
	w, err := content.LoadWorld(root, world.SpawnID)
	if err != nil {
		t.Fatalf("content.LoadWorld(%s): %v", root, err)
	}
	if w.Rooms.Len() < 40 {
		t.Fatalf("real dalbitgol catalog: %d rooms, want ≥40 (fixture?)", w.Rooms.Len())
	}
	if w.Items == nil || w.Items.Len() == 0 {
		t.Fatal("LoadWorld returned no items")
	}
	if _, ok := w.Items.Get("hammer"); !ok {
		t.Fatal("hammer missing from items")
	}
	if !groundHas(w.Ground, smithRoom, "hammer") {
		t.Fatalf("smithy ground missing hammer: %v", w.Ground[smithRoom])
	}
	r, ok := w.Rooms.Room(smithRoom)
	if !ok {
		t.Fatalf("%s missing after LoadWorld", smithRoom)
	}
	if !roomHasFlag(r, "forge") {
		t.Fatalf("%s missing forge flag after spawns: %v", smithRoom, r.Flags)
	}
	cafe, ok := w.Rooms.Room(speechRoom)
	if !ok {
		t.Fatalf("%s missing after LoadWorld", speechRoom)
	}
	if !roomHasFlag(cafe, "salon") {
		t.Fatalf("%s missing salon flag: %v", speechRoom, cafe.Flags)
	}

	skills, err := skill.Load(filepath.Join(root, "skills"))
	if err != nil {
		t.Fatalf("skill.Load: %v", err)
	}
	if skills.Len() < 14 {
		t.Fatalf("skills=%d, want ≥14", skills.Len())
	}
	if _, ok := skills.Lookup("smith"); !ok {
		t.Fatal("smith missing")
	}
	if _, ok := skills.Lookup("언변"); !ok {
		t.Fatal("speech/언변 missing")
	}
	if skills.Title(skills.NewSheet().WithRank("smith", 15)).KO != smithTitle {
		t.Fatalf("smith-15 title=%q want %q", skills.Title(skills.NewSheet().WithRank("smith", 15)).KO, smithTitle)
	}
	if skills.Title(skills.NewSheet().WithRank("speech", 15)).KO != speechTitle {
		t.Fatalf("speech-15 title=%q want %q", skills.Title(skills.NewSheet().WithRank("speech", 15)).KO, speechTitle)
	}
	return w, skills
}

func walkTo(t *testing.T, c *telnetClient, cat *world.Catalog, from, dest string) {
	t.Helper()
	path := shortestDirs(cat, from, dest)
	if path == nil {
		c.h.failf(t, "%s no path %s → %s", c.name, from, dest)
	}
	here := from
	for _, dir := range path {
		next, ok := cat.Exit(here, dir)
		if !ok {
			c.h.failf(t, "%s catalog exit %s %s missing", c.name, here, dir)
		}
		room, ok := cat.Room(next)
		if !ok {
			c.h.failf(t, "%s invented room %s", c.name, next)
		}
		c.sendPaced(t, dir)
		c.expectLine(t, strings.TrimSpace(room.Name.KO))
		here = next
	}
	if here != dest {
		c.h.failf(t, "%s walk ended in %s want %s", c.name, here, dest)
	}
}

func pickAndEquip(t *testing.T, c *telnetClient, names ...string) {
	t.Helper()
	var last string
	for _, name := range names {
		c.sendPaced(t, "get "+name)
		got, hit := c.readUntilAny(t, "집었습니다", "여기에는 그것이 없습니다")
		last = got
		if hit == "집었습니다" {
			c.sendPaced(t, "equip "+name)
			c.expectLine(t, "갖췄습니다")
			return
		}
	}
	c.h.failf(t, "%s could not get %v\nlast recv: %q", c.name, names, last)
}

func practiceUntilTitle(t *testing.T, c *telnetClient, skills *skill.Catalog, cmd, title string) int {
	t.Helper()
	sk, ok := skills.Lookup(cmd)
	if !ok {
		c.h.failf(t, "%s lookup %q", c.name, cmd)
	}
	needles := append(append([]string{}, sk.Gain...), sk.Miss...)
	needles = append(needles, rateLimited, "그런 숙련은 없습니다", "모르는 말입니다")
	loops := 0
	for loops < maxPractice {
		c.sendPaced(t, cmd)
		got, hit := c.readUntilAny(t, needles...)
		switch hit {
		case "그런 숙련은 없습니다", "모르는 말입니다":
			c.h.failf(t, "%s unknown skill for %q\nlast recv: %q", c.name, cmd, got)
		case rateLimited:
			time.Sleep(200 * time.Millisecond)
			continue
		}
		loops++
		sheet := readSheet(t, c, "숙련")
		if strings.Contains(sheet, title) {
			return loops
		}
	}
	sheet := readSheet(t, c, "숙련")
	if strings.Contains(sheet, title) {
		return loops
	}
	c.h.failf(t, "%s title %q never appeared after %d %q\nsheet: %q", c.name, title, loops, cmd, sheet)
	return loops
}

func readSheet(t *testing.T, c *telnetClient, cmd string) string {
	t.Helper()
	c.sendPaced(t, cmd)
	got := c.readUntil(t, "능력:")
	if i := strings.IndexAny(c.buf, "\r\n"); i >= 0 {
		got += c.buf[:i]
		c.buf = c.buf[i:]
	}
	return got
}

func compactSheet(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\r", " ")), " ")
}

func sheetHasSkill(dump, ko, id string) bool {
	for _, line := range strings.Split(dump, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "숙련:") {
			continue
		}
		return strings.Contains(line, ko) || strings.Contains(line, id)
	}
	return strings.Contains(dump, ko) || strings.Contains(dump, id)
}

func (c *telnetClient) sendPaced(t *testing.T, line string) {
	t.Helper()
	time.Sleep(cmdPace)
	c.send(t, line)
}

func shortestDirs(cat *world.Catalog, from, dest string) []string {
	if from == dest {
		return []string{}
	}
	type node struct {
		id  string
		via []string
	}
	seen := map[string]struct{}{from: {}}
	q := []node{{id: from}}
	order := []string{"north", "south", "east", "west", "up", "down"}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, dir := range order {
			next, ok := cat.Exit(cur.id, dir)
			if !ok {
				continue
			}
			if _, dup := seen[next]; dup {
				continue
			}
			path := append(append([]string{}, cur.via...), dir)
			if next == dest {
				return path
			}
			seen[next] = struct{}{}
			q = append(q, node{id: next, via: path})
		}
	}
	return nil
}

func groundHas(ground map[string][]world.Stack, room, item string) bool {
	for _, s := range ground[room] {
		if s.ID == item && s.Qty > 0 {
			return true
		}
	}
	return false
}

func roomHasFlag(r world.Room, flag string) bool {
	for _, f := range r.Flags {
		if f == flag {
			return true
		}
	}
	return false
}
