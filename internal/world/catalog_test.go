package world

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func sample() []Room {
	return []Room{
		{ID: "a", Name: Localized{KO: "에이"}, Description: Localized{KO: "설명"},
			Exits: map[string]string{"east": "b"}, Flags: []string{"safe"},
			Ambient: []Localized{{KO: "바람"}}, Foreshadow: []string{"FS-014"}},
		{ID: "b", Name: Localized{KO: "비", EN: "Bee"}, Description: Localized{KO: "설명"},
			Exits: map[string]string{"west": "a"}},
	}
}

func TestCatalog(t *testing.T) {
	cat, err := NewCatalog(sample(), "a")
	if err != nil {
		t.Fatal(err)
	}
	if cat.Spawn() != "a" || cat.Len() != 2 {
		t.Fatalf("spawn=%s len=%d", cat.Spawn(), cat.Len())
	}
	if ids := cat.IDs(); len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("ids=%v", ids)
	}
	if dest, ok := cat.Exit("a", "east"); !ok || dest != "b" {
		t.Fatalf("exit %q %v", dest, ok)
	}
	if _, ok := cat.Exit("a", "south"); ok {
		t.Fatal("missing dir")
	}
	if _, ok := cat.Exit("no", "east"); ok {
		t.Fatal("missing room exit")
	}
	if _, ok := cat.Room("no"); ok {
		t.Fatal("missing room")
	}

	in := sample()
	frozen, err := NewCatalog(in, "a")
	if err != nil {
		t.Fatal(err)
	}
	in[0].Exits["east"] = "hacked"
	got, _ := frozen.Room("a")
	got.Exits["east"] = "mutated"
	got.Flags[0] = "mutated"
	again, _ := frozen.Room("a")
	if again.Exits["east"] != "b" || again.Flags[0] != "safe" {
		t.Fatalf("mutated: %+v", again)
	}
}

func TestNewCatalogErrors(t *testing.T) {
	if _, err := NewCatalog(sample(), ""); err == nil {
		t.Fatal("empty spawn")
	}
	if _, err := NewCatalog(sample(), "missing"); err == nil {
		t.Fatal("missing spawn")
	}
	if _, err := NewCatalog([]Room{{ID: ""}}, "x"); err == nil {
		t.Fatal("empty id")
	}
	if _, err := NewCatalog([]Room{{ID: "a"}, {ID: "a"}}, "a"); err == nil {
		t.Fatal("dup")
	}
	if _, err := NewCatalog([]Room{{ID: "a", Exits: map[string]string{"north": "ghost"}}}, "a"); err == nil {
		t.Fatal("bad exit")
	}
}

func TestLocalizedText(t *testing.T) {
	both := Localized{KO: "한글", EN: "English"}
	if both.Text("ko") != "한글" || both.Text("en") != "English" {
		t.Fatalf("%+v", both)
	}
	if (Localized{KO: "한글"}).Text("en") != "한글" {
		t.Fatal("en fallback")
	}
}

func TestConcurrentReads(t *testing.T) {
	cat, err := NewCatalog(sample(), "a")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cat.Room("a")
			_, _ = cat.Exit("a", "east")
			_ = cat.IDs()
		}()
	}
	wg.Wait()
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
			t.Errorf("%s: catalog must not have a mutex", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
