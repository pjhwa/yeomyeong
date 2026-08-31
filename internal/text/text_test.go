package text

import "testing"

func TestTKoreanAndFallback(t *testing.T) {
	if got := T(LocaleKO, CmdUnknown); got != "무슨 말인지 모르겠어요. 보다, 종료" {
		t.Fatalf("unknown: %q", got)
	}
	if got := T(LocaleEN, MoveNoExit); got != "그쪽으로는 갈 수 없어요." {
		t.Fatalf("en fallback: %q", got)
	}
	if got := T(Default, SysSeated, "갑"); got != "갑 님이 들어왔어요." {
		t.Fatalf("seated: %q", got)
	}
	if got := T(Default, SysLeave, "을"); got != "을 님이 나갔어요." {
		t.Fatalf("leave: %q", got)
	}
	if got := T(Default, SysRateLimit); got != "너무 빨리 입력했어요. 잠깐만 기다리세요." {
		t.Fatalf("rate: %q", got)
	}
	if got := T(Default, PracticeGain); got != "조금 익숙해진 것 같다." {
		t.Fatalf("gain fallback: %q", got)
	}
	if got := T(Default, PracticeMiss); got != "아직 손에 안 익는다." {
		t.Fatalf("miss: %q", got)
	}
	if got := T(Default, PracticeUnknown); got != "그런 기술은 없어요." {
		t.Fatalf("unknown skill: %q", got)
	}
	if got := T(Default, SheetTitle, "아무개"); got != "불리는 이름: 아무개" {
		t.Fatalf("title: %q", got)
	}
	if got := T(Default, SheetWallet, 12); got != "주머니: 12냥" {
		t.Fatalf("wallet: %q", got)
	}
	if EulReul("쇠망치") != "를" || EulReul("식칼") != "을" {
		t.Fatalf("eul/reul: %s %s", EulReul("쇠망치"), EulReul("식칼"))
	}
	if got := T(LocaleEN, "missing.key"); got != "missing.key" {
		t.Fatalf("missing: %q", got)
	}
}

func TestDirLabel(t *testing.T) {
	if got := DirLabel(Default, "north"); got != "북쪽" {
		t.Fatalf("north: %q", got)
	}
	if got := DirLabel(LocaleEN, "up"); got != "위" {
		t.Fatalf("en fallback up: %q", got)
	}
	if got := DirLabel(Default, "sideways"); got != "sideways" {
		t.Fatalf("unknown dir: %q", got)
	}
}

func TestENOverride(t *testing.T) {
	prev := catalog[CmdUnknown]
	catalog[CmdUnknown] = entry{ko: prev.ko, en: "Unknown command."}
	t.Cleanup(func() { catalog[CmdUnknown] = prev })
	if got := T(LocaleEN, CmdUnknown); got != "Unknown command." {
		t.Fatalf("en override: %q", got)
	}
	if got := T(LocaleKO, CmdUnknown); got != prev.ko {
		t.Fatalf("ko unchanged: %q", got)
	}
}
