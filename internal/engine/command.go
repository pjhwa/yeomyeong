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

// Say broadcasts a line to every roster connection.
type Say struct {
	ConnID ConnID
	Text   string
}

// LeaveWorld removes a player on quit or disconnect.
type LeaveWorld struct {
	ConnID ConnID
}

func (EnterWorld) command() {}
func (Say) command()        {}
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
