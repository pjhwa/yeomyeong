// Package engine implements the single-writer game loop (PLAN.md §4.2).
// The loop is the only writer of Player.RoomID. The YAML catalog is
// immutable and read-only (EVENT-BUS.md). Auth I/O stays off this
// goroutine (D-012).
package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

const (
	// Tick is the game-loop period.
	Tick = 100 * time.Millisecond
	// CommandQueueSize is the bounded inbound command buffer.
	CommandQueueSize = 256
	// OutboundSize is the per-connection event buffer. Overflow drops oldest.
	OutboundSize = 64
)

// Loop is the sole writer of world state. Connection goroutines only Submit
// commands and drain the outbound channel returned by Attach.
type Loop struct {
	cmds chan Command
	log  *slog.Logger

	// catalog is immutable after construction. nil means no rooms.
	catalog *yworld.Catalog

	// Owned exclusively by the Run goroutine.
	world    world
	outbound map[ConnID]chan Event
}

// New constructs an idle loop with no room catalog. Call Run from a single goroutine.
func New(log *slog.Logger) *Loop {
	return NewWithCatalog(log, nil)
}

// NewWithCatalog constructs a loop that seats players at cat.Spawn() and
// serves Look/Move from the immutable graph. cat may be nil.
func NewWithCatalog(log *slog.Logger, cat *yworld.Catalog) *Loop {
	if log == nil {
		log = slog.Default()
	}
	return &Loop{
		cmds:     make(chan Command, CommandQueueSize),
		log:      log,
		catalog:  cat,
		world:    world{roster: make(map[ConnID]Player)},
		outbound: make(map[ConnID]chan Event),
	}
}

// Submit enqueues cmd. It is safe for concurrent use and never blocks:
// a full queue returns false and the adapter treats that as rate_limited.
func (l *Loop) Submit(cmd Command) bool {
	if cmd == nil {
		return false
	}
	select {
	case l.cmds <- cmd:
		return true
	default:
		return false
	}
}

// Run is the game-loop goroutine. It returns after ctx is cancelled and
// any commands already in the queue have been drained.
func (l *Loop) Run(ctx context.Context) {
	ticker := time.NewTicker(Tick)
	defer ticker.Stop()
	l.log.Info("game loop started", "tick", Tick)
	for {
		select {
		case <-ctx.Done():
			n := l.drain()
			l.log.Info("game loop stopped", "drained", n)
			return
		case cmd := <-l.cmds:
			l.handle(cmd)
		case <-ticker.C:
			start := time.Now()
			n := l.drain()
			if d := time.Since(start); d > Tick {
				l.log.Warn("tick overran", "dur", d, "commands", n)
			}
		}
	}
}

