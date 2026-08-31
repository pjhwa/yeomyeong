package craft

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/world"
	"gopkg.in/yaml.v3"
)

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
		l.KO, l.EN = strings.TrimSpace(raw.KO), strings.TrimSpace(raw.EN)
		return nil
	default:
		return fmt.Errorf("localized string must be a Korean scalar or {ko, en}")
	}
}

type yamlNode struct {
	Room       string `yaml:"room"`
	Skill      string `yaml:"skill"`
	Item       string `yaml:"item"`
	Stock      *int   `yaml:"stock"`
	RegenTicks *int   `yaml:"regen_ticks"`
}

type yamlPile struct {
	ID string `yaml:"id"`
	N  *int   `yaml:"n"`
}

type yamlRecipe struct {
	ID    string     `yaml:"id"`
	Skill string     `yaml:"skill"`
	Flag  string     `yaml:"flag"`
	Tool  string     `yaml:"tool"`
	In    []yamlPile `yaml:"in"`
	Out   yamlPile   `yaml:"out"`
	Gain  []string   `yaml:"gain"`
	Miss  []string   `yaml:"miss"`
}

var knownStation = map[string]struct{}{
	"": {}, "forge": {}, "kitchen": {}, "press": {}, "clinic": {},
}

// Load reads dir/nodes.yaml and dir/recipes.yaml (each optional).
// Missing dir yields an empty catalog. rooms/items/skills, when non-nil,
// are required to contain every referenced id.
func Load(dir string, rooms *world.Catalog, items *world.Items, skills *skill.Catalog) (*Catalog, error) {
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyCatalog(), nil
		}
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s: is not a directory", dir)
	}
	nodes, err := loadNodes(filepath.Join(dir, "nodes.yaml"), rooms, items, skills)
	if err != nil {
		return nil, err
	}
	recipes, err := loadRecipes(filepath.Join(dir, "recipes.yaml"), items, skills)
	if err != nil {
		return nil, err
	}
	return newCatalog(nodes, recipes)
}

func loadNodes(path string, rooms *world.Catalog, items *world.Items, skills *skill.Catalog) ([]Node, error) {
	raw, err := readYAMLNodes(path)
	if err != nil {
		return nil, err
	}
	out := make([]Node, 0, len(raw))
	seen := map[string]struct{}{}
	for i, yn := range raw {
		n, err := toNode(yn)
		if err != nil {
			return nil, fmt.Errorf("%s: node %d: %w", path, i, err)
		}
		k := nodeKey(n.Room, n.Item)
		if _, dup := seen[k]; dup {
			return nil, fmt.Errorf("%s: duplicate node %s %s", path, n.Room, n.Item)
		}
		seen[k] = struct{}{}
		if rooms != nil {
			if _, ok := rooms.Room(n.Room); !ok {
				return nil, fmt.Errorf("%s: node %s: unknown room", path, n.Room)
			}
		}
		if items != nil {
			if _, ok := items.Get(n.Item); !ok {
				return nil, fmt.Errorf("%s: node %s: unknown item %s", path, n.Room, n.Item)
			}
		}
		if skills != nil {
			sk, ok := skills.Skill(n.Skill)
			if !ok {
				return nil, fmt.Errorf("%s: node %s: unknown skill %s", path, n.Room, n.Skill)
			}
			if sk.Group != "gather" {
				return nil, fmt.Errorf("%s: node %s: skill %s is not gather", path, n.Room, n.Skill)
			}
		}
		out = append(out, n)
	}
	return out, nil
}

func readYAMLNodes(path string) ([]yamlNode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw []yamlNode
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return raw, nil
}

func toNode(yn yamlNode) (Node, error) {
	room := strings.TrimSpace(yn.Room)
	item := strings.TrimSpace(yn.Item)
	sk := strings.TrimSpace(yn.Skill)
	if room == "" || item == "" || sk == "" {
		return Node{}, fmt.Errorf("room, skill, and item are required")
	}
	stock := 4
	if yn.Stock != nil {
		if *yn.Stock < 0 {
			return Node{}, fmt.Errorf("negative stock")
		}
		stock = *yn.Stock
	}
	regen := 50
	if yn.RegenTicks != nil {
		if *yn.RegenTicks < 0 {
			return Node{}, fmt.Errorf("negative regen_ticks")
		}
		regen = *yn.RegenTicks
	}
	return Node{Room: room, Skill: sk, Item: item, Stock: stock, RegenTicks: regen}, nil
}

