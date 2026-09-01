# Decisions

LEAD records every binding ruling here. The commissioner (Jerry) should be
able to reconstruct project state from this file and [QUESTIONS.md](QUESTIONS.md)
alone.

Format:

```
## D-NNN — short title
- Date: YYYY-MM-DD
- Status: accepted | superseded by D-NNN
- Decider: LEAD
- Context: why a decision was needed
- Decision: what we will do
- Consequences: what this forces or forbids
```

---

## D-001 — Repository name is `yeomyeong`

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §10.1 recommended renaming `pjhwa/aethermoor` because the
  Western-fantasy name no longer matches the 1920s alternate-history setting.
  The remote is already `https://github.com/pjhwa/yeomyeong`.
- Decision: Keep `yeomyeong`. Do not introduce `dawnguild` or `dalnae` as the
  repository name. GitHub continues to redirect the old `aethermoor` URL.
- Consequences: Module path, Docker image names, and Go import paths use
  `yeomyeong`. The word *Aethermoor* is on the content denylist so it cannot
  leak into world text.

## D-002 — Empty-repo bootstrap commit on `main`

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: The GitHub repository existed with zero commits. PRs, branch
  protection, and a `dev` branch all require a `main` ref.
- Decision: Land `PLAN.md` as the first commit on `main` (direct push, once).
  All subsequent changes, including the community-files foundation, go through
  a PR. After that first push, `main` is protected.
- Consequences: One historically necessary direct push. Documented here so it
  is not treated as a process regression.

## D-003 — Agents are labelled, not assigned as GitHub users

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: ENGINE / GAMEPLAY / NARRATIVE and the rest are roles, not GitHub
  accounts. The assignee field cannot name them.
- Decision: Every issue states `Owner: <AGENT>` in the body and carries the
  matching `area:*` label. The GitHub assignee is the commissioner or whoever
  is driving the human session. Commit trailers still use
  `Co-Authored-By: <AGENT> <agent@yeomyeong.dev>`.
- Consequences: Work tracking is by label + body, not by the assignee column.
  See Q-003 if dedicated bot accounts are created later.

## D-004 — Branch flow: work → `dev` → `main`

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §10.4 specifies `main` ← `dev` ← work branches.
- Decision: Feature PRs target `dev`. `dev` is promoted to `main` only at
  milestone completion (or for a documented hotfix). Exception: the M0
  foundation PR (`lead/m0-repo-foundation`) targets `main` because `dev` is
  created from the same bootstrap and has no prior history.
- Consequences: M0 engine work PRs target `dev`. Do not open feature PRs
  against `main`.

## D-005 — Merge policy

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §10.6 requires squash merges and LEAD-only merge rights.
- Decision: Squash merge only. Merge and rebase commits are disabled.
  Source branches delete on merge. Only LEAD merges.
- Consequences: GitHub history on `main`/`dev` is one commit per PR.
  Conventional-commit PR titles become the squash subject.

## D-006 — Branch protection and the self-approval gap

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §10.4 requires a PR, a green CI, and at least one LEAD
  approval. GitHub does not let the author approve their own PR. In this
  phase LEAD is also the author of bootstrap PRs.
- Decision: Protect `main` with required pull requests, required status check
  `build`, no force-push, no deletions. `enforce_admins` stays **false** so
  LEAD can merge a self-authored PR with an admin merge after reviewing it in
  writing (this file or the PR body). Required approving review count is 1
  once a second actor exists; until then LEAD uses the admin exception and
  records it.
- Consequences: Direct pushes to `main` are forbidden for everyone, including
  LEAD, except the already-completed D-002 bootstrap. CI context name is the
  `build` job in `.github/workflows/ci.yml`. `content-lint` is **not** a
  required check — path filters would otherwise block PRs that do not run it.

## D-007 — CI is skip-safe until `go.mod` exists

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: Foundation lands before ENGINE creates a module. A hard `go build`
  would make the first PRs unmergeable and prevent enabling required checks.
