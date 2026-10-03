# ATC role: implementation plan

Ordered tasks, each about a day or less, grouped into milestones. A task
lists the requirements it implements or contributes to (**Covers**, same
abbreviation rule as `scenarios.md`: an ID without prefix takes the last
prefix), the files it touches, the tests to write (unit + textsim), and
when it is done. Every requirement of `requirements.md` appears in at least
one task (checked by the generator of `README.md`).

Rules for every task (definition of done, common part):
- `make check` passes (`-tags nolibopusfile`, `-race`); no network in unit tests.
- New behaviour has table-driven unit tests in the package that owns it; no edits to other packages' tests.
- Spoken text only through `internal/atc/phrase` → `speech/phraseology`.
- Contract changes to shared packages (`controller`, `radio`, `sim`, `world`, `nlu`, `speech`, `mappack`) only after the orchestrator agreed (design §7.14); additive where possible.
- The task's requirement IDs are listed in the commit body.

Milestones:

| Milestone | Goal | Exit |
|---|---|---|
| M0 Foundations | contracts, NLU pack, phrase renderer, world/weather adapters, config skeleton, registry, airspace, test tooling | textsim `-role atc` answers a radio check at the Nellis fixture |
| M1 MVP | tower + ground at one field (Nellis) with the overhead, formations, sequencing, go-arounds, readbacks, emergencies, NORDO, jurisdiction | scenarios S-01…S-21 pass; live checklist M1 |
| M2 Approach / departure radar | split positions, handoffs, clearance delivery, radar contact, vectors, flight following, AI traffic, Class B/C/D, hostile boundary, arresting gear, F10 map core with the airspace, hostile and AWACS layers | S-22, S-33, S-34, S-37, S-40, S-41 pass; live checklist M2 |
| M3 ATIS and NOTAMs | ATIS broadcast, letters, NOTAMs with effects, configurable remarks, runtime API | S-24, S-35 pass |
| M4 Guard | outreach on own frequency and guard, limits, IDENT, combat hand-off | S-23 passes |
| M5 Terminal procedures | leg model, canonical encoding (OPE) and the DCS compiler, generator, selection by capability, vectorizer, holding, missed approaches, conformance, safety alerts, PAR/ASR, fighter VFR routes, VFR / fix / procedure map layers | S-25, S-30, S-31, S-32, S-43, S-45 pass |
| M6 Carrier | case, marshal, Case I pattern and spin/break, Case II/III, CCA, bolters, Delta, departures | S-26…S-29, S-38 pass |
| M7 ICAO / Navy / civil variants | ICAO phraseology and rules, Navy land-field variant, civil traffic, Overwatch-managed civil traffic (P2 core; extras in M8), area control (FIR / ARTCC) | S-36, S-39, S-42, S-46 pass |
| M8 Polish and tooling (P3) | civil traffic extras (later / only if needed), SFO, helicopters, CIFP/AIXM packs, procedure derivation (IR, parsers, overlay, `derive`, `diff`, pack format 2), kneeboards, admin layer, map legend and menu | S-44 passes; per task |

---

## M0 · Foundations

#### T-M0-01 · Scaffold type additions
- **Covers:** ATC-FR-IDN-005, PHR-001, HND-001, ATI-001, ASP-013
- **Files:** `internal/atc/{agency.go,flight.go,handoff.go,atis.go,airspace.go,taxi.go,runway.go,doc.go}`
- **Tests:** unit: existing `atc_test.go` still passes; new `TestVariantService`, `TestRunwayInUseOpen` (closed ends excluded), `TestNeedsUpdateSplitRemarks`.
- **Done when:** design §3.2 additions compile; no behaviour change to existing helpers.

#### T-M0-02 · Clock, health and log helpers
- **Covers:** ATC-NFR-DEG-009, OBS-001
- **Files:** `internal/atc/{clock.go,health.go,obs.go}`
- **Tests:** unit: `TestNoTimeNow` and `TestNoPrint` lint tests over `internal/atc/...`; fake clock advance.
- **Done when:** every later package takes `atc.Clock` and `*slog.Logger`.

