package engine

import (
	"strings"

	"github.com/pjhwa/yeomyeong/internal/text"
	yworld "github.com/pjhwa/yeomyeong/internal/world"
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
	body := pickTalk(npc, p.Flags)
	if p.Flags[key] == 0 {
		p.Flags[key] = 1
		l.world.roster[c.ConnID] = p
	}
	l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: body})
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

func (l *Loop) examine(p Player, q string) {
	if npc, ok := l.findNPCHere(p.RoomID, q); ok {
		body := strings.TrimSpace(npc.Look.Text(text.Default))
		l.emit(Text{ConnID: p.ConnID, Channel: ChannelSys, Body: body})
		return
	}
	if obj, ok := l.findObjectHere(p.RoomID, q); ok {
		body := strings.TrimSpace(obj.Description.Text(text.Default))
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
