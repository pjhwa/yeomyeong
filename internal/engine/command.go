package engine

import yworld "github.com/pjhwa/yeomyeong/internal/world"

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
// Sheet is loaded by the adapter before enqueue (D-034).
type EnterWorld struct {
	ConnID    ConnID
	AccountID AccountID
	Username  string
	Session   string
	Sheet     yworld.Sheet
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

// Get moves one ground unit of ItemID into the bag if weight allows.
type Get struct {
	ConnID ConnID
	ItemID string
}

// DropItem is EVENT-BUS Drop. The disconnect event is already named Drop.
type DropItem struct {
	ConnID ConnID
	ItemID string
}

// Equip moves a wearable bag item into its catalog slot.
type Equip struct {
	ConnID ConnID
	ItemID string
}

// Unequip moves Slot (main_hand|body) back into the bag.
type Unequip struct {
	ConnID ConnID
	Slot   string
}

// Sheet asks the loop for skills/title/stats/inv text.
type Sheet struct {
	ConnID ConnID
}

// Practice is one skill trial (SKILL-TABLE.md). SkillID is a catalog id or Korean name.
type Practice struct {
	ConnID  ConnID
	SkillID string
}

// Gather harvests a YAML node in the current room.
type Gather struct {
	ConnID ConnID
	Query  string // item id/name or empty (first node)
	Skill  string // optional gather skill id/verb
}

// Craft consumes a YAML recipe. Query is recipe id or output name; empty lists options.
type Craft struct {
	ConnID ConnID
	Query  string
}

// Sell sells Qty of Query at the room's market.
type Sell struct {
	ConnID ConnID
	Query  string
	Qty    int
}

// Buy buys Qty of Query at the room's market.
type Buy struct {
	ConnID ConnID
	Query  string
	Qty    int
}

// Quote prints the current stall prices.
type Quote struct {
	ConnID ConnID
}

func (EnterWorld) command() {}
func (Say) command()        {}
func (Look) command()       {}
func (Move) command()       {}
func (LeaveWorld) command() {}
func (Get) command()        {}
func (DropItem) command()   {}
func (Equip) command()      {}
func (Unequip) command()    {}
func (Sheet) command()      {}
func (Practice) command()   {}
func (Gather) command()     {}
func (Craft) command()      {}
func (Sell) command()       {}
func (Buy) command()        {}
func (Quote) command()      {}

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
