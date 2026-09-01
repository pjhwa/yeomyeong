package world

import "testing"

func TestEmberPrereq(t *testing.T) {
	if EmberFlag != "ember" {
		t.Fatalf("ember=%s", EmberFlag)
	}
	if EmberPrereq(nil) || EmberPrereq(map[string]int{}) {
		t.Fatal("empty flags")
	}
	if EmberPrereq(map[string]int{ExaminedFlag("gangpo-pack"): 1}) {
		t.Fatal("pack alone")
	}
	if EmberPrereq(map[string]int{FirstMarketSaleFlag: 1}) {
		t.Fatal("sale alone")
	}
	if EmberPrereq(map[string]int{DawnScentFlag: 1, SmuggleSuccessCountFlag: 1}) {
		t.Fatal("scent alone")
	}
	if !EmberPrereq(map[string]int{ExaminedFlag("gangpo-pack"): 1, FirstMarketSaleFlag: 1}) {
		t.Fatal("pack+sale")
	}
	if !EmberPrereq(map[string]int{ExaminedFlag("gangpo-pack"): 1, DawnScentFlag: 1}) {
		t.Fatal("pack+scent")
	}
}

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
		TalkWhen: []TalkWhen{{Flag: "examined:test-paper", Line: Localized{KO: "신문을 봤군"}}},
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
	if len(in[0].TalkWhen) > 0 {
		in[0].TalkWhen[0].Flag = "hacked"
	}
	again := npcs.InRoom("test:start")
	if again[0].Aliases[0] != "훈장" {
		t.Fatal("must clone aliases")
	}
	if again[0].TalkWhen[0].Flag != "examined:test-paper" {
		t.Fatal("must clone talk.when")
	}
	if _, err := NewNPCs([]NPC{{ID: "a"}, {ID: "a"}}); err == nil {
		t.Fatal("dup npc")
	}
	if _, err := NewObjects([]Object{{ID: "o"}, {ID: "o"}}); err == nil {
		t.Fatal("dup object")
	}
}
