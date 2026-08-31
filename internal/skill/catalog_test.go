package skill

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLoadOK(t *testing.T) {
	cat := loadOK(t)
	if cat.Len() != 3 {
		t.Fatalf("len=%d", cat.Len())
	}
	ids := cat.IDs()
	if len(ids) != 3 || ids[0] != "test:swing" || ids[2] != "test:cook" {
		t.Fatalf("ids=%v", ids)
	}
	sk, ok := cat.Skill("test:swing")
	if !ok || sk.Name.KO != "휘두르기" || sk.Name.Text("en") != "Swing" || sk.Group != groupCombat || sk.Stat != statStr {
		t.Fatalf("swing: %+v", sk)
	}
	talk, _ := cat.Skill("test:talk")
	if talk.Name.Text("en") != "말솜씨" {
		t.Fatalf("bare: %+v", talk)
	}
	cook, _ := cat.Skill("test:cook")
	if cook.PracticeItem != "kitchen-knife" {
		t.Fatalf("cook: %+v", cook)
	}
	if _, ok := cat.Skill("missing"); ok {
		t.Fatal("missing")
	}
	rules := cat.TitleRules()
	if len(rules) != 3 || rules[2].Title.KO != "아무개" {
		t.Fatalf("titles: %+v", rules)
	}
	rules[0].Require["hacked"] = 1
	if _, ok := cat.TitleRules()[0].Require["hacked"]; ok {
		t.Fatal("mutated")
	}
}

func TestLoadM2(t *testing.T) {
	cat, err := Load(repoFile(t, filepath.Join("content", "skills", "m2.yaml")))
	if err != nil {
		t.Fatal(err)
	}
	if cat.Len() != 15 {
		t.Fatalf("skills=%d", cat.Len())
	}
	var n [4]int
	for _, id := range cat.IDs() {
		sk, _ := cat.Skill(id)
		if sk.Name.KO == "" || strings.HasPrefix(id, "test:") {
			t.Fatalf("bad %s %+v", id, sk)
		}
		switch sk.Group {
		case groupCombat:
			n[0]++
		case groupCraft:
			n[1]++
		case groupSocial:
			n[2]++
		case groupGather:
			n[3]++
		default:
			t.Fatalf("group %s", sk.Group)
		}
	}
	if n != [4]int{6, 4, 4, 1} {
		t.Fatalf("groups %v", n)
	}
	rules := cat.TitleRules()
	if n := len(rules); n == 0 || rules[n-1].Title.KO != "아무개" || len(rules[n-1].Require) != 0 {
		t.Fatalf("아무개 last: %+v", rules)
	}
	smith, ok := cat.Lookup("두드리다")
	if !ok || smith.Name.KO != "대장" || len(smith.Gain) == 0 || len(smith.Miss) == 0 {
		t.Fatalf("두드리다: %+v %v", smith, ok)
	}
	speech, ok := cat.Lookup("이야기하다")
	if !ok || speech.Name.KO != "말솜씨" || speech.PracticeFlag != "salon" {
		t.Fatalf("이야기하다: %+v %v", speech, ok)
	}
	if Band(3) != "초보" || Band(15) != "초보" || Band(20) != "수련" {
		t.Fatalf("band %s %s %s", Band(3), Band(15), Band(20))
	}
}

func TestLoadDirAndErrors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yaml"), "skills:\n  - id: test:a\n    name: 에이\n    group: combat\n    stat: str\ntitles:\n  - id: nobody\n    require: {}\n    title: { ko: 아무개 }\n")
	writeFile(t, filepath.Join(dir, "b.yaml"), "skills:\n  - id: test:b\n    name: 비\n    group: craft\n    stat: dex\n")
	writeFile(t, filepath.Join(dir, "skip.txt"), "x")
	cat, err := Load(dir)
	if err != nil || cat.Len() != 2 {
		t.Fatalf("merge: %v len=%d", err, cat.Len())
	}
	if _, err := Load(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "c.yaml"), "skills:\n  - id: test:a\n    name: 에이\n    group: combat\n    stat: str\n")
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "duplicate skill") {
		t.Fatalf("cross-file dup: %v", err)
	}
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("missing")
	}

	cases := []struct{ name, yaml, sub string }{
		{"yaml", "skills: [\n", ""},
		{"id", "skills:\n  - name: 격\n    group: combat\n    stat: str\n", "missing id"},
		{"slug", skillDoc("id: NotASlug\n    name: 격\n    group: combat\n    stat: str\n"), "slug"},
		{"name", skillDoc("id: test:x\n    group: combat\n    stat: str\n"), "name.ko"},
		{"loc", skillDoc("id: test:x\n    name: [\"격\"]\n    group: combat\n    stat: str\n"), "localized"},
		{"group", skillDoc("id: test:x\n    name: 격\n    group: wizard\n    stat: str\n"), "unknown group"},
		{"stat", skillDoc("id: test:x\n    name: 격\n    group: combat\n    stat: mana\n"), "unknown stat"},
		{"flag", skillDoc("id: test:x\n    name: 격\n    group: combat\n    stat: str\n    practice_flag: arena\n"), "practice_flag"},
		{"dup-skill", "skills:\n  - id: test:x\n    name: 격\n    group: combat\n    stat: str\n  - id: test:x\n    name: 격\n    group: combat\n    stat: str\n", "duplicate skill"},
		{"title-id", titlesAfterX("- id: NotGood\n    require: {}\n    title: { ko: 호 }\n"), "slug"},
		{"title-ko", titlesAfterX("- id: nobody\n    require: {}\n    title: { en: X }\n"), "title.ko"},
		{"req", titlesAfterX("- id: nobody\n    require: { \"\": 1 }\n    title: { ko: 호 }\n"), "empty require"},
		{"need", titlesAfterX("- id: nobody\n    require: { test:x: 101 }\n    title: { ko: 호 }\n"), "out of"},
		{"unknown", titlesAfterX("- id: ghost\n    require: { test:nope: 1 }\n    title: { ko: 호 }\n  - id: nobody\n    require: {}\n    title: { ko: 아무개 }\n"), "unknown skill"},
		{"middle", titlesAfterX("- id: nobody\n    require: {}\n    title: { ko: 아무개 }\n  - id: later\n    require: { test:x: 1 }\n    title: { ko: 뒤 }\n"), "must be last"},
		{"last", titlesAfterX("- id: only\n    require: { test:x: 1 }\n    title: { ko: 호 }\n"), "empty require"},
		{"dup-title", titlesAfterX("- id: nobody\n    require: { test:x: 1 }\n    title: { ko: 호 }\n  - id: nobody\n    require: {}\n    title: { ko: 아무개 }\n"), "duplicate title"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(writeYAML(t, tc.yaml))
			if err == nil || (tc.sub != "" && !strings.Contains(err.Error(), tc.sub)) {
				t.Fatalf("got %v want %q", err, tc.sub)
			}
		})
	}
}

