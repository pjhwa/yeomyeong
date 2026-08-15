// Package world holds the immutable room catalog. Positions stay in engine
// (EVENT-BUS.md). The catalog has no mutex; it is read-only after load.
package world

import (
	"fmt"
	"sort"
)

// SpawnID is the M1 start room (D-028). Missing after a real-tree load is fatal.
const SpawnID = "dalbitgol:gate"

// Localized is a canonical {ko, en} string. Missing en falls back to ko (D-029).
type Localized struct {
	KO string
	EN string
}

// Text returns EN when locale is "en" and EN is set; otherwise KO.
func (l Localized) Text(locale string) string {
	if locale == "en" && l.EN != "" {
		return l.EN
	}
	return l.KO
}

// Room is one YAML room after load. Exits map dir → room id.
type Room struct {
	ID           string
	Name         Localized
	Description  Localized
	Exits        map[string]string
	Flags        []string
	Market       string
	HeatModifier float64
	Ambient      []Localized
	Foreshadow   []string
}

// Catalog is an immutable room graph. Safe for concurrent reads after NewCatalog.
type Catalog struct {
	rooms map[string]Room
	spawn string
}

// NewCatalog copies rooms into a frozen graph. spawn and every exit target must exist.
func NewCatalog(rooms []Room, spawn string) (*Catalog, error) {
	if spawn == "" {
		return nil, fmt.Errorf("empty spawn")
	}
	m := make(map[string]Room, len(rooms))
	for _, r := range rooms {
		if r.ID == "" {
			return nil, fmt.Errorf("room with empty id")
		}
		if _, dup := m[r.ID]; dup {
			return nil, fmt.Errorf("duplicate room id %q", r.ID)
		}
		m[r.ID] = cloneRoom(r)
	}
	if _, ok := m[spawn]; !ok {
		return nil, fmt.Errorf("spawn %q missing", spawn)
	}
	for _, r := range m {
		for dir, dest := range r.Exits {
			if _, ok := m[dest]; !ok {
				return nil, fmt.Errorf("exit %s from %s: missing %s", dir, r.ID, dest)
			}
		}
	}
	return &Catalog{rooms: m, spawn: spawn}, nil
}

// Spawn returns the configured start room id.
func (c *Catalog) Spawn() string { return c.spawn }

// Len returns the number of rooms.
func (c *Catalog) Len() int { return len(c.rooms) }

// Room returns a defensive copy of the room.
func (c *Catalog) Room(id string) (Room, bool) {
	r, ok := c.rooms[id]
	if !ok {
		return Room{}, false
	}
	return cloneRoom(r), true
}

// IDs returns sorted room ids.
func (c *Catalog) IDs() []string {
	ids := make([]string, 0, len(c.rooms))
	for id := range c.rooms {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Exit looks up dest for from+dir.
func (c *Catalog) Exit(from, dir string) (string, bool) {
	r, ok := c.rooms[from]
	if !ok {
		return "", false
	}
	dest, ok := r.Exits[dir]
	return dest, ok
}

func cloneRoom(r Room) Room {
	out := r
	if r.Exits != nil {
		out.Exits = make(map[string]string, len(r.Exits))
		for k, v := range r.Exits {
			out.Exits[k] = v
		}
	}
	if r.Flags != nil {
		out.Flags = append([]string(nil), r.Flags...)
	}
	if r.Ambient != nil {
		out.Ambient = append([]Localized(nil), r.Ambient...)
	}
	if r.Foreshadow != nil {
		out.Foreshadow = append([]string(nil), r.Foreshadow...)
	}
	return out
}
