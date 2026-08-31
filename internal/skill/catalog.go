// Package skill loads the YAML practice catalog and applies the SKILL-TABLE
// growth curve. There are no levels, classes, or XP bars.
package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// RankCap is the maximum rank of one skill.
	RankCap = 100
	// RankSumCap is the maximum sum of all skill ranks (D-033).
	RankSumCap = 700
	// StatCap is the maximum of one stat.
	StatCap = 100
	// StatSumCap is the maximum sum of all stats.
	StatSumCap = 300

	groupCombat, groupCraft, groupSocial, groupGather = "combat", "craft", "social", "gather"
	statStr, statDex, statVit                         = "str", "dex", "vit"
	statWit, statSense, statFame                      = "wit", "sense", "fame"
)

// Localized is a canonical {ko, en} string. Missing en falls back to ko.
type Localized struct{ KO, EN string }

// Text returns EN when locale is "en" and EN is set; otherwise KO.
func (l Localized) Text(locale string) string {
	if locale == "en" && l.EN != "" {
		return l.EN
	}
	return l.KO
}

// Skill is one YAML practice definition after load.
type Skill struct {
	ID, Group, Stat, PracticeFlag, PracticeItem string
	Name                                        Localized
	Verbs                                       []string
	Gain, Miss                                  []string
}

// TitleRule is one YAML title rule. Empty Require is the default (must be last).
type TitleRule struct {
	ID       string
	Require  map[string]int
	Title    Localized
	Announce Localized
}

// Catalog is an immutable skill + title table. Safe for concurrent reads after Load.
type Catalog struct {
	skills map[string]Skill
	order  []string
	titles []TitleRule
}

var (
	skillIDRe   = regexp.MustCompile(`^(?:test:)?[a-z][a-z0-9-]*$`)
	titleIDRe   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	knownGroups = map[string]struct{}{groupCombat: {}, groupCraft: {}, groupSocial: {}, groupGather: {}}
	knownStats  = map[string]struct{}{
		statStr: {}, statDex: {}, statVit: {}, statWit: {}, statSense: {}, statFame: {},
	}
	knownFlags = map[string]struct{}{
		"safe": {}, "town": {}, "market": {}, "indoor": {}, "dark": {},
		"forge": {}, "kitchen": {}, "press": {}, "clinic": {}, "yard": {},
		"salon": {}, "checkpoint": {},
	}
	reservedVerbs = map[string]struct{}{
		"n": {}, "s": {}, "e": {}, "w": {}, "u": {}, "d": {},
		"north": {}, "south": {}, "east": {}, "west": {}, "up": {}, "down": {},
		"북": {}, "남": {}, "동": {}, "서": {}, "위": {}, "아래": {},
		"look": {}, "l": {}, "보다": {}, "살펴": {},
		"say": {}, "말": {}, "quit": {}, "종료": {},
		"skills": {}, "숙련": {}, "기술": {}, "inv": {}, "소지": {}, "가방": {},
		"practice": {}, "익히다": {},
		"get": {}, "집다": {}, "drop": {}, "놓다": {},
		"equip": {}, "들다": {}, "unequip": {}, "벗다": {},
		"go": {}, "가다": {},
		"craft": {}, "만들다": {}, "sell": {}, "팔다": {},
		"buy": {}, "사다": {}, "quote": {}, "시세": {},
		"gather": {},
	}
	statNameKO = map[string]string{
		statStr: "힘", statDex: "손재주", statVit: "맷집",
		statWit: "재치", statSense: "감응", statFame: "평판",
	}
)

// Load reads a YAML file or every *.yaml in a directory (name order).
func Load(path string) (*Catalog, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		s, t, err := parseFile(path)
		if err != nil {
			return nil, err
		}
		return newCatalog(s, t)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || (ext != ".yaml" && ext != ".yml") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var skills []Skill
	var titles []TitleRule
	for _, name := range names {
		s, t, err := parseFile(filepath.Join(path, name))
		if err != nil {
			return nil, err
		}
		skills = append(skills, s...)
		titles = append(titles, t...)
	}
	return newCatalog(skills, titles)
}

type loc struct{ KO, EN string }

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

type yamlFile struct {
	Skills []yamlSkill `yaml:"skills"`
	Titles []yamlTitle `yaml:"titles"`
}

type yamlSkill struct {
	ID           string   `yaml:"id"`
	Name         loc      `yaml:"name"`
	Group        string   `yaml:"group"`
	Stat         string   `yaml:"stat"`
	PracticeFlag string   `yaml:"practice_flag"`
	PracticeItem string   `yaml:"practice_item"`
	Verbs        []string `yaml:"verbs"`
	Gain         []string `yaml:"gain"`
	Miss         []string `yaml:"miss"`
}

