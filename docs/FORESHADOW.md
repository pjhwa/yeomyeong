# Foreshadow ledger

NARRATIVE owns this file. Every clue is registered **in the same change that
plants it**. Tightness is a recovery rate, not a vibe.

Quality gate from M4 onward (PLAN.md §9.5): **zero overdue unrecovered rows**.

## Rules

1. Write the ending first (`docs/STORY-BIBLE.md`). Derive rows backwards.
2. Every reversal is shown at least three times, in three forms (dialogue /
   object / newspaper), before it pays off.
3. After payoff, the original locations must still read as planted, not
   retconned.
4. Status values: `planted` | `reinforced` | `recovered` | `dropped`.
5. `dropped` requires a LEAD note in DECISIONS.md. Do not silently delete a row.

## Ledger

Design-level plants. No YAML rooms yet. WORLD attaches `foreshadow:` keys
when zones are written. Revisit after payoff must still find the original
form.

| ID | Clue | Planted at | Forms (min 3) | Recovers | Status |
|---|---|---|---|---|---|
| FS-001 | 한벽일보 typesetting substitutions are a Circle-site cipher in 한도규's hand | `dalbitgol:market` (board); `dalbitgol:printshop`; `hanbyeok:newsroom` | newspaper: substitution pattern on the market copy; object: the same worn slugs in his case; dialogue: he blames "닳은 활자" and offers a sick-day that matches a quiet issue | Act II (II-8) | planted |
| FS-002 | 한도규's alibi does not hold | `dalbitgol:cafe-baekya`; `dalbitgol:printshop`; `dalbitgol:station` | dialogue: café girl saw him by the station the night he "never left the press"; object: a second time-clock card stamped at the 주재소 wicket; newspaper: a night blotter item about a "print-shop errand" that the shop ledger does not list | Act II (II-8) | planted |
| FS-003 | 주재소 paper in the print-shop waste | `dalbitgol:printshop` | object: ink-stained receipt in the waste bin; dialogue: apprentice jokes that 도규 washes his hands twice after night jobs; newspaper: a classified "lost: one stamped chit" the 관아 never follows up | Act II (II-8) | planted |
| FS-004 | 한도규 is coerced; family held as surety | `dalbitgol:printshop` (drawer); `dalbitgol:inn`; `hanbyeok:surety-house` (exterior only until II-8) | object: child's hair ribbon and wages unspent; dialogue: he refuses a winter coat he can afford; newspaper: a bland "protective residence" notice with two given names blacked but countable | Act II (II-8 / II-9) | planted |
| FS-005 | 서월향 is 백야 | `dalbitgol:cafe-baekya` | dialogue: she answers to a nickname she did not give; object: the white lamp lit only when the shop is "closed"; newspaper: a 철력 6 society note that 다방 백야 "opened under a lady who would not be photographed" | Act I (I-10) | planted |
| FS-006 | Never-order guests at 백야 are Circle | `dalbitgol:cafe-baekya` | dialogue: 월향: "그 손님은 차를 안 시켜"; object: saucers with no stain, coins left the same each night; newspaper: a humor column on "the driest café in the province" | Act I (I-8 / I-10) | planted |
| FS-007 | 청람 was an imperial scholar | `dalbitgol:school` | object: a plate he keeps face-down; dialogue: he corrects an Empire primer then stops; newspaper: 철력 −2 academic list under the courtesy name **월송** | Act II (II-3) | planted |
| FS-008 | 청람 designed the stakes (as regulators) | `dalbitgol:school`; first socket (`meokgol:shaft` or `jaeanam:court`) | object: a faded five-needle sketch with vent marks, not hearts; dialogue: the Act I flinch at 쇠말뚝; newspaper: a 15-year-old technical abstract on "bleeding a fevered vein" signed 월송 | Act II (II-3) | planted |
| FS-009 | Five blood-points exist and are sick | `dalbitgol:shrine`; `dalbitgol:market` | dialogue: 곱단 points five directions before she has names; object: five rusted nails on her altar; newspaper: a shipping/survey series that happens to name mine, temple, port, fortress, drowned basin | Act II (II-2…II-7) | planted |
| FS-010 | The stakes feed a single thing in the Empire (무쇠 심장) | `dalbitgol:school`; `hanbyeok:cheoltap` chapel; `hanbyeok:newsroom` | object: 청람's sketch joins five lines to one circle he will not name; dialogue: a chapel attendant bows to a pipe; newspaper: "continental foundry output exceeds all estimates" on hymn-festival week | Act III (III-2) | planted |
| FS-011 | 쇠노래 is faith and machinery, not weather | `dalbitgol:well`; `dalbitgol:inn` radio; `hanbyeok:cheoltap` | object: well water tastes of iron when the radio hour starts; dialogue: 곱단: "제사가 아니라 파이프야"; newspaper: programme listing "시민 쇠노래" beside foundry notices | Act III (III-2) | planted |
| FS-012 | Stake sockets are also 분맥 fuse-wells | `meokgol:shaft` (vent stamp); `dalbitgol:school` (vent marks on the sketch); `hanbyeok:archive` | object: socket lip stamped VENT / 배기, later packed as a charge seat; dialogue: 청람, after II-3, says the vents should have been empty; newspaper: a dry "civil-protection drill" at all five survey posts on the same night | Act III (III-3) | planted |
| FS-013 | 류겐 doubts procedure-as-justice | `dalbitgol:market`; `dalbitgol:station`; `dalbitgol:well` | dialogue: he refuses a well search because it is not on the warrant; object: a returned toy in the lost-property box, tagged in his hand; newspaper: an unsigned letter to the editor on "procedure that forgets the street" | Act III (류겐 branch) | planted |
| FS-014 | The market board is already a battlefield | `dalbitgol:market` | object: 관아 notice with a torn corner; newspaper: the error-copy of 한벽일보 sold under it (ties FS-001); dialogue: a stall-keeper who will not look at the board after noon | Act I (I-1 / I-9) | planted |
| FS-015 | 곱단 feels ley-sickness first | `dalbitgol:shrine`; `dalbitgol:well`; `dalbitgol:market` | dialogue: she will not walk the well path; object: shrine bowl filmed with rust after rain; newspaper: a health brief on "nerveless harvests" she spat on | Act II–III | planted |
| FS-016 | Vanished peddler 강포 was a Circle courier | `dalbitgol:warehouse`; `dalbitgol:warehouse-lane`; `dalbitgol:packing-shed`; `dalbitgol:school` (청람 when); `dalbitgol:market`; `dalbitgol:cafe-baekya` | object: false-bottom straw and a saucer token in his pack remnant; object: wheel ruts in warehouse-lane; object: cargo chit on an empty 가마니 strap in packing-shed; dialogue: 월향 goes still at his name; dialogue: 오씨 점원 / 청람 if pack examined; newspaper: a missing-person line that calls him "unlicensed itinerant" and nothing else | Act I (I-2 / I-8) | planted |
| FS-017 | The 달빛골 cell is already mapped (검은 밤 incoming) | `dalbitgol:station`; `dalbitgol:printshop`; `dalbitgol:inn` | object: a carbon roster with nicknames, not legal names, in 메스너 주사의 desk; dialogue: 한도규 asks who will sleep at the shop "tomorrow"; newspaper: an extra "sanitation inspection" notice dated the morning after the night | Act I (I-9) | planted |
| FS-018 | 안나 is a third force (international opinion) | `dalbitgol:inn`; `hanbyeok:cable`; `hanbyeok:newsroom` | dialogue: she asks the question 한벽일보 will not; object: a cable flimsy stamped 북해통신; newspaper: her piece runs abroad and returns as a redacted "foreign slander" item | ongoing | planted |
| FS-019 | 한벽 철탑 is the intake valve, not a chapel | `hanbyeok:cheoltap`; `dalbitgol:school`; `garangpo:pier` | object: chapel pews face a covered grate; dialogue: a stevedore at 가랑포 on "pipes that hum toward the tower, not the sea"; newspaper: a cutaway "modern civic tower" that omits the basement | Act III (III-2) | planted |
| FS-020 | 최만석 sells both ways and can be bought for a war-chest | `dalbitgol:warehouse`; `dalbitgol:market`; `hanbyeok:newsroom` | object: double ledger (관아 tariffs / Circle rice); dialogue: he names a price before a flag; newspaper: a society note that 만석상회 underwrote both a hymn-festival float and a "winter soup kitchen" | Act III (최만석 branch) | planted |

## ID convention

`FS-NNN`, zero-padded, allocated by NARRATIVE. Gaps are fine. Do not reuse an
ID after `dropped`. Next free: `FS-021`.

Supporting face **메스너 주사** (handler) appears in FS-017. He is not one of
the eight and must not absorb 류겐's recovery.

## Milestone audit

| Milestone | Overdue unrecovered | Auditor | Date |
|---|---|---|---|
| M0 | n/a (gate starts M4) | — | — |
