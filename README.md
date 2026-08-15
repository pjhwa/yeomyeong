<div align="center">

# 여명 · YEOMYEONG

### *Liberate a nation. You may never draw a sword.*

A modern text-based MMO where progression comes from **what you actually do** —
not from grinding mobs. No levels. No classes. A living economy.
NPCs powered by LLMs that **remember you**.

Built in Go. Inspired by [DikuMUD](https://dikumud.com) (1990), sharing none of its code.

[![Build](https://img.shields.io/github/actions/workflow/status/pjhwa/yeomyeong/ci.yml?branch=main)](https://github.com/pjhwa/yeomyeong/actions)
[![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Roadmap](https://img.shields.io/badge/milestone-M0-lightgrey)](#roadmap)

**[한국어 README](README.ko.md)** · **[Play the demo](#)** · **[Discord](#)**

</div>

---

> **⚠️ Status: early development (M0).** The world described below is the design
> target, not yet the shipped state. Follow the [roadmap](#roadmap) for what's live today.

<!-- TODO(M1): replace with asciinema demo GIF — 30s: market → haggle → checkpoint → deliver -->
<div align="center">
  <img src="docs/assets/demo.gif" alt="Terminal demo" width="720">
  <br><em>30 seconds in 한벽성 — haggling at the market, then slipping past a checkpoint.</em>
</div>

## Why another MUD?

Text MUDs died for a reason: they became spreadsheets with prose. You killed
rats until a number went up. We think the format deserves better.

- **No levels, no classes.** 40+ skills grow from *use*. Swing a hammer, get better
  at smithing. Haggle, get better at haggling. A hard cap on total skill means
  no one does everything — so players actually need each other.
- **You can finish the story without fighting.** Every major obstacle has at least
  three solutions: force, craft, or words. Smuggling ink for an underground press
  advances the main narrative just as surely as a raid does.
- **The world reacts.** Prices move with supply, weather, and war. NPCs keep
  schedules. A server-wide *crackdown level* rises when players get bold, and
  patrols thicken in response.
- **NPCs that remember.** A three-tier dialogue system (scripted → open-weight
  model → premium API) gives key characters persistent memory of your past
  conversations, grounded in real game state and gated behind narrative progress.

## The setting

A 1920s-flavored alternate history. The peninsular nation of **달내 (Dalnae)**
fell to the **Iron Empire** fifteen years ago. The empire drives iron stakes into
the land's ley lines, draining the vitality of the earth — and of the people —
back to its capital.

You start as nobody in a border village. A missing courier, a newspaper with a
strange typesetting error, and a retired schoolmaster who knows more than he
admits. The secret society called **새벽회 (the Dawn Circle)** is looking for
people who notice things.

> 좁은 흙길 양옆으로 좌판이 늘어섰다. 말린 명태와 삼베 냄새가 뒤섞이고,
> 엿장수의 가위질 소리가 장단을 맞춘다. 장터 어귀의 게시판에는 총독부
> 고시문이 붙어 있는데, 누군가 모퉁이를 찢어 갔다.

*All nations, organizations, and characters are fictional. This project draws on
the spirit of resistance-era history without depicting real people or groups.*

## Try it in 60 seconds

```bash
git clone https://github.com/pjhwa/yeomyeong.git
cd yeomyeong
docker compose up
```

Then connect with either:

```bash
telnet localhost 4000        # classic
open http://localhost:8080   # web client
```

## Architecture

The concurrency model is the one design decision everything else hangs from:
**a single goroutine owns all world state.**

```
[client] ──WebSocket/Telnet──> [connection goroutine, one per player]
                                        │ commands (channel)
                                        ▼
                          ┌──────────────────────────────┐
                          │   GAME LOOP  (exactly one)   │
                          │   100ms tick                 │
                          │   • sole writer of world     │
                          │   • combat rounds, NPC AI,   │
                          │     prices, weather, heat    │
                          └──────────────────────────────┘
                                        │ events (channel)
                                        ▼
                        [per-connection output buffers → clients]
```

No mutexes on world state. No data races by construction. I/O never blocks the
simulation; database writes and LLM calls are delegated to async workers.

| Layer | Choice |
|---|---|
| Server | Go 1.24+ |
| Transport | WebSocket (primary) + Telnet (compat) |
| Client | TypeScript + React |
| Storage | PostgreSQL + in-memory world, periodic snapshots |
| Content | YAML, hot-reloadable — **zero hardcoded rooms** |
| Scripting | Lua (sandboxed) |
| Dialogue | Pluggable `DialogueProvider` — open-weight local + premium API |

## Roadmap

- [ ] **M0** — Server skeleton, WS/Telnet, accounts
- [ ] **M1** — YAML world loader, movement, first village (40 rooms)
- [ ] **M2** — Skill system: growth curves, total cap, equipment
- [ ] **M3** — Living economy: gathering → crafting → regional trade
- [ ] **M4** — Act I narrative, foreshadowing ledger, first AI-driven NPC
- [ ] **M5** — Cells (guilds), reputation, smuggling, the weekly in-game newspaper
- [ ] **M6** — Web client, balance pass, 500 concurrent load target

## Contributing

Writers are as welcome as engineers here — a well-written room is a feature.

Start with [`good first issue`](../../issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22),
read [CONTRIBUTING.md](CONTRIBUTING.md), and skim `docs/CONTENT-STYLE.md`
before writing prose.

## License & credits

MIT — see [LICENSE](LICENSE).

Inspired by **DikuMUD** (University of Copenhagen, 1990), which proved that text
alone could hold a world. No DikuMUD code is used, copied, or derived from in
this project; it is a clean-room reimagining of the idea.
