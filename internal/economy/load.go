package economy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pjhwa/yeomyeong/internal/world"
	"gopkg.in/yaml.v3"
)

type yamlMarket struct {
	ID    string     `yaml:"id"`
	Name  loc        `yaml:"name"`
	Goods []yamlGood `yaml:"goods"`
}

type yamlGood struct {
	ID     string   `yaml:"id"`
	Base   int      `yaml:"base"`
	Stock  *int     `yaml:"stock"`
	Target *int     `yaml:"target"`
	Demand *float64 `yaml:"demand"`
}

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

// Load reads every *.yaml in dir (name order). Missing dir yields an empty book.
// items, when non-nil, must contain every good id.
func Load(dir string, items *world.Items) (*Book, error) {
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return NewBook(nil), nil
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
	sort.Strings(matches)
	var all []Market
	seen := map[string]string{}
	for _, path := range matches {
		ms, err := parseFile(path)
		if err != nil {
			return nil, err
		}
		for _, m := range ms {
			if prev, dup := seen[m.ID]; dup {
				return nil, fmt.Errorf("duplicate market id %q (%s and %s)", m.ID, prev, path)
			}
			seen[m.ID] = path
			if items != nil {
				for id := range m.Goods {
					if _, ok := items.Get(id); !ok {
						return nil, fmt.Errorf("market %q: unknown item %q", m.ID, id)
					}
				}
			}
			all = append(all, m)
		}
	}
	return NewBook(all), nil
}

func parseFile(path string) ([]Market, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw []yamlMarket
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := make([]Market, 0, len(raw))
	for i, ym := range raw {
		m, err := toMarket(ym)
		if err != nil {
			return nil, fmt.Errorf("%s: market %d: %w", path, i, err)
		}
		out = append(out, m)
	}
	return out, nil
}

func toMarket(ym yamlMarket) (Market, error) {
	id := strings.TrimSpace(ym.ID)
	if id == "" || strings.ContainsAny(id, " :") {
		return Market{}, fmt.Errorf("id %q must be a slug", ym.ID)
	}
	if ym.Name.KO == "" {
		return Market{}, fmt.Errorf("market %q: name.ko is required", id)
	}
	goods := make(map[string]Good, len(ym.Goods))
	for _, yg := range ym.Goods {
		gid := strings.TrimSpace(yg.ID)
		if gid == "" {
			return Market{}, fmt.Errorf("market %q: good missing id", id)
		}
		if _, dup := goods[gid]; dup {
			return Market{}, fmt.Errorf("market %q: duplicate good %q", id, gid)
		}
		if yg.Base < 1 {
			return Market{}, fmt.Errorf("market %q: good %q: base must be ≥ 1", id, gid)
		}
		stock := 0
		if yg.Stock != nil {
			if *yg.Stock < 0 {
				return Market{}, fmt.Errorf("market %q: good %q: negative stock", id, gid)
			}
			stock = *yg.Stock
		}
		target := stock
		if yg.Target != nil {
			if *yg.Target < 0 {
				return Market{}, fmt.Errorf("market %q: good %q: negative target", id, gid)
			}
			target = *yg.Target
		}
		demand := 1.0
		if yg.Demand != nil {
			if *yg.Demand <= 0 {
				return Market{}, fmt.Errorf("market %q: good %q: demand must be > 0", id, gid)
			}
			demand = *yg.Demand
		}
		goods[gid] = Good{ID: gid, Base: yg.Base, Stock: stock, Target: target, Demand: demand}
	}
	if len(goods) == 0 {
		return Market{}, fmt.Errorf("market %q: no goods", id)
	}
	return Market{ID: id, Name: ym.Name.KO, Goods: goods}, nil
}
