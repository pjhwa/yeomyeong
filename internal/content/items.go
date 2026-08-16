package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pjhwa/yeomyeong/internal/world"
	"gopkg.in/yaml.v3"
)

var (
	ErrUnknownItem = errors.New("unknown item")
	ErrUnknownSlot = errors.New("unknown slot")
	itemIDRe       = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	knownSlots     = map[string]struct{}{
		world.SlotNone: {}, world.SlotMainHand: {}, world.SlotBody: {},
	}
)

type yamlItem struct {
	ID          string   `yaml:"id"`
	Name        loc      `yaml:"name"`
	Description loc      `yaml:"description"`
	Slot        string   `yaml:"slot"`
	Skills      []string `yaml:"skills"`
	Weight      *int     `yaml:"weight"`
}

type yamlSpawn struct {
	Room     string   `yaml:"room"`
	FlagsAdd []string `yaml:"flags_add"`
	Items    []string `yaml:"items"`
}

// World is rooms + items + boot ground piles (ground is world state).
type World struct {
	Rooms  *world.Catalog
	Items  *world.Items
	Ground map[string][]world.Stack
}

// LoadWorld loads rooms, optional items/, and optional zone spawns.yaml.
func LoadWorld(root, spawnID string) (*World, error) {
	rooms, order, err := loadRooms(root, spawnID)
	if err != nil {
		return nil, err
	}
	items, err := loadItems(root)
	if err != nil {
		return nil, err
	}
	ground, err := applySpawns(root, rooms, items)
	if err != nil {
		return nil, err
	}
	for i, r := range order {
		order[i] = rooms[r.ID]
	}
	if err := checkForeshadow(root, order); err != nil {
		return nil, err
	}
	cat, err := world.NewCatalog(order, spawnID)
	if err != nil {
		return nil, err
	}
	if err := checkReachable(cat, spawnID); err != nil {
		return nil, err
	}
	return &World{Rooms: cat, Items: items, Ground: ground}, nil
}

func loadRooms(root, spawnID string) (map[string]world.Room, []world.Room, error) {
	if spawnID == "" {
		return nil, nil, fmt.Errorf("%w: empty id", ErrSpawnMissing)
	}
	byID := make(map[string]world.Room)
	var order []world.Room
	err := eachZoneFile(root, "rooms.yaml", func(path, zone string) error {
		parsed, err := parseRoomsFile(path, zone)
		if err != nil {
			return err
		}
		for _, r := range parsed {
			if _, dup := byID[r.ID]; dup {
				return fmt.Errorf("duplicate room id %q", r.ID)
			}
			byID[r.ID] = r
			order = append(order, r)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if _, ok := byID[spawnID]; !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrSpawnMissing, spawnID)
	}
	for _, r := range order {
		for dir, dest := range r.Exits {
			if _, ok := byID[dest]; !ok {
				return nil, nil, fmt.Errorf("%w: %s %s -> %s", ErrUnknownExit, r.ID, dir, dest)
			}
		}
	}
	return byID, order, nil
}

func loadItems(root string) (*world.Items, error) {
	dir := filepath.Join(root, "items")
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return world.NewItems(nil)
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s: is not a directory", dir)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	var all []world.Item
	seen := map[string]string{}
	for _, path := range matches {
		items, err := parseItemsFile(path)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			if prev, dup := seen[it.ID]; dup {
				return nil, fmt.Errorf("duplicate item id %q (%s and %s)", it.ID, prev, path)
			}
			seen[it.ID] = path
			all = append(all, it)
		}
	}
	return world.NewItems(all)
}

func parseItemsFile(path string) ([]world.Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []yamlItem
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := make([]world.Item, 0, len(raw))
	for i, yi := range raw {
		it, err := toItem(yi)
		if err != nil {
			return nil, fmt.Errorf("%s: item %d: %w", path, i, err)
		}
		out = append(out, it)
	}
	return out, nil
}

func toItem(yi yamlItem) (world.Item, error) {
	if yi.ID == "" || !itemIDRe.MatchString(yi.ID) {
		return world.Item{}, fmt.Errorf("id %q must be [a-z][a-z0-9-]*", yi.ID)
	}
	if yi.Name.KO == "" || yi.Description.KO == "" {
		return world.Item{}, fmt.Errorf("item %q: name.ko and description.ko required", yi.ID)
	}
	slot := strings.TrimSpace(yi.Slot)
	if slot == "" {
		slot = world.SlotNone
	}
	if _, ok := knownSlots[slot]; !ok {
		return world.Item{}, fmt.Errorf("%w: %s on %s", ErrUnknownSlot, slot, yi.ID)
	}
	weight := 1
	if yi.Weight != nil {
		if *yi.Weight < 0 {
			return world.Item{}, fmt.Errorf("item %q: negative weight", yi.ID)
		}
		weight = *yi.Weight
	}
	return world.Item{
		ID: yi.ID, Name: world.Localized{KO: yi.Name.KO, EN: yi.Name.EN},
		Description: world.Localized{KO: yi.Description.KO, EN: yi.Description.EN},
		Slot:        slot, Skills: append([]string(nil), yi.Skills...), Weight: weight,
	}, nil
}

func applySpawns(root string, rooms map[string]world.Room, items *world.Items) (map[string][]world.Stack, error) {
	ground := map[string][]world.Stack{}
	err := eachZoneFile(root, "spawns.yaml", func(path, _ string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var raw []yamlSpawn
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for i, ys := range raw {
			if err := applyOneSpawn(ys, rooms, items, ground); err != nil {
				return fmt.Errorf("%s: spawn %d: %w", path, i, err)
			}
		}
		return nil
	})
	return ground, err
}

func applyOneSpawn(ys yamlSpawn, rooms map[string]world.Room, items *world.Items, ground map[string][]world.Stack) error {
	if ys.Room == "" {
		return fmt.Errorf("missing room")
	}
	r, ok := rooms[ys.Room]
	if !ok {
		return fmt.Errorf("unknown room %q", ys.Room)
	}
	for _, f := range ys.FlagsAdd {
		if _, ok := knownFlags[f]; !ok {
			return fmt.Errorf("%w: %s on %s", ErrUnknownFlag, f, ys.Room)
		}
		if !hasFlag(r.Flags, f) {
			r.Flags = append(r.Flags, f)
		}
	}
	for _, id := range ys.Items {
		if _, ok := items.Get(id); !ok {
			return fmt.Errorf("%w: %s on %s", ErrUnknownItem, id, ys.Room)
		}
		ground[ys.Room] = world.AddStack(ground[ys.Room], id, 1)
	}
	rooms[ys.Room] = r
	return nil
}

func eachZoneFile(root, name string, fn func(path, zone string) error) error {
	zonesDir := filepath.Join(root, "zones")
	entries, err := os.ReadDir(zonesDir)
	if err != nil {
		return fmt.Errorf("read zones: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || !zoneRe.MatchString(e.Name()) {
			continue
		}
		path := filepath.Join(zonesDir, e.Name(), name)
		st, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if st.IsDir() {
			return fmt.Errorf("%s: is a directory", path)
		}
		if err := fn(path, e.Name()); err != nil {
			return err
		}
	}
	return nil
}

func hasFlag(flags []string, f string) bool {
	for _, x := range flags {
		if x == f {
			return true
		}
	}
	return false
}