- Decision: `ci.yml` no-ops (exit 0) when `go.mod` is absent. `content-lint`
  no-ops when there are no lint targets. Both become strict the moment those
  files appear.
- Consequences: ENGINE's first module PR is what turns CI into a real gate.
  Do not weaken the skip after `go.mod` exists.

## D-008 — Repository stays public; promotion waits for M4

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §10.8 says flip public after M1 and promote after M4.
  The repository is already public.
- Decision: Leave it public. Do not advertise (HN, Reddit, GeekNews) until M4
  acceptance criteria are met: playable Act I, demo GIF, reachable demo host.
- Consequences: Early clones will see an unfinished tree. README already
  marks status as M0. Revisit only if the commissioner answers Q-002 with
  "make it private until M1."

## D-009 — Copyright holder on the MIT license

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §6 selects MIT and names Jerry Park as commissioner.
- Decision: `LICENSE` reads `Copyright (c) 2026 Jerry Park`.
- Consequences: Third-party contributions remain MIT. No CLA in M0.

## D-010 — Story bible and foreshadow ledger start as scaffolds

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §2.3–2.5 and §9.2 require the ending to be locked before
  clues are planted. NARRATIVE writes the draft in parallel with M0, not as
  a blocker for the server skeleton.
- Decision: `docs/STORY-BIBLE.md` and `docs/FORESHADOW.md` ship as structured
  empty ledgers in the foundation PR. NARRATIVE fills them under a separate
  issue. The ending is written first; foreshadow rows are derived backwards.
- Consequences: WORLD does not write quests or plant `foreshadow:` keys until
  NARRATIVE's draft is LEAD-approved.

## D-011 — Repository About blurb and topics

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §10.2.
- Decision: Description and topics match the plan verbatim (description first
  sentence carries the "liberate without drawing a sword" hook). Website
  field stays empty until a demo URL exists (Q-001).
- Consequences: Topics include `korean-history` and `alternate-history` as
  specified. World text itself still obeys §2.1 — topics are catalog metadata,
  not in-game nouns.

## D-012 — Auth I/O stays off the game loop

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: Account create/login needs argon2id and (later) Postgres. Those
  must not stall the 100ms tick. World state is only the in-memory roster.
- Decision: Connection goroutines (or a persist worker) hash passwords and
  call `AccountStore`. Only a successful auth enqueues `EnterWorld`.
  `Say` / `LeaveWorld` are the other M0 commands. See EVENT-BUS.md.
- Consequences: The loop package has no import of `database/sql` or argon2.
  Persist is not world state and may use its own mutexes.

## D-013 — M0 world is a roster, not rooms

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: M0 done-when is “two clients connect and talk.” Rooms are M1.
- Decision: The only in-memory world data in M0 is the logged-in roster.
  `say` broadcasts to everyone on that roster. No room graph, no YAML
  content, no hardcoded plaza.
- Consequences: WORLD is idle. ENGINE must not invent a “void room” entity
  that later collides with the YAML loader.

## D-014 — In-memory AccountStore when DATABASE_URL is empty

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: CI and unit tests cannot depend on a live Postgres.
- Decision: `AccountStore` is an interface. Empty `DATABASE_URL` selects
  the in-memory implementation. Postgres is implemented in the persist
  issue and used only when the URL is set.
- Consequences: Default `docker compose` / local `go run` without a URL
  still satisfies M0. Data vanishes on process exit — acceptable until M1.

## D-015 — Module path and process config

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: First `go.mod` must not churn.
- Decision: Module is `github.com/pjhwa/yeomyeong`. Env:
  `YEOMYEONG_TELNET_ADDR` (`:4001`), `YEOMYEONG_WS_ADDR` (`:8080`),
  `DATABASE_URL`, `YEOMYEONG_LOG_LEVEL` (`info`).
- Consequences: Import paths and Docker image names follow the module.

## D-016 — M0 prompts are Korean literals

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §7.7 wants i18n keys from M1. Inventing a key system
  in M0 would delay the skeleton.
