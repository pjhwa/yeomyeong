package world

import (
	"fmt"
	"sort"
)

// Wear slots on the M2 sheet (CONTENT-SCHEMA).
const (
	SlotNone     = "none"
	SlotMainHand = "main_hand"
	SlotBody     = "body"
)

// Item is one catalog entry from content/items/*.yaml. Immutable after NewItems.
type Item struct {
	ID          string
	Name        Localized
	Description Localized
	Slot        string
	Skills      []string
	Weight      int
}

// Items is an immutable item catalog. Safe for concurrent reads after NewItems.
type Items struct {
	byID map[string]Item
}

// NewItems copies items into a frozen catalog. Empty input yields an empty catalog.
func NewItems(items []Item) (*Items, error) {
	m := make(map[string]Item, len(items))
	for _, it := range items {
		if it.ID == "" {
			return nil, fmt.Errorf("item with empty id")
		}
		if _, dup := m[it.ID]; dup {
			return nil, fmt.Errorf("duplicate item id %q", it.ID)
		}
		if it.Weight < 0 {
			return nil, fmt.Errorf("item %q: negative weight", it.ID)
		}
		m[it.ID] = cloneItem(it)
	}
	return &Items{byID: m}, nil
}

// Get returns a defensive copy of the item.
func (c *Items) Get(id string) (Item, bool) {
	if c == nil {
		return Item{}, false
	}
	it, ok := c.byID[id]
	if !ok {
		return Item{}, false
	}
	return cloneItem(it), true
}

// Len returns the number of items.
func (c *Items) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byID)
}

// IDs returns sorted item ids.
func (c *Items) IDs() []string {
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

// Find looks up by id or Korean/English name.
func (c *Items) Find(q string) (Item, bool) {
	if c == nil {
		return Item{}, false
	}
	if it, ok := c.Get(q); ok {
		return it, true
	}
	for _, id := range c.IDs() {
		it := c.byID[id]
		if it.Name.KO == q || it.Name.EN == q {
			return cloneItem(it), true
		}
	}
	return Item{}, false
}

func cloneItem(it Item) Item {
	out := it
	if it.Skills != nil {
		out.Skills = append([]string(nil), it.Skills...)
	}
	return out
}

// Stack is one ground (or later bag) pile. Qty is ≥ 1 when stored.
type Stack struct {
	ID  string `json:"id"`
	Qty int    `json:"n"`
}

// CloneStacks copies a pile list. Nil becomes an empty slice.
func CloneStacks(in []Stack) []Stack {
	if in == nil {
		return []Stack{}
	}
	out := make([]Stack, len(in))
	copy(out, in)
	return out
}

// CloneGround copies room → piles. Nil becomes an empty map.
func CloneGround(in map[string][]Stack) map[string][]Stack {
	out := make(map[string][]Stack, len(in))
	for id, piles := range in {
		out[id] = CloneStacks(piles)
	}
	return out
}

// AddStack merges n of id into bag. n ≤ 0 is a no-op.
func AddStack(bag []Stack, id string, n int) []Stack {
	if n <= 0 || id == "" {
		return bag
	}
	for i := range bag {
		if bag[i].ID == id {
			bag[i].Qty += n
			return bag
		}
	}
	return append(bag, Stack{ID: id, Qty: n})
}

// TakeStack removes n of id. ok is false when the pile is missing or short.
func TakeStack(bag []Stack, id string, n int) ([]Stack, bool) {
	if n <= 0 || id == "" {
		return bag, false
	}
	for i := range bag {
		if bag[i].ID != id || bag[i].Qty < n {
			continue
		}
		bag[i].Qty -= n
		if bag[i].Qty == 0 {
			return append(bag[:i], bag[i+1:]...), true
		}
		return bag, true
	}
	return bag, false
}
