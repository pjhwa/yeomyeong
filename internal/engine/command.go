package engine

// ConnID identifies a live connection. Assigned by the net adapter.
type ConnID string

// AccountID is the persist-layer account key. Opaque to the loop.
type AccountID string

// Command is a value enqueued by adapters. The loop is the only consumer.
// Implementations live in this package (unexported method).
type Command interface {
	command()
}

// EnterWorld inserts a player on the roster after persist auth succeeds.
type EnterWorld struct {
	ConnID    ConnID
	AccountID AccountID
	Username  string
	Session   string
}

// Say delivers a line to roster connections in the same room (D-030).
type Say struct {
	ConnID ConnID
	Text   string
}

// Look asks the loop for the caller's current room card.
type Look struct {
	ConnID ConnID
}

// Move asks the loop to walk Dir (north/south/east/west/up/down).
type Move struct {
	ConnID ConnID
	Dir    string
}

// LeaveWorld removes a player on quit or disconnect.
type LeaveWorld struct {
	ConnID ConnID
}

func (EnterWorld) command() {}
func (Say) command()        {}
func (Look) command()       {}
func (Move) command()       {}
func (LeaveWorld) command() {}

// Control requests served by the same queue so they observe FIFO order
// relative to world-mutating commands. Not part of the public bus.

type snapReq struct {
	resp chan Snapshot
}

type attachReq struct {
	id   ConnID
	resp chan chan Event
}

type detachReq struct {
	id   ConnID
	done chan struct{}
}

func (snapReq) command()   {}
func (attachReq) command() {}
func (detachReq) command() {}