#### T-M0-03 · ATC intent kinds and slot extractors
- **Covers:** ATC-NFR-TST-008, ATC-FR-PAT-015
- **Files:** `internal/nlu/intent.go` (kinds), `internal/nlu/entity/{runway.go,altitude.go,squawk.go,frequency.go,atis.go,fuel.go,heading.go}`, tests
- **Tests:** unit: extractor tables ("two one left" → 21L, "angels six" → 6000, "state five point four" → 5.4, "squawk four one zero one", "information Bravo", "one two four point niner five").
- **Done when:** `packcheck` reports no missing kinds or extractors for the atc pack.

#### T-M0-04 · ATC pack: examples, lexicon, embed
- **Covers:** ATC-NFR-TST-008, ATC-FR-PAT-015
- **Files:** `internal/nlu/packs/atc/{pack.yaml,examples.tsv,templates.txt}`, `internal/nlu/packs/packs.go` (embed)
- **Tests:** `go run ./cmd/packcheck -dir internal/nlu/packs/atc`; classifier held-out precision ≥ 0.99 on MVP kinds; lexicon terms of P §3.5 and T §3.9 present.
- **Done when:** the pack is embedded and enabled behind `atc.enabled`.

#### T-M0-05 · qud slots and `Question.Kinds`
- **Covers:** ATC-FR-COM-009
- **Files:** `internal/controller/qud/qud.go` (+ tests) — orchestrator-approved additive change (OD-012)
- **Tests:** unit: "Initial" matches a question with `Kinds: [report_pattern:initial]`; new slots Runway/Altitude/Heading/Frequency/Fuel/ATIS match their bags.
- **Done when:** existing qud tests pass unchanged.

#### T-M0-06 · Phraseology helpers and `RoleATC`
- **Covers:** ATC-FR-PHR-009, PHR-011, WX-003, WX-004
- **Files:** `internal/speech/phraseology/{atc.go,atc_test.go}`, `internal/speech/speech.go` (`RoleATC`)
- **Tests:** unit: `FlightLevel(230)`, `Altimeter(29.92)`, `QNH(1013)`, `Wind(210,8,0)`, `Wind(220,15,27)`, `Clock(11)`, `Fuel(5.4)`, `Angels(6)`, `TimeZ(14,35)`.
- **Done when:** no ATC code needs to spell a number by hand.

#### T-M0-07 · Phrase templates and renderer
- **Covers:** ATC-FR-PHR-002, PHR-005, PHR-008, PHR-009, PHR-010, PHR-013
- **Files:** `internal/atc/phrase/{templates.go,render.go,variant.go}`, `internal/atc/phrase/data/{general,ground,tower}.tmpl`
- **Tests:** unit: every id renders in FAA and ICAO or has a variant tag; `TestNoDigitsInPhrases`; yes/no templates start with "Affirm"/"Negative"; composed ids tagged; first-contact addressing adds the station name.
- **Done when:** the M1 template ids of design §11 exist.

#### T-M0-08 · World view snapshot
- **Covers:** ATC-NFR-PERF-004, ATC-FR-IDN-004
- **Files:** `internal/atc/world.go`, `internal/sim/dcsgrpc/atc.go` (player flag, group, carrier velocity if missing)
- **Tests:** unit: snapshot from a fake `world.Store`; `TestNoPerFacilityPolling` (calls/s independent of facility count).
- **Done when:** facilities read one shared ≤ 2 Hz snapshot.

#### T-M0-09 · Weather adapter
- **Covers:** ATC-FR-WX-001, WX-002, ATC-NFR-DEG-008
- **Files:** `internal/atc/wx.go`, `internal/sim/dcsgrpc/atmosphere.go`, `internal/sim/sim.go` (additive `AtmosphereReader`)
- **Tests:** unit: mission weather JSON → `atc.Weather` (clouds, visibility, fog, QNH); magnetic conversion 222°T → "two one zero" at 11° E; missing weather → items omitted.
- **Done when:** `wx.At(airbase)` returns fresh (≤ 60 s) or flagged-stale weather.

#### T-M0-10 · Mission config: `OVERWATCH.atc` skeleton
- **Covers:** ATC-FR-CFG-001, CFG-002, CFG-003, CFG-004, CFG-005, CFG-009, CFG-013, ATC-NFR-DEG-007
- **Files:** `internal/missionconfig/{atc.go,atc_test.go}`, `docs/overwatch-mission-config.lua` (atc section)
- **Tests:** unit: schema table tests (type errors with Lua paths, did-you-mean, unknown fields as warnings, `atc = 5` → defaults); announcement text.
- **Done when:** the Nevada fixture without `atc` yields zero-config; S-20 issue lines match.

