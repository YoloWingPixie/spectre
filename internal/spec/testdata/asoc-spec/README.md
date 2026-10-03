# ASOC role: specification (SDD)

This folder is the spec-driven-development specification of Overwatch's
**ASOC** agency: the Air Support Operations Center / battle-management
agency that owns the flow of a combat sortie and the airspace it flies in.
It is written for line-by-line review in `cmd/specview`, in the format of
`docs/atc-spec`. Nothing here changes code.

**The decision (user brief, 2026-10-02).** A new agency type `asoc` is
added next to the JTAC ("Landshark", `internal/jtac`), the AWACS
("Darkstar", `internal/awacs`), ATC (`internal/atc`, removable) and the AI
flights (`internal/aipilot`). The ASOC does check-in / check-out and
tasking against a mission-defined tasking order (ATO), owns the airspace
control order (ACO: kill boxes, ROZs, AR tracks and anchors, CAP stations,
gateways, FSCMs), clears flights into and out of it, and routes them: to
a JTAC for CAS, to the AIC (Darkstar) for the air-to-air picture, to
tankers, and back to ATC at a gateway.

**The constraint (user, 2026-10-02, overrides the first brief).** The
AWACS module stays **exactly as it is**. Nothing moves out of Darkstar;
its behaviour, NLU and responsibilities do not change. The ASOC is purely
additive: it hands flights to and from Darkstar through the existing
interfaces and frequencies ("contact Darkstar <freq>", the way ATC's
operating-area release already does), a flight that checks in with
Darkstar directly is handled by Darkstar exactly as today, and every
shared-core seam the ASOC reads is read-only and neutral, with no
behaviour change in `internal/awacs`. "AIC" in this spec is simply
Darkstar's existing role (air intercept control); there is no refactor.

## 1. How to read it

| File | What it holds | Read it when |
|---|---|---|
| `README.md` | decision, conventions, legends, counts | first |
| `requirements.md` | doctrine grounding, interplay with Darkstar, then 176 requirements in EARS form with acceptance criteria, exact phraseology, source, priority, dependencies and design elements | reviewing behaviour |
| `design.md` | packages and Go types, shared-core seams, ATO / ACO data model and drawing grammar, NLU pack, phrase catalogue, full example radio dialogues, configuration, removal | reviewing the architecture |
| `scenarios.md` | 11 acceptance scenarios (textsim scripts) with checks and coverage | checking behaviour end to end |
| `tasks.md` | 38 tasks in 6 milestones with tests, files and done criteria; live checklists | planning |
| `open-decisions.md` | 17 decisions (16 open for the user), each with options and a recommendation | deciding |

Suggested order: §1–§3 of `requirements.md` (doctrine, what the ASOC is
and is not, interplay with Darkstar), then `open-decisions.md`, then the
example dialogues of `design.md` §9, then the MVP requirements.

## 2. Conventions

### 2.1 IDs

| Kind | Form | Example |
|---|---|---|
| Functional requirement | `ASOC-FR-<AREA>-<nnn>` | `ASOC-FR-CHK-002` |
| Non-functional requirement | `ASOC-NFR-<AREA>-<nnn>` (PERF, OBS, TST, DEG) | `ASOC-NFR-PERF-001` |
| Scenario | `S-<nn>` | `S-03` |
| Task | `T-M<milestone>-<nn>` | `T-M1-04` |
| Open decision | `OD-<nnn>` | `OD-001` |
| Design element | `DE-<NAME>` | `DE-ACO` |

In `scenarios.md` and `tasks.md`, **Covers** lists abbreviate IDs: an ID
without the `ASOC-FR-` / `ASOC-NFR-` prefix takes the last prefix written.
IDs are stable: a removed requirement keeps its number retired.

### 2.2 Requirement fields

`Priority`, `Status`, `EARS`, `Acceptance`, `Source`, `Depends`, `Design`,
as in the ATC spec. EARS patterns: ubiquitous ("The ASOC shall …"),
event-driven ("When …"), state-driven ("While …"), unwanted behaviour
("If …, then …") and optional feature ("Where …").

Unless an acceptance criterion says otherwise, examples use the **Nevada
example** of `design.md` §9.1: ground ASOC **Overlord** on 264.0 AM (a
CRC radar site, unit `CRC-1`), AIC **Darkstar** on 251.0 AM (E-3, unit
`AWACS-1`), JTAC **Landshark** on 238.5 AM, tanker **Texaco 1** (KC-135,
AR track Shell, FL220–240, 276.1, TACAN 31Y), kill boxes **88AY** and
**88AZ** (surface to 18,000 ft), ROZ **Reaper**, CAP **Tiger**, contact
point **Bobcat**, gateways **Coyote** and **Tikaboo**, ATC **Nellis
Departure / Approach** 124.95 and **Los Angeles Center** 133.95, and the
flights **Hawg 1** (A-10C 2-ship, CAS, mission 5101), **Viper 1** (F-16C
2-ship, kill box interdiction, mission 4101), **Eagle 2** (F-15C 2-ship,
escort, mission 2101) and the AI flight **Uzi 1** (F-16C, CAP).

## 3. Source tags

| Tag | Document |
|---|---|
| [UB] | the user's brief of 2026-10-02 (the ASOC decision) |
| [UC] | the user's constraint of 2026-10-02 (AWACS unchanged, ASOC additive) |
| [UR] | standing user rules (voice identity, shared logic, no scope widening) |
| [AC §x] | `docs/atc-area-control.md` (§13 operating areas) |
| [ML] | `docs/mission-lifecycle.md` |
| [SM §x] | `docs/state-model.md` |

## 4. Counts

Requirements: **176** (164 functional, 12 non-functional) — **77 MVP**,
**84 P2**, **15 P3**. Status: 54 decided, 105 derived, 17 open (each with a
recommendation in `open-decisions.md`).

Scenarios: 11. Tasks: 38 (M0 6, M1 9, M2 8, M3 5, M4 6, M5 4).
Open decisions: 17 (16 open, OD-017 resolved: specview generalised, T-M0-06 done).

## 5. Viewing in specview

```sh
go run -tags nolibopusfile ./cmd/specview -port 8765 -dir docs/asoc-spec   # http://127.0.0.1:8765/
```

The viewer takes the `ASOC` prefix from the requirement IDs and the page
titles from this README's title; review files go to
`docs/asoc-spec/review/`, separate from the ATC spec's.

## 6. Review checklist

- No requirement changes Darkstar's behaviour, NLU or code (`ASOC-FR-AIC-001`).
- No requirement lets the ASOC give an AIC call (picture, BRAA, declare, commit, threat) or a JTAC call (nine-line, cleared hot).
- No requirement lets the ASOC address a flight that has not checked in with it.
- Each shared concern names the one core utility it uses.
- `derived` requirements and every `open → OD` choice are the places to push back.
