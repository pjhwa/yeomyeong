// Package craft is gather nodes and recipes (PLAN.md §3.4).
// Catalog is immutable after Load. Stock is mutable and owned by the loop.
package craft

import (
	"strings"
)

// Node is one harvest point. Stock in the catalog is the boot maximum.
type Node struct {
	Room, Skill, Item string
	Stock, RegenTicks int
}

// Input is one consumed pile.
type Input struct {
	ID  string
	Qty int
}

// Recipe consumes inputs and yields Out.
type Recipe struct {
	ID, Skill, Flag, Tool string
	In                    []Input
	Out                   Input
	Gain, Miss            []string
}

// Catalog is an immutable node+recipe table. Safe for concurrent reads after Load.
type Catalog struct {
	nodes   []Node
	byRoom  map[string][]int
	recipes []Recipe
	byOut   map[string]int
}

// Nodes returns a copy of the node list (file order).
func (c *Catalog) Nodes() []Node {
	if c == nil {
		return nil
	}
	return append([]Node(nil), c.nodes...)
}

// Recipes returns a copy of the recipe list (file order).
func (c *Catalog) Recipes() []Recipe {
	if c == nil {
		return nil
	}
	out := make([]Recipe, len(c.recipes))
	for i, r := range c.recipes {
		out[i] = cloneRecipe(r)
	}
	return out
}

// NodesIn are harvest points in room (catalog order).
func (c *Catalog) NodesIn(room string) []Node {
	if c == nil {
		return nil
	}
	idx := c.byRoom[room]
	out := make([]Node, 0, len(idx))
	for _, i := range idx {
		out = append(out, c.nodes[i])
	}
	return out
}

// Node looks up room+item.
func (c *Catalog) Node(room, item string) (Node, bool) {
	for _, n := range c.NodesIn(room) {
		if n.Item == item {
			return n, true
		}
	}
	return Node{}, false
}

// LookupRecipe finds by id or by output item id (exact).
func (c *Catalog) LookupRecipe(q string) (Recipe, bool) {
	if c == nil {
		return Recipe{}, false
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return Recipe{}, false
	}
	for _, r := range c.recipes {
		if r.ID == q || r.Out.ID == q {
			return cloneRecipe(r), true
		}
	}
	return Recipe{}, false
}

// RecipesHere are recipes the room's flags can host (empty recipe flag is anywhere).
func (c *Catalog) RecipesHere(roomFlags []string) []Recipe {
	if c == nil {
		return nil
	}
	var out []Recipe
	for _, r := range c.recipes {
		if r.Flag == "" || hasFlag(roomFlags, r.Flag) {
			out = append(out, cloneRecipe(r))
		}
	}
	return out
}

func hasFlag(flags []string, f string) bool {
	for _, x := range flags {
		if x == f {
			return true
		}
	}
	return false
}

func cloneRecipe(r Recipe) Recipe {
	out := r
	if r.In != nil {
		out.In = append([]Input(nil), r.In...)
	}
	if r.Gain != nil {
		out.Gain = append([]string(nil), r.Gain...)
	}
	if r.Miss != nil {
		out.Miss = append([]string(nil), r.Miss...)
	}
	return out
}

func emptyCatalog() *Catalog {
	return &Catalog{byRoom: map[string][]int{}, byOut: map[string]int{}}
}
