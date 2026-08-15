## Purpose

<!-- One or two sentences. What player-facing or operator-facing problem does this solve? -->

## Summary of changes

<!-- Bullet list. Name packages, YAML files, and interface docs. -->

## Related issue

Closes #

## Test plan

<!-- Commands a reviewer can run. Paste a terminal transcript for net/auth work. -->

- [ ] `go test ./...` (skip if this PR has no Go)
- [ ] Content linter clean (skip if this PR has no `content/`)
- [ ] Manual check:

## Non-negotiable principles

False checks are a process-severity defect. Untick and explain in the notes if a box does not apply.

- [ ] **Single-writer.** World state is mutated only by the single game-loop goroutine. No new mutexes on world data.
- [ ] **No levels / classes / XP bars.** Growth remains use-based skill (0–100) with total cap 700.
- [ ] **Three solutions.** Every major obstacle added or changed has force, craft/skill, and social paths. No combat-only gate.
- [ ] **Legal and ethical guardrails.** No real people or organizations, no foreign proper nouns, no direct torture or massacre.
- [ ] **Data-driven content.** No rooms, NPCs, or items hardcoded in source. They live in YAML.

## Notes

<!-- Interface contract links, follow-up issues, screenshots, terminal logs. -->

## Size

- Changed lines (approx):
- Split required if over 1,000. Target 600.
