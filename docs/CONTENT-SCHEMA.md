# CONTENT-SCHEMA

WORLD ↔ ENGINE / GAMEPLAY. Rooms, NPCs, and items are YAML only
(PLAN.md §4.4, principle 5). M1 defines **rooms**. NPC/item schemas wait
for later milestones.

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
content/zones/<zone>/rooms.yaml
content/zones/<zone>/spawns.yaml   # extra flags + ground item ids
```

`npcs.yaml` and `quests.yaml` remain reserved.

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
      좁은 흙길 양옆으로 좌판이 늘어섰다. 말린 명태와 삼베 냄새가 뒤섞이고,
      엿장수의 가위질 소리가 장단을 맞춘다. 장터 어귀의 게시판에는 관아
      고시문이 붙어 있는데, 누군가 모퉁이를 찢어 갔다.
  exits:
    north: dalbitgol:smithy
    east: dalbitgol:inn
  flags: [safe, town, market]
  market: dalbitgol
  heat_modifier: 0.8
  ambient:
    - ko: "지게꾼이 무거운 짐을 지고 지나간다."
    - ko: "저잣거리 소문이 바람결에 실려 온다."
  foreshadow: [FS-014]
```

| Field | Required | Rules |
|---|---|---|
| `id` | yes | `zone:slug`. Zone must match the directory. Unique across the load. |
| `name` | yes | Localized. Non-empty `ko`. |
| `description` | yes | Localized. 2–4 sentences in `ko` (style, not schema). |
| `exits` | no | Keys: `north` `south` `east` `west` `up` `down`. Values: existing room ids (checked after the full load). |
| `flags` | no | Closed set: `safe`, `town`, `market`, `indoor`, `dark`, `forge`, `kitchen`, `press`, `clinic`, `yard`, `salon`. Unknown flag = load error. |
| `market` | no | Slug of a price table. Unused in M1; stored for M3. |
| `heat_modifier` | no | Float, default `1.0`. Unused in M1; stored for M5. |
| `ambient` | no | List of localized strings. Flavor only in M1. |
| `foreshadow` | no | List of `FS-NNN` that **already exist** in `docs/FORESHADOW.md`. Unknown ID = load error. |

## Item object (`content/items/*.yaml`)

```yaml
- id: hammer
  name: { ko: "쇠망치" }
  description: { ko: "자루가 반질한 쇠망치다. 대장간 냄새를 아직 품고 있다." }
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
