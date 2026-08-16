// Package net implements connection adapters. M0 is the Telnet listener
// (issue #4). WebSocket is issue #12 and is not started here.
package net

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdnet "net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/pjhwa/yeomyeong/internal/engine"
	"github.com/pjhwa/yeomyeong/internal/persist"
	"github.com/pjhwa/yeomyeong/internal/text"
)

// Korean auth prompts remain WIRE-PROTOCOL literals. Sys lines come from
// internal/text (D-016 → D-029) without wording changes.
const (
	banner       = "여명 · YEOMYEONG"
	promptName   = "계정 이름:"
	promptPass   = "암호:"
	promptCreate = "그 이름은 장부에 없습니다. 새로 만드시겠습니까? (y/n)"
	msgBadCreds  = "이름이나 암호가 맞지 않습니다."
	promptCmd    = ">"

	maxLine   = 4096
	cmdRate   = 20
	cmdWindow = time.Second
	readIdle  = 5 * time.Minute
	iac       = 255
	iacSe     = 240
	iacSb     = 250
	iacWill   = 251
	iacWont   = 252
	iacDo     = 253
	iacDont   = 254
	optEcho   = 1
	optSGA    = 3
	telnetBS  = 0x08
	telnetDEL = 0x7f
)

var connSeq atomic.Uint64

// Server is the Telnet listener. It never mutates the roster.
type Server struct {
	Addr  string
	Loop  *engine.Loop
	Store persist.AccountStore
	Log   *slog.Logger

	ln    stdnet.Listener
	ready chan struct{}
}

// NewServer constructs a Telnet server that binds addr (":0" is allowed).
func NewServer(addr string, loop *engine.Loop, store persist.AccountStore, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		Addr:  addr,
		Loop:  loop,
		Store: store,
		Log:   log,
		ready: make(chan struct{}),
	}
}

// BoundAddr is the address the listener actually bound.
func (s *Server) BoundAddr() string {
	if s.ln != nil {
		return s.ln.Addr().String()
	}
	return s.Addr
}

// Serve listens until ctx is cancelled. It returns nil on shutdown.
func (s *Server) Serve(ctx context.Context) error {
	ln, err := listenTCP(s.Addr, s.Log, "telnet")
	if err != nil {
		return err
	}
	s.ln = ln
	if s.ready != nil {
		close(s.ready)
	}
	s.Log.Info("telnet listening", "addr", ln.Addr().String())

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	var wg sync.WaitGroup
	for {
		c, err := ln.Accept()
		if err != nil {
			wg.Wait()
			if ctx.Err() != nil || errors.Is(err, stdnet.ErrClosed) {
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handle(ctx, c)
		}()
	}
}

func (s *Server) handle(ctx context.Context, raw stdnet.Conn) {
	defer func() { _ = raw.Close() }()
	id := engine.ConnID(fmt.Sprintf("telnet-%d", connSeq.Add(1)))
	if err := writeNegotiate(raw); err != nil {
		return
	}
	sess := &session{
		id:    id,
		raw:   raw,
		r:     bufio.NewReader(raw),
		loop:  s.Loop,
		store: s.Store,
	}

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = raw.SetDeadline(time.Now())
		case <-stop:
		}
	}()

	defer func() {
		s.Loop.Submit(engine.LeaveWorld{ConnID: id})
		// Bound by server ctx so shutdown does not wait on a dead loop.
		dctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		_ = s.Loop.Detach(dctx, id)
	}()

	acc, tok, err := sess.auth(ctx)
	if err != nil {
		return
	}
	sess.user = acc.Username

	sheet, err := s.Store.LoadSheet(ctx, acc.ID)
	if err != nil {
		return
	}
	out, err := s.Loop.Attach(ctx, id)
	if err != nil {
		return
	}
	if !s.Loop.Submit(engine.EnterWorld{
		ConnID:    id,
		AccountID: engine.AccountID(acc.ID),
		Username:  acc.Username,
		Session:   tok.Token,
		Sheet:     sheet,
	}) {
		_ = sess.writeLine(text.T(text.Default, text.SysRateLimit))
		return
	}
	if err := sess.awaitSeated(ctx, out, acc.Username); err != nil {
		return
	}
	if err := sess.writeLine(promptCmd); err != nil {
		return
	}
	go sess.drain(ctx, out)
	sess.commands(ctx)
}

