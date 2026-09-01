package engine

import (
	"strconv"
	"strings"

	"github.com/pjhwa/yeomyeong/internal/craft"
	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

var stationKO = map[string]string{
	"forge":   "대장간",
	"kitchen": "부엌",
	"press":   "인쇄소",
	"clinic":  "약방",
}

func (l *Loop) livelihoodTick() {
	if l.nodes != nil {
		l.nodes.Regen(l.ticks)
	}
	if l.markets != nil && l.ticks > 0 && l.ticks%uint64(MarketTickEvery) == 0 {
		l.markets.Tick()
	}
}

func (l *Loop) gather(c Gather) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	if l.craft == nil || l.nodes == nil {
		l.sys(p.ConnID, text.GatherNone, text.CodeNoNode)
		return
	}
	skID := ""
	if c.Skill != "" {
		if sk, ok := l.GatherSkill(c.Skill); ok {
			skID = sk.ID
		}
	}
	node, ok := l.pickNode(p.RoomID, c.Query, skID)
	if !ok {
		l.sys(p.ConnID, text.GatherNone, text.CodeNoNode)
		return
	}
	if l.nodes.Remaining(node.Room, node.Item) <= 0 {
		l.sys(p.ConnID, text.GatherEmpty, text.CodeNoStock)
		return
	}
	if l.carryWeight(p)+l.itemWeight(node.Item) > BagCap {
		l.sys(p.ConnID, text.GatherHeavy, text.CodeTooHeavy)
		return
	}
	if !l.nodes.Take(node.Room, node.Item) {
		l.sys(p.ConnID, text.GatherEmpty, text.CodeNoStock)
		return
	}
	p.Bag = yworld.AddStack(p.Bag, node.Item, 1)
	name := l.itemName(node.Item)
	l.world.roster[c.ConnID] = p
	l.sysf(p.ConnID, text.GatherOK, name, text.EulReul(name))
	l.applyGain(&p, node.Skill, true)
	l.world.roster[c.ConnID] = p
}

func (l *Loop) pickNode(room, query, skillID string) (craft.Node, bool) {
	nodes := l.craft.NodesIn(room)
	if len(nodes) == 0 {
		return craft.Node{}, false
	}
	q := strings.TrimSpace(query)
	if q != "" {
		id := l.catalogItemID(q)
		if id == "" {
			id = q
		}
		for _, n := range nodes {
			if n.Item == id && (skillID == "" || n.Skill == skillID) {
				return n, true
			}
		}
		return craft.Node{}, false
	}
	for _, n := range nodes {
		if skillID != "" && n.Skill != skillID {
			continue
		}
		if l.nodes.Remaining(n.Room, n.Item) > 0 {
			return n, true
		}
	}
	for _, n := range nodes {
		if skillID == "" || n.Skill == skillID {
			return n, true
		}
	}
	return craft.Node{}, false
}

func (l *Loop) doCraft(c Craft) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	if l.craft == nil {
		l.sys(p.ConnID, text.CraftNone, text.CodeNoRecipe)
		return
	}
	q := strings.TrimSpace(c.Query)
	if q == "" {
		l.listCrafts(p)
		return
	}
	rec, ok := l.findRecipe(q)
	if !ok {
		l.sys(p.ConnID, text.CraftUnknown, text.CodeNoRecipe)
		return
	}
	flags := l.roomFlags(p.RoomID)
	if rec.Flag != "" && !hasFlag(flags, rec.Flag) {
		where := rec.Flag
		if n, ok := stationKO[rec.Flag]; ok {
			where = n
		}
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.CraftNeedRoom, where), Code: text.CodeNoRecipe})
		return
	}
	if rec.Tool != "" && !holdsItem(p, rec.Tool) {
		name := l.itemName(rec.Tool)
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.CraftNeedTool, name, text.EulReul(name)), Code: text.CodeNeedMat})
		return
	}
	for _, in := range rec.In {
		if qtyOfBag(p.Bag, in.ID) < in.Qty {
			name := l.itemName(in.ID)
			l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.CraftNeedMat, name, text.EulReul(name)), Code: text.CodeNeedMat})
			return
		}
	}
	net := l.itemWeight(rec.Out.ID) * rec.Out.Qty
	for _, in := range rec.In {
		net -= l.itemWeight(in.ID) * in.Qty
	}
	if l.carryWeight(p)+net > BagCap {
		l.sys(p.ConnID, text.CraftHeavy, text.CodeTooHeavy)
		return
	}
	bag := p.Bag
	for _, in := range rec.In {
		var ok bool
		bag, ok = yworld.TakeStack(bag, in.ID, in.Qty)
		if !ok {
			l.sys(p.ConnID, text.CraftNeedMat, text.CodeNeedMat)
			return
		}
	}
	p.Bag = yworld.AddStack(bag, rec.Out.ID, rec.Out.Qty)
	matched := rec.Flag != "" && hasFlag(flags, rec.Flag)
	l.world.roster[c.ConnID] = p
	line := skill.LineAt(rec.Gain, 0)
	if line == "" {
		name := l.itemName(rec.Out.ID)
		line = text.T(text.Default, text.CraftOK, name, text.EulReul(name))
	}
	l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: line})
	l.applyGain(&p, rec.Skill, matched)
	l.world.roster[c.ConnID] = p
}

