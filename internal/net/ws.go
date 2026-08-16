package net

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdnet "net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pjhwa/yeomyeong/internal/engine"
	"github.com/pjhwa/yeomyeong/internal/persist"
	"github.com/pjhwa/yeomyeong/internal/text"
)

// WIRE-PROTOCOL frame types and sys/auth.err codes.
const (
	protoV = 1

	typeAuthCreate  = "auth.create"
	typeAuthLogin   = "auth.login"
	typeCmdSay      = "cmd.say"
	typeCmdLook     = "cmd.look"
	typeCmdMove     = "cmd.move"
	typeCmdPractice = "cmd.practice"
	typeCmdSkills   = "cmd.skills"
	typeCmdInv      = "cmd.inv"
	typeCmdGet      = "cmd.get"
	typeCmdDrop     = "cmd.drop"
	typeCmdEquip    = "cmd.equip"
	typeCmdUnequip  = "cmd.unequip"
	typeCmdQuit     = "cmd.quit"
	typeAuthOK      = "auth.ok"
	typeAuthErr     = "auth.err"
	typeText        = "text"
	typeRoom        = "room"
	typeSys         = "sys"

	codeBadFrame    = "bad_frame"
	codeRateLimited = "rate_limited"
	codeNotAuth     = "not_authenticated"
	codeInternal    = "internal"
	codeBadDir      = "bad_dir"

	maxPayload = 4096
	maxFrame   = 8192
	wsPath     = "/ws"
)

var (
	errFrameTooLarge = errors.New("frame too large")
	errWSQuit        = errors.New("quit")
)

// Allow any Origin in M0: there is no browser client yet (CLIENT / M6).
var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// WS is the WebSocket HTTP listener. It never mutates the roster.
type WS struct {
	Addr  string
	Loop  *engine.Loop
	Store persist.AccountStore
	Log   *slog.Logger

	ln    stdnet.Listener
	ready chan struct{}
}

// NewWS constructs a WebSocket server that binds addr (":0" is allowed).
func NewWS(addr string, loop *engine.Loop, store persist.AccountStore, log *slog.Logger) *WS {
	if log == nil {
		log = slog.Default()
	}
	return &WS{
		Addr:  addr,
		Loop:  loop,
		Store: store,
		Log:   log,
		ready: make(chan struct{}),
	}
}

// BoundAddr is the address the listener actually bound.
func (w *WS) BoundAddr() string {
	if w.ln != nil {
		return w.ln.Addr().String()
	}
	return w.Addr
}

// Serve listens until ctx is cancelled. It returns nil on shutdown.
func (w *WS) Serve(ctx context.Context) error {
	ln, err := stdnet.Listen("tcp", w.Addr)
	if err != nil {
		return err
	}
	w.ln = ln
	if w.ready != nil {
		close(w.ready)
	}
	w.Log.Info("ws listening", "addr", ln.Addr().String())

	mux := http.NewServeMux()
	var wg sync.WaitGroup
	mux.HandleFunc("GET "+wsPath, func(rw http.ResponseWriter, r *http.Request) {
		wg.Add(1)
		defer wg.Done()
		w.handle(ctx, rw, r)
	})

	httpSrv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(stdnet.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		shctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shctx)
	}()

	err = httpSrv.Serve(ln)
	wg.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) && ctx.Err() == nil {
		return err
	}
	return nil
}

func (w *WS) handle(ctx context.Context, rw http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(rw, r, nil)
	if err != nil {
		w.Log.Debug("ws upgrade", "err", err)
		return
	}
	defer func() { _ = conn.Close() }()

	id := engine.ConnID(fmt.Sprintf("ws-%d", connSeq.Add(1)))
	sess := &wsSession{
		id:    id,
		conn:  conn,
		loop:  w.Loop,
		store: w.Store,
	}

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()

	defer func() {
		w.Loop.Submit(engine.LeaveWorld{ConnID: id})
		if !sess.attached {
			return
		}
		dctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		_ = w.Loop.Detach(dctx, id)
	}()

	sess.readLoop(ctx)
}