#### T-M0-11 · Server config `atc:` section
- **Covers:** ATC-FR-CFG-012
- **Files:** `internal/config/{atc.go,atc_test.go}`, `config.example.yaml`
- **Tests:** unit: defaults of design §8.3; unknown keys warn.
- **Done when:** `atc.enabled: false` keeps the role off.

#### T-M0-12 · Tower frequencies from `radio.lua` (mappack owner)
- **Covers:** ATC-FR-FAC-001
- **Files:** `internal/mappack/airfield.go` (`ParseRadio` frequencies) — owned by the mappack agent (OD-004)
- **Tests:** unit (mappack): Nellis radio fixture → 327.0 and 132.55 AM.
- **Done when:** `AirfieldRadio` carries frequencies.

#### T-M0-13 · Registry
- **Covers:** ATC-FR-FAC-001, FAC-002, FAC-010, FAC-011, FAC-013, CFG-006, CFG-003, PHR-001
- **Files:** `internal/atc/{registry.go,registry_test.go}`
- **Tests:** unit: Nevada fixture → Nellis combined facility on 327.0/132.55 named "Nellis Tower", FAA/military/USAF; split freqs produce two facilities; duplicate frequency with a JTAC → issue; deterministic voices.
- **Done when:** build < 2 s on the Nevada fixture.

#### T-M0-14 · Airspace build
- **Covers:** ATC-FR-ASP-001, ASP-002, ASP-003, ASP-006
- **Files:** `internal/atc/airspace/{build.go,build_test.go}`
- **Tests:** unit: synthesized C at Nellis; mission `class` wins; membership inside core / below shelf; `BenchmarkContains` < 2 µs.
- **Done when:** `Registry.Airspace` has Class C Nellis (synthesized). Superseded 2026-10-01: Nellis is an estimated Class D (military fields are D by heuristic; the FAA has no Nellis Class D, it lies in the Las Vegas Class B), published FAA airspace comes first (DE-ASP).

#### T-M0-15 · textsim `-role atc` skeleton
- **Covers:** ATC-NFR-TST-003
- **Files:** `cmd/textsim/{atc.go,atc_cmds.go}`, `internal/world/testdata/nevada-atc.yaml`, `internal/mappack/mappacktest` (Nellis extract fixture)
- **Tests:** a smoke script `atc-smoke.txt` (radio check); `@proc` / `@seq` parse.
- **Done when:** `go run ./cmd/textsim -role atc -script …/atc-smoke.txt` passes.

#### T-M0-16 · Trajectory builder
- **Covers:** ATC-NFR-TST-002
- **Files:** `internal/atc/proc/traj/{traj.go,traj_test.go}`
- **Tests:** unit: `straight`, `turn`, `climb` produce 1 Hz samples on a fake clock; a pattern trajectory for runway 21L.
- **Done when:** textsim `/fly` uses it.

---

## M1 · MVP: tower and ground at Nellis with the overhead

#### T-M1-01 · Facility controller skeleton
- **Covers:** ATC-FR-IDN-001, IDN-011, IDN-012, FAC-014, FAC-020, PHR-010
- **Files:** `internal/atc/facility/{facility.go,handle.go,channel.go}`
- **Tests:** unit: unknown voice → "say callsign"; blocked; air-to-air ignored; reply on the call's frequency; radio check. textsim: S-01 (first half).
- **Done when:** a `Channel` per frequency satisfies `controller.Controller`, `Hinter`, `Roler`, `SpeakerLabeler`.

#### T-M1-02 · Flight table and identity
- **Covers:** ATC-FR-IDN-002, IDN-003, IDN-004, IDN-005, IDN-006, IDN-009, FRM-001
- **Files:** `internal/atc/{flights.go,flights_test.go}`
- **Tests:** unit: bind, correction window, amend, lock; unit bound only after callsign; group → flight with members; WSO voice on the same member; lifetime drops. textsim: S-01.
- **Done when:** `/proc` shows flights and members.

#### T-M1-36 · Scripts S-01, S-02, S-03
- **Covers:** ATC-NFR-TST-004
- **Files:** `cmd/textsim/testdata/atc-{identity,taxi-readback,departure-vfr}.txt`
- **Tests:** textsim runs in `go test ./cmd/textsim`.
- **Done when:** green in CI.

