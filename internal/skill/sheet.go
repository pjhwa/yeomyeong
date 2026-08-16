package skill

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
)

// Sheet is one character's ranks and stats. TryGain returns a new value;
// the receiver is not mutated. Rank/stat maps are unexported so callers
// cannot bypass the caps.
type Sheet struct {
	cat   *Catalog
	ranks map[string]int
	stats map[string]int
}

// Rank is the stored rank for id, or 0.
func (s Sheet) Rank(id string) int { return s.ranks[id] }

// Stat is the stored stat for id, or 0.
func (s Sheet) Stat(id string) int { return s.stats[id] }

// RankSum is the sum of all stored ranks (cap 700).
func (s Sheet) RankSum() int { return sumMap(s.ranks) }

// StatSum is the sum of all stored stats (cap 300).
func (s Sheet) StatSum() int { return sumMap(s.stats) }

// Title is the first matching catalog rule, or empty if unbound.
func (s Sheet) Title() Localized {
	if s.cat == nil {
		return Localized{}
	}
	return s.cat.Title(s)
}

// Ranks returns a copy of stored ranks.
func (s Sheet) Ranks() map[string]int { return copyInts(s.ranks) }

// Stats returns a copy of stored stats.
func (s Sheet) Stats() map[string]int { return copyInts(s.stats) }

// Bind attaches ranks and stats to this catalog. Nil maps become empty.
func (c *Catalog) Bind(ranks, stats map[string]int) Sheet {
	return Sheet{cat: c, ranks: copyInts(ranks), stats: copyInts(stats)}
}

// Difficulty is the practice trial difficulty (SKILL-TABLE.md).
// A matching room flag or held item uses the current rank; otherwise max(0, rank-25).
func Difficulty(rank int, matched bool) int {
	if rank < 0 {
		rank = 0
	}
	if matched {
		return rank
	}
	if rank < 25 {
		return 0
	}
	return rank - 25
}

// Always is a deterministic rng hook that always succeeds a Bernoulli trial.
func Always() float64 { return 0 }

// Never is a deterministic rng hook that always fails a Bernoulli trial.
func Never() float64 { return 1 }

func copyInts(m map[string]int) map[string]int {
	out := make(map[string]int, len(m))
	for k, v := range m {
		if v > 0 {
			out[k] = v
		}
	}
	return out
}

// WithRank returns a copy with id set to n. n<=0 deletes the key.
func (s Sheet) WithRank(id string, n int) Sheet {
	out := s.clone()
	if n <= 0 {
		delete(out.ranks, id)
	} else {
		out.ranks[id] = n
	}
	return out
}

// WithStat returns a copy with id set to n. n<=0 deletes the key.
func (s Sheet) WithStat(id string, n int) Sheet {
	out := s.clone()
	if n <= 0 {
		delete(out.stats, id)
	} else {
		out.stats[id] = n
	}
	return out
}

// Chance is the Bernoulli p for one practice trial (SKILL-TABLE.md).
// Rank-sum / stat caps are applied by TryGain, not here.
func Chance(rank, difficulty int) float64 {
	if rank >= RankCap {
		return 0
	}
	if rank < 0 {
		rank = 0
	}
	d := float64(rank-difficulty) / 18
	proximity := math.Exp(-0.5 * d * d)
	falloff := 1 / (1 + float64(rank)/12)
	return 0.55 * falloff * proximity
}

// TryGain is one independent Bernoulli trial. On success the rank
// increases by 1. rng must return a value in [0,1); nil uses DefaultRand.
func (s Sheet) TryGain(skillID string, difficulty int, rng func() float64) (gained bool, sheet Sheet) {
	out := s.clone()
	if s.cat == nil {
		return false, out
	}
	sk, ok := s.cat.Skill(skillID)
	if !ok {
		return false, out
	}
	rank := out.Rank(skillID)
	if rank >= RankCap || out.RankSum() >= RankSumCap {
		return false, out
	}
	if rng == nil {
		rng = DefaultRand
	}
	if rng() >= Chance(rank, difficulty) {
		return false, out
	}
	out.ranks[skillID] = rank + 1
	tryStatGain(out.stats, sk.Stat, rng)
	return true, out
}

func tryStatGain(stats map[string]int, statID string, rng func() float64) {
	if statID == "" {
		return
	}
	cur := stats[statID]
	if cur >= StatCap || sumMap(stats) >= StatSumCap {
		return
	}
	p := 0.25 * (1 - float64(cur)/float64(StatCap))
	if rng() >= p {
		return
	}
	stats[statID] = cur + 1
}

// DefaultRand is a process-wide [0,1) source from crypto/rand.
// It is not seeded from time.Now per call.
func DefaultRand() float64 {
	var buf [8]byte
	if _, err := crand.Read(buf[:]); err != nil {
		panic("skill: crypto/rand: " + err.Error())
	}
	u := binary.LittleEndian.Uint64(buf[:]) >> 11
	return float64(u) / (1 << 53)
}

func (s Sheet) clone() Sheet {
	out := Sheet{cat: s.cat, ranks: make(map[string]int, len(s.ranks)), stats: make(map[string]int, len(s.stats))}
	for k, v := range s.ranks {
		out.ranks[k] = v
	}
	for k, v := range s.stats {
		out.stats[k] = v
	}
	return out
}

func sumMap(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}
