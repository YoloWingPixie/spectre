# ASOC role: implementation plan

Ordered tasks, each about a day or less, grouped into milestones. A task
lists the requirements it implements (**Covers**: an ID without prefix
takes the last prefix written), the files it touches, the tests, and when
it is done. Every requirement of `requirements.md` appears in at least one
task.

Rules for every task (definition of done, common part):
- `make check` passes (`-tags nolibopusfile`, `-race`); no network in unit tests.
- Table-driven unit tests in the package that owns the behaviour; no edits to other packages' tests (hook owners extend their own).
- **No change under `internal/awacs`.** `TestAWACSUntouchedByASOC` passes in every task.
- Every change outside `internal/asoc` and `cmd/*/asoc*.go` is tagged `// asoc hook` and listed in `design.md` §12.2.
- Spoken text only through `internal/asoc/phrase` → `speech/phraseology`.
- Contract changes to shared packages only as listed in `design.md` §3.3, after the orchestrator agreed.

Milestones:

| Milestone | Goal | Exit |
|---|---|---|
| M0 Foundations | skeleton, removal guards, config, phrases, NLU pack skeleton, wiring, textsim role, specview | textsim `-role asoc` answers a radio check |
| M1 MVP | identity, check-in, ATO (table and inferred), routing to the JTAC and Darkstar, kill box clearances, check-out, lifecycle, golden tests | S-01…S-05 pass; live checklist M1 |
| M2 ACO and airspace | ROZ, AR, CP, FSCM, windows, runtime statuses, monitoring, tankers, Darkstar interplay, F10 layer | S-06…S-08 pass; live checklist M2 |
| M3 ATC seam | gateways in ATC, ATC → ASOC and ASOC → ATC hand-offs | S-09 passes; live checklist M3 |
| M4 Platforms and AI | coverage, co-location, SRS binding, AI flights, JTAC brief | S-10…S-11 pass; live checklist M4 |
| M5 Polish (P3) | picture seam and re-task, several ASOCs, keypads, FSCL, clips, extras | per task |

---

## M0 · Foundations

#### T-M0-01 · Package skeleton and removal guards
- **Covers:** ASOC-FR-RMV-001, RMV-002, RMV-003, AIC-001
- **Files:** `internal/asoc/{doc.go,imports_test.go,hooks_test.go,untouched_test.go}`
- **Tests:** `TestNothingImportsASOC`, `TestASOCImportsNoRole`, `TestASOCHooksListed` (parses `docs/asoc-spec/design.md` §12.2), `TestAWACSUntouchedByASOC`.
- **Done when:** the four guards pass on the empty skeleton.

#### T-M0-02 · Mission and server config
- **Covers:** ASOC-FR-CFG-001, CFG-002, CFG-003
- **Files:** `internal/asoc/config/{config.go,parse.go,validate.go}` (+ tests), `internal/config/config.go` (hook: `asoc:` section)
- **Tests:** table tests over Lua payload fixtures: defaults, duplicate frequency, callsign clash, bad platform, unknown JTAC; `RegisterSection("asoc")`.
- **Done when:** design §8.1–8.2 messages are produced verbatim.

#### T-M0-03 · Phrase catalogue
- **Covers:** ASOC-FR-PHR-001, PHR-002, PHR-003, PHR-004, PHR-005, ACO-011, ATO-007, ROL-002, JTC-006
- **Files:** `internal/asoc/phrase/{catalogue.go,render.go}`, `internal/asoc/phrase/data/asoc.tmpl`
- **Tests:** `TestNoDigitsInPhrases`, `TestNoForeignCalls` (no AIC / JTAC wording), `TestSpellKillbox` ("88AY"), `TestFreq`, `TestBlocks` (angels, flight levels, surface).
- **Done when:** every id of design §9.3 renders.

#### T-M0-04 · NLU kinds, slots and route
- **Covers:** ASOC-FR-NLU-001, NLU-002, NLU-003, NLU-004
- **Files:** `internal/asoc/nlu/{kinds.go,slots.go,parser.go,register.go}`, `internal/asoc/nlu/pack/{pack.yaml,examples.tsv,templates.txt}`
- **Tests:** extractor tables (mission number, CGRS names, blocks, playtime); `packcheck -dir internal/asoc/nlu/pack`; `route.Parser` test: 251.0 parses as before.
- **Done when:** `packs.RegisterExtension` accepts the pack.

