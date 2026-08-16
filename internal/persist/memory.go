package persist

import (
	"context"
	"sync"
	"time"

	"github.com/pjhwa/yeomyeong/internal/world"
)

type memAccount struct {
	Account
	hash  string
	sheet world.Sheet
}

// Memory is the in-memory AccountStore. It may use a mutex: it is not world state (D-012).
type Memory struct {
	mu       sync.Mutex
	now      func() time.Time
	byFold   map[string]*memAccount
	byID     map[string]*memAccount
	sessions map[string]Session
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		now:      time.Now,
		byFold:   make(map[string]*memAccount),
		byID:     make(map[string]*memAccount),
		sessions: make(map[string]Session),
	}
}

// Create registers a new account. Username uniqueness is Unicode simple-fold.
func (m *Memory) Create(ctx context.Context, username, password string) (Account, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	if err := ValidateUsername(username); err != nil {
		return Account{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Account{}, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return Account{}, err
	}
	id, err := newID()
	if err != nil {
		return Account{}, err
	}
	fold := foldUsername(username)
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byFold[fold]; exists {
		return Account{}, ErrNameTaken
	}
	acc := Account{ID: id, Username: username, CreatedAt: m.now()}
	rec := &memAccount{Account: acc, hash: hash, sheet: world.CloneSheet(world.Sheet{})}
	m.byFold[fold] = rec
	m.byID[id] = rec
	return acc, nil
}

// Exists reports whether username is registered (Unicode simple-fold).
func (m *Memory) Exists(ctx context.Context, username string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	fold := foldUsername(username)
	m.mu.Lock()
	_, ok := m.byFold[fold]
	m.mu.Unlock()
	return ok, nil
}

// Authenticate verifies username+password. Every failure is ErrBadCredentials.
func (m *Memory) Authenticate(ctx context.Context, username, password string) (Account, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	fold := foldUsername(username)
	m.mu.Lock()
	rec, ok := m.byFold[fold]
	var hash string
	var acc Account
	if ok {
		hash = rec.hash
		acc = rec.Account
	}
	m.mu.Unlock()
	if !ok {
		hash = dummyPasswordHash()
	}
	if !verifyPassword(hash, password) || !ok {
		return Account{}, ErrBadCredentials
	}
	return acc, nil
}

// IssueSession records an unguessable token for accountID.
func (m *Memory) IssueSession(ctx context.Context, accountID string, ttl time.Duration) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	if accountID == "" {
		return Session{}, ErrNotFound
	}
	token, err := newToken()
	if err != nil {
		return Session{}, err
	}
	ttl = sessionTTL(ttl)
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[accountID]
	if !ok {
		return Session{}, ErrNotFound
	}
	sess := Session{
		Token:     token,
		AccountID: rec.ID,
		Username:  rec.Username,
		ExpiresAt: m.now().Add(ttl),
	}
	m.sessions[token] = sess
	return sess, nil
}

// LookupSession returns a live session. Missing, revoked, and expired tokens fail.
func (m *Memory) LookupSession(ctx context.Context, token string) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	sess, ok := m.sessions[token]
	if !ok || !m.now().Before(sess.ExpiresAt) {
		return Session{}, ErrSessionNotFound
	}
	return sess, nil
}

// RevokeSession drops token. Missing tokens succeed.
func (m *Memory) RevokeSession(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
	return nil
}

// LoadSheet returns a copy of the account sheet. Missing accounts are ErrNotFound.
func (m *Memory) LoadSheet(ctx context.Context, accountID string) (world.Sheet, error) {
	if err := ctx.Err(); err != nil {
		return world.Sheet{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[accountID]
	if !ok {
		return world.Sheet{}, ErrNotFound
	}
	return world.CloneSheet(rec.sheet), nil
}

// SaveSheet replaces the account sheet. Missing accounts are ErrNotFound.
func (m *Memory) SaveSheet(ctx context.Context, accountID string, sheet world.Sheet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[accountID]
	if !ok {
		return ErrNotFound
	}
	rec.sheet = world.CloneSheet(sheet)
	return nil
}