func loadRecipes(path string, items *world.Items, skills *skill.Catalog) ([]Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw []yamlRecipe
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := make([]Recipe, 0, len(raw))
	seen := map[string]struct{}{}
	for i, yr := range raw {
		r, err := toRecipe(yr)
		if err != nil {
			return nil, fmt.Errorf("%s: recipe %d: %w", path, i, err)
		}
		if _, dup := seen[r.ID]; dup {
			return nil, fmt.Errorf("%s: duplicate recipe id %q", path, r.ID)
		}
		seen[r.ID] = struct{}{}
		if skills != nil {
			if _, ok := skills.Skill(r.Skill); !ok {
				return nil, fmt.Errorf("%s: recipe %s: unknown skill %s", path, r.ID, r.Skill)
			}
		}
		if items != nil {
			if _, ok := items.Get(r.Out.ID); !ok {
				return nil, fmt.Errorf("%s: recipe %s: unknown out %s", path, r.ID, r.Out.ID)
			}
			for _, in := range r.In {
				if _, ok := items.Get(in.ID); !ok {
					return nil, fmt.Errorf("%s: recipe %s: unknown in %s", path, r.ID, in.ID)
				}
			}
			if r.Tool != "" {
				if _, ok := items.Get(r.Tool); !ok {
					return nil, fmt.Errorf("%s: recipe %s: unknown tool %s", path, r.ID, r.Tool)
				}
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func toRecipe(yr yamlRecipe) (Recipe, error) {
	id := strings.TrimSpace(yr.ID)
	sk := strings.TrimSpace(yr.Skill)
	if id == "" || sk == "" {
		return Recipe{}, fmt.Errorf("id and skill are required")
	}
	flag := strings.TrimSpace(yr.Flag)
	if _, ok := knownStation[flag]; !ok {
		return Recipe{}, fmt.Errorf("recipe %q: unknown flag %q", id, flag)
	}
	if strings.TrimSpace(yr.Out.ID) == "" {
		return Recipe{}, fmt.Errorf("recipe %q: out.id is required", id)
	}
	outN := 1
	if yr.Out.N != nil {
		if *yr.Out.N < 1 {
			return Recipe{}, fmt.Errorf("recipe %q: out.n must be ≥ 1", id)
		}
		outN = *yr.Out.N
	}
	if len(yr.In) == 0 {
		return Recipe{}, fmt.Errorf("recipe %q: in is required", id)
	}
	in := make([]Input, 0, len(yr.In))
	for _, p := range yr.In {
		pid := strings.TrimSpace(p.ID)
		if pid == "" {
			return Recipe{}, fmt.Errorf("recipe %q: input missing id", id)
		}
		n := 1
		if p.N != nil {
			if *p.N < 1 {
				return Recipe{}, fmt.Errorf("recipe %q: input %s n must be ≥ 1", id, pid)
			}
			n = *p.N
		}
		in = append(in, Input{ID: pid, Qty: n})
	}
	return Recipe{
		ID: id, Skill: sk, Flag: flag, Tool: strings.TrimSpace(yr.Tool),
		In: in, Out: Input{ID: strings.TrimSpace(yr.Out.ID), Qty: outN},
		Gain: cleanLines(yr.Gain), Miss: cleanLines(yr.Miss),
	}, nil
}

func cleanLines(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func newCatalog(nodes []Node, recipes []Recipe) (*Catalog, error) {
	c := emptyCatalog()
	c.nodes = append([]Node(nil), nodes...)
	for i, n := range c.nodes {
		c.byRoom[n.Room] = append(c.byRoom[n.Room], i)
	}
	c.recipes = make([]Recipe, 0, len(recipes))
	for _, r := range recipes {
		if _, dup := c.byOut[r.Out.ID]; dup {
			return nil, fmt.Errorf("duplicate recipe output %q", r.Out.ID)
		}
		c.byOut[r.Out.ID] = len(c.recipes)
		c.recipes = append(c.recipes, cloneRecipe(r))
	}
	return c, nil
}
