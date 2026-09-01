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

func TestPracticeSmithAtForgeAndSpeech(t *testing.T) {
	l, cat := startPractice(t, skill.Always)
	smith, ok := cat.Skill("smith")
	if !ok {
		t.Fatal("m2 smith missing")
	}
	speech, ok := cat.Skill("speech")
	if !ok {
		t.Fatal("m2 speech missing")
	}

	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	l.Submit(Move{ConnID: "a", Dir: "east"})
	l.Submit(Practice{ConnID: "a", SkillID: smith.ID})
	l.Submit(Practice{ConnID: "a", SkillID: speech.Name.KO})
	snap := mustSnapshot(t, l)
	p := snap.Players[0]
	if p.Skills[smith.ID] != 1 {
		t.Fatalf("forge smith rank=%d", p.Skills[smith.ID])
	}
	if p.Skills[speech.ID] != 1 {
		t.Fatalf("speech rank=%d", p.Skills[speech.ID])
	}
	evs := drain(out)
	if !hasBody(evs, skill.LineAt(smith.Gain, 1)) {
		t.Fatalf("missing smith gain: %#v", evs)
	}
	if !hasBody(evs, skill.LineAt(speech.Gain, 1)) {
		t.Fatalf("missing speech gain: %#v", evs)
	}
}

func TestPracticeHammerRaisesSmith(t *testing.T) {
	l, cat := startPractice(t, skill.Always)
	smith, _ := cat.Skill("smith")
	mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Bag: []yworld.Stack{{ID: smith.PracticeItem, Qty: 1}}}})
	l.Submit(Practice{ConnID: "a", SkillID: smith.ID})
	snap := mustSnapshot(t, l)
	if snap.Players[0].Skills[smith.ID] != 1 {
		t.Fatalf("hammer smith rank=%d", snap.Players[0].Skills[smith.ID])
	}
}

func TestTitlesDifferAtFifteen(t *testing.T) {
	l, cat := startPractice(t, skill.Always)
	smith, _ := cat.Skill("smith")
	speech, _ := cat.Skill("speech")
	outA := mustAttach(t, l, "a")
	outB := mustAttach(t, l, "b")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Skills: map[string]int{smith.ID: 15}}})
	l.Submit(EnterWorld{ConnID: "b", AccountID: "2", Username: "을", Session: "s",
		Sheet: yworld.Sheet{Skills: map[string]int{speech.ID: 15}}})
	l.Submit(Sheet{ConnID: "a"})
	l.Submit(Sheet{ConnID: "b"})
	_ = mustSnapshot(t, l)
	if !hasBody(drain(outA), text.T(text.Default, text.SheetTitle, "달빛골의 대장장이")) {
		t.Fatal("smith title")
	}
	if !hasBody(drain(outB), text.T(text.Default, text.SheetTitle, "말 잘하는 사람")) {
		t.Fatal("speech title")
	}
}

func TestPracticeMismatchLowerP(t *testing.T) {
	const rank = 50
	l, cat := startPractice(t, func() float64 {
		return skill.Chance(rank, rank) - 1e-9
	})
	smith, _ := cat.Skill("smith")
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Skills: map[string]int{smith.ID: rank}}})
	l.Submit(Practice{ConnID: "a", SkillID: smith.ID})
	if snap := mustSnapshot(t, l); snap.Players[0].Skills[smith.ID] != rank {
		t.Fatalf("unmatched must miss, rank=%d", snap.Players[0].Skills[smith.ID])
	}
	if !hasBody(drain(out), skill.LineAt(smith.Miss, rank)) {
		t.Fatal("want miss line")
	}

	l.Submit(Move{ConnID: "a", Dir: "east"})
	l.Submit(Practice{ConnID: "a", SkillID: smith.ID})
	if snap := mustSnapshot(t, l); snap.Players[0].Skills[smith.ID] != rank+1 {
		t.Fatalf("matched must gain, rank=%d", snap.Players[0].Skills[smith.ID])
	}
}

func TestTitleAnnounceAndKoreanSheet(t *testing.T) {
	l, cat := startPractice(t, skill.Always)
	smith, _ := cat.Skill("smith")
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s",
		Sheet: yworld.Sheet{Skills: map[string]int{smith.ID: 14}}})
	l.Submit(Practice{ConnID: "a", SkillID: "두드리다"})
	l.Submit(Sheet{ConnID: "a"})
	_ = mustSnapshot(t, l)
	evs := drain(out)
	if !hasBody(evs, "사람들이 이제 달빛골의 대장장이라고 불러요.") {
		t.Fatalf("missing title announce: %#v", evs)
	}
	if !hasBody(evs, text.T(text.Default, text.SheetTitle, "달빛골의 대장장이")) {
		t.Fatal("title")
	}
	var skillsLine string
	for _, ev := range evs {
		if tx, ok := ev.(Text); ok && strings.HasPrefix(tx.Body, "기술:") {
			skillsLine = tx.Body
		}
	}
	if !strings.Contains(skillsLine, "대장 초보") {
		t.Fatalf("want Korean band, got %q", skillsLine)
	}
	if strings.Contains(skillsLine, "smith") || strings.Contains(skillsLine, "15") {
		t.Fatalf("sheet leaked id/number: %q", skillsLine)
	}
}

func TestPracticeUnknownAndGhost(t *testing.T) {
	l, _ := startPractice(t, skill.Always)
	out := mustAttach(t, l, "a")
	l.Submit(EnterWorld{ConnID: "a", AccountID: "1", Username: "갑", Session: "s"})
	l.Submit(Practice{ConnID: "a", SkillID: "nope"})
	l.Submit(Practice{ConnID: "ghost", SkillID: "smith"})
	_ = mustSnapshot(t, l)
	if !hasBody(drain(out), text.T(text.Default, text.PracticeUnknown)) {
		t.Fatal("want unknown")
	}
	l.Submit(Sheet{ConnID: "a"})
	_ = mustSnapshot(t, l)
	if !hasBody(drain(out), text.T(text.Default, text.SheetTitle, "아무개")) {
		t.Fatal("default title")
	}
}

func startPractice(t *testing.T, rng func() float64) (*Loop, *skill.Catalog) {
	t.Helper()
	w, err := content.LoadWorld(filepath.Join("..", "content", "testdata", "valid"), "test:start")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := skill.Load(filepath.Join("..", "..", "content", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	l := NewWithWorld(discardLog(), w.Rooms, w.Items, w.Ground, nil).WithSkills(cat).WithRand(rng)
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
	return l, cat
}

func hasBody(evs []Event, body string) bool {
	for _, ev := range evs {
		if tx, ok := ev.(Text); ok && tx.Body == body {
			return true
		}
	}
	return false
}
