package engine

// Event is delivered to one connection's outbound buffer. The loop never
// writes to a socket.
type Event interface {
	Target() ConnID
	event()
}

const (
	ChannelSay = "say"
	ChannelSys = "sys"
)

// Text is a player-visible line (EVENT-BUS.md).
type Text struct {
	ConnID  ConnID
	Channel string // say | sys
	From    string
	Body    string
}

// Target returns the connection that should receive this line.
func (e Text) Target() ConnID { return e.ConnID }

func (Text) event() {}

// Drop tells the adapter to close the socket after flushing the outbound buffer.
type Drop struct {
	ConnID ConnID
}

// Target returns the connection that should be closed.
func (e Drop) Target() ConnID { return e.ConnID }

func (Drop) event() {}