type yamlTitle struct {
	ID       string         `yaml:"id"`
	Require  map[string]int `yaml:"require"`
	Title    loc            `yaml:"title"`
	Announce loc            `yaml:"announce"`
}

func parseFile(path string) ([]Skill, []TitleRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var raw yamlFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	skills := make([]Skill, 0, len(raw.Skills))
	for i, ys := range raw.Skills {
		s, err := toSkill(ys)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: skill %d: %w", path, i, err)
		}
		skills = append(skills, s)
	}
	titles := make([]TitleRule, 0, len(raw.Titles))
	for i, yt := range raw.Titles {
		tr, err := toTitle(yt)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: title %d: %w", path, i, err)
		}
		titles = append(titles, tr)
	}
	return skills, titles, nil
}

func toSkill(ys yamlSkill) (Skill, error) {
	id := strings.TrimSpace(ys.ID)
	if id == "" {
		return Skill{}, fmt.Errorf("missing id")
	}
	if !skillIDRe.MatchString(id) {
		return Skill{}, fmt.Errorf("id %q must be a slug or test:slug", id)
	}
	if ys.Name.KO == "" {
		return Skill{}, fmt.Errorf("skill %q: name.ko is required", id)
	}
	group, stat := strings.TrimSpace(ys.Group), strings.TrimSpace(ys.Stat)
	if _, ok := knownGroups[group]; !ok {
		return Skill{}, fmt.Errorf("skill %q: unknown group %q", id, group)
	}
	if _, ok := knownStats[stat]; !ok {
		return Skill{}, fmt.Errorf("skill %q: unknown stat %q", id, stat)
	}
	flag := strings.TrimSpace(ys.PracticeFlag)
	if flag != "" {
		if _, ok := knownFlags[flag]; !ok {
			return Skill{}, fmt.Errorf("skill %q: unknown practice_flag %q", id, flag)
		}
	}
	verbs := cleanLines(ys.Verbs)
	for _, v := range verbs {
		if _, bad := reservedVerbs[v]; bad {
			return Skill{}, fmt.Errorf("skill %q: verb %q is a reserved command", id, v)
		}
		if _, bad := reservedVerbs[strings.ToLower(v)]; bad {
			return Skill{}, fmt.Errorf("skill %q: verb %q is a reserved command", id, v)
		}
	}
	return Skill{ID: id, Name: Localized{KO: ys.Name.KO, EN: ys.Name.EN}, Group: group, Stat: stat,
		PracticeFlag: flag, PracticeItem: strings.TrimSpace(ys.PracticeItem),
		Verbs: verbs, Gain: cleanLines(ys.Gain), Miss: cleanLines(ys.Miss)}, nil
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

func toTitle(yt yamlTitle) (TitleRule, error) {
	id := strings.TrimSpace(yt.ID)
	if id == "" {
		return TitleRule{}, fmt.Errorf("missing id")
	}
	if !titleIDRe.MatchString(id) {
		return TitleRule{}, fmt.Errorf("id %q must be a slug", id)
	}
	if yt.Title.KO == "" {
		return TitleRule{}, fmt.Errorf("title %q: title.ko is required", id)
	}
	req := make(map[string]int, len(yt.Require))
	for k, v := range yt.Require {
		k = strings.TrimSpace(k)
		if k == "" {
			return TitleRule{}, fmt.Errorf("title %q: empty require key", id)
		}
		if v < 0 || v > RankCap {
			return TitleRule{}, fmt.Errorf("title %q: require %s=%d out of 0–%d", id, k, v, RankCap)
		}
		req[k] = v
	}
	return TitleRule{ID: id, Require: req,
		Title:    Localized{KO: yt.Title.KO, EN: yt.Title.EN},
		Announce: Localized{KO: yt.Announce.KO, EN: yt.Announce.EN}}, nil
}

func newCatalog(skills []Skill, titles []TitleRule) (*Catalog, error) {
	m := make(map[string]Skill, len(skills))
	order := make([]string, 0, len(skills))
	verbs := make(map[string]string)
	for _, s := range skills {
		if _, dup := m[s.ID]; dup {
			return nil, fmt.Errorf("duplicate skill id %q", s.ID)
		}
		m[s.ID] = s
		order = append(order, s.ID)
		for _, v := range s.Verbs {
			if other, dup := verbs[v]; dup {
				return nil, fmt.Errorf("duplicate verb %q (%s and %s)", v, other, s.ID)
			}
			verbs[v] = s.ID
		}
	}
	seen := make(map[string]struct{}, len(titles))
	out := make([]TitleRule, 0, len(titles))
	for i, tr := range titles {
		if _, dup := seen[tr.ID]; dup {
			return nil, fmt.Errorf("duplicate title id %q", tr.ID)
		}
		seen[tr.ID] = struct{}{}
		for sid := range tr.Require {
			if _, ok := m[sid]; !ok {
				return nil, fmt.Errorf("title %q: unknown skill %q", tr.ID, sid)
			}
		}
		if len(tr.Require) == 0 && i != len(titles)-1 {
			return nil, fmt.Errorf("title %q: empty require must be last", tr.ID)
		}
		out = append(out, cloneTitle(tr))
	}
	if n := len(titles); n > 0 && len(titles[n-1].Require) != 0 {
		return nil, fmt.Errorf("title %q: last rule must have empty require", titles[n-1].ID)
	}
	return &Catalog{skills: m, order: order, titles: out}, nil
}

// Len returns the number of skills.
func (c *Catalog) Len() int {
	if c == nil {
		return 0
	}
	return len(c.order)
}

// IDs returns skill ids in file order.
func (c *Catalog) IDs() []string {
	if c == nil {
		return nil
	}
	return append([]string(nil), c.order...)
}

// Skill returns a copy of the definition.
func (c *Catalog) Skill(id string) (Skill, bool) {
	if c == nil {
		return Skill{}, false
	}
	s, ok := c.skills[id]
	return s, ok
}

// Lookup finds a skill by id (case-insensitive) or Korean/English name.
func (c *Catalog) Lookup(q string) (Skill, bool) {
	if c == nil {
		return Skill{}, false
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return Skill{}, false
	}
	if s, ok := c.skills[q]; ok {
		return s, true
	}
	low := strings.ToLower(q)
	if s, ok := c.skills[low]; ok {
		return s, true
	}
	for _, id := range c.order {
		s := c.skills[id]
		if s.Name.KO == q || s.Name.EN == q || strings.EqualFold(s.ID, q) {
			return s, true
		}
		for _, v := range s.Verbs {
			if v == q {
				return s, true
			}
		}
	}
	return Skill{}, false
}

// LineAt picks a stable line for rank (or 0). Empty list returns "".
func LineAt(lines []string, rank int) string {
	if len(lines) == 0 {
		return ""
	}
	if rank < 0 {
		rank = 0
	}
	return lines[rank%len(lines)]
}

// Band is the player-facing rank word (SKILL-TABLE). Not a number.
func Band(rank int) string {
	switch {
	case rank >= 100:
		return "달인"
	case rank >= 90:
		return "명인"
	case rank >= 70:
		return "노련"
	case rank >= 45:
		return "능숙"
	case rank >= 20:
		return "익숙"
	default:
		return "초보"
	}
}

// StatName is the Korean label for a stat id, or the id itself.
func StatName(id string) string {
	if n, ok := statNameKO[id]; ok {
		return n
	}
	return id
}

// TitleRules returns a defensive copy of the title list (file order).
func (c *Catalog) TitleRules() []TitleRule {
	if c == nil {
		return nil
	}
	out := make([]TitleRule, len(c.titles))
	for i, tr := range c.titles {
		out[i] = cloneTitle(tr)
	}
	return out
}

// NewSheet returns an empty ranks+stats sheet bound to this catalog.
func (c *Catalog) NewSheet() Sheet { return Sheet{cat: c} }

// Title returns the first matching title in file order.
func (c *Catalog) Title(s Sheet) Localized {
	if c == nil {
		return Localized{}
	}
	for _, tr := range c.titles {
		if titleMatches(tr, s) {
			return tr.Title
		}
	}
	return Localized{}
}

// Announce is the first matching title's spoken line, or empty.
func (c *Catalog) Announce(s Sheet) Localized {
	if c == nil {
		return Localized{}
	}
	for _, tr := range c.titles {
		if titleMatches(tr, s) {
			return tr.Announce
		}
	}
	return Localized{}
}

func titleMatches(tr TitleRule, s Sheet) bool {
	for id, need := range tr.Require {
		if s.Rank(id) < need {
			return false
		}
	}
	return true
}

func cloneTitle(tr TitleRule) TitleRule {
	out := tr
	if tr.Require != nil {
		out.Require = make(map[string]int, len(tr.Require))
		for k, v := range tr.Require {
			out.Require[k] = v
		}
	}
	return out
}
