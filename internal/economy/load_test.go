package economy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pjhwa/yeomyeong/internal/world"
)

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.yaml"), `
- id: dalbitgol
  name: 달빛골 장터
  goods:
    - {id: herb, base: 3, stock: 10, target: 8, demand: 0.8}
    - {id: nail, base: 10, stock: 2, target: 4, demand: 1.4}
- id: solgol
  name: {ko: 솔골 장터}
  goods:
    - {id: herb, base: 3, stock: 2, target: 5, demand: 1.5}
`)
	items, err := world.NewItems([]world.Item{
		{ID: "herb", Name: world.Localized{KO: "쑥"}, Weight: 1},
		{ID: "nail", Name: world.Localized{KO: "쇠못"}, Weight: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Load(dir, items)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.IDs(); len(got) != 2 || got[0] != "dalbitgol" {
		t.Fatalf("ids %v", got)
	}
	p1, _ := b.Quote("dalbitgol", "herb")
	p2, _ := b.Quote("solgol", "herb")
	if p1 >= p2 {
		t.Fatalf("loaded spread %d %d", p1, p2)
	}
}

func TestLoadMissingDir(t *testing.T) {
	b, err := Load(filepath.Join(t.TempDir(), "nope"), nil)
	if err != nil || len(b.IDs()) != 0 {
		t.Fatalf("%v %+v", err, b)
	}
}

func TestLoadErrors(t *testing.T) {
	items, _ := world.NewItems([]world.Item{{ID: "herb", Name: world.Localized{KO: "쑥"}}})
	cases := []struct{ name, yaml, sub string }{
		{"yaml", "- [\n", ""},
		{"id", "- name: 장터\n  goods: [{id: herb, base: 1, stock: 1}]\n", "slug"},
		{"name", "- id: m\n  goods: [{id: herb, base: 1, stock: 1}]\n", "name.ko"},
		{"nogoods", "- id: m\n  name: 장터\n  goods: []\n", "no goods"},
		{"base", "- id: m\n  name: 장터\n  goods: [{id: herb, base: 0, stock: 1}]\n", "base"},
		{"dupgood", "- id: m\n  name: 장터\n  goods: [{id: herb, base: 1, stock: 1}, {id: herb, base: 1, stock: 1}]\n", "duplicate good"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, filepath.Join(dir, "m.yaml"), tc.yaml)
			_, err := Load(dir, items)
			if err == nil || (tc.sub != "" && !strings.Contains(err.Error(), tc.sub)) {
				t.Fatalf("got %v want %q", err, tc.sub)
			}
		})
	}
	t.Run("unknown-item", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "m.yaml"), "- id: m\n  name: 장터\n  goods: [{id: ghost, base: 1, stock: 1}]\n")
		_, err := Load(dir, items)
		if err == nil || !strings.Contains(err.Error(), "unknown item") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("dup-market", func(t *testing.T) {
		dir := t.TempDir()
		write(t, filepath.Join(dir, "a.yaml"), "- id: m\n  name: 장터\n  goods: [{id: herb, base: 1, stock: 1}]\n")
		write(t, filepath.Join(dir, "b.yaml"), "- id: m\n  name: 장터\n  goods: [{id: herb, base: 1, stock: 1}]\n")
		_, err := Load(dir, items)
		if err == nil || !strings.Contains(err.Error(), "duplicate market") {
			t.Fatalf("got %v", err)
		}
	})
}

func write(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
