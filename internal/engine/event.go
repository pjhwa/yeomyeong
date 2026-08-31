package engine

// Event is delivered to one connection's outbound buffer. The loop never
// writes to a socket.
type Event interface {
	Target() ConnID
	event()
}

const (
	ChannelSay  = "say"
	ChannelSys  = "sys"
	ChannelRoom = "room"
)

// Text is a player-visible line (EVENT-BUS.md).
type Text struct {
	ConnID  ConnID
	Channel string // say | sys | room
	From    string
	Body    string
	Code    string // optional WIRE-PROTOCOL sys code (no_exit)
}

// Target returns the connection that should receive this line.
func (e Text) Target() ConnID { return e.ConnID }

func (Text) event() {}

// Room is the full room card after enter, look, or a successful move.
type Room struct {
	ConnID      ConnID
	ID          string
	Name        string
	Description string
	Exits       map[string]string // dir → destination display name
	Who         []string          // other usernames in the room
	NPCs        []string          // scripted NPC display names in the room
	Ground      []string          // catalog display names, one entry per unit
}

// Target returns the connection that should receive this card.
func (e Room) Target() ConnID { return e.ConnID }

func (Room) event() {}

// Drop tells the adapter to close the socket after flushing the outbound buffer.
type Drop struct {
	ConnID ConnID
}

// Target returns the connection that should be closed.
func (e Drop) Target() ConnID { return e.ConnID }

func (Drop) event() {}
