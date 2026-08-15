package engine

// Player is a logged-in roster entry. No mutex; only the loop goroutine
// may read or write this value while it lives in world.roster.
type Player struct {
	ConnID    ConnID
	AccountID AccountID
	Username  string
	Session   string
}

// Snapshot is a copy of the roster produced by the loop (EVENT-BUS.md).
// Callers must not assume it stays current.
type Snapshot struct {
	Players []Player
}

// world is the in-memory simulation. M0 is roster-only (D-013): no rooms,
// NPCs, or items. There is no mutex; the game-loop goroutine is the sole
// reader and writer.
type world struct {
	roster map[ConnID]Player
}

func (w *world) copyPlayers() []Player {
	out := make([]Player, 0, len(w.roster))
	for _, p := range w.roster {
		out = append(out, p)
	}
	return out
}
