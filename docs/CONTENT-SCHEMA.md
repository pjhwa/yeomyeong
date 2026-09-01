# CONTENT-SCHEMA

WORLD ↔ ENGINE / GAMEPLAY. Rooms, NPCs, objects, and items are YAML only
(PLAN.md §4.4, principle 5).

Related: [CONTENT-STYLE.md](CONTENT-STYLE.md), [FORESHADOW.md](FORESHADOW.md),
[EVENT-BUS.md](EVENT-BUS.md).

## Layout

```
content/zones/<zone>/rooms.yaml
```

`<zone>` is a lowercase slug (`dalbitgol`).

```
content/skills/*.yaml          # GAMEPLAY — ranks, titles (SKILL-TABLE.md)
content/items/*.yaml           # WORLD names + ENGINE slots
content/craft/nodes.yaml       # gather points (M3)
content/craft/recipes.yaml     # consume/produce (M3)
content/economy/markets.yaml   # regional price tables (M3)
content/zones/<zone>/rooms.yaml
content/zones/<zone>/spawns.yaml   # extra flags + ground item ids
content/zones/<zone>/objects.yaml  # examine targets (보다)
content/zones/<zone>/npcs.yaml     # T0 scripted NPCs (대화)
```

`quests.yaml` remains reserved.

## Localized string

Korean is required. English is optional and falls back to Korean (D-029).

```yaml
name:
  ko: "달빛골 장터"
  en: "Dalbitgol Market"   # optional
```

A bare string is accepted as Korean-only:

```yaml
name: "달빛골 장터"
```

The loader canonicalizes both forms to `{ko, en}`.

## Room object

```yaml
- id: dalbitgol:market
  name:
    ko: "달빛골 장터"
  description:
    ko: >
      좁은 흙길 양옆으로 좌판이 늘어섰다. 참기름 고소한 냄새가 나고,
      저울추가 나무판을 친다. 장터 어귀 게시판에는 관아 안내문이 붙어
      있는데, 누군가 모퉁이를 찢어 갔다.
  exits:
    north: dalbitgol:smithy
    east: dalbitgol:inn
  flags: [safe, town, market]
  market: dalbitgol
  heat_modifier: 0.8
  ambient:
    - ko: "지게꾼이 무거운 짐을 지고 지나간다."
    - ko: "장터 소문이 바람결에 실려 온다."
  foreshadow: [FS-014]
```

| Field | Required | Rules |
|---|---|---|
| `id` | yes | `zone:slug`. Zone must match the directory. Unique across the load. |
| `name` | yes | Localized. Non-empty `ko`. |
| `description` | yes | Localized. 2–4 sentences in `ko` (style, not schema). |
| `exits` | no | Keys: `north` `south` `east` `west` `up` `down`. Values: existing room ids (checked after the full load). |
| `flags` | no | Closed set: `safe`, `town`, `market`, `indoor`, `dark`, `forge`, `kitchen`, `press`, `clinic`, `yard`, `salon`, `checkpoint`. Unknown flag = load error. |
| `market` | no | Slug of a price table in `content/economy/markets.yaml`. 시세/팔다/사다 use this stall. |
| `heat_modifier` | no | Float, default `1.0`. Unused in M1; stored for M5. |
| `ambient` | no | List of localized strings. Flavor only in M1. |
| `foreshadow` | no | List of `FS-NNN` that **already exist** in `docs/FORESHADOW.md`. Unknown ID = load error. |

## Examine target (`content/zones/<zone>/objects.yaml`)

Scenery, not inventory. `보다 신문` matches an alias in the current room.

```yaml
- id: hanbyeok-ilbo
  room: dalbitgol:market
  name: { ko: "한벽일보" }
  aliases: [한벽일보, 신문, 게시판]
  description:
    ko: >
      한벽일보 한 줄에 같은 활자가 두 번 찍혀 있다.
  after_examine:
    ko: "바깥에서 덧문이 한 번 닫힌다."
  foreshadow: [FS-001, FS-014]
```

Room cards list each scenery `name.ko` as `살펴볼 것:` (text key `room.objects`),
parallel to `사람:` / `바닥:`.

| Field | Required | Rules |
|---|---|---|
| `id` | yes | `[a-z][a-z0-9-]*`, unique across the load |
| `room` | yes | existing room id; zone must match the directory |
| `name` | yes | Localized. Non-empty `ko`. Shown on the room card (`살펴볼 것:`) |
| `aliases` | no | `보다` / `look` targets. Also matches `id` and `name` |
| `description` | yes | Localized. 2–4 sentences, ≥2 senses (style) |
| `after_examine` | no | Localized ambient line after a successful `보다`. Korean-only OK. Empty = none. The loop prints `description` (or a matching `description_when`) every time; this line fires once per player (`Flags["examined:<id>"]`). Not an NPC. |
| `description_when` | no | List of `{flag, ko}` (`en` optional). On `보다`, the **first** entry whose `flag` is >0 on the player sheet replaces `description` (D-047). Empty/missing `flag` = load error. |
| `foreshadow` | no | existing `FS-NNN` ids |

## Scripted NPC (`content/zones/<zone>/npcs.yaml`)

T0 YAML dialogue. Not LLM. `대화 청람` / `talk 청람` / `말걸다`.

```yaml
- id: cheongram
  name: { ko: "청람 선생" }
  room: dalbitgol:school
  aliases: [청람, 선생]
  look: { ko: "낡은 코트 소매에 잉크가 묻어 있다." }
  talk:
    first: { ko: "처음 보는 얼굴이군, 자네." }
    second: { ko: "또 왔군, 자네." }
    when:
      - flag: first_market_sale
        ko: "장터에서 물건을 넘겼다고 들었네, 자네."
  foreshadow: [FS-016]
```

