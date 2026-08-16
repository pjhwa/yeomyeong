# Questions for the commissioner

Only items that need Jerry's judgment live here. LEAD decides everything else
and writes it in [DECISIONS.md](DECISIONS.md). How to play a finished
milestone, and how to send that review, is [COMMISSIONER.md](COMMISSIONER.md).

Format:

```
## Q-NNN — short title
- Date: YYYY-MM-DD
- Status: open | answered
- Needed by: milestone or date
- Question:
- Options (if any):
- Answer:
```

---

## Q-001 — Demo and Discord URLs

- Date: 2026-08-15
- Status: open
- Needed by: M1 (demo GIF + `docker compose up`); Discord can wait until public promotion (M4)
- Question: What URLs should replace the `#` placeholders in README.md
  (`Play the demo`, `Discord`)?
- Options: leave as `#` until they exist (current default)
- Answer:

## Q-002 — Keep the repository public before M1?

- Date: 2026-08-15
- Status: open
- Needed by: whenever you care about first impressions
- Question: PLAN.md §10.8 prefers public-after-M1. The GitHub repo is already
  public (D-008 keeps it that way). Do you want it flipped private until M1
  ships a walkable village?
- Options:
  1. Stay public (LEAD default, D-008)
  2. Private until M1 complete
- Answer:

## Q-003 — Dedicated GitHub identities for sub-agents?

- Date: 2026-08-15
- Status: open
- Needed by: optional; current process works without this (D-003)
- Question: Should ENGINE / NARRATIVE / etc. get bot or machine-user accounts
  so GitHub assignees and required reviews map 1:1 onto agents?
- Options:
  1. No — labels + `Owner:` line are enough (current)
  2. Yes — create bot accounts and CODEOWNERS later
- Answer:

## Q-004 — Code of Conduct / security contact email

- Date: 2026-08-15
- Status: open
- Needed by: M1 public interest, or the first external contributor
- Question: `CODE_OF_CONDUCT.md` and `SECURITY.md` currently point at GitHub
  private advisories and the repository owner. Do you want a dedicated email
  (e.g. `conduct@…`, `security@…`)?
- Options:
  1. GitHub-only is enough for now (current)
  2. Publish an email once you have one
- Answer:
