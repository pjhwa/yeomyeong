# M3 slice notes — 여명 / YEOMYEONG

This is a **vertical slice**, not PLAN.md §3.4 complete. The README M3
checkbox stays open. Binding rulings: D-042, D-043, D-044.

## What shipped

Gather a YAML node → consume a YAML recipe → sell at a stall whose
price moves with stock. Two markets so the same good is cheap in one
place and dear in the other. Walking bulk through a checkpoint may
skim a toll. No combat.

## Files

**New packages**

- `internal/craft` — immutable nodes + recipes; loop-owned `Stock`
- `internal/economy` — live price book (`Quote` / `Sell` / `Buy` / `Tick`)

**Engine / persist / net**

- `internal/engine/livelihood.go` — gather, craft, quote, sell, buy, toll
- `internal/engine/command.go` — `Gather` `Craft` `Sell` `Buy` `Quote`
- `internal/engine/loop.go` — tick regen + market drift; `WithCraft` / `WithMarkets`
- `internal/world/sheet.go` — `Nyang`
- `internal/persist/sheet.go`, `migrations/003_nyang.sql`
- `internal/net/telnet.go`, `internal/net/ws.go` — Korean verbs + WS types
- `internal/content/livelihood.go` — boot loader
- `cmd/server/main.go` — wires craft + markets into the loop

**YAML**

- `content/skills/m2.yaml` — `forage` (채집), titles 채집꾼 / 보부상 / 못 벼리는 사람
- `content/items/m3-goods.yaml` — 쑥, 쇠조각, 한지, 쇠못, 쑥떡, 전단, 약첩
- `content/craft/nodes.yaml`, `content/craft/recipes.yaml`
- `content/economy/markets.yaml` — `dalbitgol`, `solgol`
- `content/zones/solgol/rooms.yaml` — 고갯길·장터·비탈·솔밭
- `content/zones/dalbitgol/rooms.yaml` — 정거장 east → 솔골, flag `checkpoint`

**Docs**

- `docs/DECISIONS.md` D-042–D-044
- `docs/CONTENT-SCHEMA.md`, `docs/SKILL-TABLE.md`, `docs/EVENT-BUS.md`, `docs/WIRE-PROTOCOL.md`
- `docs/COMMISSIONER.md` §9 play recipe (slice, not a gate close)

## Playtest

```bash
go run ./cmd/server
nc localhost 4001
```

한글 입력은 `nc`. Full walk is in `docs/COMMISSIONER.md` §9.1.

Short loop:

1. Spawn `달빛골 마을문`. `n n w n` → 시내. `캐다` → 쇠조각.
2. `n n` → 방앗간. `줍다` → 한지. `n` → 서쪽 밭. `캐다` → 쑥.
3. Walk to 대장간. `집다 쇠망치` `들다 쇠망치` `만들다 쇠못` (쇠조각×2).
4. 장터: `시세` then `팔다 쑥`. Quote should ease after the sale.
5. Station `e` `e` → 솔골 장터. `시세` — 쑥 dearer than 달빛골. `팔다 쑥`.
6. `소지` shows `주머니: N냥`. `숙련` may show 채집꾼 / 보부상 / 못 벼리는 사람.

| Verb | Where | Result |
|---|---|---|
| `캐다` / `줍다` | 보리밭, 서쪽 밭, 시내, 방앗간, 솔골 비탈/솔밭 | item into bag |
| `만들다 쇠못` | 대장간 + 쇠망치 + 쇠조각×2 | 쇠못 |
| `만들다 쑥떡` | 주막 + 식칼 + 쑥 | 쑥떡 |
| `만들다 전단` | 인쇄소 + 활자 막대 + 한지 | 전단 |
| `만들다 약첩` | 사당(clinic) + 쑥×2 | 약첩 |
| `시세` `팔다` `사다` | 달빛골 장터, 솔골 장터 | 냥 |

`두드리다` is still practice only. It does not eat 쇠조각.

Carry **four or more** market goods into 정거장 or 솔골 고갯길: ~35% chance
of 2냥 toll, or one unit left behind if the purse is empty.

## Remaining M3 gaps (not this slice)

- Player shops / housing / offline stalls
- Smuggling, double-bottom carts, Dawn Circle trust from contraband
- Three-axis reputation (새벽회 / 제국 수배 / 상계)
- Seasonal/weather/event shocks to prices
- Player-to-player trade
- Split gather skills (채광·벌목·약초·낚시·수렵)
- Equipment wear, repair sink, masterwork-only-from-players
- Inflation metrics / QA dashboards
- Hireable escorts, detours via 독도법

M4 narrative can start; livelihood should keep running beside it. Do not
advertise M3 as done until shops or a second trade loop (smuggling) exist.
