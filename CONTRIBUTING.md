# Contributing to YEOMYEONG

Writers are as welcome as engineers. A well-written room is a feature.

Read [PLAN.md](PLAN.md) first. It is the single source of truth.
Then skim this file and the five non-negotiable principles below.

한국어가 1차 작업 언어다. 커밋 메시지와 공개 README는 영문을 병기한다.

## Non-negotiable principles

A pull request that violates any of these is rejected without debate.

1. **Single-writer game loop.** Only one game-loop goroutine mutates world state. No scattered mutexes on the world.
2. **No levels, no classes, no XP bars.** Growth is use-based skill (0–100) with a total cap of 700.
3. **Every major obstacle has at least three solutions** — force, craft/skill, and social. No combat-gated segments.
4. **No real people or organizations. No other works' proper nouns. No direct depiction of torture or massacre.**
5. **Rooms, NPCs, and items are defined in YAML.** Hardcoded content in Go is a defect.

## Who does what

LEAD is the only person who merges. Sub-agents do not merge their own PRs.

| Agent | Owns | Does not touch |
|---|---|---|
| ENGINE | `internal/engine`, `internal/net`, `internal/world`, `internal/persist` | `web/`, `content/` prose |
| GAMEPLAY | `internal/skill`, `internal/combat`, `internal/arts`, `internal/craft`, `internal/script` | economy numbers, narrative text |
| ECONOMY | `internal/economy` | combat resolver, room prose |
| NARRATIVE | `docs/FORESHADOW.md`, `docs/STORY-BIBLE.md`, `content/zones/*/quests` | engine code |
| WORLD | `content/zones`, `content/lore`, `content/personas` | any `.go` file |
| AINPC | `internal/dialogue` | world YAML, client UI |
| CLIENT | `web/` | server packages |
| QA | `tools/`, E2E, linters | other agents' source (file a report) |

If you need a change in someone else's tree, open an issue with the `needs-decision` label. Do not edit across ownership lines.

Shared surfaces (`EVENT-BUS.md`, `WIRE-PROTOCOL.md`, `CONTENT-SCHEMA.md`, `DIALOGUE-API.md`, `SKILL-TABLE.md`) are agreed in writing before implementation.

## How work starts

1. There is already an issue. If there is not, stop and open one.
2. Move the issue to In Progress and put the owning agent in the issue body (`Owner: ENGINE`).
3. Branch from `dev` (from `main` only during the M0 bootstrap):

   ```
   <agent>/<milestone>-<slug>
   ```

   Examples: `engine/m0-telnet-listener`, `world/m1-dalbitgol-market`.

4. Push at least once a day.
5. Open a PR against `dev`. The body must contain `Closes #N`.
6. Fill the five-principles checklist honestly. A false check is the highest-severity process violation.
7. Wait for LEAD review. If rejected, push more commits to the same branch. Do not open a replacement PR.

Milestone work does not start until the previous milestone's completion criteria (PLAN.md §5) and quality gates (PLAN.md §9.5) have passed.

## Commits

[Conventional Commits](https://www.conventionalcommits.org):

```
feat(economy): add regional price simulation
fix(combat): correct parry chance rounding
content(dalbitgol): add 12 market rooms
docs(story): register FS-014 in foreshadow ledger
chore(ci): skip golangci-lint when go.mod is absent
```

Every commit carries an agent trailer:

```
Co-Authored-By: ENGINE <engine@yeomyeong.dev>
```

Valid agent names: `LEAD`, `ENGINE`, `GAMEPLAY`, `ECONOMY`, `NARRATIVE`, `WORLD`, `AINPC`, `CLIENT`, `QA`.

## Pull requests

- Aim for ~600 changed lines. Split above 1,000.
- Required sections live in `.github/PULL_REQUEST_TEMPLATE.md`.
- Merge method is **squash only**. LEAD merges. The branch is deleted after merge.
- CI must be green: `go build`, `go vet`, `golangci-lint`, unit tests, coverage (once `go.mod` exists), and the content linter when `content/` changes.

LEAD reviews within 24 hours. Rejections come with a concrete defect list, not "please improve."

## Content writers

1. Read `docs/CONTENT-STYLE.md` (written in M1) before any prose.
2. Every room is 2–4 sentences, at least two senses, no repeated clichés.
3. Period vocabulary is welcome (전차, 등사기, 고무신, 주재소). Dense Sino-Korean is not.
4. Register every planted clue in `docs/FORESHADOW.md` in the same change.
5. Prefer a `good first issue` labelled `type:content` if you are new.

## Design arguments

Design debate happens on issues labelled `needs-decision`, not in PR review threads. LEAD records the ruling in `docs/DECISIONS.md`. Open questions for the commissioner go in `docs/QUESTIONS.md` only. Milestone playtests follow `docs/COMMISSIONER.md`.

Features that are not in PLAN.md are not implemented. Park them in `docs/PROPOSALS.md` for the next commissioner review.
