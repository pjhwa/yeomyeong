# CONTENT-STYLE

WORLD owns this file. LEAD approves it **before** any `content/zones/**`.
Schema is [CONTENT-SCHEMA.md](CONTENT-SCHEMA.md). Plot is
[STORY-BIBLE.md](STORY-BIBLE.md). Clues are [FORESHADOW.md](FORESHADOW.md).
WORLD does not invent plot or foreshadow IDs.

읽는 맛이 곧 그래픽이다. The room text is the render.

## Voice

Write as if the player is standing there, talking about what they can see
and touch — **everyday modern Korean** (D-045). The setting is 1910s–1920s
달내; the diction is not. Do not write textbook 1920s literary Korean.

- **2–4 sentences** in `description.ko`. One sentence is a caption. Five is
  a paragraph the look-loop will dump. Count Korean periods, not YAML lines.
- **At least two senses.** Sight plus one of sound, smell, touch, or taste.
  Two visual adjectives are still one sense.
- **One specific object** the player could look at again later — a torn
  corner, a strap, a slug of type. Generic stalls and "an old alley" do not
  count.
- **No repeated clichés.** If a smell, cry, or notice has been spent, invent
  the next room from a different sense and a different object.
- **Spent, do not reuse:** the old market sentence (*말린 명태와 삼베 냄새*,
  엿장수 가위질) is retired. Do not put it in live rooms or new samples.
- **Words people use now.** Prefer 가게 / 시장 / 길 / 경찰 / 검문 / 돈 /
  가방 / 만들다 / 캐다. Cut rare 한자어, 고어, and 문어체 (엿장수 가위질,
  윗도리, 문설주, 오래라, 한 켜, 품귀). Commands are common verbs a player
  would type (`보다`, `캐다`, `만들다`, `팔다`), not 언변 or 익히다.
- Period objects are seasoning when they are still the thing in the room.
  Do not pour the whole allowed list into one yard.

Keep unique proper nouns and world terms: 달내, 무쇠 제국, 새벽회, 달빛골,
솔골, 지맥, 쇠말뚝, 한벽일보, 다방 백야, 만석상회. Those are setting, not
diction.

Korean is required. English `en` is optional and falls back to Korean
(D-029). Do not ship an English-only room.

## Room recipe

| Must | Must not |
|---|---|
| 2–4 sentences | A mood with no object |
| ≥2 distinct senses | The same spent smell in the next stall |
| One concrete object | Abstract 한 / 기개 / 통한 |
| Present tense, on the ground | A history lecture or a quest brief |
| Everyday spoken Korean | 문어체, 고어, rare 한자어 |
| Aftermath and residue | Torture, massacre, a death scene |

A later beat may rewrite the same room (gloves on the anvil, a café shuttered
for "sanitation"). Write the **present** look. Do not spoil Act II in an
Act I description.

## Period vocabulary

Welcome when they earn their keep as **objects in the room**, not as a
writing style:

전차, 등사기, 고무신, 주재소, 관아, 철탑, 다방, 활자, 지게, 통행증, 전신,
원지

Occupation offices are **관아**. The intake tower is **한벽 철탑** (D-019).
A village post is **주재소**, not a real-world bureau. A checkpoint is
**검문** / **검문소**. A posted 관아 paper is **안내문**, not 고시문.

Dense Sino-Korean is forbidden even when it is historically flavored.
If a sentence would sit in a textbook — 국권피탈, 민족정기, 무단통치, 철권,
분견대, 봉랍, 교정쇄, 식자대 — cut it and name the gravel, the ink, or the
strap instead. Say 경찰 / 검문 when the player is meeting a check; keep
주재소 when that building is the room.

## Forbidden

PLAN.md §2.1 is the rule. `tools/contentlint/denylist.txt` is the machine
list. `content-lint` scans this file; do not quote denylisted names here to
"explain" them.

- **No real historical persons or organizations.** No real colonial-era
  city names used as political entities. Spirit of the era, invented nouns.
- **No proper nouns from other works** (books, games, films).
- **No 총독부** as a proper noun (D-019). Write **관아** or **한벽 철탑**.
- **No direct torture or massacre.** Tragedy is implication, residue, and
  aftermath — a dry well, a silent radio hour, a blank masthead (D-020).
  돌쇠 dies on every path; the room shows the empty stool, not the blow.
- **No hatred aimed at a real-world nation or people.** The occupier is the
  fictional 무쇠 제국. Empire faces stay three-dimensional.