func (l *Loop) listCrafts(p Player) {
	recs := l.craft.RecipesHere(l.roomFlags(p.RoomID))
	if len(recs) == 0 {
		l.sys(p.ConnID, text.CraftNone, text.CodeNoRecipe)
		return
	}
	parts := make([]string, 0, len(recs))
	for _, r := range recs {
		parts = append(parts, l.itemName(r.Out.ID))
	}
	l.sysf(p.ConnID, text.CraftWhat, strings.Join(parts, ", "))
}

func (l *Loop) quote(c Quote) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	slug, ok := l.roomMarket(p.RoomID)
	if !ok {
		l.sys(p.ConnID, text.MarketNone, text.CodeNoMarket)
		return
	}
	rows := l.markets.List(slug)
	if len(rows) == 0 {
		l.sys(p.ConnID, text.MarketNone, text.CodeNoMarket)
		return
	}
	l.sysf(p.ConnID, text.QuoteHeader, l.markets.MarketName(slug))
	for _, row := range rows {
		name := l.itemName(row.ID)
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.QuoteLine, name, row.Price, quoteFlavor(row.Stock, row.Target))})
	}
}

func (l *Loop) sell(c Sell) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	slug, ok := l.roomMarket(p.RoomID)
	if !ok {
		l.sys(p.ConnID, text.MarketNone, text.CodeNoMarket)
		return
	}
	qty := c.Qty
	if qty < 1 {
		qty = 1
	}
	itemID := l.resolveItemID(c.Query, p.Bag)
	if itemID == "" {
		l.sys(p.ConnID, text.SellMissing, text.CodeNotFound)
		return
	}
	have := qtyOfBag(p.Bag, itemID)
	if have == 0 {
		l.sys(p.ConnID, text.SellMissing, text.CodeNotFound)
		return
	}
	if have < qty {
		l.sys(p.ConnID, text.SellShort, text.CodeNeedMat)
		return
	}
	if _, listed := l.markets.Quote(slug, itemID); !listed {
		l.sys(p.ConnID, text.SellNone, text.CodeNoMarket)
		return
	}
	bag, ok := yworld.TakeStack(p.Bag, itemID, qty)
	if !ok {
		l.sys(p.ConnID, text.SellMissing, text.CodeNotFound)
		return
	}
	paid, ok := l.markets.Sell(slug, itemID, qty)
	if !ok {
		p.Bag = yworld.AddStack(bag, itemID, qty)
		l.world.roster[c.ConnID] = p
		l.sys(p.ConnID, text.SellNone, text.CodeNoMarket)
		return
	}
	p.Bag = bag
	p.Nyang += paid
	if p.Flags == nil {
		p.Flags = map[string]int{}
	}
	firstSale := p.Flags[yworld.FirstMarketSaleFlag] == 0
	if firstSale {
		p.Flags[yworld.FirstMarketSaleFlag] = 1
	}
	l.world.roster[c.ConnID] = p
	name := l.itemName(itemID)
	l.sysf(p.ConnID, text.SellOK, name, text.EulReul(name), paid)
	if firstSale {
		l.sysf(p.ConnID, text.FirstSaleRumor)
	}
	l.applyGain(&p, "haggle", true)
	l.world.roster[c.ConnID] = p
}

func (l *Loop) buy(c Buy) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	slug, ok := l.roomMarket(p.RoomID)
	if !ok {
		l.sys(p.ConnID, text.MarketNone, text.CodeNoMarket)
		return
	}
	qty := c.Qty
	if qty < 1 {
		qty = 1
	}
	itemID := l.catalogItemID(c.Query)
	if itemID == "" {
		itemID = strings.TrimSpace(c.Query)
	}
	unit, listed := l.markets.Quote(slug, itemID)
	if !listed {
		l.sys(p.ConnID, text.BuyNone, text.CodeNoMarket)
		return
	}
	if l.markets.Stock(slug, itemID) < qty {
		l.sys(p.ConnID, text.BuyEmpty, text.CodeNoStock)
		return
	}
	cost := unit * qty
	if p.Nyang < cost {
		l.sys(p.ConnID, text.BuyPoor, text.CodeTooPoor)
		return
	}
	if l.carryWeight(p)+l.itemWeight(itemID)*qty > BagCap {
		l.sys(p.ConnID, text.BuyHeavy, text.CodeTooHeavy)
		return
	}
	charged, ok := l.markets.Buy(slug, itemID, qty)
	if !ok {
		l.sys(p.ConnID, text.BuyEmpty, text.CodeNoStock)
		return
	}
	p.Nyang -= charged
	p.Bag = yworld.AddStack(p.Bag, itemID, qty)
	l.world.roster[c.ConnID] = p
	name := l.itemName(itemID)
	l.sysf(p.ConnID, text.BuyOK, name, text.EulReul(name), charged)
	l.applyGain(&p, "haggle", true)
	l.world.roster[c.ConnID] = p
}