#### T-M1-37 · Scripts S-04, S-05, S-06, S-07
- **Covers:** ATC-NFR-TST-004
- **Files:** `cmd/textsim/testdata/atc-{overhead-single,overhead-4ship,goaround-flightmate,closed}.txt`
- **Tests:** as above.
- **Done when:** green.

## M2 · Approach and departure radar

#### T-M2-01 · Split positions, redirect, regional approach
- **Covers:** ATC-FR-FAC-003, FAC-005, FAC-015, ASP-007
- **Files:** `internal/atc/registry.go`, `internal/atc/facility/handle.go` (redirect)
- **Tests:** unit: split registry; `Responsible` redirect; merged terminal airspace. textsim: S-40.
- **Done when:** S-40 passes.

## M3 · ATIS and NOTAMs

#### T-M3-01 · ATIS content and rendering
- **Covers:** ATC-FR-ATI-001, ATI-003, ATI-010, ATI-012, ATC-NFR-TST-007
- **Files:** `internal/atc/atis/{generator.go,render.go}`, goldens
- **Tests:** unit goldens (FAA; good-weather omission; threats; long remarks trimmed).
- **Done when:** golden of S-35 matches.

## M4 · Guard outreach

#### T-M4-01 · SRS guard receivers
- **Covers:** ATC-FR-GUA-004
- **Files:** `internal/radio/radio.go` (`ClientRadios.Guards`, orchestrator, OD-011), `internal/radio/srs`, `internal/atc/guard.go`
- **Tests:** unit: `GuardsFor` with secondary receivers.
- **Done when:** a radio with `secFreq` 243.0 qualifies.

## M5 · Terminal procedures

#### T-M5-01 · `term` model and YAML/Lua loader
- **Covers:** ATC-FR-TPD-001, TPD-009
- **Files:** `internal/atc/term/{model.go,yaml.go,lua.go}` (+ tests)
- **Tests:** unit: every PT parses; VEGAS1 YAML = Lua; disable by id.
- **Done when:** the T §4.5 examples load.

## M6 · Carrier

#### T-M6-01 · Ship frame and carrier geometry
- **Covers:** ATC-FR-CVC-005, ASP-015
- **Files:** `internal/atc/cv/ship.go`, `internal/atc/proc/frame.go` (ship/final frames), `internal/atc/cv/data/carriers.yaml`
- **Tests:** unit: moving frame, relative velocity, turning ship; control zone/area membership.
- **Done when:** textsim ship turns 20° without breaking predicates.

#### T-M6-21 · Scripts S-26…S-29, S-38 and live checklist M6
- **Covers:** ATC-NFR-TST-004
- **Files:** `cmd/textsim/testdata/atc-{case1-spin,case3-marshal,bolter,case3-delta,case3-departure}.txt`, `docs/atc-spec/live-checklist-m6.md`
- **Tests:** textsim + live (Hornet Case I with spin, Case III night recovery).
- **Done when:** green; timing constants measured (OD-014).

---

## M7 · ICAO, Navy and civil variants

#### T-M7-01 · ICAO templates and variant rules
- **Covers:** ATC-FR-PHR-003, ATI-002, ASP-016, SEQ-010
- **Files:** `internal/atc/phrase/data/*.tmpl` (ICAO), `internal/atc/phrase/variant.go`
- **Tests:** unit: D §8 ICAO column; transition level; ICAO separation rule.
- **Done when:** every template has an ICAO form.

## M8 · Polish and tooling (P3)

#### T-M8-01 · SFO / PFO
- **Covers:** ATC-FR-PAT-017
- **Files:** `internal/atc/proc/data/sfo.yaml`, profile `sfo`
- **Tests:** unit: weather gate; high/low key; traj.
- **Done when:** textsim SFO variant passes.

## Requirement → task index

Generated from the **Covers** lines.

| Requirement | Pri | Tasks |
|---|---|---|
| ATC-FR-IDN-001 | MVP | T-M1-01 |
| ATC-FR-IDN-002 | MVP | T-M1-02 |
| ATC-FR-IDN-003 | MVP | T-M1-02 |
| ATC-FR-IDN-004 | MVP | T-M0-08, T-M1-02 |
| ATC-FR-IDN-005 | MVP | T-M0-01, T-M1-02 |
| ATC-FR-IDN-006 | MVP | T-M1-02, T-M1-09 |
| ATC-FR-FAC-018 | P3 | T-M8-12 |
