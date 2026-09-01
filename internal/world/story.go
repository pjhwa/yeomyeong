package world

import (
	"fmt"
	"sort"
	"strings"
)

// TalkFlag is the per-player sheet flag set after the first talk with npcID.
func TalkFlag(npcID string) string {
	if npcID == "" {
		return ""
	}
	return npcID + "_talked"
}

// ExaminedFlag is the per-player sheet flag set after the first after_examine
// reaction for objectID. Description still prints on later looks.
func ExaminedFlag(objectID string) string {
	if objectID == "" {
		return ""
	}
	return "examined:" + objectID
}

// FirstMarketSaleFlag is set once after the player's first successful market sell.
const FirstMarketSaleFlag = "first_market_sale"

// TalkWhen is one optional flag-gated talk line (D-046). Flag is an opaque
// sheet key; the loop picks the first entry whose Flags[Flag] > 0.
type TalkWhen struct {
	Flag string
	Line Localized
}

// DescWhen is one optional flag-gated object description (D-047). Same pick
// rule as TalkWhen: first matching flag >0 wins over the base Description.
type DescWhen struct {
	Flag string
	Line Localized
}

// NPC is a scripted character from content/zones/<zone>/npcs.yaml.
type NPC struct {
	ID         string
	Room       string
	Name       Localized
	Aliases    []string
	Look       Localized
	TalkFirst  Localized
	TalkSecond Localized
	TalkWhen   []TalkWhen
	Foreshadow []string
}

// Object is a scenery examine target from content/zones/<zone>/objects.yaml.
type Object struct {
	ID             string
	Room           string
	Name           Localized
	Aliases        []string
	Description    Localized
	DescriptionWhen []DescWhen
	AfterExamine   Localized
	Foreshadow     []string
}

// NPCs is an immutable NPC catalog. Safe for concurrent reads after NewNPCs.
type NPCs struct {
	byID   map[string]NPC
	byRoom map[string][]string
}

// Objects is an immutable examine-target catalog.
type Objects struct {
	byID   map[string]Object
	byRoom map[string][]string
}

// NewNPCs copies NPCs into a frozen catalog. Duplicate ids fail.
func NewNPCs(list []NPC) (*NPCs, error) {
	m := make(map[string]NPC, len(list))
	rooms := make(map[string][]string)
	for _, n := range list {
		if n.ID == "" {
			return nil, fmt.Errorf("npc with empty id")
		}
		if _, dup := m[n.ID]; dup {
			return nil, fmt.Errorf("duplicate npc id %q", n.ID)
		}
		m[n.ID] = cloneNPC(n)
		if n.Room != "" {
			rooms[n.Room] = append(rooms[n.Room], n.ID)
		}
	}
	return &NPCs{byID: m, byRoom: rooms}, nil
}

// NewObjects copies objects into a frozen catalog. Duplicate ids fail.
func NewObjects(list []Object) (*Objects, error) {
	m := make(map[string]Object, len(list))
	rooms := make(map[string][]string)
	for _, o := range list {
		if o.ID == "" {
			return nil, fmt.Errorf("object with empty id")
		}
		if _, dup := m[o.ID]; dup {
			return nil, fmt.Errorf("duplicate object id %q", o.ID)
		}
		m[o.ID] = cloneObject(o)
		if o.Room != "" {
			rooms[o.Room] = append(rooms[o.Room], o.ID)
		}
	}
	return &Objects{byID: m, byRoom: rooms}, nil
}

// Len returns the number of NPCs.
func (c *NPCs) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byID)
}

// Len returns the number of objects.
func (c *Objects) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byID)
}

// InRoom returns NPCs standing in room, sorted by Korean name.
func (c *NPCs) InRoom(room string) []NPC {
	if c == nil || room == "" {
		return nil
	}
	ids := c.byRoom[room]
	out := make([]NPC, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneNPC(c.byID[id]))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name.KO == out[j].Name.KO {
			return out[i].ID < out[j].ID
		}
		return out[i].Name.KO < out[j].Name.KO
	})
	return out
}

// InRoom returns objects in room, sorted by Korean name.
func (c *Objects) InRoom(room string) []Object {
	if c == nil || room == "" {
		return nil
	}
	ids := c.byRoom[room]
	out := make([]Object, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneObject(c.byID[id]))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name.KO == out[j].Name.KO {
			return out[i].ID < out[j].ID
		}
		return out[i].Name.KO < out[j].Name.KO
	})
	return out
}

// FindInRoom matches q against id, name, or aliases among NPCs in room.
func (c *NPCs) FindInRoom(room, q string) (NPC, bool) {
	for _, n := range c.InRoom(room) {
		if matchQuery(q, n.ID, n.Name, n.Aliases) {
			return n, true
		}
	}
	return NPC{}, false
}

// FindInRoom matches q against id, name, or aliases among objects in room.
func (c *Objects) FindInRoom(room, q string) (Object, bool) {
	for _, o := range c.InRoom(room) {
		if matchQuery(q, o.ID, o.Name, o.Aliases) {
			return o, true
		}
	}
	return Object{}, false
}

// Find matches q against any NPC (any room).
func (c *NPCs) Find(q string) (NPC, bool) {
	if c == nil {
		return NPC{}, false
	}
	for _, id := range c.IDs() {
		n := c.byID[id]
		if matchQuery(q, n.ID, n.Name, n.Aliases) {
			return cloneNPC(n), true
		}
	}
	return NPC{}, false
}

// IDs returns sorted NPC ids.
func (c *NPCs) IDs() []string {
	if c == nil {
		return nil
	}
	ids := make([]string, 0, len(c.byID))
	for id := range c.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// IDs returns sorted object ids.
func (c *Objects) IDs() []string {
	if c == nil {
		return nil
	}
	ids := make([]string, 0, len(c.byID))
	for id := range c.byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func matchQuery(q, id string, name Localized, aliases []string) bool {
	q = strings.TrimSpace(q)
	if q == "" {
		return false
	}
	if strings.EqualFold(q, id) {
		return true
	}
	if q == name.KO || (name.EN != "" && (q == name.EN || strings.EqualFold(q, name.EN))) {
		return true
	}
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if q == a || strings.EqualFold(q, a) {
			return true
		}
	}
	return false
}

func cloneNPC(n NPC) NPC {
	out := n
	if n.Aliases != nil {
		out.Aliases = append([]string(nil), n.Aliases...)
	}
	if n.TalkWhen != nil {
		out.TalkWhen = append([]TalkWhen(nil), n.TalkWhen...)
	}
	if n.Foreshadow != nil {
		out.Foreshadow = append([]string(nil), n.Foreshadow...)
	}
	return out
}

func cloneObject(o Object) Object {
	out := o
	if o.Aliases != nil {
		out.Aliases = append([]string(nil), o.Aliases...)
	}
	if o.DescriptionWhen != nil {
		out.DescriptionWhen = append([]DescWhen(nil), o.DescriptionWhen...)
	}
	if o.Foreshadow != nil {
		out.Foreshadow = append([]string(nil), o.Foreshadow...)
	}
	return out
}
