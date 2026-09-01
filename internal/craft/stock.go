package craft

func nodeKey(room, item string) string { return room + "\x00" + item }

// Stock is remaining harvest at each node. The loop is the only writer.
type Stock struct {
	left, max, every map[string]int
}

// NewStock copies boot remaining from the catalog.
func (c *Catalog) NewStock() *Stock {
	s := &Stock{
		left:  map[string]int{},
		max:   map[string]int{},
		every: map[string]int{},
	}
	if c == nil {
		return s
	}
	for _, n := range c.nodes {
		k := nodeKey(n.Room, n.Item)
		s.left[k] = n.Stock
		s.max[k] = n.Stock
		s.every[k] = n.RegenTicks
	}
	return s
}

// Remaining is the live pile, or 0.
func (s *Stock) Remaining(room, item string) int {
	if s == nil {
		return 0
	}
	return s.left[nodeKey(room, item)]
}

// Take removes one unit. ok is false when the node is empty or unknown.
func (s *Stock) Take(room, item string) bool {
	if s == nil {
		return false
	}
	k := nodeKey(room, item)
	if s.left[k] <= 0 {
		return false
	}
	s.left[k]--
	return true
}

// Put restores one unit (bag full after a take). No-op at max.
func (s *Stock) Put(room, item string) {
	if s == nil {
		return
	}
	k := nodeKey(room, item)
	if s.left[k] < s.max[k] {
		s.left[k]++
	}
}

// Regen restores one unit on nodes whose interval divides tick.
func (s *Stock) Regen(tick uint64) {
	if s == nil || tick == 0 {
		return
	}
	for k, every := range s.every {
		if every <= 0 {
			continue
		}
		if tick%uint64(every) != 0 {
			continue
		}
		if s.left[k] < s.max[k] {
			s.left[k]++
		}
	}
}
