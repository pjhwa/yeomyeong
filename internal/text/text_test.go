package text

import "testing"

func TestTKoreanAndFallback(t *testing.T) {
	if got := T(LocaleKO, CmdUnknown); got != "모르는 말입니다. say / quit" {
		t.Fatalf("unknown: %q", got)
	}
	if got := T(LocaleEN, MoveNoExit); got != "그쪽으로는 갈 수 없습니다." {
		t.Fatalf("en fallback: %q", got)
	}
	if got := T(Default, SysSeated, "갑"); got != "갑 님이 자리에 앉았습니다." {
		t.Fatalf("seated: %q", got)
	}
	if got := T(Default, SysLeave, "을"); got != "을 님이 자리를 떴습니다." {
		t.Fatalf("leave: %q", got)
	}
	if got := T(Default, SysRateLimit); got != "rate_limited" {
		t.Fatalf("rate: %q", got)
	}
	if got := T(Default, PracticeMiss); got != "아무것도 자리 잡지 않는다." {
		t.Fatalf("miss: %q", got)
	}
	if got := T(Default, SheetTitle, "아무개"); got != "호칭: 아무개" {
		t.Fatalf("title: %q", got)
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