- Decision: Telnet prompts and system lines are Korean string literals.
  M1 replaces them with keyed text. No English-only player-facing path.
- Consequences: Tests assert on the Korean strings in WIRE-PROTOCOL.md.

## D-017 — Username and password rules

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: Need a stable validation rule before persist lands.
- Decision: Username 2–16 runes, Hangul syllables / letters / digits / `_`,
  uniqueness is Unicode simple-fold. Password 8–72 bytes. Login failures
  always return `bad_credentials` (no user enumeration).
- Consequences: Content names (NPC ids) are a different namespace and
  unused in M0.

## D-018 — Close issues on merge-to-dev

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: GitHub auto-closes `Closes #N` only when the PR lands on the
  default branch (`main`). Work PRs target `dev` (D-004).
- Decision: LEAD closes the issue when the implementing PR is squash-merged
  to `dev`, and cites the PR. Milestone promote `dev` → `main` is separate.
- Consequences: Issue state tracks `dev`, not `main`.

## D-019 — Occupation administration is 관아; the tower is 한벽 철탑

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md's sample room used 총독부. NARRATIVE dodged it as too
  close to a real institution (issue #14 / PR #16).
- Decision: In-world occupation offices are **관아**. The intake valve is
  **한벽 철탑**. WORLD must not introduce 총독부 as a proper noun.
- Consequences: M1 room prose follows this. The PLAN.md YAML example is
  historical; new files do not copy that word.

## D-020 — 분맥 is a server-wide binary fail state

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: NARRATIVE asked whether 분맥 should be a sliding aftermath.
- Decision: Completing the primer cauterizes the five blood-points
  (server-wide). Stopping it allows reverse flow. Aftermath is wells,
  silent radio, empty masthead — never a massacre scene.
- Consequences: M5–M6 raid design treats the primer cutoff as a binary
  gate, not a damage slider.

## D-021 — Game loop handles commands immediately and also drains on tick

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: EVENT-BUS.md listed drain as tick work. Immediate handle lowers
  say latency without a second writer (PR #17).
- Decision: The loop `select`s on the command channel *and* drains the
  queue on each 100ms tick. Overrun logs a warning. Still one writer.
- Consequences: Adapters must not assume a full tick of delay.

## D-022 — argon2id parameters (OWASP 2024 interactive)

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: RFC 9106's 64 MiB option made `-race` hashes too slow.
- Decision: time=2, memory=19456 KiB (19 MiB), threads=1, key=32, salt=16.
  Stored as a PHC string. Verify uses the stored params.
- Consequences: Retuning later does not need a second hash algorithm.

## D-023 — WebSocket library and origin policy

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: M0 has no browser client.
- Decision: `github.com/gorilla/websocket` v1.5.3. `CheckOrigin` allows all
  until CLIENT exists (M6). Revisit then.
- Consequences: A hostile page can open `/ws` against a local demo. Acceptable
  for M0; not acceptable once a public demo is advertised.

## D-024 — AccountStore.Exists is Telnet glue only

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: WIRE-PROTOCOL Telnet must ask “새로 만드시겠습니까?” for unknown
  names. `Authenticate` must not enumerate.
- Decision: `Exists` is on `AccountStore` for the create prompt. Login
  failures remain a single `bad_credentials`.
- Consequences: WebSocket create/login stay as distinct `auth.create` /
  `auth.login` types and do not need `Exists`.

## D-025 — Supporting face 메스너 주사; zone prefixes reserved

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: NARRATIVE added a handler so 류겐 is not the mole (PR #16).
- Decision: **메스너 주사** is a supporting face, not one of the eight.
  Blood-point prefixes 먹골갱 / 재안암 / 가랑포 / 흰재보 / 잠수성 are
  reserved. WORLD may rename rooms; prefixes stay.
- Consequences: AINPC persona cards for the eight do not absorb the handler.
  한도규 disposal set (spare-and-use / spare-untrusted / exile / Circle
  execution) is accepted.

## D-026 — M0 accepted

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §5 done-when: two clients connect at once and talk.
  §9.5 gates that apply before M3/M4.
- Decision: M0 is complete on `dev` and is promoted to `main`. Next work
  is M1. Do not start M1 systems until this decision is on `main`.
- Consequences: Rooms, i18n keys, and YAML loaders may begin. Foreshadow
  `foreshadow:` keys may be attached to rooms from FS-001–020.

## D-027 — Realign `dev` to `main` after the M0 squash-promote

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: Promoting `dev` → `main` as one squash (PR #24) plus
  `delete_branch_on_merge` deleted `dev` and left the old tip unreachable.
- Decision: Recreate `dev` at `main` (`308aa2b`) before any M1 work.
  Future milestone promotes still squash to `main`, then immediately reset
  `dev` to the new `main` tip.
- Consequences: Do not open M1 PRs against a pre-promote `dev`.

## D-028 — Spawn room is `dalbitgol:gate`

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: M1 needs a single spawn. The market is a story beat (I-1), not
  a spawn — new users should walk into it.
- Decision: `EnterWorld` always places the player in `dalbitgol:gate`.
  Missing spawn is a boot failure.
- Consequences: WORLD must write that room. ENGINE must not invent a
  fallback plaza in Go.

## D-029 — i18n: `ko` required, `en` falls back to `ko`

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §7.7 wants keys from M1. Full English room prose for
  40 rooms would delay the village.
- Decision: `internal/text` holds system-message keys. Room YAML uses
  localized strings (`ko` required, `en` optional). Missing `en` uses `ko`.
  Default locale is `ko`.
- Consequences: M0 Korean literals move onto keys without wording changes.
  WORLD is not blocked on English.

## D-030 — `say` is room-scoped from M1

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: Global say made M0 talk easy. A walkable village with global
  chat would leak position-free shouting into later heat/stealth design.
- Decision: `Say` delivers only to roster entries with the same `RoomID`.
  The whole-server channel is reserved for M5 (전신망, paid).
- Consequences: M0 two-client smoke that assumed global say must sit in
  the same room (spawn) — still passes if both stay at the gate.

## D-031 — M1 accepted

- Date: 2026-08-15
- Status: accepted
- Decider: LEAD
- Context: PLAN.md §5 M1 done-when: a new user walks 달빛골 and reads
  descriptions. Style guide, schema, loader, look/move, 40 rooms, QA walk.
- Decision: M1 is complete. Promote `dev` → `main`, then reset `dev` to
  that tip (D-027). Next work is M2 (skills). Do not start M2 until this
  decision is on `main`.
- Consequences: CONTENT-STYLE is binding for all later zones. Spawn stays
  `dalbitgol:gate`. YAML rooms remain the only room source.

## D-032 — M2 has skills, not a combat-round engine

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: PLAN.md M2 lists "전투 6" skills. A full 3-second combat
  resolver would blow the line budget and is not required for the
  done-when ("two skill mixes feel like different games").
- Decision: Ship the six combat *skills* as practiceable ranks. Do **not**
  ship rounds, stance, cooldown, or hit locations. Those wait for a later
  GAMEPLAY issue (not M2).
- Consequences: QA proves difference via titles and practice/check
  outcomes, not via a duel.

## D-033 — No lock/down commands in M2

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: PLAN.md allows skill lock/down. Two review characters will not
  hit the 700 cap.
- Decision: Enforce the 700 / 300 caps in the gain function. Do not add
  `lock` / `down` verbs yet.
- Consequences: A later issue may add them without changing the curve.

## D-034 — Account sheet is 1:1 with the login

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: Need two durable characters for commissioner review.
- Decision: Skills, stats, bag, and equipment persist on the **account**.
  Still one body per login. Load on `EnterWorld`, save on `LeaveWorld`.
- Consequences: Reconnect must restore the sheet. In-memory store is
  enough for CI; Postgres JSON columns when `DATABASE_URL` is set.

## D-035 — M2 practice set is fourteen YAML skills

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: PLAN.md's 40-skill list is the long-term table. M2 asks for
  전투 6 + 생산 4 + 생활 4.
- Decision: The fourteen ids in SKILL-TABLE.md are the M2 set. Adding a
  fifteenth in this milestone is scope creep (park it in PROPOSALS).
- Consequences: Title rules and QA paths only mention those ids.

## D-036 — M2 accepted

- Date: 2026-08-16
- Status: superseded by D-040
- Decider: LEAD
- Context: PLAN.md §5 M2 done-when: two skill mixes feel like different
  games. Commissioner review is required.
- Decision: M2 is complete on `dev`. Promote to `main`, reset `dev`.
  Combat rounds remain out of scope (D-032).
- Consequences: Next work is M3 (living economy). Do not start M3 until
  this decision is on `main`. Commissioner play recipe is
  [COMMISSIONER.md](COMMISSIONER.md) §8.

## D-037 — Telnet default is :4001; busy ports step forward

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: On the commissioner's Mac, OrbStack already listens on :4000,
  so `go run ./cmd/server` failed with `bind: address already in use`.
- Decision: Default `YEOMYEONG_TELNET_ADDR` is **`:4001`**. If the
  requested TCP port is in use, the listener tries the next ports (up to
  10) and logs the bound address.
- Consequences: `telnet localhost 4001`. Override with the env var. Read
  the `telnet listening` log line if a fallback fired.

## D-038 — Telnet takes echo; passwords stay off the wire

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: Commissioner play on macOS `telnet localhost 4001` showed
  passwords in the clear and Hangul / `look` input as C0 garbage
  (`c^E^Sc^C`, `l'^Qk^K$`). The adapter dropped inbound IAC but never
  sent `WILL ECHO`, so the client kept local echo. BSD telnet plus a
  Hangul IME cannot compose syllables; that is a client limit, not a
  missing UTF-8 decoder (room cards already print Hangul).
- Decision: On accept, send `IAC WILL ECHO` and `IAC WILL SGA`. The
  server echoes typed bytes except at `암호:`. Backspace/DEL erase the
  last rune; other C0 controls are dropped. The banner adds
  `한글이 깨지면: nc localhost <port>`. Inbound IAC is still skipped —
  this is not a full option engine. WIRE-PROTOCOL is updated to match.
- Consequences: `nc localhost 4001` is the Hangul-safe classic client.
  macOS `telnet` is supported for ASCII movement and English verbs.
  Do not add linemode or a second input parser.

## D-039 — Commissioner playtest lives in COMMISSIONER.md

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: After M2 the commissioner could connect but had no written
  recipe for what to try or what kind of reply counts as a review.
  PLAN.md §9.5 requires commissioner review at M2, M4, and M6.
- Decision: [`docs/COMMISSIONER.md`](COMMISSIONER.md) is the playbook.
  Each milestone section states the PLAN §5 done-when in player language,
  a connect recipe, and a verdict template (bug / gate miss / taste).
  Official gates stay M2/M4/M6; M0/M1/M3/M5 use the same shape. Unshipped
  sections list judgment questions only — LEAD fills exact verbs when
  declaring that milestone complete, in the same file, same PR as closeout.
- Consequences: Closeout reports link this file instead of a one-off
  chat recipe. Taste notes are not a reject reason. A missing recipe
  block on a closeout is a LEAD defect.

## D-040 — M2 playtest failed; practice is an action, not 익히다

- Date: 2026-08-16
- Status: accepted
- Decider: LEAD
- Context: Commissioner M2 review: A and B did not feel like different
  people; `익히다 대장` is not spoken Korean; the loop was a button that
  raised a number. PLAN §5 done-when is unmet. D-036 is withdrawn.
- Decision: Stay on M2. Do not start M3. Player commands are YAML
  **verbs** (`두드리다`, `이야기하다`). Output is per-skill `gain`/`miss`
  prose with **no rank number**. The sheet prints `대장 미숙`, not
  `smith 15`. A title change is announced. Speech matches flag `salon`
  (다방 백야, 주막), not every `town` street. `익히다` / `practice`
  remain hidden aliases.
- Consequences: [COMMISSIONER.md](COMMISSIONER.md) §8 uses the new verbs.
  Re-review M2 after this lands. A second fail stays on M2.

## D-041 — Player-facing Korean is everyday speech

- Date: 2026-08-16
- Status: accepted; room diction superseded by D-045
- Decider: LEAD
- Context: Commissioner: `그런 숙련은 없습니다` and many other lines are
  not how people talk. D-016 froze M0 wording; that freeze is lifted
  for player-facing strings.
- Decision: System lines, prompts, skill names, verbs, bands, and
  practice/title announce use ordinary spoken Korean. Particle `을/를`
  is chosen by the last Hangul batchim, not `을(를)`. Protocol *codes*
  (`rate_limited`) stay English; the visible sentence is Korean.
  Room descriptions stay literary (CONTENT-STYLE) — **lifted by D-045**.
  WIRE-PROTOCOL transcripts follow the new wording.
- Consequences: Tests assert the new sentences. `그대`, `장부`,
  `호칭`, `주손`, `을(를)`, and English `rate_limited` as a player
  line are defects.

## D-042 — M3 is a livelihood slice, not the full §3.4 economy

- Date: 2026-08-31
- Status: accepted
- Decider: LEAD
- Context: PLAN.md M3 lists gathering, crafting, regional prices,
  transport risk, smuggling, player shops, and three-axis reputation.
  COMMISSIONER.md §9 done-when is gather → craft → sell/carry with no
  combat. Shipping the whole section would explode the line budget.
- Decision: Ship a vertical slice: YAML gather nodes, YAML recipes on
  the four M2 craft skills, two markets (달빛골 / 솔골) with live
  stock, a purse in 냥, and one gather skill `forage` (채집, verbs
  `캐다` `줍다`). Practice verbs (`두드리다`, `썰다`, …) still do not
  consume or produce. `만들다` is the consume/produce command.
  Splitting 채광/약초/낚시, player shops, smuggling, and reputation
  wait. Do not mark the M3 checkbox done.
- Consequences: Titles `못 벼리는 사람` (smith 8), `보부상` (haggle 8),
  `달빛골의 채집꾼` (forage 8) sit under the M2 15-rank titles.
  Packages are `internal/craft` and `internal/economy`. The loop is
  still the only writer.

## D-043 — Posted prices come from stock and demand, not a shop NPC

- Date: 2026-08-31
- Status: accepted
- Decider: LEAD
- Context: COMMISSIONER.md asks whether prices feel dynamic. Hardcoded
  buy/sell lists would be a fixed stall.
- Decision: Each market lists `base`, `stock`, `target`, `demand` in
  YAML. The quote is
  `max(1, round(base * demand * (1 + 0.5 * (target - stock) / max(target,1))))`
  with a 0.2 factor floor. Selling raises stock (price eases). Buying
  lowers it. Every 10 loop ticks, stock walks one step toward `target`
  (quiet NPC flow). `시세` prints the Korean stall name, a 냥 figure,
  and a flavor clause (오늘은 흔하다 / 값은 평소랑 비슷하다 / 오늘은
  귀하다). No English player line; no rank numbers.
- Consequences: The same good must quote differently at 달빛골 vs 솔골
  at boot. Tests lock that spread and that a sell drops the next quote.

## D-044 — Checkpoint toll when carrying bulk; no player shops

- Date: 2026-08-31
- Status: accepted
- Decider: LEAD
- Context: PLAN.md wants transport risk on the trade road. Full bandit
  encounters or player storefronts are out of slice scope.
- Decision: Rooms may take flag `checkpoint` (달빛골 정거장, 솔골
  고갯길). Walking **into** one while the bag holds **4+** units of
  any market-listed good rolls `P=0.35`. On a hit, pay 2냥 if the
  purse can; otherwise one traded unit is left at the post. No combat,
  no heat, no smuggling skill check. Player shops are not in this
  slice.
- Consequences: The 보부상 loop is gather/craft in 달빛골, walk east
  from the station, sell dearer in 솔골 — with a chance the post
  skims the load.

## D-045 — Rooms and remaining live copy are everyday Korean

- Date: 2026-08-31
- Status: accepted
- Decider: LEAD
- Context: D-041 converted system lines but left rooms literary
  (CONTENT-STYLE). Live rooms, items, skill verbs, bands, quotes, and
  tolls still used 문어체, 고어, and rare 한자어 (윗도리, 문설주,
  고시문, 교정쇄, 분견대, 품귀, 언변, 숙련할 기술 중에…). Players
  should read the world like contemporary daily speech, not a 1920s
  textbook, while the occupation setting stays.
- Decision: All player-facing copy — rooms, items, NPCs, skills, help,
  commands, and system lines — is everyday modern Korean. Setting
  nouns stay (달내, 무쇠 제국, 새벽회, 지맥, 쇠말뚝, 관아, 주재소,
  등사기, 통행증). Prefer 가게/시장/길/경찰/검문/돈/가방/만들다/캐다.
  Commands are common verbs (`보다`, `캐다`, `만들다`, `팔다`, `사다`,
  `가방`). Rank bands print 초보/익숙/능숙/노련/명인/달인. Stats print
  힘/손재주/맷집/재치/감응/평판 (감응 stays as the 지맥 sense). The
  sensory room recipe (2–4 sentences, ≥2 senses, one object) is
  unchanged. PLAN.md historical essays are not rewritten.
- Consequences: CONTENT-STYLE voice is everyday, not literary. Tests
  assert the new sentences. `엿장수 가위질`, `고시문`, `교정쇄`,
  `분견대`, `언변` as a typed verb, and remaining 문어체 in live YAML
  or Go player strings are defects.

## D-046 — Optional `talk.when` flag gates

- Date: 2026-09-01
- Status: accepted
- Decider: LEAD
- Context: After the warehouse pack examine (`examined:gangpo-pack`,
  FS-016), Act I needs T0 dialogue that notices the pack without a
  second NPC verb or a quest VM. Engine talk only had `first` /
  `second`.
- Decision: NPC YAML may list optional `talk.when` entries. Each
  entry is `{flag, ko}` (`en` optional). On `대화`, pick the **first**
  `when` entry whose `flag` is present and **>0** on the player sheet;
  otherwise keep today's first/second. Still set `{id}_talked` on the
  first talk, even if a when-line was used. `talk.first` and
  `talk.second` stay required. Empty or missing `when.flag` is a load
  error. The flag string is an opaque sheet key; do not require it to
  exist at load time.
- Consequences: CONTENT-SCHEMA documents the field. Loader and the
  single-writer loop implement it. Do not grow `when` into a predicate
  language in this slice. Unknown flag names are allowed (they simply
  never match until something writes that sheet key).

## D-047 — First market sale gates Act I courier soft-hook

- Date: 2026-09-01
- Status: accepted
- Decider: LEAD
- Context: P0 first-session loop needs livelihood between warehouse pack
  examine and concrete courier-trail clues. Pack examine alone opened the
  detective path too early.
- Decision: After a successful market `팔다` that pays nyang, set sheet flag
  `first_market_sale` once (also acceptable if `nyang >= 2` after that sell).
  Concrete trail object `when` lines, 청람/오씨 concrete `talk.when`, and
  장터 board/room `when` descriptions gate on this flag. Pack examine only
  bridges to 시세/캐다/만들다. Soft-hook durable surfaces are 청람 talk
  third and 장터 안내문/묘사 — no 월향 NPC/YAML in this slice. Object and
  room YAML may list optional `when` entries with the same pick rule as
  `talk.when` (D-046).
- Consequences: CONTENT-SCHEMA documents object/room `when`. Smoke and
  story tests assert bland-before / concrete-after sale.

