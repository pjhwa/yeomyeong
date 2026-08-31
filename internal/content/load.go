// Package content loads YAML rooms from content/zones/<zone>/rooms.yaml
// (docs/CONTENT-SCHEMA.md). Test fixtures use the test: zone, never 달빛골.
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

// Sentinel load failures (CONTENT-SCHEMA). The process must not boot through these.
var (
	ErrSpawnMissing      = errors.New("spawn missing")
	ErrUnknownExit       = errors.New("unknown exit target")
	ErrUnknownFlag       = errors.New("unknown flag")
	ErrUnknownForeshadow = errors.New("unknown foreshadow")
	ErrUnreachable       = errors.New("unreachable room")
)

var (
	zoneRe    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	roomIDRe  = regexp.MustCompile(`^([a-z][a-z0-9-]*):([a-z][a-z0-9-]*)$`)
	ledgerRow = regexp.MustCompile(`(?m)^\|\s*(FS-\d{3})\s*\|`)
)

var knownDirs = map[string]struct{}{
	"north": {}, "south": {}, "east": {}, "west": {}, "up": {}, "down": {},
}

var knownFlags = map[string]struct{}{
	"safe": {}, "town": {}, "market": {}, "indoor": {}, "dark": {},
	"forge": {}, "kitchen": {}, "press": {}, "clinic": {}, "yard": {},
	"salon": {}, "checkpoint": {},
}

// loc is a YAML localized string: a bare Korean scalar or {ko, en}.
type loc struct {
	KO string
	EN string
}

func (l *loc) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		var s string
		if err := n.Decode(&s); err != nil {
			return err
		}
		l.KO = strings.TrimSpace(s)
		return nil
	case yaml.MappingNode:
		var raw struct {
			KO string `yaml:"ko"`
			EN string `yaml:"en"`
		}
		if err := n.Decode(&raw); err != nil {
			return err
		}
		l.KO = strings.TrimSpace(raw.KO)
		l.EN = strings.TrimSpace(raw.EN)
		return nil
	default:
		return fmt.Errorf("localized string must be a Korean scalar or {ko, en}")
	}
}

type yamlRoom struct {
	ID           string            `yaml:"id"`
	Name         loc               `yaml:"name"`
	Description  loc               `yaml:"description"`
	Exits        map[string]string `yaml:"exits"`
	Flags        []string          `yaml:"flags"`
	Market       string            `yaml:"market"`
	HeatModifier *float64          `yaml:"heat_modifier"`
	Ambient      []loc             `yaml:"ambient"`
	Foreshadow   []string          `yaml:"foreshadow"`
}

// Load reads rooms, optional items, and optional spawns, then returns the room catalog.
// spawnID must exist (dalbitgol:gate on the real tree; testdata uses test:start).
func Load(root, spawnID string) (*world.Catalog, error) {
	w, err := LoadWorld(root, spawnID)
	if err != nil {
		return nil, err
	}
	return w.Rooms, nil
}

// Reachable reports room ids reachable from start (including start).
// An unknown start yields an empty set.
func Reachable(cat *world.Catalog, start string) map[string]bool {
	seen := make(map[string]bool)
	if cat == nil {
		return seen
	}
	if _, ok := cat.Room(start); !ok {
		return seen
	}
	q := []string{start}
	seen[start] = true
	for len(q) > 0 {
		id := q[0]
		q = q[1:]
		r, _ := cat.Room(id)
		for _, dest := range r.Exits {
			if !seen[dest] {
				seen[dest] = true
				q = append(q, dest)
			}
		}
	}
	return seen
}

func parseRoomsFile(path, zone string) ([]world.Room, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []yamlRoom
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := make([]world.Room, 0, len(raw))
	for i, yr := range raw {
		r, err := toRoom(yr, zone)
		if err != nil {
			return nil, fmt.Errorf("%s: room %d: %w", path, i, err)
		}
		out = append(out, r)
	}
	return out, nil
}