func TestTitlePick(t *testing.T) {
	cat := loadOK(t)
	s := cat.NewSheet()
	if s.Title().KO != "아무개" {
		t.Fatalf("default %+v", s.Title())
	}
	if cat.Title(s.WithRank("test:talk", 15)).KO != "말을 부리는 자" {
		t.Fatal("talk")
	}
	if cat.Title(s.WithRank("test:cook", 15)).KO != "부뚜막의 손" {
		t.Fatal("craft title")
	}
	both := s.WithRank("test:cook", 15).WithRank("test:talk", 15)
	if both.Title().KO != "부뚜막의 손" {
		t.Fatal("first match")
	}
	if cat.Title(s.WithRank("test:cook", 14)).KO != "아무개" {
		t.Fatal("under")
	}
	if (Sheet{}).Title() != (Localized{}) {
		t.Fatal("unbound")
	}
	var c *Catalog
	if c.Len() != 0 || c.IDs() != nil || c.TitleRules() != nil || c.Title(Sheet{}) != (Localized{}) {
		t.Fatal("nil catalog")
	}
	if _, ok := c.Skill("x"); ok {
		t.Fatal("nil skill")
	}
	if _, ok := c.Lookup("test:swing"); ok {
		t.Fatal("nil lookup")
	}
}

func TestLookupIDAndName(t *testing.T) {
	cat := loadOK(t)
	if sk, ok := cat.Lookup("test:swing"); !ok || sk.Name.KO != "휘두르기" {
		t.Fatalf("id: %+v %v", sk, ok)
	}
	if sk, ok := cat.Lookup("TEST:SWING"); !ok || sk.ID != "test:swing" {
		t.Fatalf("fold: %+v %v", sk, ok)
	}
	if sk, ok := cat.Lookup("말솜씨"); !ok || sk.ID != "test:talk" {
		t.Fatalf("ko: %+v %v", sk, ok)
	}
	if sk, ok := cat.Lookup("Swing"); !ok || sk.ID != "test:swing" {
		t.Fatalf("en: %+v %v", sk, ok)
	}
	if _, ok := cat.Lookup("nope"); ok {
		t.Fatal("missing")
	}
	if _, ok := cat.Lookup("  "); ok {
		t.Fatal("blank")
	}
}

func TestNoLevelClassXPOrHardcodedIDs(t *testing.T) {
	cat, err := Load(repoFile(t, filepath.Join("content", "skills", "m2.yaml")))
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]struct{}{}
	for _, id := range cat.IDs() {
		ids[id] = struct{}{}
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for name, f := range pkg.Files {
			test := strings.HasSuffix(name, "_test.go")
			ast.Inspect(f, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && !test {
					switch id.Name {
					case "Level", "XPBar", "Class":
						t.Errorf("%s: forbidden identifier %s", name, id.Name)
					}
				}
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if _, hit := ids[s]; hit {
					t.Errorf("%s: hardcoded skill id %q", name, s)
				}
				return true
			})
		}
	}
}

func loadOK(t *testing.T) *Catalog {
	t.Helper()
	cat, err := Load(filepath.Join("testdata", "ok.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func writeYAML(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "skills.yaml")
	writeFile(t, p, body)
	return p
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skillDoc(fields string) string { return "skills:\n  - " + fields }

func titlesAfterX(body string) string {
	return "skills:\n  - id: test:x\n    name: 격\n    group: combat\n    stat: str\ntitles:\n  " + body
}

func repoFile(t *testing.T, rel string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(dir, rel)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("not found: %s", rel)
		}
		dir = parent
	}
}
