package persist

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pgUniqueViolation = "23505"
	pgFKViolation     = "23503"
)

// schemaStmts is the M0 DDL. Keep migrations/001_init.sql identical.
var schemaStmts = []string{
	`CREATE TABLE IF NOT EXISTS accounts (
		id            TEXT PRIMARY KEY,
		username      TEXT NOT NULL,
		username_fold TEXT NOT NULL,
		password_hash TEXT NOT NULL,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS accounts_username_fold_uidx ON accounts (username_fold)`,
	`CREATE TABLE IF NOT EXISTS sessions (
		token      TEXT PRIMARY KEY,
		account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
		expires_at TIMESTAMPTZ NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS sessions_account_id_idx ON sessions (account_id)`,
	`CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at)`,
}

// Postgres is the PostgreSQL AccountStore. Used only when DATABASE_URL is set.
type Postgres struct {
	pool *pgxpool.Pool
}

// OpenPostgres connects, pings, and applies IF NOT EXISTS schema.
func OpenPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	p := &Postgres{pool: pool}
	if err := p.applySchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

// Close releases the connection pool.
func (p *Postgres) Close() error {
	p.pool.Close()
	return nil
}

func (p *Postgres) applySchema(ctx context.Context) error {
	for _, stmt := range schemaStmts {
		if _, err := p.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// Create inserts a new account. Unique fold collisions become ErrNameTaken.
func (p *Postgres) Create(ctx context.Context, username, password string) (Account, error) {
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
	var acc Account
	err = p.pool.QueryRow(ctx, `
		INSERT INTO accounts (id, username, username_fold, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, created_at
	`, id, username, foldUsername(username), hash).Scan(&acc.ID, &acc.Username, &acc.CreatedAt)
	if pgCode(err) == pgUniqueViolation {
		return Account{}, ErrNameTaken
	}
	return acc, err
}

// Authenticate verifies username+password. Every failure is ErrBadCredentials.
func (p *Postgres) Authenticate(ctx context.Context, username, password string) (Account, error) {
	var acc Account
	var hash string
	err := p.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, created_at
		FROM accounts WHERE username_fold = $1
	`, foldUsername(username)).Scan(&acc.ID, &acc.Username, &hash, &acc.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = verifyPassword(dummyPasswordHash(), password)
		return Account{}, ErrBadCredentials
	}
	if err != nil {
		return Account{}, err
	}
	if !verifyPassword(hash, password) {
		return Account{}, ErrBadCredentials
	}
	return acc, nil
}

// IssueSession records an unguessable token for accountID.
func (p *Postgres) IssueSession(ctx context.Context, accountID string, ttl time.Duration) (Session, error) {
	if accountID == "" {
		return Session{}, ErrNotFound
	}
	token, err := newToken()
	if err != nil {
		return Session{}, err
	}
	expires := time.Now().Add(sessionTTL(ttl))
	var username string
	err = p.pool.QueryRow(ctx, `
		INSERT INTO sessions (token, account_id, expires_at)
		SELECT $1, id, $2 FROM accounts WHERE id = $3
		RETURNING (SELECT username FROM accounts WHERE id = $3)
	`, token, expires, accountID).Scan(&username)
	if errors.Is(err, pgx.ErrNoRows) || pgCode(err) == pgFKViolation {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	return Session{Token: token, AccountID: accountID, Username: username, ExpiresAt: expires}, nil
}

// LookupSession returns a live session. Missing, revoked, and expired tokens fail.
func (p *Postgres) LookupSession(ctx context.Context, token string) (Session, error) {
	var sess Session
	err := p.pool.QueryRow(ctx, `
		SELECT s.token, s.account_id, a.username, s.expires_at
		FROM sessions s
		JOIN accounts a ON a.id = s.account_id
		WHERE s.token = $1 AND s.expires_at > $2
	`, token, time.Now()).Scan(&sess.Token, &sess.AccountID, &sess.Username, &sess.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	return sess, err
}

// RevokeSession drops token. Missing tokens succeed.
func (p *Postgres) RevokeSession(ctx context.Context, token string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func pgCode(err error) string {
	var e *pgconn.PgError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
