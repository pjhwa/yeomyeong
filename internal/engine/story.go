package engine

import (
	"strings"

	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
)

const (
	cafeHandID             = "cafe-hand"
	cheongramID            = "cheongram"
	leafletID              = "leaflet"
	cafeBaekyaRoom         = "dalbitgol:cafe-baekya"
	packingShedRoom        = "dalbitgol:packing-shed"
	cheongramEmberRiskLine = "이제 그쪽에서 자네를 부른다네. 우물길이나 다방 문을 한 번 더 보게. 말은 아끼게."
	cafeHandEmberAck       = "카운터 뒤에서 누가 사라진 보부상 이야기를 하고 있어요. 점원이 잔을 닦다 말고 고개를 끄덕여요."
	emberDropAmbient       = "전단을 내려놓자 누가 작은 소리로 말해요. 우물길이나 다방 문을 한 번 더 보라는 투예요."
	emberHideAmbient       = "검문 너머로 누가 우물길 쪽을 턱짓해요. 다방 문도 한 번 더 보라는 투예요."
)

func (l *Loop) talk(c Talk) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	q := strings.TrimSpace(c.NPC)
	if q == "" {
		l.sys(p.ConnID, text.TalkMissing, text.CodeNotFound)
		return
	}
	npc, ok := l.findNPCHere(p.RoomID, q)
	if !ok {
		l.sys(p.ConnID, text.TalkMissing, text.CodeNotFound)
		return
	}
	if p.Flags == nil {
		p.Flags = map[string]int{}
	}
	key := yworld.TalkFlag(npc.ID)
	body := l.talkBody(npc, &p)
	if p.Flags[key] == 0 {
		p.Flags[key] = 1
	}
	l.world.roster[c.ConnID] = p
	l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: body})
}

func (l *Loop) talkBody(npc yworld.NPC, p *Player) string {
	if npc.ID == cheongramID && p.Flags[yworld.EmberFlag] == 0 &&
		yworld.EmberPrereq(p.Flags) &&
		p.Flags[yworld.DawnScentFlag] > 0 &&
		p.Flags[yworld.SmuggleSuccessCountFlag] >= 1 {
		p.Flags[yworld.EmberFlag] = 1
		return cheongramEmberRiskLine
	}
	if npc.ID == cafeHandID && p.Flags[yworld.EmberFlag] == 0 && !yworld.EmberPrereq(p.Flags) {
		return ungatedTalk(npc, p.Flags)
	}
	body := pickTalk(npc, p.Flags)
	if npc.ID == cafeHandID && maybeGrantEmber(p) {
		if isUngatedTalk(npc, body) {
			return cafeHandEmberAck
		}
	}
	return body
}

func ungatedTalk(npc yworld.NPC, flags map[string]int) string {
	if flags[yworld.TalkFlag(npc.ID)] > 0 {
		if second := strings.TrimSpace(npc.TalkSecond.Text(text.Default)); second != "" {
			return second
		}
	}
	return strings.TrimSpace(npc.TalkFirst.Text(text.Default))
}

func isUngatedTalk(npc yworld.NPC, body string) bool {
	return body == strings.TrimSpace(npc.TalkFirst.Text(text.Default)) ||
		body == strings.TrimSpace(npc.TalkSecond.Text(text.Default))
}

func maybeGrantEmber(p *Player) bool {
	if p == nil {
		return false
	}
	if p.Flags == nil {
		p.Flags = map[string]int{}
	}
	if p.Flags[yworld.EmberFlag] > 0 || !yworld.EmberPrereq(p.Flags) {
		return false
	}
	p.Flags[yworld.EmberFlag] = 1
	return true
}

func emberDropRoom(room string) bool {
	return room == packingShedRoom || room == cafeBaekyaRoom
}

// pickTalk returns the first talk.when line whose flag is >0, else first/second (D-046).
func pickTalk(npc yworld.NPC, flags map[string]int) string {
	for _, w := range npc.TalkWhen {
		if flags[w.Flag] > 0 {
			if body := strings.TrimSpace(w.Line.Text(text.Default)); body != "" {
				return body
			}
		}
	}
	if flags[yworld.TalkFlag(npc.ID)] > 0 {
		if second := strings.TrimSpace(npc.TalkSecond.Text(text.Default)); second != "" {
			return second
		}
	}
	return strings.TrimSpace(npc.TalkFirst.Text(text.Default))
}

// pickObjectDesc returns the first object.when line whose flag is >0, else description.
func pickObjectDesc(obj yworld.Object, flags map[string]int) string {
	for _, w := range obj.DescWhen {
		if flags[w.Flag] > 0 {
			if body := strings.TrimSpace(w.Line.Text(text.Default)); body != "" {
				return body
			}
		}
	}
	return strings.TrimSpace(obj.Description.Text(text.Default))
}

// pickRoomDesc returns the first room.when line whose flag is >0, else description.
func pickRoomDesc(r yworld.Room, flags map[string]int) string {
	for _, w := range r.DescWhen {
		if flags[w.Flag] > 0 {
			if body := strings.TrimSpace(w.Line.Text(text.Default)); body != "" {
				return body
			}
		}
	}
	return strings.TrimSpace(r.Description.Text(text.Default))
}

func (l *Loop) examine(p Player, q string) {
	if npc, ok := l.findNPCHere(p.RoomID, q); ok {
		body := strings.TrimSpace(npc.Look.Text(text.Default))
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: body})
		return
	}
	if obj, ok := l.findObjectHere(p.RoomID, q); ok {
		body := pickObjectDesc(obj, p.Flags)
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: body})
		reaction := strings.TrimSpace(obj.AfterExamine.Text(text.Default))
		if reaction == "" {
			return
		}
		if p.Flags == nil {
			p.Flags = map[string]int{}
		}
		key := yworld.ExaminedFlag(obj.ID)
		if p.Flags[key] > 0 {
			return
		}
		p.Flags[key] = 1
		l.world.roster[p.ConnID] = p
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: reaction})
		return
	}
	if l.npcs != nil {
		if _, ok := l.npcs.Find(q); ok {
			l.sys(p.ConnID, text.TalkMissing, text.CodeNotFound)
			return
		}
	}
	l.sys(p.ConnID, text.ExamineMissing, text.CodeNotFound)
}

func (l *Loop) findNPCHere(room, q string) (yworld.NPC, bool) {
	if l.npcs == nil {
		return yworld.NPC{}, false
	}
	return l.npcs.FindInRoom(room, q)
}

func (l *Loop) findObjectHere(room, q string) (yworld.Object, bool) {
	if l.objects == nil {
		return yworld.Object{}, false
	}
	return l.objects.FindInRoom(room, q)
}
