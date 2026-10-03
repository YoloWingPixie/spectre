# ATC role: specification (SDD)

This folder is the spec-driven-development specification of Overwatch's
air traffic control role, written for line-by-line review. It turns the
decisions of `docs/atc-design.md`, `docs/atc-procedures.md`,
`docs/atc-terminal-procedures.md` (with its §10, the canonical procedure
encoding), `docs/f10-map-layer.md` and `docs/atc-gaps.md` (plus the scaffold
in `internal/atc`) into numbered, testable requirements, an architecture,
an implementation plan, acceptance scenarios and a list of open decisions.
Nothing here changes code.

## 1. How to read it

| File | What it holds | Read it when |
|---|---|---|
| `README.md` | conventions, legends, counts | first |
| `requirements.md` | 644 requirements in EARS form, grouped by area, each with acceptance criteria, exact phraseology, source, priority, dependencies and design elements | reviewing behaviour |
| `scenarios.md` | 46 scripted radio exchanges (time, position, speaker, line, checks) and the coverage matrix | checking behaviour end to end |
| `design.md` | packages and Go types (with the scaffold changes), data flow, state machines, configuration schema, interfaces, degraded modes, budgets, traceability | reviewing the architecture |
| `tasks.md` | 207 tasks (≤ ~1 day each) in 9 milestones, each with requirements, files, tests and done criteria; requirement → task index | planning |
| `open-decisions.md` | 95 decisions (all resolved) with options and recommendations, and the requirements each blocks | deciding |

Suggested order: skim the area index of `requirements.md`, read the MVP
requirements (priority `MVP`) with scenarios S-01…S-21 beside them, then
`open-decisions.md`, then
`design.md` and `tasks.md`.

## 2. Conventions

### 2.1 IDs

| Kind | Form | Example |
|---|---|---|
| Functional requirement | `ATC-FR-<AREA>-<nnn>` | `ATC-FR-TWR-012` |
| Non-functional requirement | `ATC-NFR-<AREA>-<nnn>` (PERF, OBS, TST, DEG) | `ATC-NFR-PERF-001` |
| Scenario | `S-<nn>` (variants `S-02b`) | `S-06` |
| Task | `T-M<milestone>-<nn>` | `T-M1-17` |
| Open decision | `OD-<nnn>` | `OD-047` |
| Design element | `DE-<NAME>` | `DE-SEQ` |

In `scenarios.md` and `tasks.md`, **Covers** lists abbreviate IDs: an ID
without the `ATC-FR-` / `ATC-NFR-` prefix takes the last prefix written, and
`CNF-001…006` is a range.

IDs are stable: a removed requirement keeps its number retired; new ones
take the next free number in their area.

### 2.2 Requirement fields

`Priority`, `Status`, `EARS`, `Acceptance`, `Source`, `Depends`, `Design`.
EARS patterns used: ubiquitous ("The … shall …"), event-driven ("When …,
the … shall …"), state-driven ("While …"), unwanted behaviour ("If …, then
the … shall …") and optional feature ("Where …").

Unless an acceptance criterion says otherwise, phraseology examples use
the FAA military (USAF) variant at Nellis, runway 21L, wind 210/8,
altimeter 29.92, and the flight "Viper 1-1" (F-16C 2-ship).

## 3. Source tags

| Tag | Document |
|---|---|
| [D §x] | `docs/atc-design.md` (D1–D12 = its Decisions 1–12) |
| [P §x] | `docs/atc-procedures.md` |
| [T §x] | `docs/atc-terminal-procedures.md` |
| [G §x] | `docs/atc-gaps.md` (gap n = row n of its §2 table) |
| [MC] | `docs/mission-config.md` |
| [CT §x] | `docs/atc-civil-traffic.md` (Overwatch-managed civil traffic) |

## 4. Counts

Requirements: **644** (604 functional, 40 non-functional) —
**200 MVP**, **382 P2**, **62 P3**. Status: 575 decided,
69 derived, 0 open (77 requirements were blocked by one of 72
decisions, all resolved 2026-10-01 by accepting the recommendations).

| Area | Code | MVP | P2 | P3 | Total |
|---|---|---|---|---|---|
| Identity and sessions | IDN | 9 | 2 | 1 | 12 |
| Facilities, staffing, frequencies | FAC | 10 | 10 | 1 | 21 |
| Airspace and jurisdiction | ASP | 11 | 6 | 1 | 18 |
| Clearance delivery | CLR | 1 | 9 | 1 | 11 |
| Ground and taxi | GND | 12 | 2 | 1 | 15 |
| Tower: runway operations | TWR | 18 | 3 | 3 | 24 |

Scenarios: 46 (S-01…S-21 are the MVP set; every MVP requirement is
covered, see the matrix at the end of `scenarios.md`).
Tasks: 207 (M0 16, M1 42, M2 33, M3 13, M4 7, M5 29, M6 21, M7 25, M8 21); every requirement is covered by at least one task.
Open decisions: 0 of 95 (all resolved; kept for traceability).

Generated tables (the area index, the traceability table of `design.md`,
the coverage matrix of `scenarios.md`, the requirement → task index of
`tasks.md`, the "Blocks" lines of `open-decisions.md` and the counts above)
were derived mechanically from the requirement fields and the **Covers**
lines; edit the sources, not the tables.

## 5. Review checklist

- Each EARS sentence names one trigger and one facility response.
- Each acceptance criterion is checkable by a unit test or a textsim script line.
- Every phrase example goes through `speech/phraseology` (no digits in phrases).
- No requirement lets a facility speak about an aircraft outside its airspace (except the guard boundary warning and traffic calls to its own flights, ATC-FR-AIT-014).
- Nothing is ever transmitted to an AI aircraft; aircraft inside hostile airspace are never called as traffic.
- No requirement separates aircraft of the same flight.
- `derived` requirements and every `open → OD` choice are the places to push back.
