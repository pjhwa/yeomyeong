package e2e

import (
	"strings"
	"testing"
)

// Two-client simultaneous talk smoke (issue #13, PLAN.md §5 M0).
// In-process: engine.Loop + net.NewServer + net.NewWS + persist.Memory.

func TestTwoTelnetClientsTalk(t *testing.T) {
	h := startHarness(t)

	a := h.dialTelnet(t, "telnet-A")
	b := h.dialTelnet(t, "telnet-B")
	a.createUser(t, "갑을", "password1")
	b.createUser(t, "병정", "password2")
	h.waitRoster(t, 2)

	a.send(t, "say 안녕")
	want := "[말] 갑을: 안녕"
	a.expectLine(t, want)
	b.expectLine(t, want)
}

func TestTelnetAndWebsocketTalk(t *testing.T) {
	h := startHarness(t)

	tn := h.dialTelnet(t, "telnet-A")
	tn.createUser(t, "갑을", "password1")

	ws := h.dialWS(t, "ws-B")
	ws.createUser(t, "병정", "password2", "c1")
	h.waitRoster(t, 2)

	ws.send(t, typeSay, "s1", map[string]string{"text": "안녕"})
	tn.expectLine(t, "[말] 병정: 안녕")
	ws.expectSay(t, "병정", "안녕")

	tn.send(t, "say 반갑소")
	tn.expectLine(t, "[말] 갑을: 반갑소")
	ws.expectSay(t, "갑을", "반갑소")
}

func TestFailurePrintsTranscript(t *testing.T) {
	tr := &transcript{}
	tr.add("telnet-A SEND %q", "say 안녕")
	tr.add("telnet-B RECV %q", ">")
	got := formatFail(tr, `telnet-B did not receive "[말] 갑을: 안녕"`)
	if !strings.Contains(got, "--- reproduction transcript ---") {
		t.Fatal("failure must include transcript header")
	}
	if !strings.Contains(got, `telnet-A SEND "say 안녕"`) {
		t.Fatalf("failure must include sent line, got:\n%s", got)
	}
	if !strings.Contains(got, `telnet-B RECV ">"`) {
		t.Fatalf("failure must include received line, got:\n%s", got)
	}
	if strings.Count(got, "not equal") > 0 && !strings.Contains(got, "transcript") {
		t.Fatal("failure must not be a bare not-equal")
	}
}