| Field | Required | Rules |
|---|---|---|
| `id` | yes | `[a-z][a-z0-9-]*`, unique. First talk sets sheet flag `{id}_talked` |
| `room` | yes | existing room id; zone must match the directory |
| `name` | yes | Localized. Shown on the room card (`사람:`) |
| `aliases` | yes | at least one. `대화` / `보다` targets |
| `look` | yes | `보다 <alias>` blurb |
| `talk.first` / `talk.second` | yes | Korean. Second line is for a player who already talked |
| `talk.when` | no | List of `{flag, ko}` (`en` optional). On `대화`, the **first** entry whose `flag` is >0 on the player sheet wins over first/second (D-046). Empty/missing `flag` = load error. Flag is an opaque sheet key. first/second stay required. |
| `foreshadow` | no | existing `FS-NNN` ids |

Unknown NPC: `여기는 그 사람이 없어요.`

## Item object (`content/items/*.yaml`)

```yaml
- id: hammer
  name: { ko: "쇠망치" }
  description: { ko: "자루가 반질한 쇠망치다. 아직 대장간 냄새가 난다." }
  slot: main_hand          # none | main_hand | body
  skills: [smith]          # practice bonus tags
  weight: 2
```

| Field | Required | Rules |
|---|---|---|
| `id` | yes | `[a-z][a-z0-9-]*`, unique |
| `name` / `description` | yes | Localized. Style: short, one object, no 총독부 |
| `slot` | no | default `none` (not wearable) |
| `skills` | no | skill ids from SKILL-TABLE |
| `weight` | no | integer ≥ 0, default 1. Bag cap is **20** weight |

## Zone spawns (`content/zones/<zone>/spawns.yaml`)

Does not rewrite room prose. Applied after rooms load.

```yaml
- room: dalbitgol:smithy
  flags_add: [forge]
  items: [hammer]
```

| Field | Required | Rules |
|---|---|---|
| `room` | yes | existing room id |
| `flags_add` | no | subset of the closed flag set |
| `items` | no | item ids that exist. Copied onto the **room ground** at boot (world state), not into the catalog |

## Gather node (`content/craft/nodes.yaml`)

```yaml
- room: dalbitgol:barley-field
  skill: forage
  item: mugwort
  stock: 6
  regen_ticks: 40
```

| Field | Required | Rules |
|---|---|---|
| `room` | yes | existing room id |
| `skill` | yes | skill id, group `gather` |
| `item` | yes | item id |
| `stock` | no | boot remaining and max, default 4, ≥ 0 |
| `regen_ticks` | no | restore 1 toward max every N loop ticks (100ms), default 50. 0 = no regen |

Player verbs are the skill's YAML `verbs` (`캐다`, `줍다`). Empty node → wait; wrong room → `여기서는 집을 게 없다.`

## Recipe (`content/craft/recipes.yaml`)

```yaml
- id: iron-nail
  skill: smith
  flag: forge
  tool: hammer
  in: [{id: scrap-iron, n: 2}]
  out: {id: iron-nail, n: 1}
  gain: ["쇠못 머리가 둥글게 잡혔다. 모루가 아직 뜨겁다."]
```

| Field | Required | Rules |
|---|---|---|
| `id` | yes | unique slug |
| `skill` | yes | existing skill id (use-based gain on success) |
| `flag` | no | station: `forge` `kitchen` `press` `clinic`. Hard gate. |
| `tool` | no | item that must be held (bag or slot), not consumed |
| `in` | yes | consumed piles |
| `out` | yes | produced pile |
| `gain` / `miss` | no | Korean prose, no rank numbers |

`만들다` lists recipes the room can host. Practice verbs (`두드리다`) do **not** consume (D-042).

## Market (`content/economy/markets.yaml`)

```yaml
- id: dalbitgol
  name: 달빛골 장터
  goods:
    - {id: mugwort, base: 3, stock: 18, target: 16, demand: 0.7}
```

| Field | Required | Rules |
|---|---|---|
| `id` | yes | slug; rooms point here with `market:` |
| `name` | yes | Korean stall name |
| `goods[].id` | yes | item id |
| `goods[].base` | yes | ≥ 1. Not the posted price — quote uses D-043 |
| `goods[].stock` | no | live pile, default 0 |
| `goods[].target` | no | NPC restock level, default stock |
| `goods[].demand` | no | > 0, default 1 |

At least two markets so the same good can quote differently. Selling raises stock (price eases); buying lowers it. Tick walks stock one step toward `target`.

Rooms with flag `checkpoint` may take a toll when a player walks in carrying **4+** units of market goods (D-044).

## Graph rules

- Every exit target must exist.
- One-way exits are allowed (a cliff).
- The spawn room `dalbitgol:gate` (D-028) must exist after a full load.
- Every room in `dalbitgol` must be reachable from the spawn (QA / loader check).
- Cycles are allowed.

## What must not appear

- Real historical persons or organizations; other works' proper nouns.
- The word **총독부** as a proper noun (D-019). Use **관아** / **한벽 철탑**.
- Hardcoded rooms in Go. Test fixtures live under `internal/content/testdata/`
  and use ids in the `test:` zone.

## Load failures

The process **must not boot** if:

- YAML does not parse
- a required field is missing
- an exit points at a missing id
- spawn is missing
- a `foreshadow` id is not in the ledger
- a flag is unknown

Hot reload is not required in M1 (boot load only).
