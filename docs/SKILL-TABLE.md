# SKILL-TABLE

GAMEPLAY ↔ ECONOMY / ENGINE. Agreed before implementation (PLAN.md §9.4).
M2 ships **14 skills** (전투 6 + 생산 4 + 생활 4). The remaining ~26 wait.

There are **no levels, no classes, no XP bars.** Rank is 0–100 per skill.
The player-facing identity is a **title**, not a number.

Related: [CONTENT-SCHEMA.md](CONTENT-SCHEMA.md), [EVENT-BUS.md](EVENT-BUS.md).

## M2 skills

| id | ko | group | stat | practice_flag | practice_item |
|---|---|---|---|---|---|
| unarmed | 격술 | combat | str | yard | — |
| sword | 검술 | combat | str | yard | wooden-sword |
| bow | 궁술 | combat | dex | yard | — |
| dodge | 회피 | combat | dex | yard | — |
| guard | 방어 | combat | vit | yard | — |
| firstaid | 응급처치 | combat | sense | clinic | bandage |
| smith | 대장 | craft | str | forge | hammer |
| cook | 요리 | craft | sense | kitchen | kitchen-knife |
| print | 인쇄 | craft | dex | press | composing-stick |
| remedy | 약제 | craft | sense | clinic | — |
| haggle | 흥정 | social | fame | market | — |
| speech | 언변 | social | wit | town | — |
| stealth | 잠입 | social | dex | dark | — |
| lockpick | 자물쇠 | social | dex | — | lockpick |

`practice_flag` / `practice_item` are **matching bonuses**, not hard gates.
You may practice anywhere. Matching the room flag **or** holding the item
sets `difficulty` to the character's current rank (best gain). Mismatch
sets `difficulty` to `max(0, rank-25)` (easy, almost no gain at mid-rank).

## Rank bands (display only)

| band | ko | range |
|---|---|---|
| raw | 미숙 | 0–19 |
| trained | 수련 | 20–44 |
| able | 능숙 | 45–69 |
| veteran | 노련 | 70–89 |
| master | 명인 | 90–99 |
| peak | 경지 | 100 |

경지 is not globally unique in M2 (no server-wide slot table yet).

## Growth

Each `Practice` is one independent Bernoulli trial. On success the rank
increases by **1**. There is no visible XP bar and no fractional player
stat.

```
if rank >= 100: p = 0
if sum(all ranks) >= 700: p = 0
proximity = exp( -0.5 * ((rank - difficulty) / 18)^2 )
falloff   = 1 / (1 + rank/12)          // log-like
p = 0.55 * falloff * proximity
```

Deterministic tests inject a `Rand` (or a `Always(bool)` hook). Production
uses `crypto/rand` or `math/rand/v2` seeded at process start — **not**
per-command time.Now() jitter that tests cannot pin.

A failed trial still emits a "nothing settles" line. Macro-grinding an
easy station at high rank must measure `p ≈ 0`.

## Stats

Six stats, 0–100 each, **sum cap 300**: 힘 `str`, 손재주 `dex`, 맷집 `vit`,
기지 `wit`, 감응 `sense`, 인망 `fame`.

On a successful skill gain, the tagged stat increases by 1 with
`p = 0.25 * (1 - stat/100)`, refused if the sum would exceed 300.

## Titles

YAML list, first matching rule in file order wins (priority = order).

```yaml
- id: smith-local
  require: { smith: 15 }
  title: { ko: "달빛골의 대장장이" }
- id: speaker
  require: { speech: 15 }
  title: { ko: "말을 부리는 자" }
- id: nobody
  require: {}
  title: { ko: "아무개" }
```

An empty `require` is the default and must be last. No class ids.

## Caps

| what | cap |
|---|---|
| one skill | 100 |
| sum of skill ranks | **700** |
| one stat | 100 |
| sum of stats | **300** |

`lock` / `down` commands are **not** in M2 (D-033). The cap still blocks
gains.

## What M2 does not ship

- Combat rounds, cooldown, stance, hit locations (D-032)
- The other ~26 skills
- Player-to-player trade (M3)
- Crafting recipes that consume resources (M3). Practice does not spawn
  products except as already-world items.
