package persist

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/pjhwa/yeomyeong/internal/world"
)

func TestOpenEmptyURLIsMemory(t *testing.T) {
	s, err := Open(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*Memory); !ok {
		t.Fatalf("got %T", s)
	}
}

func TestAuthErrorCodes(t *testing.T) {
	want := map[error]string{
		ErrBadUsername:    "bad_username",
		ErrBadPassword:    "bad_password",
		ErrNameTaken:      "name_taken",
		ErrBadCredentials: "bad_credentials",
	}
	for err, code := range want {
		if err.Error() != code {
			t.Errorf("%v.Error()=%q want %q", err, err.Error(), code)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	ok := []string{"ab", "유저", "User_1", "가A1_", "가나다라마바사아자차카타파하ab"}
	if utf8.RuneCountInString(ok[len(ok)-1]) != 16 {
		t.Fatal("fixture must be 16 runes")
	}
	for _, name := range ok {
		if err := ValidateUsername(name); err != nil {
			t.Errorf("ValidateUsername(%q)=%v", name, err)
		}
	}
	bad := []string{"", "a", "abcdefghijklmnopq", "a-b", "a b", "ㄱㄱ", "유저!", "ab.cd", "１２", "한"}
	for _, name := range bad {
		if err := ValidateUsername(name); !errors.Is(err, ErrBadUsername) {
			t.Errorf("ValidateUsername(%q)=%v want bad_username", name, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("1234567"); !errors.Is(err, ErrBadPassword) {
		t.Fatalf("7 bytes: %v", err)
	}
	if err := ValidatePassword("12345678"); err != nil {
		t.Fatalf("8 bytes: %v", err)
	}
	if err := ValidatePassword(strings.Repeat("x", 72)); err != nil {
		t.Fatalf("72 bytes: %v", err)
	}
	if err := ValidatePassword(strings.Repeat("x", 73)); !errors.Is(err, ErrBadPassword) {
		t.Fatalf("73 bytes: %v", err)
	}
}

func TestMemoryStore(t *testing.T) {
	testAccountStore(t, NewMemory())
}

func testAccountStore(t *testing.T, s AccountStore) {
	t.Helper()
	ctx := context.Background()

	acc, err := s.Create(ctx, "갑을", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	if acc.ID == "" || acc.Username != "갑을" || acc.CreatedAt.IsZero() {
		t.Fatalf("create: %+v", acc)
	}

	got, err := s.Authenticate(ctx, "갑을", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != acc.ID || got.Username != "갑을" {
		t.Fatalf("auth: %+v", got)
	}
	if ok, err := s.Exists(ctx, "갑을"); err != nil || !ok {
		t.Fatalf("exists known: %v %v", ok, err)
	}
	if ok, err := s.Exists(ctx, "갑乙"); err != nil || ok {
		t.Fatalf("exists missing: %v %v", ok, err)
	}

	if _, err := s.Create(ctx, "Player", "password1"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"player", "PLAYER", "PlAyEr"} {
		if _, err := s.Create(ctx, name, "password1"); !errors.Is(err, ErrNameTaken) {
			t.Fatalf("dup %q: %v", name, err)
		}
	}
	folded, err := s.Authenticate(ctx, "PLAYER", "password1")
	if err != nil || folded.Username != "Player" {
		t.Fatalf("case-fold login: %+v %v", folded, err)
	}

	if _, err := s.Create(ctx, "x", "password1"); !errors.Is(err, ErrBadUsername) {
		t.Fatalf("short name: %v", err)
	}
	if _, err := s.Create(ctx, "okname", "short"); !errors.Is(err, ErrBadPassword) {
		t.Fatalf("short password: %v", err)
	}

	if _, err := s.Authenticate(ctx, "갑을", "wrongpass"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("bad password: %v", err)
	}
	if _, err := s.Authenticate(ctx, "nobody", "secret12"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("unknown user: %v", err)
	}
	if _, err := s.Authenticate(ctx, "!", "secret12"); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("invalid login name: %v", err)
	}

	sess, err := s.IssueSession(ctx, acc.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Token == "" || sess.AccountID != acc.ID || sess.Username != "갑을" {
		t.Fatalf("issue: %+v", sess)
	}
	look, err := s.LookupSession(ctx, sess.Token)
	if err != nil || look.Token != sess.Token || look.AccountID != acc.ID {
		t.Fatalf("lookup: %+v %v", look, err)
	}
	if err := s.RevokeSession(ctx, sess.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LookupSession(ctx, sess.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("revoked: %v", err)
	}
	if _, err := s.LookupSession(ctx, "no-such-token"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := s.IssueSession(ctx, "missing-account", time.Hour); !errors.Is(err, ErrNotFound) {
		t.Fatalf("issue missing: %v", err)
	}

	empty, err := s.LoadSheet(ctx, acc.ID)
	if err != nil || len(empty.Bag) != 0 || len(empty.Skills) != 0 {
		t.Fatalf("empty sheet: %+v %v", empty, err)
	}
	want := world.Sheet{
		Skills: map[string]int{"smith": 4},
		Stats:  map[string]int{"str": 1},
		Bag:    []world.Stack{{ID: "pebble", Qty: 2}},
		Equip:  world.Equipment{MainHand: "rod"},
		Nyang:  12,
	}
	if err := s.SaveSheet(ctx, acc.ID, want); err != nil {
		t.Fatal(err)
	}
	gotSheet, err := s.LoadSheet(ctx, acc.ID)
	if err != nil || gotSheet.Skills["smith"] != 4 || gotSheet.Stats["str"] != 1 ||
		len(gotSheet.Bag) != 1 || gotSheet.Bag[0] != (world.Stack{ID: "pebble", Qty: 2}) ||
		gotSheet.Equip.MainHand != "rod" || gotSheet.Nyang != 12 {
		t.Fatalf("load sheet: %+v %v", gotSheet, err)
	}
	gotSheet.Bag[0].ID = "hacked"
	again, err := s.LoadSheet(ctx, acc.ID)
	if err != nil || again.Bag[0].ID != "pebble" {
		t.Fatalf("sheet must copy: %+v %v", again, err)
	}
	if _, err := s.LoadSheet(ctx, "missing-account"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("load missing: %v", err)
	}
	if err := s.SaveSheet(ctx, "missing-account", want); !errors.Is(err, ErrNotFound) {
		t.Fatalf("save missing: %v", err)
	}
}

func TestExpiredSessionRejected(t *testing.T) {
	m := NewMemory()
	fixed := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return fixed }
	ctx := context.Background()
	acc, err := m.Create(ctx, "clock", "password1")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := m.IssueSession(ctx, acc.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.LookupSession(ctx, sess.Token); err != nil {
		t.Fatal(err)
	}
	m.now = func() time.Time { return fixed.Add(time.Hour) }
	if _, err := m.LookupSession(ctx, sess.Token); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("at expiry: %v", err)
	}
}

func TestMemoryCancelledContext(t *testing.T) {
	m := NewMemory()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Create(ctx, "ab", "password1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := m.Authenticate(ctx, "ab", "password1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("auth: %v", err)
	}
	if _, err := m.Exists(ctx, "ab"); !errors.Is(err, context.Canceled) {
		t.Fatalf("exists: %v", err)
	}
	if _, err := m.IssueSession(ctx, "x", time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("issue: %v", err)
	}
	if _, err := m.LookupSession(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Fatalf("lookup: %v", err)
	}
	if err := m.RevokeSession(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := m.LoadSheet(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Fatalf("load sheet: %v", err)
	}
	if err := m.SaveSheet(ctx, "x", world.Sheet{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("save sheet: %v", err)
	}
}

func TestMemoryRace(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	acc, err := m.Create(ctx, "race", "password1")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_, _ = m.Authenticate(ctx, "race", "password1")
			_, _ = m.Authenticate(ctx, "race", "wrongpass")
		}()
		go func() {
			defer wg.Done()
			sess, err := m.IssueSession(ctx, acc.ID, time.Hour)
			if err != nil {
				return
			}
			_, _ = m.LookupSession(ctx, sess.Token)
			_ = m.RevokeSession(ctx, sess.Token)
		}()
		go func(i int) {
			defer wg.Done()
			_, _ = m.Create(ctx, "race", "password1")
			_, _ = m.Create(ctx, "u"+strings.Repeat("x", i%4+1), "password1")
		}(i)
	}
	wg.Wait()
}

func TestPHCRoundTripAndRejectsOtherHashes(t *testing.T) {
	h, err := hashPassword("password1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("phc: %s", h)
	}
	if !verifyPassword(h, "password1") {
		t.Fatal("verify ok")
	}
	if verifyPassword(h, "password2") {
		t.Fatal("verify should fail")
	}
	if verifyPassword("$bcrypt$not-used", "password1") {
		t.Fatal("must not accept a second hash")
	}
}

func TestMigrationHasRequiredSchema(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(body)
		b.WriteByte('\n')
	}
	s := b.String()
	for _, need := range []string{
		"CREATE TABLE IF NOT EXISTS accounts",
		"username", "username_fold", "password_hash", "created_at",
		"CREATE TABLE IF NOT EXISTS sessions",
		"token", "account_id", "expires_at",
		"skills JSONB", "stats JSONB", "bag JSONB", "equipment JSONB",
	} {
		if !strings.Contains(s, need) {
			t.Errorf("migrations missing %q", need)
		}
	}
	for _, stmt := range schemaStmts {
		if !strings.Contains(s, strings.TrimSpace(strings.Split(stmt, "\n")[0])) {
			t.Errorf("schemaStmts drift vs migrations: %s", stmt[:min(40, len(stmt))])
		}
	}
}

func TestEngineDoesNotImportPersist(t *testing.T) {
	check := func(dir string, forbidden []string) {
		t.Helper()
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, needle := range forbidden {
				if bytes.Contains(b, []byte(needle)) {
					t.Errorf("%s must not contain %q", path, needle)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	check(filepath.Join("..", "engine"), []string{
		"github.com/pjhwa/yeomyeong/internal/persist",
		"golang.org/x/crypto/argon2",
		"database/sql",
		"github.com/jackc/pgx",
	})
	check(filepath.Join("..", "..", "cmd", "server"), []string{
		"golang.org/x/crypto/argon2",
	})
}
