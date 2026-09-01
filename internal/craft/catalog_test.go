package craft

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/world"
)

func TestLoadAndStock(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "nodes.yaml"), `
- {room: test:start, skill: forage, item: herb, stock: 2, regen_ticks: 3}
- {room: test:yard, skill: forage, item: ore, stock: 1, regen_ticks: 0}
`)
	write(t, filepath.Join(dir, "recipes.yaml"), `
- id: nail
  skill: smith
  flag: forge
  tool: hammer
  in: [{id: ore, n: 2}]
  out: {id: nail, n: 1}
  gain: ["못이 됐다."]
`)
	rooms, err := world.NewCatalog([]world.Room{
		{ID: "test:start", Name: world.Localized{KO: "시작"}, Description: world.Localized{KO: "설명이다."}},
		{ID: "test:yard", Name: world.Localized{KO: "마당"}, Description: world.Localized{KO: "설명이다."}, Exits: map[string]string{"south": "test:start"}},
	}, "test:start")
	if err != nil {
		t.Fatal(err)
	}
	items, err := world.NewItems([]world.Item{
		{ID: "herb", Name: world.Localized{KO: "쑥"}, Weight: 1},
		{ID: "ore", Name: world.Localized{KO: "쇠조각"}, Weight: 1},
		{ID: "nail", Name: world.Localized{KO: "쇠못"}, Weight: 1},
		{ID: "hammer", Name: world.Localized{KO: "쇠망치"}, Weight: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	sk, err := skill.Load(filepath.Join("..", "..", "content", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	cat, err := Load(dir, rooms, items, sk)
	if err != nil {
		t.Fatal(err)
	}
	if n := cat.NodesIn("test:start"); len(n) != 1 || n[0].Item != "herb" {
		t.Fatalf("nodes %+v", n)
	}
	r, ok := cat.LookupRecipe("nail")
	if !ok || r.Out.ID != "nail" || len(r.In) != 1 || r.Tool != "hammer" {
		t.Fatalf("recipe %+v", r)
	}
	if _, ok := cat.LookupRecipe("herb"); ok {
		t.Fatal("herb is not a recipe")
	}
	here := cat.RecipesHere([]string{"forge"})
	if len(here) != 1 {
		t.Fatalf("here %d", len(here))
	}
	if len(cat.RecipesHere([]string{"kitchen"})) != 0 {
		t.Fatal("kitchen")
	}

	st := cat.NewStock()
	if st.Remaining("test:start", "herb") != 2 {
		t.Fatal("boot stock")
	}
	if !st.Take("test:start", "herb") {
		t.Fatal("take 1")
	}
	if !st.Take("test:start", "herb") {
		t.Fatal("take 2")
	}
	if st.Take("test:start", "herb") {
		t.Fatal("empty")
	}
	st.Regen(3)
	if st.Remaining("test:start", "herb") != 1 {
		t.Fatalf("regen %d", st.Remaining("test:start", "herb"))
	}
	st.Put("test:start", "herb")
	if st.Remaining("test:start", "herb") != 2 {
		t.Fatal("put")
	}
	st.Take("test:yard", "ore")
	st.Regen(3)
	if st.Remaining("test:yard", "ore") != 0 {
		t.Fatal("regen 0 means no restore")
	}
}

func TestLoadMissingDir(t *testing.T) {
	cat, err := Load(filepath.Join(t.TempDir(), "nope"), nil, nil, nil)
	if err != nil || len(cat.Nodes()) != 0 || len(cat.Recipes()) != 0 {
		t.Fatalf("%v %+v", err, cat)
	}
}

func TestLoadErrors(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "nodes.yaml"), "- {room: missing, skill: forage, item: herb, stock: 1}\n")
	rooms, _ := world.NewCatalog([]world.Room{
		{ID: "test:start", Name: world.Localized{KO: "시작"}, Description: world.Localized{KO: "설명이다."}},
	}, "test:start")
	items, _ := world.NewItems([]world.Item{{ID: "herb", Name: world.Localized{KO: "쑥"}}})
	sk, err := skill.Load(filepath.Join("..", "..", "content", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, rooms, items, sk); err == nil || !strings.Contains(err.Error(), "unknown room") {
		t.Fatalf("room: %v", err)
	}
}

func TestNoMutex(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(b, []byte("sync.Mutex")) || bytes.Contains(b, []byte("sync.RWMutex")) {
			t.Errorf("%s: craft catalog is loop-owned; no mutex", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
