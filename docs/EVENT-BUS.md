# EVENT-BUS

ENGINE ↔ GAMEPLAY / ECONOMY / AINPC.  
M1 adds rooms, look, and move. Later systems subscribe to the same bus;
they do not grow their own world writers.

Related: [WIRE-PROTOCOL.md](WIRE-PROTOCOL.md), PLAN.md §4.2.

## Single-writer

The game-loop goroutine is the **only** goroutine that:

- inserts / removes players on the in-memory roster
- reads the roster to decide who receives a `say`
- writes `Player.RoomID` (position)
- writes skill ranks, stats, bag, equipment, room ground piles, nyang, player flags
- writes market stock and gather-node remaining
- changes any future world field (heat, …)

The **room catalog** loaded from YAML is immutable after boot. The loop
reads it; it does not mutate room definitions. Positions are world state.

Connection goroutines, persist workers, and LLM workers **never** take a
mutex on world data — because world data has no mutex. They send a
`Command` in and receive `Event`s out.

```
conn goroutine ──Command──▶ game loop ──Event──▶ per-conn outbound buffer
                     ▲
                     └── persist / auth happens BEFORE EnterWorld
```

## Commands (M0–M3)

Enqueued by adapters (`internal/net`). Value types, no shared pointers into
the world.

| Name | Fields | Who enqueues | Loop effect |
|---|---|---|---|
| `EnterWorld` | `ConnID`, `AccountID`, `Username`, `Session` | net, **after** persist auth succeeds | Insert roster entry in spawn `dalbitgol:gate` (D-028); emit seated `Sys`; emit `Room` card to the newcomer |
| `Say` | `ConnID`, `Text` | net, only if that conn is in the roster | Emit `Text{channel:say}` to every roster conn **in the same room** (M1: say is no longer global) |
| `Look` | `ConnID`, `Target?` | net | Empty target: emit `Room` card (NPCs, scenery names, ground). Set: examine NPC/object, emit Description `Text`; object `after_examine` emits a second sys `Text` once (`Flags["examined:<id>"]`) |
| `Talk` | `ConnID`, `NPC` | net | Scripted YAML talk. First matching `talk.when` whose sheet flag is >0, else first/second (D-046). First talk sets `Sheet.Flags["{id}_talked"]` even if a when-line was used. `cafe-hand` with EmberPrereq sets `ember=1`; 청람 risk (EmberPrereq + `dawn_scent` + `smuggle_success_count>=1`) overrides the line and sets `ember` (D-049) |
| `Move` | `ConnID`, `Dir` | net | If exit exists, set `RoomID`, emit `Room` to mover. Else emit `Sys` `no_exit` |
| `Practice` | `ConnID`, `SkillID` | net | GAMEPLAY roll; loop writes new ranks/stats; emit Text |
| `Get` | `ConnID`, `ItemID` | net | Move one ground stack into bag if weight allows |
| `Drop` | `ConnID`, `ItemID` | net | Move one bag stack to ground. Leaflet drop in packing-shed or café-baekya with EmberPrereq sets `ember=1` after the normal drop (D-049) |
| `Equip` | `ConnID`, `ItemID` | net | Bag → slot if `slot` is wearable |
| `Unequip` | `ConnID`, `Slot` | net | Slot → bag |
| `Sheet` | `ConnID` | net | Emit skills/title/stats/inv/purse text |
| `Gather` | `ConnID`, `Query`, `Skill` | net | Harvest a YAML node; write bag + forage rank |
| `Craft` | `ConnID`, `Query` | net | Consume a YAML recipe; write bag + craft rank |
| `Sell` / `Buy` | `ConnID`, `Query`, `Qty` | net | Trade at the room's `market` slug; write nyang + stock |
| `Quote` | `ConnID` | net | Emit current stall prices |
| `Hide` | `ConnID`, `Query?` | net | At `checkpoint` with contraband: stealth-flavored conceal. Success sets `dawn_scent` / `smuggle_pass`; fail confiscates one unit or fines 냥 (D-048). Successful leaflet hide with EmberPrereq also sets `ember=1` (D-049) |
| `LeaveWorld` | `ConnID` | net, on `quit` or disconnect | Persist sheet; delete roster; emit `Sys` to remaining **in that room** |

Auth create/login is **not** a loop command. Hashing and store I/O run in
the connection goroutine (or a persist worker). Only a successful auth
produces `EnterWorld`.

Unknown `ConnID` on `Say` / `LeaveWorld`: no-op (already gone).

## Events (M0–M1)

| Name | Fields | Delivery |
|---|---|---|
| `Text` | `ConnID` (target), `Channel` (`say`\|`sys`\|`room`), `From`, `Body` | That connection's outbound buffer |
| `Room` | `ConnID`, `ID`, `Name`, `Description`, `Exits` (dir→display name), `Who`, `NPCs`, `Objects`, `Ground` | That connection; adapters format per WIRE-PROTOCOL |
| `Drop` | `ConnID` | Adapter closes the socket after flush |

The loop never writes to a socket. It only appends to an outbound queue
the adapter drains.

## Tick

Period: **100ms**. Tick work:

1. Drain the command channel (bounded; see below).
2. Regen gather nodes whose `regen_ticks` divide the tick count.
3. Every 10 ticks, walk each market good one step toward `target` (NPC flow).
4. No combat, weather, or NPC AI yet.

If the drain exceeds the tick budget, log a warning and continue. Do not
spawn extra loop goroutines.

## Channel contract

- Command queue: buffered (`256`). A full queue: the adapter treats it as
  `rate_limited` / drops the command. The adapter must not block the
  caller for more than a short send timeout.
- Per-connection outbound: buffered (`64`). Overflow drops the oldest
  event and logs. The loop does not block on a slow client.

## Invariants tests must lock in

1. `go test -race` is clean with many goroutines calling `Submit`.
2. Roster maps are not accessed from test helpers except through the loop
   (or a test-only snapshot method that **copies** under the loop's own
   tick — a `Snapshot()` command/request, never a naked map read).
3. No `sync.Mutex` / `RWMutex` on `World`, `Roster`, or player structs.

## Reserved for later milestones

Do not implement these in M1; names are reserved so GAMEPLAY/ECONOMY do
not invent parallel buses:

- `UseSkill`, `CombatIntent`
- `HeatTick`, `WeatherTick`
- `DialogueRequest` / `DialogueResult` (AINPC workers → loop)

Price drift is not a command. The loop calls `Book.Tick` on its own
goroutine (D-043).