#### T-M0-05 · Wiring, overlay sets, textsim role
- **Covers:** ASOC-FR-ROL-001, ROL-003, VOX-001, VOX-002, RMV-005
- **Files:** `cmd/overwatch/{asoc.go,main.go (hook),overlay.go (hook),voicealloc.go (hook)}`, `internal/speech/speech.go` (hook), `internal/voicepool/voicepool.go` (hook), `cmd/textsim/{asoc.go,main.go (hook)}`, `internal/asoc/agency/agency.go` (radio check only)
- **Tests:** overlay with two named extra sets; frequency clash; textsim radio check script.
- **Done when:** the M0 exit holds and every hook is inert without an `asoc` table.

#### T-M0-06 · specview reads `docs/asoc-spec`
- **Covers:** –
- **Files:** `internal/specview/{spec.go,link.go,review.go,server.go}` (OD-017; not an ASOC hook: a tooling generalisation)
- **Tests:** `spec_test.go` loads both `docs/atc-spec` and `docs/asoc-spec`; IDs `ASOC-FR-…` link; short IDs take the last full prefix.
- **Done when:** `go run -tags nolibopusfile ./cmd/specview -dir docs/asoc-spec` serves the spec with titles from its README.

---

## M1 · MVP

#### T-M1-01 · Identity and sessions
- **Covers:** ASOC-FR-IDN-001, IDN-002, IDN-003, IDN-004, IDN-005, IDN-006, ROL-004
- **Files:** `internal/asoc/agency/{identity.go,session.go}`
- **Tests:** unidentified caller; binding and correction window via `controller/ident`; whois only after identification; red caller; expiry.
- **Done when:** S-01 passes.

#### T-M1-02 · ATO lines and matching
- **Covers:** ASOC-FR-ATO-001, ATO-003, ATO-006
- **Files:** `internal/asoc/ato/{ato.go,match.go}`
- **Tests:** match by number, callsign, group; conflicting number asked once; line reuse after check-out.
- **Done when:** design §5.2 table loads with its warnings.

#### T-M1-03 · ATO inference and unfragged flights
- **Covers:** ASOC-FR-ATO-002, ATO-004
- **Files:** `internal/asoc/ato/infer.go`, `internal/missionconfig` payload read (no change if group tasks are already in `Payload`; else a tagged query field)
- **Tests:** the task table of design §5.3; numbering per first digit; unfragged CAS / kill box / DCA choices.
- **Done when:** a mission with only client groups and a JTAC gives sensible lines.

#### T-M1-04 · Check-in
- **Covers:** ASOC-FR-CHK-001, CHK-002, CHK-004, CHK-006, CHK-007, CHK-008, NLU-005
- **Files:** `internal/asoc/agency/checkin.go`
- **Tests:** full MNPOPCA; as fragged; missing items asked once; "not radar contact" with no unit; two-transmission split; back with you; enrichment arriving late fills silently.
- **Done when:** S-02's check-in lines match.