- **No rooms, NPCs, or items in Go.** YAML only. Test fixtures belong under
  `internal/content/testdata/` in the `test:` zone.

## Naming

Ids are `zone:slug`.

- `zone` is a lowercase slug and **must match** `content/zones/<zone>/`.
- `slug` is lowercase ASCII `[a-z][a-z0-9-]*`. No Hangul in the id.
- Unique across the whole load.
- Display names are Korean (`name.ko`). `name.en` is optional.

```yaml
id: dalbitgol:gate
name:
  ko: "달빛골 마을문"
  en: "Dalbitgol Gate"    # optional
```

Reserved zone prefixes (D-025; WORLD may rename rooms, not prefixes):
`dalbitgol`, `hanbyeok`, `meokgol`, `jaeanam`, `garangpo`, `huinjaebo`,
`jamseoseong`. Spawn is `dalbitgol:gate` (D-028). That room is live content
and is **not** written in this document.

## Foreshadow

Attach only IDs that **already exist** in [FORESHADOW.md](FORESHADOW.md).
Never invent `FS-NNN`. NARRATIVE allocates; WORLD hangs the key on the room
that actually holds the plant. Unknown id is a boot failure.

```yaml
foreshadow: [FS-014]
```

- Copy the id exactly (`FS-014`, zero-padded).
- The room text must carry the plant the ledger already named — for FS-014,
  a 관아 notice with a torn corner (object), not a new riddle.
- One room may list several ids if the ledger plants them there.
- A new clue is a NARRATIVE change to the ledger **in the same PR** as the
  YAML. Next free as of this writing is `FS-021`; do not mint it here.
- `dropped` rows stay unused. Do not revive an id.

If the ledger does not mention the room, omit the field.

## Ambient

`ambient` lines are scenery people, not NPC entities.

- Flavor only in M1. The loader does not turn them into actors.
- Unnamed, passing, repeatable: a porter shifting a 지게, a runner with
  등사기 sheets, a tram bell beyond the wall.
- **Not** a named face, a secret, a shop that can be hailed, or a line that
  belongs in later `npcs.yaml`.
- If the person has a name, a schedule, or a recovery beat, they are an NPC.
  Do not smuggle 청람 or 돌쇠 into `ambient`.

```yaml
ambient:
  - ko: "심부름꾼이 등사기 원지를 품에 안고 마당을 가로지른다."
  - ko: "담 너머에서 전차 종소리가 한 번 끊긴다."
```

## Example (not live content)

The block below is a **schema-shaped illustration**. It is not a room.
Do not copy it into `content/zones/`. Do not load an `example:` zone.

```yaml
# EXAMPLE ONLY — not live content. Do not add to content/zones/.
- id: example:yamen-yard
  name:
    ko: "관아 앞마당"
  description:
    ko: >
      관아 앞마당은 자갈이 닳아 하얗고, 낮에도 그늘이 짧다. 회벽에는
      등사기 잉크가 덜 마른 안내문이 겹쳐 붙었고, 맨 앞장 모퉁이만
      손가락 넓이로 찢어져 있다. 고무신 밑창이 자갈을 문지르는 소리가
      담 너머 전차 종보다 가깝다. 누가 두고 간 지게가 문기둥에 기대어
      있고, 멜빵 가죽이 햇볕에 뜨겁다.
  exits:
    south: example:alley
  flags: [town]
  ambient:
    - ko: "심부름꾼이 등사기 원지를 품에 안고 마당을 가로지른다."
    - ko: "담 너머에서 전차 종소리가 한 번 끊긴다."
```

Why this passes the recipe: four sentences; sight, sound, and touch; the
specific object is the 지게 and its hot strap (the torn 안내문 is a second
anchor, not a lecture); 관아 not 총독부; everyday diction with period
objects; no spent market-smell sentence; ambient figures have no names.

A live room that the ledger already plants would add, for example,
`foreshadow: [FS-014]` on `dalbitgol:market` — only when that YAML is
written, and only for ids already in the ledger.

## Writer checklist

- [ ] 2–4 sentences, ≥2 senses, one specific object
- [ ] Everyday Korean; no spent cliché, no dense Sino-Korean, no 총독부, no 문어체
- [ ] `id` is `zone:slug`; `name.ko` is what the player reads
- [ ] `foreshadow` ids exist in FORESHADOW.md or the field is omitted
- [ ] `ambient` is scenery, not an NPC
- [ ] `python3 tools/contentlint/lint.py` exits 0
