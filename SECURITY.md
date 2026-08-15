# Security Policy

## Supported versions

YEOMYEONG is pre-alpha (milestone M0). Security fixes land on `main` only.
There are no long-lived release branches yet.

## What to report

Please report anything that lets a player:

- Authenticate as someone else, or recover another account
- Mutate world state from outside the single game-loop writer
- Inject into Lua scripts or LLM prompts (NPC dialogue, tool calls)
- Bypass rate limits, input validation, or the dialogue tool-call sandbox
- Read another player's private messages, session tokens, or password hashes
- Exhaust the process (unbounded allocations, unchecked Lua CPU, LLM cost bombs)

Gameplay balance, economy inflation, and lore mistakes are ordinary issues, not
security reports.

## How to report

Do **not** open a public issue.

1. Open a [private vulnerability advisory](https://github.com/pjhwa/yeomyeong/security/advisories/new)
   on this repository, or
2. Contact the repository owner via GitHub if advisories are unavailable.

Include:

- A short description of the impact
- Steps to reproduce, or a proof of concept that does **not** include an exploit payload against a live host
- Affected commit SHA or milestone, if known
- Whether you plan to disclose independently, and on what timeline

## What happens next

- LEAD acknowledges within 72 hours.
- We confirm, patch on a private branch if needed, and credit you unless you ask otherwise.
- Passwords use argon2id. Session tokens are unguessable. User input is validated on the server. Lua has no filesystem or network access and a per-command CPU budget. LLM output never writes game state directly — tool calls are server-validated.

## Public disclosure

Please give us a reasonable window to ship a fix before publishing details.
We will not request more than 90 days for issues reported in good faith.
