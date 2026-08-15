// Package persist implements account and session storage (D-012, D-014, D-017).
// Hashing and store I/O run outside the game loop. Empty DATABASE_URL selects
// the in-memory driver.
package persist

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
)

const (
	minUsernameRunes = 2
	maxUsernameRunes = 16
	minPasswordBytes = 8
	maxPasswordBytes = 72

	// DefaultSessionTTL is used when IssueSession is given a non-positive ttl.
	// M0 does not use sessions for reconnect.
	DefaultSessionTTL = 24 * time.Hour

	hangulSyllableFirst = '가'
	hangulSyllableLast  = '힣'
)

// Sentinel errors. The first four match WIRE-PROTOCOL auth.err codes so net
// can map with errors.Is. Authenticate never returns a more specific failure
// than ErrBadCredentials (no user enumeration).
var (
	ErrBadUsername     = errors.New("bad_username")
	ErrBadPassword     = errors.New("bad_password")
	ErrNameTaken       = errors.New("name_taken")
	ErrBadCredentials  = errors.New("bad_credentials")
	ErrNotFound        = errors.New("not_found")
	ErrSessionNotFound = errors.New("session_not_found")
)

// Account is a stored player identity. The password hash never leaves the store.
type Account struct {
	ID        string
	Username  string
	CreatedAt time.Time
}

// Session is an opaque login token. Username is joined for the adapter.
type Session struct {
	Token     string
	AccountID string
	Username  string
	ExpiresAt time.Time
}

// AccountStore is the persist-side auth API. The game loop must not call it.
type AccountStore interface {
	Create(ctx context.Context, username, password string) (Account, error)
	Authenticate(ctx context.Context, username, password string) (Account, error)
	IssueSession(ctx context.Context, accountID string, ttl time.Duration) (Session, error)
	LookupSession(ctx context.Context, token string) (Session, error)
	RevokeSession(ctx context.Context, token string) error
}

var (
	_ AccountStore = (*Memory)(nil)
	_ AccountStore = (*Postgres)(nil)
)

// Open returns the in-memory store when databaseURL is empty (D-014).
func Open(ctx context.Context, databaseURL string) (AccountStore, error) {
	if databaseURL == "" {
		return NewMemory(), nil
	}
	return OpenPostgres(ctx, databaseURL)
}

// ValidateUsername enforces D-017 / WIRE-PROTOCOL: 2–16 runes; Hangul
// syllables (가–힣), ASCII letters, digits, or '_'.
func ValidateUsername(username string) error {
	n := 0
	for _, r := range username {
		n++
		if n > maxUsernameRunes || !validUsernameRune(r) {
			return ErrBadUsername
		}
	}
	if n < minUsernameRunes {
		return ErrBadUsername
	}
	return nil
}

// ValidatePassword enforces 8–72 bytes (WIRE-PROTOCOL).
func ValidatePassword(password string) error {
	n := len(password)
	if n < minPasswordBytes || n > maxPasswordBytes {
		return ErrBadPassword
	}
	return nil
}

func validUsernameRune(r rune) bool {
	switch {
	case r >= hangulSyllableFirst && r <= hangulSyllableLast:
		return true
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9', r == '_':
		return true
	default:
		return false
	}
}

// foldUsername is Unicode simple-fold, canonicalized to the smallest rune
// in each fold cycle. Hangul syllables have no case; ASCII maps to a–z.
func foldUsername(s string) string {
	return strings.Map(foldRune, s)
}

func foldRune(r rune) rune {
	min := r
	for c := unicode.SimpleFold(r); c != r; c = unicode.SimpleFold(c) {
		if c < min {
			min = c
		}
	}
	return min
}

func sessionTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return DefaultSessionTTL
	}
	return ttl
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func newToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
