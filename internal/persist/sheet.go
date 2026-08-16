package persist

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pjhwa/yeomyeong/internal/world"
)

// LoadSheet returns a copy of the account sheet. Missing accounts are ErrNotFound.
func (p *Postgres) LoadSheet(ctx context.Context, accountID string) (world.Sheet, error) {
	var skills, stats, bag, equip []byte
	err := p.pool.QueryRow(ctx, `
		SELECT skills, stats, bag, equipment FROM accounts WHERE id = $1
	`, accountID).Scan(&skills, &stats, &bag, &equip)
	if errors.Is(err, pgx.ErrNoRows) {
		return world.Sheet{}, ErrNotFound
	}
	if err != nil {
		return world.Sheet{}, err
	}
	return decodeSheet(skills, stats, bag, equip)
}

// SaveSheet replaces the account sheet. Missing accounts are ErrNotFound.
func (p *Postgres) SaveSheet(ctx context.Context, accountID string, sheet world.Sheet) error {
	sh := world.CloneSheet(sheet)
	skills, err := json.Marshal(sh.Skills)
	if err != nil {
		return err
	}
	stats, err := json.Marshal(sh.Stats)
	if err != nil {
		return err
	}
	bag, err := json.Marshal(sh.Bag)
	if err != nil {
		return err
	}
	equip, err := json.Marshal(sh.Equip)
	if err != nil {
		return err
	}
	tag, err := p.pool.Exec(ctx, `UPDATE accounts SET skills=$2, stats=$3, bag=$4, equipment=$5 WHERE id=$1`,
		accountID, skills, stats, bag, equip)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func decodeSheet(skills, stats, bag, equip []byte) (world.Sheet, error) {
	sh := world.CloneSheet(world.Sheet{})
	if err := unmarshalJSON(skills, &sh.Skills); err != nil {
		return world.Sheet{}, err
	}
	if err := unmarshalJSON(stats, &sh.Stats); err != nil {
		return world.Sheet{}, err
	}
	if err := unmarshalJSON(bag, &sh.Bag); err != nil {
		return world.Sheet{}, err
	}
	if err := unmarshalJSON(equip, &sh.Equip); err != nil {
		return world.Sheet{}, err
	}
	return world.CloneSheet(sh), nil
}

func unmarshalJSON(raw []byte, dst any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, dst)
}

type sheetJob struct {
	id    string
	sheet world.Sheet
}

// AsyncSaver enqueues sheet writes so the game loop never blocks on I/O.
type AsyncSaver struct {
	store AccountStore
	log   *slog.Logger
	ch    chan sheetJob
	done  chan struct{}
}

// NewAsyncSaver starts a single persist worker. Close after the loop stops.
func NewAsyncSaver(store AccountStore, log *slog.Logger) *AsyncSaver {
	if log == nil {
		log = slog.Default()
	}
	s := &AsyncSaver{
		store: store,
		log:   log,
		ch:    make(chan sheetJob, 64),
		done:  make(chan struct{}),
	}
	go s.run()
	return s
}

// SaveAsync copies sheet and enqueues a write. A full queue drops the save.
func (s *AsyncSaver) SaveAsync(accountID string, sheet world.Sheet) {
	if s == nil {
		return
	}
	select {
	case s.ch <- sheetJob{id: accountID, sheet: world.CloneSheet(sheet)}:
	default:
		s.log.Warn("sheet save dropped", "account", accountID)
	}
}

// Close stops the worker after queued saves finish.
func (s *AsyncSaver) Close() {
	if s == nil {
		return
	}
	close(s.ch)
	<-s.done
}

func (s *AsyncSaver) run() {
	defer close(s.done)
	for job := range s.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.store.SaveSheet(ctx, job.id, job.sheet); err != nil {
			s.log.Error("save sheet", "account", job.id, "err", err)
		}
		cancel()
	}
}
