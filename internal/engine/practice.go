package engine

import (
	"github.com/pjhwa/yeomyeong/internal/skill"
	"github.com/pjhwa/yeomyeong/internal/text"
)

func (l *Loop) practice(c Practice) {
	p, ok := l.world.roster[c.ConnID]
	if !ok {
		return
	}
	if l.skills == nil {
		l.sysf(p.ConnID, text.PracticeUnknown)
		return
	}
	sk, ok := l.skills.Lookup(c.SkillID)
	if !ok {
		l.sysf(p.ConnID, text.PracticeUnknown)
		return
	}
	sheet := l.skills.Bind(p.Skills, p.Stats)
	rank := sheet.Rank(sk.ID)
	diff := skill.Difficulty(rank, l.practiceMatched(p, sk))
	gained, next := sheet.TryGain(sk.ID, diff, l.rng)
	p.Skills = next.Ranks()
	p.Stats = next.Stats()
	l.world.roster[c.ConnID] = p
	if gained {
		l.sysf(p.ConnID, text.PracticeGain, sk.Name.Text(text.Default), next.Rank(sk.ID))
		return
	}
	l.sysf(p.ConnID, text.PracticeMiss)
}

func (l *Loop) practiceMatched(p Player, sk skill.Skill) bool {
	if sk.PracticeFlag != "" && l.roomHasFlag(p.RoomID, sk.PracticeFlag) {
		return true
	}
	for _, id := range heldItemIDs(p) {
		if sk.PracticeItem != "" && id == sk.PracticeItem {
			return true
		}
		it, ok := l.item(id)
		if !ok {
			continue
		}
		for _, tagged := range it.Skills {
			if tagged == sk.ID {
				return true
			}
		}
	}
	return false
}

func (l *Loop) roomHasFlag(roomID, flag string) bool {
	if l.catalog == nil || flag == "" {
		return false
	}
	r, ok := l.catalog.Room(roomID)
	if !ok {
		return false
	}
	for _, f := range r.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

func heldItemIDs(p Player) []string {
	ids := make([]string, 0, len(p.Bag)+2)
	for _, s := range p.Bag {
		if s.Qty > 0 {
			ids = append(ids, s.ID)
		}
	}
	if p.Equip.MainHand != "" {
		ids = append(ids, p.Equip.MainHand)
	}
	if p.Equip.Body != "" {
		ids = append(ids, p.Equip.Body)
	}
	return ids
}
