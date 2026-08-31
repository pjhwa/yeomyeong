package content

import (
	"fmt"
	"os"
	"strings"

	"github.com/pjhwa/yeomyeong/internal/world"
	"gopkg.in/yaml.v3"
)

type yamlObject struct {
	ID          string   `yaml:"id"`
	Room        string   `yaml:"room"`
	Name        loc      `yaml:"name"`
	Aliases     []string `yaml:"aliases"`
	Description loc      `yaml:"description"`
	Foreshadow  []string `yaml:"foreshadow"`
}

type yamlNPC struct {
	ID         string   `yaml:"id"`
	Room       string   `yaml:"room"`
	Name       loc      `yaml:"name"`
	Aliases    []string `yaml:"aliases"`
	Look       loc      `yaml:"look"`
	Talk       yamlTalk `yaml:"talk"`
	Foreshadow []string `yaml:"foreshadow"`
}

type yamlTalk struct {
	First  loc `yaml:"first"`
	Second loc `yaml:"second"`
}

func loadObjects(root string, rooms map[string]world.Room) ([]world.Object, error) {
	var all []world.Object
	seen := map[string]string{}
	err := eachZoneFile(root, "objects.yaml", func(path, zone string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var raw []yamlObject
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for i, yo := range raw {
			o, err := toObject(yo, zone, rooms)
			if err != nil {
				return fmt.Errorf("%s: object %d: %w", path, i, err)
			}
			if prev, dup := seen[o.ID]; dup {
				return fmt.Errorf("duplicate object id %q (%s and %s)", o.ID, prev, path)
			}
			seen[o.ID] = path
			all = append(all, o)
		}
		return nil
	})
	return all, err
}

func loadNPCs(root string, rooms map[string]world.Room) ([]world.NPC, error) {
	var all []world.NPC
	seen := map[string]string{}
	err := eachZoneFile(root, "npcs.yaml", func(path, zone string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var raw []yamlNPC
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for i, yn := range raw {
			n, err := toNPC(yn, zone, rooms)
			if err != nil {
				return fmt.Errorf("%s: npc %d: %w", path, i, err)
			}
			if prev, dup := seen[n.ID]; dup {
				return fmt.Errorf("duplicate npc id %q (%s and %s)", n.ID, prev, path)
			}
			seen[n.ID] = path
			all = append(all, n)
		}
		return nil
	})
	return all, err
}

func toObject(yo yamlObject, zone string, rooms map[string]world.Room) (world.Object, error) {
	if yo.ID == "" || !itemIDRe.MatchString(yo.ID) {
		return world.Object{}, fmt.Errorf("id %q must be [a-z][a-z0-9-]*", yo.ID)
	}
	if yo.Room == "" {
		return world.Object{}, fmt.Errorf("object %q: missing room", yo.ID)
	}
	if _, ok := rooms[yo.Room]; !ok {
		return world.Object{}, fmt.Errorf("object %q: unknown room %q", yo.ID, yo.Room)
	}
	if zoneOf(yo.Room) != zone {
		return world.Object{}, fmt.Errorf("object %q: room %q does not match directory %q", yo.ID, yo.Room, zone)
	}
	if yo.Name.KO == "" {
		return world.Object{}, fmt.Errorf("object %q: name.ko is required", yo.ID)
	}
	if yo.Description.KO == "" {
		return world.Object{}, fmt.Errorf("object %q: description.ko is required", yo.ID)
	}
	aliases, err := cleanAliases(yo.Aliases)
	if err != nil {
		return world.Object{}, fmt.Errorf("object %q: %w", yo.ID, err)
	}
	return world.Object{
		ID:          yo.ID,
		Room:        yo.Room,
		Name:        world.Localized{KO: yo.Name.KO, EN: yo.Name.EN},
		Aliases:     aliases,
		Description: world.Localized{KO: yo.Description.KO, EN: yo.Description.EN},
		Foreshadow:  append([]string(nil), yo.Foreshadow...),
	}, nil
}

func toNPC(yn yamlNPC, zone string, rooms map[string]world.Room) (world.NPC, error) {
	if yn.ID == "" || !itemIDRe.MatchString(yn.ID) {
		return world.NPC{}, fmt.Errorf("id %q must be [a-z][a-z0-9-]*", yn.ID)
	}
	if yn.Room == "" {
		return world.NPC{}, fmt.Errorf("npc %q: missing room", yn.ID)
	}
	if _, ok := rooms[yn.Room]; !ok {
		return world.NPC{}, fmt.Errorf("npc %q: unknown room %q", yn.ID, yn.Room)
	}
	if zoneOf(yn.Room) != zone {
		return world.NPC{}, fmt.Errorf("npc %q: room %q does not match directory %q", yn.ID, yn.Room, zone)
	}
	if yn.Name.KO == "" {
		return world.NPC{}, fmt.Errorf("npc %q: name.ko is required", yn.ID)
	}
	if yn.Look.KO == "" {
		return world.NPC{}, fmt.Errorf("npc %q: look.ko is required", yn.ID)
	}
	if yn.Talk.First.KO == "" || yn.Talk.Second.KO == "" {
		return world.NPC{}, fmt.Errorf("npc %q: talk.first.ko and talk.second.ko are required", yn.ID)
	}
	aliases, err := cleanAliases(yn.Aliases)
	if err != nil {
		return world.NPC{}, fmt.Errorf("npc %q: %w", yn.ID, err)
	}
	if len(aliases) == 0 {
		return world.NPC{}, fmt.Errorf("npc %q: aliases required", yn.ID)
	}
	return world.NPC{
		ID:         yn.ID,
		Room:       yn.Room,
		Name:       world.Localized{KO: yn.Name.KO, EN: yn.Name.EN},
		Aliases:    aliases,
		Look:       world.Localized{KO: yn.Look.KO, EN: yn.Look.EN},
		TalkFirst:  world.Localized{KO: yn.Talk.First.KO, EN: yn.Talk.First.EN},
		TalkSecond: world.Localized{KO: yn.Talk.Second.KO, EN: yn.Talk.Second.EN},
		Foreshadow: append([]string(nil), yn.Foreshadow...),
	}, nil
}

func cleanAliases(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, a := range in {
		a = strings.TrimSpace(a)
		if a == "" {
			return nil, fmt.Errorf("empty alias")
		}
		if _, dup := seen[a]; dup {
			return nil, fmt.Errorf("duplicate alias %q", a)
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	return out, nil
}

func foreshadowOf(npcs []world.NPC, objects []world.Object) []string {
	var out []string
	for _, n := range npcs {
		out = append(out, n.Foreshadow...)
	}
	for _, o := range objects {
		out = append(out, o.Foreshadow...)
	}
	return out
}