func (l *Loop) maybeToll(p *Player, dest string) string {
	if l.catalog == nil || l.markets == nil {
		return ""
	}
	r, ok := l.catalog.Room(dest)
	if !ok || !hasFlag(r.Flags, "checkpoint") {
		return ""
	}
	if l.tradeQty(*p) < TradeBulk {
		return ""
	}
	rng := l.rng
	if rng == nil {
		rng = skill.DefaultRand
	}
	if rng() >= TollChance {
		return ""
	}
	if p.Nyang >= TollNyang {
		p.Nyang -= TollNyang
		return text.T(text.Default, text.TollPay, TollNyang)
	}
	id := l.firstTradeItem(*p)
	if id == "" {
		return ""
	}
	bag, ok := yworld.TakeStack(p.Bag, id, 1)
	if !ok {
		return ""
	}
	p.Bag = bag
	name := l.itemName(id)
	return text.T(text.Default, text.TollTake, name, text.EulReul(name))
}

func (l *Loop) tradeQty(p Player) int {
	n := 0
	for _, s := range p.Bag {
		if l.markets != nil && l.markets.Trades(s.ID) {
			n += s.Qty
		}
	}
	return n
}

func (l *Loop) firstTradeItem(p Player) string {
	for _, s := range p.Bag {
		if s.Qty > 0 && l.markets.Trades(s.ID) {
			return s.ID
		}
	}
	return ""
}

func (l *Loop) roomMarket(roomID string) (string, bool) {
	if l.catalog == nil || l.markets == nil {
		return "", false
	}
	r, ok := l.catalog.Room(roomID)
	if !ok || strings.TrimSpace(r.Market) == "" {
		return "", false
	}
	if !l.markets.HasMarket(r.Market) {
		return "", false
	}
	return r.Market, true
}

func (l *Loop) roomFlags(roomID string) []string {
	if l.catalog == nil {
		return nil
	}
	r, ok := l.catalog.Room(roomID)
	if !ok {
		return nil
	}
	return r.Flags
}

func (l *Loop) findRecipe(q string) (craft.Recipe, bool) {
	if l.craft == nil {
		return craft.Recipe{}, false
	}
	if r, ok := l.craft.LookupRecipe(q); ok {
		return r, true
	}
	if id := l.catalogItemID(q); id != "" {
		return l.craft.LookupRecipe(id)
	}
	return craft.Recipe{}, false
}

func (l *Loop) catalogItemID(q string) string {
	q = strings.TrimSpace(q)
	if q == "" || l.items == nil {
		return ""
	}
	it, ok := l.items.Find(q)
	if !ok {
		return ""
	}
	return it.ID
}

func (l *Loop) applyGain(p *Player, skillID string, matched bool) {
	if l.skills == nil || skillID == "" || p == nil {
		return
	}
	if _, ok := l.skills.Skill(skillID); !ok {
		return
	}
	sheet := l.skills.Bind(p.Skills, p.Stats)
	oldTitle := sheet.Title().Text(text.Default)
	rank := sheet.Rank(skillID)
	diff := skill.Difficulty(rank, matched)
	gained, next := sheet.TryGain(skillID, diff, l.rng)
	p.Skills = next.Ranks()
	p.Stats = next.Stats()
	if !gained {
		return
	}
	if sk, ok := l.skills.Skill(skillID); ok {
		if line := skill.LineAt(sk.Gain, next.Rank(sk.ID)); line != "" {
			l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: line})
		}
	}
	newTitle := next.Title().Text(text.Default)
	if newTitle != "" && newTitle != oldTitle {
		if ann := l.skills.Announce(next).Text(text.Default); ann != "" {
			l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: ann})
		}
	}
}

func quoteFlavor(stock, target int) string {
	switch {
	case stock >= target+3:
		return text.T(text.Default, text.QuoteHigh)
	case stock <= target-3:
		return text.T(text.Default, text.QuoteLow)
	default:
		return text.T(text.Default, text.QuoteMid)
	}
}

func holdsItem(p Player, id string) bool {
	if p.Equip.MainHand == id || p.Equip.Body == id {
		return true
	}
	return qtyOfBag(p.Bag, id) > 0
}

func qtyOfBag(bag []yworld.Stack, id string) int {
	for _, s := range bag {
		if s.ID == id {
			return s.Qty
		}
	}
	return 0
}

func hasFlag(flags []string, f string) bool {
	for _, x := range flags {
		if x == f {
			return true
		}
	}
	return false
}

// ParseNameQty splits "쑥 3" into name+qty. Qty defaults to 1.
func ParseNameQty(rest string) (string, int) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", 1
	}
	parts := strings.Fields(rest)
	if len(parts) >= 2 {
		if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil && n > 0 {
			return strings.Join(parts[:len(parts)-1], " "), n
		}
	}
	return rest, 1
}