#### T-M1-05 · ACO core and the block book
- **Covers:** ASOC-FR-ACO-001, ACO-002, ACO-004, ACO-007, CLR-003
- **Files:** `internal/asoc/aco/{aco.go,build.go,status.go,geom.go,book.go}`
- **Tests:** `KILLBOX 88AY 0-180` parsed through `missionconfig.ParseAreaName` (same name as the AWACS's); table over drawing precedence; default status; stacking from the top; full box.
- **Done when:** the Nevada fixture ACO builds.

#### T-M1-06 · Kill box clearances
- **Covers:** ASOC-FR-CLR-001, CLR-002, RTE-002
- **Files:** `internal/asoc/agency/clear.go`
- **Tests:** open, closed, cold, NFA remark, alternative box.
- **Done when:** S-03 passes (without readback, M2).

#### T-M1-07 · Routing and hand-offs
- **Covers:** ASOC-FR-RTE-001, RTE-003, RTE-004, RTE-005, RTE-006, RTE-008, JTC-001, JTC-004, AIC-002, AIC-003, AIC-005
- **Files:** `internal/asoc/agency/{route.go,aic.go,jtac.go}`, `cmd/overwatch/asoc.go` (receivers from the overlay: `receiverOn`)
- **Tests:** offer then contact; refusal fallback; push silence; referral of picture requests; hand-back; the AWACS's `Offer` log line through the real `awacs.AWACS` in a `cmd/overwatch` test.
- **Done when:** S-02 and S-04 pass.

#### T-M1-08 · Check-out
- **Covers:** ASOC-FR-OUT-001, OUT-002, OUT-003
- **Files:** `internal/asoc/agency/checkout.go`
- **Tests:** RTB, bingo, winchester; IFREP logged; blocks and line freed.
- **Done when:** S-05 (first half) passes.

#### T-M1-09 · Lifecycle, golden tests, M1 scenarios
- **Covers:** ASOC-FR-LCY-001, LCY-002, LCY-003, LCY-004, LCY-005, GTW-007, RMV-004, ASOC-NFR-TST-001, TST-002, TST-003, PERF-001, PERF-002, OBS-001, DEG-001
- **Files:** `internal/asoc/role.go`, `cmd/overwatch/{asoc_lifecycle.go,lifecycle.go (hook)}`, `cmd/textsim/testdata/asoc-{identity,cas,killbox,aic,checkout}.txt`, golden tests in `cmd/overwatch` / `cmd/textsim`
- **Tests:** `TestMissionPartsRegistered`, `TestControllersAreMissionResetters`, restart/ready script, `TestGoldenWithoutASOC`, `TestGoldenWithIdleASOC`, `BenchmarkCheckIn`, feed-down check-in.
- **Done when:** S-01…S-05 pass and live checklist M1 is flown.

---

## M2 · ACO and airspace

#### T-M2-01 · More ACMs, windows, validation
- **Covers:** ASOC-FR-ACO-003, ACO-006, ACO-008, ACO-009, ACO-010
- **Files:** `internal/asoc/aco/build.go`, prefix registration in `cmd/overwatch/asoc.go`
- **Tests:** each prefix of design §6.2 with good and bad shapes; windows in NOTAM forms; one ACO per coalition.
- **Done when:** design §6.2–6.3 fixtures build.

#### T-M2-02 · Runtime statuses
- **Covers:** ASOC-FR-ACO-005, CFG-005, CFG-008
- **Files:** `internal/asoc/aco/status.go`, `internal/asoc/config/runtime.go`, watch keys through `missionconfig` (existing hook), `docs/overwatch-mission-config.lua` (runtime block)
- **Tests:** flag values 0–3; API newest wins; config change keeps sessions.
- **Done when:** a flag flip is applied within 5 s on the fake clock.

#### T-M2-03 · Monitoring
- **Covers:** ASOC-FR-MON-001, MON-002, MON-003, MON-004, MON-005, MON-006, CLR-006, CLR-008
- **Files:** `internal/asoc/agency/monitor.go`
- **Tests:** projection into a closed box once per 5 min; ROZ warning; block deviation 300 ft / 30 s; broadcast after directives; no call out of coverage; quiet run.
- **Done when:** S-07 passes.

#### T-M2-04 · Other clearances and status words
- **Covers:** ASOC-FR-CLR-004, CLR-005, CLR-007, CLR-009, CHK-003, CHK-005, CHK-010
- **Files:** `internal/asoc/agency/{clear.go,checkin.go}`
- **Tests:** ROZ transit refusal with an altitude; block change; CAP clearance only without lane staffing (OD-009); readback correction; with-exception; ALPHA CHECK; status words.
- **Done when:** S-03 passes with readbacks.

#### T-M2-05 · Tankers
- **Covers:** ASOC-FR-AR-001, AR-002, AR-003, AR-004, AR-005, AR-006
- **Files:** `internal/asoc/aco/tanker.go`, `internal/asoc/agency/tanker.go`, `internal/missionconfig/query.go` (hook: Refueling group frequency, TACAN)
- **Tests:** boom / drogue by type table; on-station test; queue blocks; off-tanker re-clearance.
- **Done when:** S-06 passes.

#### T-M2-06 · Darkstar and JTAC interplay
- **Covers:** ASOC-FR-AIC-004, AIC-006, AIC-007, CHK-009, OUT-004, OUT-005, RTE-007, RTE-009, RTE-010, JTC-005
- **Files:** `internal/asoc/agency/{aic.go,jtac.go,session.go}`
- **Tests:** roster "with Darkstar"; `AirThreat` hold and release; overheard "defending"; back from Darkstar; JTAC load stack; remarks string; landed flight ends silently.
- **Done when:** S-08 passes with the real AWACS controller in textsim.

#### T-M2-07 · F10 `aco` layer
- **Covers:** ASOC-FR-MAP-001, MAP-002, MAP-003, MAP-004, MAP-005
- **Files:** `internal/asoc/mapfeed.go`, `internal/f10map/{feature.go,config.go}` (hooks), `cmd/overwatch/{asoc_map.go,f10map.go (hook)}`
- **Tests:** producer tables; status-label-only when the awacs layer draws the box; `Version` bump on status change.
- **Done when:** live: the labels change in the F10 map within 10 s.

#### T-M2-08 · NLU quality, comm discipline, options
- **Covers:** ASOC-FR-NLU-006, NLU-007, PHR-006, PHR-007, ATO-005, ATO-008, CFG-004, CFG-006, CFG-007, ASOC-NFR-OBS-002, DEG-002, DEG-003, DEG-004
- **Files:** `internal/asoc/nlu/pack/*`, `internal/asoc/agency/arbiter.go`, `cmd/overwatch/asoc_schema.go`, `internal/missionschema/registry.go` (hook)
- **Tests:** `TestASOCHeldOut` ≥ 0.98; arbiter classes; say again; vul early / late; mission number required; enablement matrix; schema export; LLM-down run of S-02; no-AIC and no-JTAC variants.
- **Done when:** S-06…S-08 pass and live checklist M2 is flown.

---

## M3 · ATC seam

#### T-M3-01 · Gateways in ATC
- **Covers:** ASOC-FR-GTW-001, GTW-002
- **Files:** `internal/atc/{airspace.go,opsarea.go}`, `internal/atc/phrase/data/area.tmpl` (hooks; ATC owner)
- **Tests:** ATC: `GATEWAY Coyote 180-280` is an ops area; the clearance phrase; ATC's own suite unchanged otherwise.
- **Done when:** "cleared to gateway Coyote … expect due regard at Coyote" renders.

#### T-M3-02 · Neutral seams and the combat preference
- **Covers:** ASOC-FR-GTW-003, GTW-004, GTW-008
- **Files:** `cmd/overwatch/{handoffs.go,atc.go (hook),atc_areas.go (hook),asoc.go}`, `internal/asoc/gateway.go`
- **Tests:** `atcCombat` with and without `asocCombat`; gateway release names Overlord; builds with either role deleted (scratch build in the test via build tags or a script).
- **Done when:** the release of S-09 is offered to and accepted by Overlord.

#### T-M3-03 · RTB through the ATC directory
- **Covers:** ASOC-FR-GTW-005
- **Files:** `internal/asoc/agency/checkout.go`, `cmd/overwatch/asoc.go` (pass `atcRole.handoffDirectory`)
- **Tests:** nearest gateway to the home plate; `ReceiverAt` → LA Center; offer lands in the mailbox.
- **Done when:** S-09's RTB lines match.

#### T-M3-04 · Gateway names in ATC return requests
- **Covers:** ASOC-FR-GTW-006
- **Files:** `internal/atc/nlu/names.go` (hook)
- **Tests:** "gateway Coyote, request clearance to Nellis" → pop-up IFR return.
- **Done when:** S-09 passes.

#### T-M3-05 · M3 live checklist
- **Covers:** –
- **Files:** `docs/asoc-spec/tasks.md` (checklist results), `cmd/textsim/testdata/asoc-gateway.txt`
- **Tests:** live flight Nellis → Coyote → 88AY → Coyote → Nellis.
- **Done when:** live checklist M3 is flown.

---

## M4 · Platforms and AI flights

#### T-M4-01 · Coverage models
- **Covers:** ASOC-FR-COV-001, COV-002, COV-004, MON-005
- **Files:** `internal/asoc/coverage/{coverage.go,ground.go,air.go}`
- **Tests:** horizon formula; terrain mask through the `sensor` LOS cache on a terrain fixture; airborne centre follows the unit; omniscient mode.
- **Done when:** S-10 part 2 coverage checks pass.

#### T-M4-02 · Platform lost; coverage is not identity
- **Covers:** ASOC-FR-COV-005, COV-006
- **Files:** `internal/asoc/agency/run.go`, `internal/asoc/gateway.go`
- **Tests:** dead platform → silent, refuses offers, leaves the directory; back when the unit returns.
- **Done when:** S-10 part 2 passes.

#### T-M4-03 · Co-located agency
- **Covers:** ASOC-FR-ROL-005, ROL-006, ROL-008, COV-003, VOX-003
- **Files:** `internal/asoc/config/validate.go`, `cmd/overwatch/{asoc.go,voicealloc.go (hook)}`
- **Tests:** same callsign allowed only with `platform = "awacs"`; separate voice books; voice exclusion.
- **Done when:** S-10 part 1 passes.

#### T-M4-04 · SRS client at the platform
- **Covers:** ASOC-FR-VOX-004
- **Files:** `cmd/overwatch/asoc.go` (uses `radio.StationBinder`)
- **Tests:** fake radio records the binding.
- **Done when:** live: the client appears at CRC-1 in SRS.

#### T-M4-05 · AI flights check in with the ASOC
- **Covers:** ASOC-FR-AIF-001, AIF-002, AIF-003, AIF-004, AIF-005, AIF-006
- **Files:** `internal/aipilot/{calls.go,ears.go,agent.go}` (hooks; aipilot owner), `cmd/overwatch/aiflights.go` (hook), `internal/aipilot/removal_test.go` (`TestAIPilotDoesNotDependOnASOC`)
- **Tests:** templates parse through the ASOC parser; ears parse "contact Darkstar"; agent sequence ASOC → Darkstar → ASOC.
- **Done when:** S-11 passes.

#### T-M4-06 · JTAC hand-off receiver and brief
- **Covers:** ASOC-FR-JTC-002, JTC-003
- **Files:** `internal/controller/handoff.go` (hook: `CheckIn`), `internal/jtac/{handoff.go,checkin.go}` (hooks; JTAC owner)
- **Tests:** JTAC: as fragged with a brief → situation update; without → today's questions; existing JTAC tests unchanged.
- **Done when:** S-02 shows the abbreviated JTAC check-in (if OD-001 (b) is chosen).

---

## M5 · Polish (P3)

#### T-M5-01 · Air picture seam and re-task
- **Covers:** ASOC-FR-ATO-010
- **Files:** `internal/airpic/airpic.go`, `cmd/overwatch/airpic_awacs.go`, `internal/asoc/agency/aic.go`
- **Tests:** adapter copies only perceived fields (a test fails on any `tactics.Group` field it does not classify); re-task never names a group.
- **Done when:** the P3 variant of design §9.4.5 works in textsim.

#### T-M5-02 · Several ASOCs, keypads, FSCL, gateway squawk
- **Covers:** ASOC-FR-ROL-007, ACO-012, CLR-010, GTW-009
- **Files:** `internal/asoc/{agency/route.go,aco/geom.go,agency/clear.go}`
- **Tests:** AOR hand-over; keypad geometry; FSCL remark; squawk line.
- **Done when:** unit tests pass.

#### T-M5-03 · Extras
- **Covers:** ASOC-FR-ATO-009, NLU-008, AIC-008, VOX-005, VOX-006, MAP-006, CFG-009, ASOC-NFR-OBS-003
- **Files:** various in `internal/asoc`, `cmd/overwatch/asoc.go` (clip list subcommand), `docs/overwatch-mission-config.lua`, `cmd/textsim/asoc.go`
- **Tests:** per item.
- **Done when:** each item's acceptance passes.

#### T-M5-04 · Live acceptance and fold-in
- **Covers:** –
- **Files:** `docs/asoc-spec/*`
- **Tests:** a full live mission with all agencies.
- **Done when:** findings are folded into the spec.

---

## Live checklists

### M1 (ground ASOC, Nevada)
- [ ] SRS shows "Overlord" on 264.0; radio check without callsign gets "say callsign".
- [ ] A-10C client: as-fragged check-in → contact point, block, "contact Landshark"; pushing gets no reply; Landshark takes the check-in.
- [ ] F-16C: kill box clearance; a second player is stacked below; a closed box is refused.
- [ ] F-15C: escort → "contact Darkstar"; Darkstar's check-in reply is as it was before the ASOC existed.
- [ ] Check-out with IFREP; a mission restart drops everything and nothing old is transmitted.

### M2
- [ ] Flag flip closes 88AY with a flight inside: directive, broadcast, F10 label within 10 s.
- [ ] ROZ warning; block deviation call; tanker flow with DCS's own tanker comms.
- [ ] Darkstar comes up on 264.0 with a THREAT for a flight working the ASOC; the ASOC holds its calls.

### M3
- [ ] Nellis Clearance to gateway Coyote with "expect due regard"; LA Center releases to Overlord; RTB back through Coyote to LA Center; pop-up IFR home.

### M4
- [ ] Co-located "Darkstar" on 264.0 next to the AWACS on 251.0; different voices.
- [ ] Ground site terrain masking at low level in realistic mode; platform destroyed → silence.
- [ ] AI flight Uzi 1 checks in with Overlord, is pushed to Darkstar, checks out through Overlord.
