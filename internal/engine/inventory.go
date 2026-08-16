package engine

import (
	"sort"
	"strconv"
	"strings"

	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

func (l *Loop) get(c Get) {
	p, ok := l.world.roster[c.ConnID]
	if !ok || c.ItemID == "" {
		return
	}
	room := p.RoomID
	itemID := l.resolveItemID(c.ItemID, l.world.ground[room])
	piles, ok := yworld.TakeStack(l.world.ground[room], itemID, 1)
	if !ok {
		l.sys(p.ConnID, text.GetMissing, text.CodeNotFound)
		return
	}
	if l.carryWeight(p)+l.itemWeight(itemID) > BagCap {
		l.world.ground[room] = yworld.AddStack(l.world.ground[room], itemID, 1)
		l.sys(p.ConnID, text.GetHeavy, text.CodeTooHeavy)
		return
	}
	if len(piles) == 0 {
		delete(l.world.ground, room)
	} else {
		l.world.ground[room] = piles
	}
	p.Bag = yworld.AddStack(p.Bag, itemID, 1)
	l.world.roster[c.ConnID] = p
	l.sysf(p.ConnID, text.GetOK, l.itemName(itemID))
}

func (l *Loop) dropItem(c DropItem) {
	p, ok := l.world.roster[c.ConnID]
	if !ok || c.ItemID == "" {
		return
	}
	itemID := l.resolveItemID(c.ItemID, p.Bag)
	bag, ok := yworld.TakeStack(p.Bag, itemID, 1)
	if !ok {
		l.sys(p.ConnID, text.DropMissing, text.CodeNotFound)
		return
	}
	p.Bag = bag
	l.world.roster[c.ConnID] = p
	l.world.ground[p.RoomID] = yworld.AddStack(l.world.ground[p.RoomID], itemID, 1)
	l.sysf(p.ConnID, text.DropOK, l.itemName(itemID))
}

func (l *Loop) equip(c Equip) {
	p, ok := l.world.roster[c.ConnID]
	if !ok || c.ItemID == "" {
		return
	}
	itemID := l.resolveItemID(c.ItemID, p.Bag)
	it, ok := l.item(itemID)
	if !ok || it.Slot == yworld.SlotNone || it.Slot == "" {
		code, key := text.CodeNotWearable, text.EquipNotWear
		if !hasStack(p.Bag, itemID) {
			code, key = text.CodeNotFound, text.EquipMissing
		}
		l.sys(p.ConnID, key, code)
		return
	}
	bag, ok := yworld.TakeStack(p.Bag, itemID, 1)
	if !ok {
		l.sys(p.ConnID, text.EquipMissing, text.CodeNotFound)
		return
	}
	p.Bag = bag
	if prev := slotGet(p.Equip, it.Slot); prev != "" {
		p.Bag = yworld.AddStack(p.Bag, prev, 1)
	}
	p.Equip = slotSet(p.Equip, it.Slot, itemID)
	l.world.roster[c.ConnID] = p
	l.sysf(p.ConnID, text.EquipOK, l.itemName(itemID))
}

func (l *Loop) unequip(c Unequip) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	id := slotGet(p.Equip, canonicalSlot(c.Slot))
	if id == "" {
		l.sys(p.ConnID, text.UnequipEmpty, text.CodeEmptySlot)
		return
	}
	p.Equip = slotSet(p.Equip, canonicalSlot(c.Slot), "")
	p.Bag = yworld.AddStack(p.Bag, id, 1)
	l.world.roster[c.ConnID] = p
	l.sysf(p.ConnID, text.UnequipOK, l.itemName(id))
}

func (l *Loop) sheet(c Sheet) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	if title := l.playerTitle(p); title != "" {
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.SheetTitle, title)})
	}
	l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.SheetInv, joinStacks(p.Bag, l.itemName))})
	l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.SheetEquip, l.formatEquip(p.Equip))})
	l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.SheetSkills, joinInts(p.Skills))})
	l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: text.T(text.Default, text.SheetStats, joinInts(p.Stats))})
}

