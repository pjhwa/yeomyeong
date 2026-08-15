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
  `YEOMYEONG_TELNET_ADDR` (`:4000`), `YEOMYEONG_WS_ADDR` (`:8080`),
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
