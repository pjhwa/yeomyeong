package engine

import yworld "github.com/pjhwa/yeomyeong/internal/world"

// Player is a logged-in roster entry. No mutex; only the loop goroutine
// may read or write this value while it lives in world.roster.
// RoomID is position (EVENT-BUS.md); the YAML catalog is immutable in package world.
type Player struct {
	ConnID    ConnID
	AccountID AccountID
	Username  string
	Session   string
	RoomID    string
	Skills    map[string]int
	Stats     map[string]int
	Bag       []yworld.Stack
	Equip     yworld.Equipment
	Nyang     int
}

// Snapshot is a copy of the roster produced by the loop (EVENT-BUS.md).
// Callers must not assume it stays current.
type Snapshot struct {
	Players []Player
	Ground  map[string][]yworld.Stack
}

// world is the in-memory simulation. Positions live here; room definitions
// do not. There is no mutex; the game-loop goroutine is the sole reader
// and writer.
type world struct {
	roster map[ConnID]Player
	ground map[string][]yworld.Stack
}

func (w *world) copyPlayers() []Player {
	out := make([]Player, 0, len(w.roster))
	for _, p := range w.roster {
		out = append(out, clonePlayer(p))
	}
	return out
}

func (w *world) copyGround() map[string][]yworld.Stack {
	return yworld.CloneGround(w.ground)
}

func clonePlayer(p Player) Player {
	out := p
	out.Skills = yworld.CloneInts(p.Skills)
	out.Stats = yworld.CloneInts(p.Stats)
	out.Bag = yworld.CloneStacks(p.Bag)
	return out
}

func (p Player) sheet() yworld.Sheet {
	return yworld.Sheet{Skills: p.Skills, Stats: p.Stats, Bag: p.Bag, Equip: p.Equip, Nyang: p.Nyang}
}
