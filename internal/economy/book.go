// Package economy is the regional price book (PLAN.md §3.4).
// The game loop is the only writer of Book. There is no mutex.
package economy

import (
	"math"
	"sort"
)

// Elasticity is how hard stock-vs-target pulls the quote.
// Formula constants live in Go (like the skill curve); prices do not.
const Elasticity = 0.5

// Good is one commodity's live stock in one market.
type Good struct {
	ID     string
	Base   int
	Stock  int
	Target int
	Demand float64
}

// Market is one region's live table.
type Market struct {
	ID    string
	Name  string
	Goods map[string]Good
}

// QuoteRow is one line of 시세.
type QuoteRow struct {
	ID     string
	Price  int
	Stock  int
	Target int
}

// Book is mutable market state. Construct via Load or NewBook; the loop owns it.
type Book struct {
	order   []string
	markets map[string]*Market
}

// NewBook copies specs into a live book. Caller must not mutate the input after.
func NewBook(markets []Market) *Book {
	b := &Book{markets: make(map[string]*Market, len(markets))}
	for _, m := range markets {
		cp := &Market{ID: m.ID, Name: m.Name, Goods: make(map[string]Good, len(m.Goods))}
		for id, g := range m.Goods {
			g.ID = id
			if g.Demand <= 0 {
				g.Demand = 1
			}
			if g.Base < 1 {
				g.Base = 1
			}
			cp.Goods[id] = g
		}
		b.markets[m.ID] = cp
		b.order = append(b.order, m.ID)
	}
	return b
}

// HasMarket reports whether slug is a loaded market.
func (b *Book) HasMarket(slug string) bool {
	if b == nil {
		return false
	}
	_, ok := b.markets[slug]
	return ok
}

// MarketName is the Korean stall name, or slug.
func (b *Book) MarketName(slug string) string {
	if b == nil {
		return slug
	}
	m, ok := b.markets[slug]
	if !ok || m.Name == "" {
		return slug
	}
	return m.Name
}

// Trades reports whether any market lists item.
func (b *Book) Trades(item string) bool {
	if b == nil || item == "" {
		return false
	}
	for _, m := range b.markets {
		if _, ok := m.Goods[item]; ok {
			return true
		}
	}
	return false
}

// Quote is the current unit price. ok is false when the stall does not list item.
func (b *Book) Quote(market, item string) (int, bool) {
	g, ok := b.good(market, item)
	if !ok {
		return 0, false
	}
	return Price(g.Base, g.Demand, g.Stock, g.Target), true
}

// List returns goods in this market, sorted by id.
func (b *Book) List(market string) []QuoteRow {
	if b == nil {
		return nil
	}
	m, ok := b.markets[market]
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(m.Goods))
	for id := range m.Goods {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]QuoteRow, 0, len(ids))
	for _, id := range ids {
		g := m.Goods[id]
		out = append(out, QuoteRow{
			ID: id, Price: Price(g.Base, g.Demand, g.Stock, g.Target),
			Stock: g.Stock, Target: g.Target,
		})
	}
	return out
}

// Sell adds n to stall stock and returns the coin paid to the seller.
func (b *Book) Sell(market, item string, n int) (int, bool) {
	g, ok := b.good(market, item)
	if !ok || n <= 0 {
		return 0, false
	}
	unit := Price(g.Base, g.Demand, g.Stock, g.Target)
	g.Stock += n
	b.set(market, g)
	return unit * n, true
}

// Buy removes n from stall stock and returns the coin charged.
func (b *Book) Buy(market, item string, n int) (int, bool) {
	g, ok := b.good(market, item)
	if !ok || n <= 0 || g.Stock < n {
		return 0, false
	}
	unit := Price(g.Base, g.Demand, g.Stock, g.Target)
	g.Stock -= n
	b.set(market, g)
	return unit * n, true
}

// Stock is the live pile, or 0.
func (b *Book) Stock(market, item string) int {
	g, ok := b.good(market, item)
	if !ok {
		return 0
	}
	return g.Stock
}

// Tick walks each good one step toward its target (NPC flow).
func (b *Book) Tick() {
	if b == nil {
		return
	}
	for _, m := range b.markets {
		for id, g := range m.Goods {
			switch {
			case g.Stock < g.Target:
				g.Stock++
			case g.Stock > g.Target:
				g.Stock--
			}
			m.Goods[id] = g
		}
	}
}

// IDs returns market slugs in file order.
func (b *Book) IDs() []string {
	if b == nil {
		return nil
	}
	return append([]string(nil), b.order...)
}

// Price is the quoted unit price. Documented in D-042.
func Price(base int, demand float64, stock, target int) int {
	if base < 1 {
		base = 1
	}
	if demand <= 0 {
		demand = 1
	}
	t := target
	if t < 1 {
		t = 1
	}
	factor := 1 + Elasticity*float64(target-stock)/float64(t)
	if factor < 0.2 {
		factor = 0.2
	}
	p := int(math.Round(float64(base) * demand * factor))
	if p < 1 {
		p = 1
	}
	return p
}

func (b *Book) good(market, item string) (Good, bool) {
	if b == nil || market == "" || item == "" {
		return Good{}, false
	}
	m, ok := b.markets[market]
	if !ok {
		return Good{}, false
	}
	g, ok := m.Goods[item]
	return g, ok
}

func (b *Book) set(market string, g Good) {
	b.markets[market].Goods[g.ID] = g
}