// Attach allocates (or returns) the outbound event buffer for id.
// The request is served by the loop so the outbound map stays single-writer.
func (l *Loop) Attach(ctx context.Context, id ConnID) (<-chan Event, error) {
	resp := make(chan chan Event, 1)
	if err := l.enqueue(ctx, attachReq{id: id, resp: resp}); err != nil {
		return nil, err
	}
	select {
	case ch := <-resp:
		return ch, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Detach closes and forgets the outbound buffer for id.
func (l *Loop) Detach(ctx context.Context, id ConnID) error {
	done := make(chan struct{})
	if err := l.enqueue(ctx, detachReq{id: id, done: done}); err != nil {
		return err
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Snapshot asks the loop to copy the roster. It never reads the map
// from the caller's goroutine.
func (l *Loop) Snapshot(ctx context.Context) (Snapshot, error) {
	resp := make(chan Snapshot, 1)
	if err := l.enqueue(ctx, snapReq{resp: resp}); err != nil {
		return Snapshot{}, err
	}
	select {
	case snap := <-resp:
		return snap, nil
	case <-ctx.Done():
		return Snapshot{}, ctx.Err()
	}
}

func (l *Loop) enqueue(ctx context.Context, cmd Command) error {
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
	for {
		if l.Submit(cmd) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(time.Millisecond)
		}
	}
}

func (l *Loop) drain() int {
	n := 0
	for {
		select {
		case cmd := <-l.cmds:
			l.handle(cmd)
			n++
		default:
			return n
		}
	}
}

func (l *Loop) handle(cmd Command) {
	switch c := cmd.(type) {
	case EnterWorld:
		l.enter(c)
	case Say:
		l.say(c)
	case Look:
		l.look(c)
	case Move:
		l.move(c)
	case LeaveWorld:
		l.leave(c)
	case attachReq:
		c.resp <- l.ensureOut(c.id)
	case detachReq:
		if ch, ok := l.outbound[c.id]; ok {
			delete(l.outbound, c.id)
			close(ch)
		}
		close(c.done)
	case snapReq:
		players := l.world.copyPlayers()
		sort.Slice(players, func(i, j int) bool {
			return players[i].ConnID < players[j].ConnID
		})
		c.resp <- Snapshot{Players: players}
	default:
		l.log.Warn("unknown command", "type", fmt.Sprintf("%T", cmd))
	}
}

func (l *Loop) enter(c EnterWorld) {
	if c.ConnID == "" {
		return
	}
	if _, exists := l.world.roster[c.ConnID]; exists {
		return
	}
	l.ensureOut(c.ConnID)
	roomID := ""
	if l.catalog != nil {
		roomID = l.catalog.Spawn()
	}
	p := Player{
		ConnID:    c.ConnID,
		AccountID: c.AccountID,
		Username:  c.Username,
		Session:   c.Session,
		RoomID:    roomID,
	}
	l.world.roster[c.ConnID] = p
	l.sysInRoom(roomID, text.T(text.Default, text.SysSeated, c.Username))
	if l.catalog != nil {
		l.emit(l.roomCard(p))
	}
}

func (l *Loop) say(c Say) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	for id, other := range l.world.roster {
		if other.RoomID != p.RoomID {
			continue
		}
		l.emit(Text{ConnID: id, Channel: ChannelSay, From: p.Username, Body: c.Text})
	}
}

func (l *Loop) look(c Look) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	l.emit(l.roomCard(p))
}

func (l *Loop) move(c Move) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	dest, ok := "", false
	if l.catalog != nil {
		dest, ok = l.catalog.Exit(p.RoomID, c.Dir)
	}
	if !ok {
		l.emit(Text{
			ConnID:  p.ConnID,
			Channel: ChannelSys,
			Body:    text.T(text.Default, text.MoveNoExit),
			Code:    text.CodeNoExit,
		})
		return
	}
	p.RoomID = dest
	l.world.roster[c.ConnID] = p
	l.emit(l.roomCard(p))
}

func (l *Loop) leave(c LeaveWorld) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	delete(l.world.roster, c.ConnID)
	l.sysInRoom(p.RoomID, text.T(text.Default, text.SysLeave, p.Username))
}

func (l *Loop) sysInRoom(roomID, body string) {
	for id, other := range l.world.roster {
		if other.RoomID == roomID {
			l.emit(Text{ConnID: id, Channel: ChannelSys, Body: body})
		}
	}
}

func (l *Loop) roomCard(viewer Player) Room {
	who := l.whoElse(viewer)
	card := Room{
		ConnID: viewer.ConnID,
		ID:     viewer.RoomID,
		Exits:  map[string]string{},
		Who:    who,
	}
	if l.catalog == nil {
		return card
	}
	r, ok := l.catalog.Room(viewer.RoomID)
	if !ok {
		return card
	}
	card.Name = strings.TrimSpace(r.Name.Text(text.Default))
	card.Description = strings.TrimSpace(r.Description.Text(text.Default))
	for dir, destID := range r.Exits {
		name := destID
		if dest, ok := l.catalog.Room(destID); ok {
			name = strings.TrimSpace(dest.Name.Text(text.Default))
		}
		card.Exits[dir] = name
	}
	return card
}

func (l *Loop) whoElse(viewer Player) []string {
	who := make([]string, 0)
	for _, other := range l.world.roster {
		if other.ConnID == viewer.ConnID || other.RoomID != viewer.RoomID {
			continue
		}
		who = append(who, other.Username)
	}
	sort.Strings(who)
	return who
}

func (l *Loop) ensureOut(id ConnID) chan Event {
	if ch, ok := l.outbound[id]; ok {
		return ch
	}
	ch := make(chan Event, OutboundSize)
	l.outbound[id] = ch
	return ch
}

func (l *Loop) emit(ev Event) {
	ch, ok := l.outbound[ev.Target()]
	if !ok {
		return
	}
	select {
	case ch <- ev:
		return
	default:
	}
	// Drop oldest so a slow client cannot stall the loop.
	var dropped Event
	select {
	case dropped = <-ch:
		l.log.Warn("outbound overflow, dropped oldest",
			"conn", ev.Target(),
			"dropped", fmt.Sprintf("%T", dropped),
		)
	default:
	}
	select {
	case ch <- ev:
	default:
	}
}
