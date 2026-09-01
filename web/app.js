(() => {
  const $ = (id) => document.getElementById(id);
  const out = $("out");
  const cmd = $("cmd");
  const authErr = $("auth-err");
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const ws = new WebSocket(proto + "//" + location.host + "/ws");
  let authed = false;
  const dirs = {
    n: "north", north: "north", "북": "north",
    s: "south", south: "south", "남": "south",
    e: "east", east: "east", "동": "east",
    w: "west", west: "west", "서": "west",
    u: "up", up: "up", "위": "up",
    d: "down", down: "down", "아래": "down",
  };
  const dirLabel = { north: "북쪽", south: "남쪽", east: "동쪽", west: "서쪽", up: "위", down: "아래" };

  function append(text) {
    out.textContent += text + "\n";
    out.scrollTop = out.scrollHeight;
  }
  function send(type, payload, id) {
    ws.send(JSON.stringify({ v: 1, type, id: id || "", payload: payload || {} }));
  }
  function showErr(msg) {
    authErr.hidden = !msg;
    authErr.textContent = msg || "";
  }
  function splitCmd(line) {
    const s = line.trim();
    const i = s.search(/\s/);
    if (i < 0) return [s, ""];
    return [s.slice(0, i), s.slice(i).trim()];
  }
  function formatRoom(p) {
    const lines = [p.name || "", p.description || ""];
    const exits = p.exits || {};
    const parts = [];
    for (const d of ["north", "south", "east", "west", "up", "down"]) {
      if (exits[d]) parts.push((dirLabel[d] || d) + "(" + exits[d] + ")");
    }
    if (parts.length) lines.push("출구: " + parts.join(", "));
    if (p.who && p.who.length) lines.push("여기: " + p.who.join(", "));
    if (p.npcs && p.npcs.length) lines.push("사람: " + p.npcs.join(", "));
    if (p.objects && p.objects.length) lines.push("살펴볼 것: " + p.objects.join(", "));
    if (p.ground && p.ground.length) lines.push("바닥: " + p.ground.join(", "));
    return lines.filter(Boolean).join("\n");
  }
  function parseAndSend(line) {
    const [verb, rest] = splitCmd(line);
    if (!verb) return;
    const low = verb.toLowerCase();
    if (low === "say" || verb === "말") {
      if (!rest) return append("무슨 말인지 모르겠어요. 보다, 종료");
      return send("cmd.say", { text: rest });
    }
    if (low === "look" || low === "l" || verb === "보다" || verb === "살펴") {
      return rest ? send("cmd.look", { target: rest }) : send("cmd.look", {});
    }
    if (low === "talk" || verb === "대화" || verb === "말걸다") {
      if (!rest) return append("무슨 말인지 모르겠어요. 보다, 종료");
      return send("cmd.talk", { npc: rest });
    }
    if (low === "go" || verb === "가다") {
      const d = dirs[rest.toLowerCase()] || dirs[rest];
      if (!d) return append("무슨 말인지 모르겠어요. 보다, 종료");
      return send("cmd.move", { dir: d });
    }
    if (dirs[low] || dirs[verb]) {
      if (rest) return append("무슨 말인지 모르겠어요. 보다, 종료");
      return send("cmd.move", { dir: dirs[low] || dirs[verb] });
    }
    if (low === "quit" || verb === "종료") return send("cmd.quit", {});
    if (low === "skills" || verb === "숙련" || verb === "기술" || low === "inv" || verb === "소지" || verb === "가방") {
      return send("cmd.skills", {});
    }
    if (low === "get" || verb === "집다") return send("cmd.get", { item: rest });
    if (low === "drop" || verb === "놓다") return send("cmd.drop", { item: rest });
    if (low === "equip" || verb === "들다") return send("cmd.equip", { item: rest });
    if (low === "unequip" || verb === "벗다") return send("cmd.unequip", { slot: rest });
    if (low === "practice" || verb === "익히다") return send("cmd.practice", { skill: rest });
    if (low === "gather") return send("cmd.gather", { item: rest });
    if (low === "craft" || verb === "만들다") return send("cmd.craft", { item: rest });
    if (low === "sell" || verb === "팔다") {
      const bits = rest.split(/\s+/);
      const n = parseInt(bits[bits.length - 1], 10);
      if (bits.length > 1 && n > 0) return send("cmd.sell", { item: bits.slice(0, -1).join(" "), n });
      return send("cmd.sell", { item: rest });
    }
    if (low === "buy" || verb === "사다") return send("cmd.buy", { item: rest });
    if (low === "quote" || verb === "시세") return send("cmd.quote", {});
    append("무슨 말인지 모르겠어요. 보다, 종료");
  }

  ws.addEventListener("message", (ev) => {
    let f;
    try { f = JSON.parse(ev.data); } catch { return; }
    if (f.type === "auth.ok") {
      authed = true;
      $("auth").hidden = true;
      $("term").hidden = false;
      cmd.focus();
      return;
    }
    if (f.type === "auth.err") {
      showErr((f.payload && (f.payload.message || f.payload.code)) || "로그인에 실패했어요.");
      return;
    }
    if (f.type === "sys") {
      append((f.payload && f.payload.message) || "");
      return;
    }
    if (f.type === "text") {
      const p = f.payload || {};
      if (p.channel === "say") append("[말] " + p.from + ": " + p.text);
      else append(p.text || "");
      return;
    }
    if (f.type === "room") append(formatRoom(f.payload || {}));
  });
  ws.addEventListener("close", () => append("연결이 끊겼어요."));

  $("create").addEventListener("click", () => {
    showErr("");
    send("auth.create", { username: $("user").value, password: $("pass").value }, "c");
  });
  $("login").addEventListener("click", () => {
    showErr("");
    send("auth.login", { username: $("user").value, password: $("pass").value }, "l");
  });
  $("cmd-form").addEventListener("submit", (e) => {
    e.preventDefault();
    if (!authed) return;
    const line = cmd.value;
    cmd.value = "";
    if (!line.trim()) return;
    parseAndSend(line);
  });
})();