func toRoom(yr yamlRoom, zone string) (world.Room, error) {
	if yr.ID == "" {
		return world.Room{}, fmt.Errorf("missing id")
	}
	m := roomIDRe.FindStringSubmatch(yr.ID)
	if m == nil {
		return world.Room{}, fmt.Errorf("id %q must be zone:slug", yr.ID)
	}
	if m[1] != zone {
		return world.Room{}, fmt.Errorf("id %q: zone does not match directory %q", yr.ID, zone)
	}
	if yr.Name.KO == "" {
		return world.Room{}, fmt.Errorf("room %q: name.ko is required", yr.ID)
	}
	if yr.Description.KO == "" {
		return world.Room{}, fmt.Errorf("room %q: description.ko is required", yr.ID)
	}
	for dir := range yr.Exits {
		if _, ok := knownDirs[dir]; !ok {
			return world.Room{}, fmt.Errorf("room %q: unknown exit direction %q", yr.ID, dir)
		}
		if yr.Exits[dir] == "" {
			return world.Room{}, fmt.Errorf("room %q: empty exit target for %s", yr.ID, dir)
		}
	}
	for _, f := range yr.Flags {
		if _, ok := knownFlags[f]; !ok {
			return world.Room{}, fmt.Errorf("%w: %s on %s", ErrUnknownFlag, f, yr.ID)
		}
	}
	ambient := make([]world.Localized, 0, len(yr.Ambient))
	for _, a := range yr.Ambient {
		if a.KO == "" {
			return world.Room{}, fmt.Errorf("room %q: ambient entry missing ko", yr.ID)
		}
		ambient = append(ambient, world.Localized{KO: a.KO, EN: a.EN})
	}
	heat := 1.0
	if yr.HeatModifier != nil {
		heat = *yr.HeatModifier
	}
	return world.Room{
		ID:           yr.ID,
		Name:         world.Localized{KO: yr.Name.KO, EN: yr.Name.EN},
		Description:  world.Localized{KO: yr.Description.KO, EN: yr.Description.EN},
		Exits:        yr.Exits,
		Flags:        yr.Flags,
		Market:       strings.TrimSpace(yr.Market),
		HeatModifier: heat,
		Ambient:      ambient,
		Foreshadow:   yr.Foreshadow,
	}, nil
}

func checkForeshadow(root string, rooms []world.Room) error {
	used := make([]string, 0)
	seen := make(map[string]struct{})
	for _, r := range rooms {
		for _, id := range r.Foreshadow {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			used = append(used, id)
		}
	}
	if len(used) == 0 {
		return nil
	}
	path := findLedger(root)
	if path == "" {
		return fmt.Errorf("docs/FORESHADOW.md not found; cannot validate %v", used)
	}
	ledger, err := parseLedger(path)
	if err != nil {
		return err
	}
	for _, id := range used {
		if _, ok := ledger[id]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownForeshadow, id)
		}
	}
	return nil
}

func checkReachable(cat *world.Catalog, spawn string) error {
	seen := Reachable(cat, spawn)
	zone := zoneOf(spawn)
	for _, id := range cat.IDs() {
		if zoneOf(id) != zone {
			continue
		}
		if !seen[id] {
			return fmt.Errorf("%w: %s from %s", ErrUnreachable, id, spawn)
		}
	}
	return nil
}

func findLedger(from string) string {
	dir, err := filepath.Abs(from)
	if err != nil {
		return ""
	}
	for {
		cand := filepath.Join(dir, "docs", "FORESHADOW.md")
		if st, err := os.Stat(cand); err == nil && !st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func parseLedger(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]struct{})
	for _, m := range ledgerRow.FindAllSubmatch(data, -1) {
		ids[string(m[1])] = struct{}{}
	}
	return ids, nil
}

func zoneOf(id string) string {
	z, _, _ := strings.Cut(id, ":")
	return z
}
