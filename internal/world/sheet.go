package world

// Equipment is the two M2 wear slots. Empty string means vacant.
type Equipment struct {
	MainHand string `json:"main_hand,omitempty"`
	Body     string `json:"body,omitempty"`
}

// Sheet is the durable account body (D-034). Skills/stats are opaque maps.
type Sheet struct {
	Skills map[string]int `json:"skills"`
	Stats  map[string]int `json:"stats"`
	Bag    []Stack        `json:"bag"`
	Equip  Equipment      `json:"equipment"`
	Nyang  int            `json:"nyang,omitempty"`
}

// CloneSheet deep-copies a sheet. Nil maps become empty maps; nil bag becomes empty.
func CloneSheet(s Sheet) Sheet {
	nyang := s.Nyang
	if nyang < 0 {
		nyang = 0
	}
	return Sheet{
		Skills: CloneInts(s.Skills),
		Stats:  CloneInts(s.Stats),
		Bag:    CloneStacks(s.Bag),
		Equip:  s.Equip,
		Nyang:  nyang,
	}
}

// CloneInts copies a rank/stat map. Nil becomes an empty map.
func CloneInts(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
