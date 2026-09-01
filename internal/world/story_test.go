package world

import "testing"

func TestTalkFlagAndMatch(t *testing.T) {
	if TalkFlag("cheongram") != "cheongram_talked" {
		t.Fatalf("flag=%s", TalkFlag("cheongram"))
	}
	if ExaminedFlag("gangpo-pack") != "examined:gangpo-pack" {
		t.Fatalf("examined=%s", ExaminedFlag("gangpo-pack"))
	}
	npcs, err := NewNPCs([]NPC{{
		ID: "tutor", Room: "test:start", Name: Localized{KO: "훈장"},
		Aliases: []string{"훈장", "선생"}, TalkFirst: Localized{KO: "안녕"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := npcs.FindInRoom("test:start", "선생"); !ok {
		t.Fatal("alias")
	}
	if _, ok := npcs.Find("훈장"); !ok {
		t.Fatal("find any")
	}
	in := npcs.InRoom("test:start")
	in[0].Aliases[0] = "hacked"
	again := npcs.InRoom("test:start")
	if again[0].Aliases[0] != "훈장" {
		t.Fatal("must clone aliases")
	}
	if _, err := NewNPCs([]NPC{{ID: "a"}, {ID: "a"}}); err == nil {
		t.Fatal("dup npc")
	}
	if _, err := NewObjects([]Object{{ID: "o"}, {ID: "o"}}); err == nil {
		t.Fatal("dup object")
	}
}