func (l *Loop) playerTitle(p Player) string {
	if l.skills == nil {
		return ""
	}
	t := l.skills.Bind(p.Skills, p.Stats).Title().Text(text.Default)
	if t == "" {
		return text.T(text.Default, text.SheetNone)
	}
	return t
}

func (l *Loop) resolveItemID(query string, piles []yworld.Stack) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	for _, s := range piles {
		if s.ID == q {
			return s.ID
		}
	}
	for _, s := range piles {
		if l.itemName(s.ID) == q {
			return s.ID
		}
	}
	return q
}

func canonicalSlot(slot string) string {
	s := strings.TrimSpace(slot)
	switch strings.ToLower(s) {
	case yworld.SlotMainHand, "주손":
		return yworld.SlotMainHand
	case yworld.SlotBody, "몸":
		return yworld.SlotBody
	}
	return s
}

func (l *Loop) item(id string) (yworld.Item, bool) {
	if l.items == nil {
		return yworld.Item{}, false
	}
	return l.items.Get(id)
}

func (l *Loop) itemName(id string) string {
	if it, ok := l.item(id); ok {
		if n := strings.TrimSpace(it.Name.Text(text.Default)); n != "" {
			return n
		}
	}
	return id
}

func (l *Loop) itemWeight(id string) int {
	if it, ok := l.item(id); ok {
		return it.Weight
	}
	return 1
}

func (l *Loop) carryWeight(p Player) int {
	n := 0
	for _, s := range p.Bag {
		n += l.itemWeight(s.ID) * s.Qty
	}
	if p.Equip.MainHand != "" {
		n += l.itemWeight(p.Equip.MainHand)
	}
	if p.Equip.Body != "" {
		n += l.itemWeight(p.Equip.Body)
	}
	return n
}

func (l *Loop) groundNames(roomID string) []string {
	var names []string
	for _, s := range l.world.ground[roomID] {
		n := l.itemName(s.ID)
		for i := 0; i < s.Qty; i++ {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func (l *Loop) formatEquip(eq yworld.Equipment) string {
	hand, body := text.T(text.Default, text.SheetNone), text.T(text.Default, text.SheetNone)
	if eq.MainHand != "" {
		hand = l.itemName(eq.MainHand)
	}
	if eq.Body != "" {
		body = l.itemName(eq.Body)
	}
	return text.T(text.Default, text.SheetEquipSlots, hand, body)
}

func (l *Loop) sys(id ConnID, key, code string) {
	l.emit(Text{ConnID: id, Channel: ChannelSys, Body: text.T(text.Default, key), Code: code})
}

func (l *Loop) sysf(id ConnID, key string, args ...any) {
	l.emit(Text{ConnID: id, Channel: ChannelSys, Body: text.T(text.Default, key, args...)})
}

func slotGet(eq yworld.Equipment, slot string) string {
	switch strings.TrimSpace(slot) {
	case yworld.SlotMainHand:
		return eq.MainHand
	case yworld.SlotBody:
		return eq.Body
	}
	return ""
}

func slotSet(eq yworld.Equipment, slot, id string) yworld.Equipment {
	switch strings.TrimSpace(slot) {
	case yworld.SlotMainHand:
		eq.MainHand = id
	case yworld.SlotBody:
		eq.Body = id
	}
	return eq
}

func joinStacks(bag []yworld.Stack, name func(string) string) string {
	if len(bag) == 0 {
		return text.T(text.Default, text.SheetNone)
	}
	parts := make([]string, 0, len(bag))
	for _, s := range bag {
		n := name(s.ID)
		if s.Qty > 1 {
			n += "×" + strconv.Itoa(s.Qty)
		}
		parts = append(parts, n)
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func joinInts(m map[string]int) string {
	if len(m) == 0 {
		return text.T(text.Default, text.SheetNone)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + " " + strconv.Itoa(m[k])
	}
	return strings.Join(parts, ", ")
}

func hasStack(bag []yworld.Stack, id string) bool {
	for _, s := range bag {
		if s.ID == id && s.Qty > 0 {
			return true
		}
	}
	return false
}