type session struct {
	id    engine.ConnID
	raw   stdnet.Conn
	r     *bufio.Reader
	mu    sync.Mutex // socket write only; not world state
	loop  *engine.Loop
	store persist.AccountStore
	user  string
	lim   limiter
}

func (s *session) auth(ctx context.Context) (persist.Account, persist.Session, error) {
	var zeroAcc persist.Account
	var zeroTok persist.Session
	if err := s.writeLine(banner); err != nil {
		return zeroAcc, zeroTok, err
	}
	if err := s.writeLine(hangulHint(s.raw.LocalAddr())); err != nil {
		return zeroAcc, zeroTok, err
	}
	for {
		if err := ctx.Err(); err != nil {
			return zeroAcc, zeroTok, err
		}
		if err := s.writeLine(promptName); err != nil {
			return zeroAcc, zeroTok, err
		}
		name, err := s.readLine()
		if err != nil {
			return zeroAcc, zeroTok, err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := s.writeLine(promptPass); err != nil {
			return zeroAcc, zeroTok, err
		}
		pass, err := s.readSecret()
		if err != nil {
			return zeroAcc, zeroTok, err
		}

		exists, err := s.store.Exists(ctx, name)
		if err != nil {
			return zeroAcc, zeroTok, err
		}

		var acc persist.Account
		if !exists {
			if err := s.writeLine(promptCreate); err != nil {
				return zeroAcc, zeroTok, err
			}
			ans, err := s.readLine()
			if err != nil {
				return zeroAcc, zeroTok, err
			}
			ans = strings.TrimSpace(ans)
			if ans != "y" && ans != "Y" {
				continue
			}
			if err := s.writeLine(promptPass); err != nil {
				return zeroAcc, zeroTok, err
			}
			pass, err = s.readSecret()
			if err != nil {
				return zeroAcc, zeroTok, err
			}
			acc, err = s.store.Create(ctx, name, pass)
			if err != nil {
				if werr := s.writeLine(msgBadCreds); werr != nil {
					return zeroAcc, zeroTok, werr
				}
				continue
			}
		} else {
			acc, err = s.store.Authenticate(ctx, name, pass)
			if err != nil {
				if werr := s.writeLine(msgBadCreds); werr != nil {
					return zeroAcc, zeroTok, werr
				}
				continue
			}
		}

		tok, err := s.store.IssueSession(ctx, acc.ID, 0)
		if err != nil {
			return zeroAcc, zeroTok, err
		}
		return acc, tok, nil
	}
}

func (s *session) awaitSeated(ctx context.Context, out <-chan engine.Event, username string) error {
	want := text.T(text.Default, text.SysSeated, username)
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-out:
			if !ok {
				return io.EOF
			}
			if err := s.writeEvent(ev); err != nil {
				return err
			}
			if tx, ok := ev.(engine.Text); ok && tx.Channel == engine.ChannelSys && tx.Body == want {
				s.flushNow(out)
				return nil
			}
		case <-timer.C:
			return errors.New("enter world timeout")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *session) flushNow(out <-chan engine.Event) {
	for {
		select {
		case ev, ok := <-out:
			if !ok {
				return
			}
			_ = s.writeEvent(ev)
		default:
			return
		}
	}
}

func (s *session) drain(ctx context.Context, out <-chan engine.Event) {
	for {
		select {
		case ev, ok := <-out:
			if !ok {
				return
			}
			if err := s.writeEvent(ev); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *session) commands(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		line, err := s.readLine()
		if err != nil {
			return
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !s.lim.allow(time.Now()) {
			if err := s.writeLine(text.T(text.Default, text.SysRateLimit)); err != nil {
				return
			}
			continue
		}
		if err := s.dispatch(line); err != nil {
			return
		}
	}
}

func (s *session) dispatch(line string) error {
	verb, rest := splitCmd(line)
	low := strings.ToLower(verb)
	switch {
	case low == "say" || verb == "말":
		if rest == "" {
			return s.writeLine(text.T(text.Default, text.CmdUnknown))
		}
		if !s.loop.Submit(engine.Say{ConnID: s.id, Text: rest}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case isLook(low, verb):
		if !s.loop.Submit(engine.Look{ConnID: s.id}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "go" || verb == "가다":
		dir, ok := parseTelnetDir(rest)
		if !ok {
			return s.writeLine(text.T(text.Default, text.CmdUnknown))
		}
		if !s.loop.Submit(engine.Move{ConnID: s.id, Dir: dir}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "skills" || verb == "숙련" || low == "inv" || verb == "소지":
		if !s.loop.Submit(engine.Sheet{ConnID: s.id}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "practice" || verb == "익히다":
		if rest == "" {
			return s.writeLine(text.T(text.Default, text.CmdUnknown))
		}
		if !s.loop.Submit(engine.Practice{ConnID: s.id, SkillID: rest}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "get" || verb == "집다":
		if rest == "" {
			return s.writeLine(text.T(text.Default, text.CmdUnknown))
		}
		if !s.loop.Submit(engine.Get{ConnID: s.id, ItemID: rest}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "drop" || verb == "놓다":
		if rest == "" {
			return s.writeLine(text.T(text.Default, text.CmdUnknown))
		}
		if !s.loop.Submit(engine.DropItem{ConnID: s.id, ItemID: rest}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "equip" || verb == "들다":
		if rest == "" {
			return s.writeLine(text.T(text.Default, text.CmdUnknown))
		}
		if !s.loop.Submit(engine.Equip{ConnID: s.id, ItemID: rest}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "unequip" || verb == "벗다":
		if rest == "" {
			return s.writeLine(text.T(text.Default, text.CmdUnknown))
		}
		if !s.loop.Submit(engine.Unequip{ConnID: s.id, Slot: rest}) {
			return s.writeLine(text.T(text.Default, text.SysRateLimit))
		}
	case low == "quit" || verb == "종료":
		if s.user != "" {
			_ = s.writeLine(text.T(text.Default, text.SysLeave, s.user))
		}
		s.loop.Submit(engine.LeaveWorld{ConnID: s.id})
		return io.EOF
	default:
		if dir, ok := parseTelnetDir(verb); ok && rest == "" {
			if !s.loop.Submit(engine.Move{ConnID: s.id, Dir: dir}) {
				return s.writeLine(text.T(text.Default, text.SysRateLimit))
			}
			return nil
		}
		if s.loop.KnowsSkill(verb) {
			if !s.loop.Submit(engine.Practice{ConnID: s.id, SkillID: verb}) {
				return s.writeLine(text.T(text.Default, text.SysRateLimit))
			}
			return nil
		}
		return s.writeLine(text.T(text.Default, text.CmdUnknown))
	}
	return nil
}

func (s *session) writeEvent(ev engine.Event) error {
	switch e := ev.(type) {
	case engine.Text:
		return s.writeLine(formatText(e))
	case engine.Room:
		for _, line := range formatRoom(e) {
			if err := s.writeLine(line); err != nil {
				return err
			}
		}
		return nil
	case engine.Drop:
		return io.EOF
	default:
		return nil
	}
}

func (s *session) writeLine(line string) error {
	return s.writeRaw(line + "\r\n")
}

func (s *session) writeRaw(p string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.raw.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := io.WriteString(s.raw, p)
	return err
}

func (s *session) readLine() (string, error) {
	_ = s.raw.SetReadDeadline(time.Now().Add(readIdle))
	return readLineInto(s.r, s.writeRaw, true)
}

func (s *session) readSecret() (string, error) {
	_ = s.raw.SetReadDeadline(time.Now().Add(readIdle))
	return readLineInto(s.r, s.writeRaw, false)
}

func writeNegotiate(c stdnet.Conn) error {
	_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := c.Write([]byte{iac, iacWill, optEcho, iac, iacWill, optSGA})
	return err
}

func hangulHint(addr stdnet.Addr) string {
	port := "4001"
	if addr != nil {
		if _, p, err := stdnet.SplitHostPort(addr.String()); err == nil && p != "" {
			port = p
		}
	}
	return "한글이 깨지면: nc localhost " + port
}

func formatText(e engine.Text) string {
	if e.Channel == engine.ChannelSay {
		return "[말] " + e.From + ": " + e.Body
	}
	return e.Body
}

var exitOrder = []string{"north", "south", "east", "west", "up", "down"}

func formatRoom(e engine.Room) []string {
	lines := []string{e.Name, e.Description}
	if s := formatExits(e.Exits); s != "" {
		lines = append(lines, s)
	}
	if len(e.Who) > 0 {
		lines = append(lines, text.T(text.Default, text.RoomHere, strings.Join(e.Who, ", ")))
	}
	if len(e.Ground) > 0 {
		lines = append(lines, text.T(text.Default, text.RoomGround, strings.Join(e.Ground, ", ")))
	}
	return lines
}

func formatExits(exits map[string]string) string {
	if len(exits) == 0 {
		return ""
	}
	parts := make([]string, 0, len(exits))
	for _, dir := range exitOrder {
		name, ok := exits[dir]
		if !ok {
			continue
		}
		parts = append(parts, text.DirLabel(text.Default, dir)+"("+name+")")
	}
	if len(parts) == 0 {
		return ""
	}
	return text.T(text.Default, text.RoomExits, strings.Join(parts, ", "))
}

func isLook(low, verb string) bool {
	return low == "look" || low == "l" || verb == "보다" || verb == "살펴"
}

var telnetDir = map[string]string{
	"n": "north", "north": "north", "북": "north",
	"s": "south", "south": "south", "남": "south",
	"e": "east", "east": "east", "동": "east",
	"w": "west", "west": "west", "서": "west",
	"u": "up", "up": "up", "위": "up",
	"d": "down", "down": "down", "아래": "down",
}

func parseTelnetDir(s string) (string, bool) {
	d, ok := telnetDir[strings.ToLower(strings.TrimSpace(s))]
	return d, ok
}

func splitCmd(line string) (verb, rest string) {
	line = strings.TrimSpace(line)
	for i, r := range line {
		if unicode.IsSpace(r) {
			return line[:i], strings.TrimSpace(line[i+utf8.RuneLen(r):])
		}
	}
	return line, ""
}

// readLine accepts CRLF, LF, CR, or CR NUL. Inbound IAC is skipped.
// Lone IAC bytes (0xFF) are dropped as WIRE-PROTOCOL requires.
func readLine(r *bufio.Reader) (string, error) {
	return readLineInto(r, nil, false)
}

func readLineInto(r *bufio.Reader, out func(string) error, echo bool) (string, error) {
	write := func(s string) error {
		if out == nil {
			return nil
		}
		return out(s)
	}
	var buf []byte
	over := false
	for {
		b, err := readFilteredByte(r)
		if err != nil {
			return "", err
		}
		if b == '\n' || b == '\r' {
			if b == '\r' {
				if r.Buffered() > 0 {
					if p, err := r.Peek(1); err == nil && (p[0] == '\n' || p[0] == 0) {
						_, _ = r.ReadByte()
					}
				}
			} else if n := len(buf); n > 0 && buf[n-1] == '\r' {
				buf = buf[:n-1]
			}
			if over {
				return "", nil
			}
			if err := write("\r\n"); err != nil {
				return "", err
			}
			return string(buf), nil
		}
		if b == telnetBS || b == telnetDEL {
			if over || len(buf) == 0 {
				continue
			}
			_, size := utf8.DecodeLastRune(buf)
			if size < 1 {
				size = 1
			}
			buf = buf[:len(buf)-size]
			if echo {
				if err := write("\b \b"); err != nil {
					return "", err
				}
			}
			continue
		}
		if b < 0x20 {
			continue
		}
		if over {
			continue
		}
		if len(buf) >= maxLine {
			over = true
			continue
		}
		buf = append(buf, b)
		if echo {
			if err := write(string([]byte{b})); err != nil {
				return "", err
			}
		}
	}
}

func readFilteredByte(r *bufio.Reader) (byte, error) {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b != iac {
			return b, nil
		}
		cmd, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		switch cmd {
		case iac:
			// IAC IAC is a literal 0xFF; drop it (WIRE-PROTOCOL).
			continue
		case iacWill, iacWont, iacDo, iacDont:
			if _, err := r.ReadByte(); err != nil {
				return 0, err
			}
		case iacSb:
			for {
				x, err := r.ReadByte()
				if err != nil {
					return 0, err
				}
				if x != iac {
					continue
				}
				y, err := r.ReadByte()
				if err != nil {
					return 0, err
				}
				if y == iacSe {
					break
				}
			}
		default:
			// IAC + one-byte command; already consumed.
		}
	}
}

// limiter is a per-connection rolling 1s window (WIRE-PROTOCOL).
type limiter struct {
	times []time.Time
}

func (l *limiter) allow(now time.Time) bool {
	cut := now.Add(-cmdWindow)
	i := 0
	for i < len(l.times) && !l.times[i].After(cut) {
		i++
	}
	if i > 0 {
		l.times = append(l.times[:0], l.times[i:]...)
	}
	if len(l.times) >= cmdRate {
		return false
	}
	l.times = append(l.times, now)
	return true
}