type wsSession struct {
	id       engine.ConnID
	conn     *websocket.Conn
	mu       sync.Mutex // socket write only; not world state
	loop     *engine.Loop
	store    persist.AccountStore
	lim      limiter
	user     string
	session  string
	authed   bool
	attached bool
}

type inFrame struct {
	V       int             `json:"v"`
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type outFrame struct {
	V       int    `json:"v"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Payload any    `json:"payload"`
}

func (s *wsSession) readLoop(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		_ = s.conn.SetReadDeadline(time.Now().Add(readIdle))
		mt, r, err := s.conn.NextReader()
		if err != nil {
			return
		}
		if mt != websocket.TextMessage {
			_, _ = io.Copy(io.Discard, r)
			if err := s.writeSys("", codeBadFrame, codeBadFrame); err != nil {
				return
			}
			continue
		}
		data, err := readLimited(r, maxFrame)
		if err != nil {
			if errors.Is(err, errFrameTooLarge) {
				if err := s.writeSys("", codeBadFrame, codeBadFrame); err != nil {
					return
				}
				continue
			}
			return
		}
		if err := s.handleFrame(ctx, data); err != nil {
			return
		}
	}
}

func (s *wsSession) handleFrame(ctx context.Context, data []byte) error {
	f, err := parseFrame(data)
	if err != nil {
		return s.writeSys("", codeBadFrame, codeBadFrame)
	}
	if f.V != protoV {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	switch f.Type {
	case typeAuthCreate, typeAuthLogin, typeCmdSay, typeCmdLook, typeCmdMove, typeCmdQuit,
		typeCmdPractice, typeCmdSkills, typeCmdInv, typeCmdGet, typeCmdDrop, typeCmdEquip, typeCmdUnequip:
		if !s.lim.allow(time.Now()) {
			return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
		}
	default:
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	switch f.Type {
	case typeAuthCreate:
		return s.doAuth(ctx, true, f)
	case typeAuthLogin:
		return s.doAuth(ctx, false, f)
	case typeCmdSay:
		return s.doSay(f)
	case typeCmdLook:
		return s.doLook(f)
	case typeCmdMove:
		return s.doMove(f)
	case typeCmdPractice:
		return s.doPractice(f)
	case typeCmdSkills, typeCmdInv:
		return s.doSheet(f)
	case typeCmdGet:
		return s.doGet(f)
	case typeCmdDrop:
		return s.doDrop(f)
	case typeCmdEquip:
		return s.doEquip(f)
	case typeCmdUnequip:
		return s.doUnequip(f)
	case typeCmdQuit:
		return s.doQuit()
	default:
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
}

func (s *wsSession) doAuth(ctx context.Context, create bool, f inFrame) error {
	var p struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodePayload(f.Payload, &p); err != nil {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	if s.authed {
		return s.writeJSON(typeAuthOK, f.ID, map[string]string{
			"username": s.user,
			"session":  s.session,
		})
	}
	var (
		acc persist.Account
		err error
	)
	if create {
		acc, err = s.store.Create(ctx, p.Username, p.Password)
	} else {
		acc, err = s.store.Authenticate(ctx, p.Username, p.Password)
	}
	if err != nil {
		code, msg := mapAuthErr(err)
		return s.writeJSON(typeAuthErr, f.ID, map[string]string{"code": code, "message": msg})
	}
	tok, err := s.store.IssueSession(ctx, acc.ID, 0)
	if err != nil {
		return s.writeSys(f.ID, codeInternal, codeInternal)
	}
	return s.enter(ctx, acc, tok, f.ID)
}

func (s *wsSession) enter(ctx context.Context, acc persist.Account, tok persist.Session, id string) error {
	out, err := s.loop.Attach(ctx, s.id)
	if err != nil {
		return s.writeSys(id, codeInternal, codeInternal)
	}
	s.attached = true
	sheet, err := s.store.LoadSheet(ctx, acc.ID)
	if err != nil {
		return s.writeSys(id, codeInternal, codeInternal)
	}
	if !s.loop.Submit(engine.EnterWorld{
		ConnID:    s.id,
		AccountID: engine.AccountID(acc.ID),
		Username:  acc.Username,
		Session:   tok.Token,
		Sheet:     sheet,
	}) {
		return s.writeSys(id, codeRateLimited, codeRateLimited)
	}
	s.authed = true
	s.user = acc.Username
	s.session = tok.Token
	go s.drain(ctx, out)
	return s.writeJSON(typeAuthOK, id, map[string]string{
		"username": acc.Username,
		"session":  tok.Token,
	})
}

func (s *wsSession) doSay(f inFrame) error {
	if !s.authed {
		return s.writeSys(f.ID, codeNotAuth, codeNotAuth)
	}
	var p struct {
		Text string `json:"text"`
	}
	if err := decodePayload(f.Payload, &p); err != nil {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	if p.Text == "" {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	if !s.loop.Submit(engine.Say{ConnID: s.id, Text: p.Text}) {
		return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
	}
	return nil
}

func (s *wsSession) doLook(f inFrame) error {
	if !s.authed {
		return s.writeSys(f.ID, codeNotAuth, codeNotAuth)
	}
	if !s.loop.Submit(engine.Look{ConnID: s.id}) {
		return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
	}
	return nil
}

func (s *wsSession) doMove(f inFrame) error {
	if !s.authed {
		return s.writeSys(f.ID, codeNotAuth, codeNotAuth)
	}
	var p struct {
		Dir string `json:"dir"`
	}
	if err := decodePayload(f.Payload, &p); err != nil {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	if _, ok := canonicalDir[p.Dir]; !ok {
		return s.writeSys(f.ID, codeBadDir, codeBadDir)
	}
	if !s.loop.Submit(engine.Move{ConnID: s.id, Dir: p.Dir}) {
		return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
	}
	return nil
}

var canonicalDir = map[string]struct{}{
	"north": {}, "south": {}, "east": {}, "west": {}, "up": {}, "down": {},
}

func (s *wsSession) doPractice(f inFrame) error {
	if !s.authed {
		return s.writeSys(f.ID, codeNotAuth, codeNotAuth)
	}
	var p struct {
		Skill string `json:"skill"`
	}
	if err := decodePayload(f.Payload, &p); err != nil || strings.TrimSpace(p.Skill) == "" {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	if !s.loop.Submit(engine.Practice{ConnID: s.id, SkillID: strings.TrimSpace(p.Skill)}) {
		return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
	}
	return nil
}

func (s *wsSession) doSheet(f inFrame) error {
	if !s.authed {
		return s.writeSys(f.ID, codeNotAuth, codeNotAuth)
	}
	if !s.loop.Submit(engine.Sheet{ConnID: s.id}) {
		return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
	}
	return nil
}

func (s *wsSession) doGet(f inFrame) error {
	return s.submitItem(f, func(id string) engine.Command { return engine.Get{ConnID: s.id, ItemID: id} })
}

func (s *wsSession) doDrop(f inFrame) error {
	return s.submitItem(f, func(id string) engine.Command { return engine.DropItem{ConnID: s.id, ItemID: id} })
}

func (s *wsSession) doEquip(f inFrame) error {
	return s.submitItem(f, func(id string) engine.Command { return engine.Equip{ConnID: s.id, ItemID: id} })
}

func (s *wsSession) submitItem(f inFrame, cmd func(string) engine.Command) error {
	if !s.authed {
		return s.writeSys(f.ID, codeNotAuth, codeNotAuth)
	}
	var p struct {
		Item string `json:"item"`
	}
	if err := decodePayload(f.Payload, &p); err != nil || strings.TrimSpace(p.Item) == "" {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	if !s.loop.Submit(cmd(strings.TrimSpace(p.Item))) {
		return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
	}
	return nil
}

func (s *wsSession) doUnequip(f inFrame) error {
	if !s.authed {
		return s.writeSys(f.ID, codeNotAuth, codeNotAuth)
	}
	var p struct {
		Slot string `json:"slot"`
	}
	if err := decodePayload(f.Payload, &p); err != nil || strings.TrimSpace(p.Slot) == "" {
		return s.writeSys(f.ID, codeBadFrame, codeBadFrame)
	}
	if !s.loop.Submit(engine.Unequip{ConnID: s.id, Slot: strings.TrimSpace(p.Slot)}) {
		return s.writeSys(f.ID, codeRateLimited, codeRateLimited)
	}
	return nil
}

func (s *wsSession) doQuit() error {
	if s.user != "" {
		_ = s.writeJSON(typeText, "", map[string]string{
			"channel": engine.ChannelSys,
			"from":    "",
			"text":    text.T(text.Default, text.SysLeave, s.user),
		})
	}
	s.loop.Submit(engine.LeaveWorld{ConnID: s.id})
	return errWSQuit
}

func (s *wsSession) drain(ctx context.Context, out <-chan engine.Event) {
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

func (s *wsSession) writeEvent(ev engine.Event) error {
	switch e := ev.(type) {
	case engine.Text:
		if e.Code != "" {
			return s.writeSys("", e.Code, e.Body)
		}
		return s.writeJSON(typeText, "", map[string]string{
			"channel": e.Channel,
			"from":    e.From,
			"text":    e.Body,
		})
	case engine.Room:
		exits := e.Exits
		if exits == nil {
			exits = map[string]string{}
		}
		who := e.Who
		if who == nil {
			who = []string{}
		}
		ground := e.Ground
		if ground == nil {
			ground = []string{}
		}
		return s.writeJSON(typeRoom, "", map[string]any{
			"id":          e.ID,
			"name":        e.Name,
			"description": e.Description,
			"exits":       exits,
			"who":         who,
			"ground":      ground,
		})
	case engine.Drop:
		return io.EOF
	default:
		return nil
	}
}

func (s *wsSession) writeSys(id, code, message string) error {
	return s.writeJSON(typeSys, id, map[string]string{"code": code, "message": message})
}

func (s *wsSession) writeJSON(typ, id string, payload any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return s.conn.WriteJSON(outFrame{V: protoV, Type: typ, ID: id, Payload: payload})
}

func parseFrame(data []byte) (inFrame, error) {
	var f inFrame
	if err := json.Unmarshal(data, &f); err != nil {
		return inFrame{}, err
	}
	return f, nil
}

func decodePayload(raw json.RawMessage, dst any) error {
	if len(raw) > maxPayload {
		return errFrameTooLarge
	}
	if len(raw) == 0 || string(raw) == "null" {
		raw = []byte("{}")
	}
	return json.Unmarshal(raw, dst)
}

func readLimited(r io.Reader, n int) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, int64(n)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > n {
		_, _ = io.Copy(io.Discard, r)
		return nil, errFrameTooLarge
	}
	return data, nil
}

func mapAuthErr(err error) (code, message string) {
	switch {
	case errors.Is(err, persist.ErrNameTaken):
		return persist.ErrNameTaken.Error(), persist.ErrNameTaken.Error()
	case errors.Is(err, persist.ErrBadCredentials):
		return persist.ErrBadCredentials.Error(), persist.ErrBadCredentials.Error()
	case errors.Is(err, persist.ErrBadUsername):
		return persist.ErrBadUsername.Error(), persist.ErrBadUsername.Error()
	case errors.Is(err, persist.ErrBadPassword):
		return persist.ErrBadPassword.Error(), persist.ErrBadPassword.Error()
	default:
		return codeInternal, codeInternal
	}
}
