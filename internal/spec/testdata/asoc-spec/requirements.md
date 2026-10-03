# ASOC role: requirements

Spec-driven-development requirements for the Overwatch ASOC agency. Read
`README.md` first for the decision, the ID scheme, the source tags and the
legends. Every requirement has the fields **Priority** (`MVP`, `P2`,
`P3`) with **Status** (`decided`, `derived`, `open → OD-nnn`), **EARS**,
**Acceptance** (numbered, testable, with exact example phraseology),
**Source**, **Depends** and **Design** (design elements, `design.md` §2).

Words used in EARS text: *the ASOC* is a running ASOC agency controller
(one per configured `asoc` entry and frequency); *Darkstar* or *the AIC*
is the coalition's existing AWACS persona (`internal/awacs`), unchanged;
*the JTAC* is a JTAC persona (`internal/jtac`); *a flight* is a flight
that has checked in with the ASOC unless stated otherwise; *the ACO* and
*the ATO* are the coalition's airspace control order and tasking order
models (`design.md` §5, §6).

## A. Doctrine grounding

Overwatch is not an operational system and this section is not doctrine.
It records which publicly released publications the behaviour is modelled
on, and where the spec relies on secondary or unverified material.
Confidence tags as in `docs/awacs-doctrine.md`: **[P]** primary, publicly
released; **[P-old]** primary but superseded; **[R]** exists but is not
publicly released (not used for wording); **[U]** cited from general
knowledge, not re-read for this spec: the edition, page or paragraph must
be confirmed before the text is quoted anywhere.

### A.1 Sources

| Tag | Publication | Date / edition | Status | Used for |
|---|---|---|---|---|
| J1 | JP 3-09.3 *Close Air Support* | 8 Jul 2009 (the newest public edition; the 2019 edition, validated 2021, is not public: `docs/jtac-doctrine.md` §1) | [P] (repo P1 of jtac-doctrine) | the CAS check-in brief (Fig V-12: mission number, number and type, position and altitude, ordnance, time on station, capabilities, abort code; "as fragged" / "with exception"); CAS aircraft routed through the theater air control system to the JTAC; contact points and holding; JTAC abort authority |
| J2 | JP 3-52 *Joint Airspace Control* | 13 Nov 2014 (public copy on irp.fas.org); a later edition may exist | [P] edition [U] | the airspace control order (ACO) implementing the airspace control plan for a defined period; airspace coordinating measures (ACMs) in its App C: restricted operations zone (ROZ), air refuelling track / anchor, CAP, kill box, coordinating altitude, transit routes |
| J3 | JP 3-30 *Joint Air Operations* | 2019 / 2021 [U] | [P] [U] | the air tasking order (ATO) and its mission numbers; the theater air control system (TACS) as the execution system |
| J4 | JP 3-09 *Joint Fire Support* | 2019 [U] | [P] [U] | kill box definition and its status: an **open** kill box allows fires without further coordination, a **closed** one does not ("fires or effects of fires are not allowed without further coordination"); blue (air-to-surface) and purple (air- and surface-to-surface) kill boxes; FSCMs (FSCL, NFA, RFA, ACA) |
| A1 | ATP 3-52.4 *ACC: Air Control Communication* MTTP | 9 Oct 2024 | [P] (repo P2 of awacs-doctrine) | check-in to a tactical C2 agency (Ch II §4): MNPOPCA order, the "as fragged" form, ALPHA CHECK; transmission formats (directive, interrogative, informative); communication priorities (Table 1) |
| A2 | ATP 1-02.1 *Brevity* MTTP | 25 Apr 2025 | [P] (repo P1 / P4) | PUSH (go to the designated frequency; no reply expected), PLAYTIME, AS FRAGGED, ALPHA CHECK, WORDS. Definitions not re-quoted here [U]; quote them from the repo's doctrine files |

**Data-model sources (the user's own repositories, not doctrine; read-only).**

| Tag | Source | Used for |
|---|---|---|
| X1 | `opord-builder` (`/mnt/c/git/opord-builder`, github.com/YoloWingPixie/opord-builder): the **proposed ATO schema v0.2** ("ATO / resources proposal", `docs/complete-schema.html`, `kind: ato`, `schema_version: '0.2'`; 103 proposal types: `ATO`, `Mission` with `mission_number`, typed `tasking` variants `preplanned_attack`, `on_call_cas`, `cap`, `escort`, `refueling`, `airborne_control`, `reconnaissance`, …, `Flight`, `Package`, `Agency`, `AgencyControl` with `check_in` and `handovers`, `Resources` catalogs `places`, `areas`, `channels`, `agencies`), and the current `AirspaceControl` of `schemas/opord.schema.json` (`geo_refs`, `airspace_zones` with kinds ROZ / HIDACZ / MEZ / FEZ / JEZ / NFZ / SAR / SAFE, `cap_tracks`, `aar_tracks`, `awacs_orbits`, `aic_sectors`, `ingress_corridors`) | the ATO model (design §5) and the ACO fields (design §6); the formal schema of ASOC-FR-ATO-011 |
| X2 | `dcs-mission-planner` (`/mnt/c/git/dcs-project`, github.com/YoloWingPixie/dcs-mission-planner): the control-measure catalogue `viewer/public/modules/controlmeasures-calc.js` (`CM_TYPES`, the canonical record `{id, type, name, geometry, altMinFt, altMaxFt, timeStart, timeEnd, agency, freq, …}`, geometry kinds point / line / corridor / polygon / circle / racetrack, `KILLBOX` kind blue / purple, `PROC_POINT` kinds IP / CP / PUSH / GATE / RV) and its research note `docs/acm-research.md` (JP 3-52 / APP-3.3.5 / USMTF ACO shapes; kill box status open / closed); the plan store `dcsops/plan.json` schema 1 (`packages`, `misc.missionNumber`, `controlMeasures`) | ACO geometry kinds and fields (design §6) |

No project named "ATO builder" was found under `/mnt/c/git` or `/mnt/c/wrk`;
X1's ATO proposal is the formalised ATO schema that matches the
description and is taken as "the ATO builder". **The user should confirm
this** (OD-020). Per the user's review, the model here is fine for now
and will be workshopped later.

**Not used.** The CRC, ASOC and AWACS AFTTP 3-1 / AFTTP 3-3 volumes and
NATO APP-7 / ATP-3.3.x are not publicly released ([AD §1]). JFIRE is not
public ([JD §1]).

### A.2 What the doctrine says the agencies do (as modelled)

| Agency | Doctrine (J1, J3, A3, F1, F2) | Overwatch |
|---|---|---|
| AOC | plans and publishes the ATO and ACO | the mission maker (the `asoc.ato` / `asoc.aco` tables, drawings) and the ATO inference of §5 of `design.md` |
| ASOC | controls air support short of the FSCL; CAS aircraft check in with it (or with the CRC / AWACS acting for the TACS), are given the current situation and routed to the JTAC; collocated with the senior Army fires element | **the ASOC agency**: check-in, ATO match, routing, kill box and ACM clearances, gateway hand-offs |
| CRC | ground radar battle management and air surveillance; often the check-in and airspace agency for fighters; terrain-limited low coverage | the ASOC (or the AIC) *voiced from a ground platform* (`platform = "ground"`) |
| AWACS (E-3, E-7) | airborne element of the TACS: surveillance, AIC, battle management; on one aircraft different crew positions work different frequencies under one callsign | Darkstar (AIC, unchanged) and optionally the ASOC role on a second frequency with the same callsign (`platform = "awacs"`) |
| TACP / JTAC | terminal attack control; receives the CAS aircraft from the ASOC, takes the check-in, passes the situation update and the nine-line | Landshark (unchanged, plus an optional hand-off receiver: OD-001) |
| Tanker | refuels on an AR track or anchor in the ACO | DCS AI tankers, referenced by the ACO; not voiced |

## B. What the ASOC is and is not

| The ASOC does | The ASOC never does (owner) |
|---|---|
| check-in, tasking, ATO matching, check-out, in-flight reports | picture, BRAA, bogey dope, declare, commit, targeting, threat, merged, faded (Darkstar) |
| ACO model and clearances (kill boxes, ROZs, AARAs, CAP blocks, entry/exit gates, FSCMs) | nine-line, talk-on, marks, cleared hot, abort, BDA requests (JTAC) |
| routing: "contact Landshark …", "contact Darkstar …", hand-offs to ATC at entry/exit gates (tanker routing deferred, OD-010) | taxi, takeoff, landing, approach, IFR clearances (ATC) |
| airspace monitoring of its own checked-in flights | DCS tasking of any AI group, weather / airbase information, guard outreach |

## C. Interplay with Darkstar (unchanged)

Darkstar keeps everything it does today. The ASOC is a peer agency:

- **Hand-off to Darkstar.** For a DCA, escort, sweep or CAP tasking the
  ASOC offers the flight to Darkstar's existing `controller.HandoffReceiver`
  (`awacs.AWACS.Offer` accepts any offer and waits for the check-in) and
  says "contact Darkstar, two five one point zero". The flight checks in
  with Darkstar as with any flight today; Darkstar answers as today.
- **Direct check-in with Darkstar.** A flight that never calls the ASOC is
  Darkstar's exactly as today. The ASOC never calls it.
- **Darkstar coming up on the ASOC frequency.** Darkstar's existing guest
  episodes (`awacs/episode.go`) already warn any roster flight on the
  frequency it was last heard on. A flight that identified itself on the
  ASOC frequency is in the roster, so Darkstar may come up on the ASOC
  frequency with a THREAT call. The ASOC holds its own routine calls to
  that flight for the episode (`controller.AirThreatListener`,
  `Overhearer`), as the JTAC does.
- **Back from Darkstar.** A flight that checks out with Darkstar ("back to
  Overlord") and calls the ASOC continues its ASOC session.
- **Shared core, read only.** Who works a flight now: the core
  `controller.Roster` (last station heard). Nothing else (OD-012
  decided: roster only, no picture seam, no dynamic re-task). No line in
  `internal/awacs` changes.
- **Darkstar's picture on the ASOC frequency.** Darkstar is not changed to
  broadcast there. The ASOC recommends monitoring Darkstar on a second
  radio when a flight's tasking needs the picture (ASOC-FR-AIC-002); an
  opt-in radio-level simulcast of Darkstar's broadcast calls onto the ASOC
  frequency, done in the Loop and not in the AWACS, is OD-019
  (ASOC-FR-AIC-009).
- **One agency on two frequencies.** With `platform = "awacs"` the ASOC
  takes Darkstar's callsign on its own frequency (a second crew position
  of the same aircraft). It is a separate controller with its own voice
  book and, by default, a different voice; Darkstar's controller is not
  modified (OD-003).

What stays shared (one copy, used by every role): voice binding
(`controller/ident`), unit resolution (`world/whois`), the roster,
hand-off contracts, `world.Lifecycle`, the drawing grammar and zone
resolution (`internal/missionconfig`), callsign normalisation and station
matching (`nlu/norm`), phraseology, the transmit queue and arbiter, the
phrase cache and voice pools, the F10 map manager, terrain and the sensor
horizon model.


## Area index

| Area | Code | MVP | P2 | P3 | Total |
|---|---|---|---|---|---|
| Role, agencies and platforms | ROL | 4 | 3 | 1 | 8 |
| Identity and sessions | IDN | 6 | 0 | 0 | 6 |
| Check-in | CHK | 6 | 4 | 0 | 10 |
| Tasking order | ATO | 6 | 2 | 2 | 10 |
| Routing and hand-offs | RTE | 7 | 3 | 0 | 10 |
| Airspace control order | ACO | 5 | 6 | 1 | 12 |

---

## 1. Role, agencies and platforms (ROL)

### ASOC-FR-ROL-001 · A new, additive agency type
- **Priority:** MVP · **Status:** decided
- **EARS:** The system shall provide an agency type `asoc` next to the JTAC, AWACS, ATC and AI flights, and shall leave the behaviour of every other role byte-identical when no ASOC is configured.
- **Acceptance:**
  1. With no `asoc` table and `asoc.enabled` unset, the golden textsim runs of the AWACS, JTAC, ATC and AI-flight scripts produce identical transcripts and logs (minus timestamps) before and after the ASOC code lands.
  2. The role registers `speech.RoleASOC = "asoc"` (additive constant) and reports `Role() == "asoc"`.
- **Source:** [UB]; [UC]
- **Depends:** –
- **Design:** DE-ROLE, DE-RMV

## 2. Identity and sessions (IDN)

### ASOC-FR-IDN-001 · Unidentified caller
- **Priority:** MVP · **Status:** decided
- **EARS:** When a voice that has not stated a callsign to this ASOC calls without one, the ASOC shall ask for the callsign and take no other action.
- **Acceptance:**
  1. "Overlord, checking in" from an unknown voice gets "Station calling Overlord, say callsign."
  2. No session, ATO match or unit binding is made (log shows none).
- **Source:** voice identity rule [UR]; [ATC] ATC-FR-IDN-001
- **Depends:** –
- **Design:** DE-IDN

## 3. Check-in (CHK)

### ASOC-FR-CHK-001 · Full check-in (MNPOPCA)
- **Priority:** MVP · **Status:** decided
- **EARS:** When a flight checks in, the ASOC shall take, in any order, the mission number, number and type, position and altitude, ordnance, playtime, capabilities and abort code it gives, and keep them on the session.
- **Acceptance:**
  1. "Overlord, Hawg 1-1, mission number five one zero one, two A-10C, over Bobcat angels one five, four GBU-12, two Mavericks, guns, playtime four five, targeting pod, abort code alpha" fills every field (log `asoc: check-in mission=5101 count=2 type=A-10C playtime=45m`).
  2. Items are parsed by the ASOC fast path at once and refined by the LLM in the background (ASOC-FR-NLU-005).
- **Source:** A1 Ch II §4 (MNPOPCA); J1 Fig V-12; [UB]
- **Depends:** ASOC-FR-IDN-002, ASOC-FR-NLU-002
- **Design:** DE-CHK

### ASOC-FR-CHK-002 · "As fragged"
- **Priority:** MVP · **Status:** decided
- **EARS:** When a flight checks in "as fragged" and its ATO line is found (ASOC-FR-ATO-003), the ASOC shall take every check-in item from the line and ask for nothing.
- **Acceptance:**
  1. "Overlord, Hawg 1-1, mission five one zero one, checking in as fragged" gets the tasking reply of ASOC-FR-CHK-006 with no question.
  2. "As fragged" with no line found is handled as an unfragged check-in (ASOC-FR-ATO-004).
- **Source:** J1 p.V-34 ("as fragged" or "with exception"); A1 Ch II §4
- **Depends:** ASOC-FR-ATO-003
- **Design:** DE-CHK, DE-ATO

## 4. Tasking order (ATO)

### ASOC-FR-ATO-001 · ATO lines from the mission
- **Priority:** MVP · **Status:** decided
- **EARS:** Where the mission defines `asoc.ato` lines, the ASOC shall load each line (mission number, optional flight callsign or DCS group, type, count, tasking, ordnance, playtime, vul window, and the tasking's targets: JTAC, contact point, kill box, CAP, escorted flight, tanker, entry/exit gates); a line without a callsign or group is an **open line** that any flight may be given.
- **Acceptance:**
  1. `{ mission = "5101", callsign = "Hawg 1", type = "A-10C", count = 2, task = "cas", jtac = "Landshark", cp = "Bobcat" }` is line 5101 (persona summary `ATO 5 lines`).
  2. `{ mission = "5102", task = "cas", jtac = "Landshark", cp = "Bobcat" }` is an open line.
  3. A line naming an unknown JTAC or kill box is kept with a warning and that target is ignored.
- **Source:** [UB]; J3 (ATO); X1 (`Mission`: `mission_number`, typed `tasking`, `flights`, `control`)
- **Depends:** ASOC-FR-CFG-002
- **Design:** DE-ATO
- **Note:** the user's review (2026-10-02): "fine as is right now", to be workshopped later against the formal schema (ASOC-FR-ATO-011).

## 5. Routing and hand-offs (RTE)

### ASOC-FR-RTE-001 · CAS to the JTAC
- **Priority:** MVP · **Status:** decided
- **EARS:** When a flight's tasking is CAS, the ASOC shall give it the contact point and block and hand it to the line's JTAC (else the nearest staffed JTAC of the coalition): "proceed contact point <CP>, block …, contact <JTAC>, <freq>".
- **Acceptance:**
  1. As ASOC-FR-CHK-006 acceptance 1.
  2. The contact point is the line's `cp`, else the `CP …` zone nearest the JTAC, else the JTAC's first IP, else none ("proceed as fragged").
- **Source:** [UB]; J1 (contact point, routing to the JTAC); A3
- **Depends:** ASOC-FR-CHK-006
- **Design:** DE-RTE, DE-JTC

## 6. Airspace control order (ACO)

### ASOC-FR-ACO-001 · Coordination measure kinds (NATO terms)
- **Priority:** MVP · **Status:** decided (OD-007, user 2026-10-02)
- **EARS:** The ACO shall model these coordination measures (CMs), named as in AJP-3.3.5 Annex B: kill box (KB, FSCM), restricted operating zone (ROZ, ACM), air-to-air refuelling area (AARA, ACM, as a track or an anchor), CAP station, entry/exit gate (EG, ARM) belonging to an AOR, contact point (CP, ARM), and the FSCMs FSCL, NFA, RFA and airspace coordination area; each with a name, a geometry, an altitude block, an activation state and an effective window.
- **Acceptance:**
  1. `aco.Kind` has exactly `kb`, `roz`, `aara`, `cap`, `eg`, `cp`, `fscl`, `nfa`, `rfa`, `aca` (airspace coordination area, written out in speech and labels; N2 uses "ACA" for the airspace control authority).
  2. Every kind maps to its N2 category and abbreviation in design §6.1 (KB → FSCM, Table B-2; ROZ, AARA → ACM, Table B-1; EG, CP → ARM, Table B-4) or is marked US-only (CAP station, NFA, RFA, airspace coordination area: J2 / J4 [U]).
  3. Geometry kinds follow X2: point, line, corridor, polygon, circle, racetrack.
- **Source:** [UB]; N2 Annex B; J2 App C; J4; X1 (`AirspaceControl`); X2 (`CM_TYPES`)
- **Depends:** –
- **Design:** DE-ACO
- **Note:** the user's review (2026-10-02): the formal schema comes from their own catalogue (ASOC-FR-ATO-011), workshop later.

## 7. Airspace clearances (CLR)

### ASOC-FR-CLR-001 · Kill box clearance, with only the relevant measures
- **Priority:** MVP · **Status:** decided (user review 2026-10-02)
- **EARS:** When a flight is tasked into, or requests, an open kill box, the ASOC shall clear it into the box with a block inside the box's block and ask for "report entering"; it shall name at most two other CMs in the clearance, chosen as the active ones that intersect the cleared volume (box and block) in the flight's window, most restrictive first (NFA, ROZ, RFA, airspace coordination area), and summarise the rest as "<n> more restrictions in effect per the ACO, say restrictions for details".
- **Acceptance:**
  1. One NFA in the box: "Viper 1-1, Overlord, eight eight alpha yankee is open, surface to one eight thousand. Cleared into eight eight alpha yankee, block angels one five to one eight. NFA Church inside the box. Report entering."
  2. Twenty NFAs and a ROZ in the box, the ROZ below the cleared block: the clearance names the two NFAs nearest the box centre and says "eighteen more restrictions in effect per the ACO, say restrictions for details"; the ROZ is not mentioned (it does not intersect the block).
  3. CMs outside the box, outside the block or outside their window are never mentioned.
- **Source:** [UB]; J4; user question 2026-10-02 ("What happens if there are 20 NFAs?")
- **Depends:** ASOC-FR-ACO-004, ASOC-FR-CLR-003
- **Design:** DE-CLR

## 8. Airspace monitoring (MON)

### ASOC-FR-MON-001 · Closed or deactivated kill box ahead: FLOW
- **Priority:** P2 · **Status:** decided (OD-008, user 2026-10-02; review 2026-10-02)
- **EARS:** While a flight is checked in with the ASOC, on its frequency and in coverage, when its straight-line path is projected to enter a closed or deactivated kill box within 2 minutes (inside the box's block), the ASOC shall warn it once with a FLOW to the heading closest to its current track (left or right, in 5° steps) whose projected path for the next 5 minutes stays out of the box.
- **Acceptance:**
  1. Viper 1 tracking 090 toward closed 88AZ, whose north edge is 4 NM north of the track: "Viper 1-1, Overlord, eight eight alpha zulu closed, flow zero six zero." (the smallest turn that clears the box, not a fixed direction).
  2. No clearing heading within 90° of the track: "…, flow" the reciprocal-side heading that leaves the box soonest.
  3. Not repeated within 5 minutes for the same box; no call to a flight on another agency's frequency (ASOC-FR-IDN-005).
- **Source:** user review 2026-10-02; A2 (FLOW: maneuver in the stated direction or heading [U wording]; Darkstar already uses a controller FLOW, [AD §4.9])
- **Depends:** ASOC-FR-COV-001
- **Design:** DE-MON

## 9. Interplay with Darkstar (AIC)

### ASOC-FR-AIC-001 · Darkstar unchanged
- **Priority:** MVP · **Status:** decided
- **EARS:** The ASOC shall require no change to `internal/awacs` (code, NLU pack, tests, behaviour) and shall reach Darkstar only through its frequency, its existing `controller.HandoffReceiver` and its exported read-only methods through a neutral adapter.
- **Acceptance:**
  1. `TestAWACSUntouchedByASOC` (internal/asoc): no file under `internal/awacs` contains "asoc" (case-insensitive) or imports `internal/asoc` or `internal/airpic`.
  2. The AWACS golden textsim scripts pass unchanged with an ASOC running on another frequency.
- **Source:** [UC]
- **Depends:** –
- **Design:** DE-AIC, DE-RMV

## 10. Interplay with the JTAC (JTC)

### ASOC-FR-JTC-001 · Hand-off to the JTAC
- **Priority:** MVP · **Status:** decided
- **EARS:** When the ASOC routes a CAS flight to a JTAC, it shall say "contact <JTAC>, <freq>" after the contact point and block, offering the hand-off first where the JTAC implements a receiver.
- **Acceptance:**
  1. As ASOC-FR-CHK-006 acceptance 1; with no JTAC receiver the line is the same and no offer is logged.
- **Source:** [UB]; J1
- **Depends:** ASOC-FR-RTE-001
- **Design:** DE-JTC

### ASOC-FR-JTC-002 · Check-in brief passed to the JTAC
- **Priority:** P2 · **Status:** decided (OD-001, user 2026-10-02)
- **EARS:** Where the JTAC accepts hand-offs, the ASOC shall pass the flight's check-in (a neutral `controller.CheckIn` value in `Handoff.State`) so the JTAC can take "checking in as fragged".
- **Acceptance:**
  1. `CheckIn{Mission:"5101", Count:2, Type:"A-10C", Ordnance:["4 GBU-12","2 AGM-65"], PlaytimeMin:45, Caps:["TGP"], Abort:"alpha"}`.
- **Source:** J1 Fig V-12; [spec]
- **Depends:** ASOC-FR-RTE-004
- **Design:** DE-JTC

## 11. Tankers (AR)

**Deferred (OD-010, user 2026-10-02).** The tanker flow is kept here for
a later milestone (M5) and is not built before then. Until it is, "request
tanker" is answered "Viper 1-1, Overlord, unable, contact the tanker
direct." (ASOC-FR-NLU-008).

### ASOC-FR-AR-001 · Tanker assets
- **Priority:** P3 · **Status:** deferred (OD-010, user 2026-10-02: "at a later time")
- **EARS:** The ASOC shall know the coalition's tankers from `asoc[i].tankers`, else ATO lines of task `tanker`, else DCS groups of task Refueling, with callsign, unit, frequency, TACAN, AR track or anchor, block and type (boom or drogue).
- **Acceptance:**
  1. `{ callsign = "Texaco 1", unit = "Texaco-1", freq = 276.1, tacan = "31Y", track = "Shell", type = "boom" }`.
  2. A DCS Refueling group's frequency and TACAN are read from the group's radio settings and beacon task (missionconfig query extension, tagged `// asoc hook`).
- **Source:** [UB]; N1 [U]
- **Depends:** ASOC-FR-ACO-003
- **Design:** DE-AR

## 12. Entry/exit gates and the ATC seam (GTW)

### ASOC-FR-GTW-001 · Entry/exit gates of an AOR, shared with ATC
- **Priority:** P2 · **Status:** decided (user review 2026-10-02)
- **EARS:** The role shall model each AOR (the ASOC's `aor` zones, else the coalition's `AOR` drawings) with one or more entry/exit gates (EG, N2: "the point to which an aircraft will be directed to commence the transit inbound/outbound"), each with an optional transit route (a `TC` / `TR` corridor or the table's `route`), altitude block and ASOC frequency; ATC shall read the same gates as operating areas of kind `gate` through the same missionconfig parse.
- **Acceptance:**
  1. `AOR West` with `EG West Coyote 180-280` and `EG West Tikaboo 160-240` → AOR "West" with gates Coyote and Tikaboo; ATC sees operating areas "Coyote" and "Tikaboo" of kind gate.
  2. A gate with no AOR word belongs to the AOR it lies on (ASOC-FR-ACO-003 acceptance 3).
  3. Spoken "gate Coyote"; the drawing prefixes `GATE` and `GATEWAY` are accepted as aliases of `EG`.
- **Source:** [UB]; N2 Table B-4 (EG); [AC §13.1]; user question 2026-10-02 ("Isn't a gateway a gateway into the AOR, and an AOR can have multiple gateways?")
- **Depends:** ASOC-FR-ACO-003
- **Design:** DE-GTW

## 13. Check-out (OUT)

### ASOC-FR-OUT-001 · Check-out
- **Priority:** MVP · **Status:** decided
- **EARS:** When a flight checks out ("checking out", "RTB", "bingo, RTB", "winchester, RTB"), the ASOC shall acknowledge, give its exit routing (a gate when its AOR has one: ASOC-FR-GTW-005; else "frequency change approved") and end its tasking.
- **Acceptance:**
  1. "Overlord, Hawg 1-1, checking out, RTB Nellis" (no gates) → "Hawg 1-1, Overlord, copy, frequency change approved."
- **Source:** [UB]; A1 (check-out) [U]
- **Depends:** ASOC-FR-CHK-008
- **Design:** DE-SESS, DE-RTE

## 14. AI flights (AIF)

### ASOC-FR-AIF-001 · AI flights check in with the ASOC
- **Priority:** P2 · **Status:** decided (OD-014, user 2026-10-02)
- **EARS:** Where an ASOC of their coalition runs, the AI flights of `ai_flights` shall check in with it first on its frequency, then follow its "contact Darkstar" to Darkstar and check in there as today (aipilot change, tagged `// asoc hook`).
- **Acceptance:**
  1. "Overlord, Uzi 1, mission two one zero two, checking in as fragged, two F-16, angels two five, playtime four zero" → "Uzi 1, Overlord, radar contact, mission two one zero two copied, defensive counterair. Contact Darkstar, two five one point zero." → "Uzi 1, pushing Darkstar." → Darkstar check-in as today.
- **Source:** [UB] ("AI flights check in with the ASOC too"); [AF §2.3]
- **Depends:** ASOC-FR-RTE-003
- **Design:** DE-AIF

## 15. NLU coverage (NLU)

### ASOC-FR-NLU-001 · Intent kinds
- **Priority:** MVP · **Status:** derived
- **EARS:** The ASOC NLU pack shall define the kinds `asoc_check_in`, `asoc_check_out`, `asoc_back_with_you`, `asoc_push`, `asoc_request_tasking`, `asoc_request_killbox`, `asoc_report_area` (entering / exiting / clear), `asoc_request_block`, `asoc_request_tanker`, `asoc_off_tanker`, `asoc_request_agency`, `asoc_request_status`, `asoc_ifrep`, `asoc_request_transit`, `asoc_say_mission`, `asoc_expected_tasking` (the answer to "what tasking were you expecting today?" and the choice among offered lines), `asoc_say_restrictions` (with "continue"), `asoc_request_gate` (an exit gate on check-out), and reuse the core kinds (radio check, say again, roger / wilco, request picture as `aic_referral`).
- **Acceptance:**
  1. `packcheck -dir internal/asoc/nlu/pack` reports no missing kind.
- **Source:** [spec]
- **Depends:** –
- **Design:** DE-NLU

## 16. Phraseology and comm discipline (PHR)

### ASOC-FR-PHR-001 · Spoken text through the shared template renderer
- **Priority:** MVP · **Status:** decided (OD-011, user 2026-10-02)
- **EARS:** The ASOC shall produce every spoken line from its template catalogue rendered by the shared phrase-template renderer (`internal/speech/phrasetmpl`, extracted from ATC's renderer and used by every service), over `speech/phraseology`, with no digits in a rendered line.
- **Acceptance:**
  1. `TestNoDigitsInPhrases` over every template and argument table.
  2. `internal/asoc/phrase` holds only templates and argument types; no renderer code of its own.
  3. ATC renders through the same package with byte-identical output (ATC's phrase golden tests unchanged).
- **Source:** AGENTS.md; OD-011 ("B, and use it with every single service")
- **Depends:** –
- **Design:** DE-PHR

## 17. Mission configuration (CFG)

### ASOC-FR-CFG-001 · The `OVERWATCH.asoc` section
- **Priority:** MVP · **Status:** decided
- **EARS:** The role shall read the mission's `OVERWATCH.asoc` list (registered with `missionconfig.RegisterSection("asoc")`), one table per agency.
- **Acceptance:**
  1. `asoc = { { callsign = "Overlord", freq = 264.0 } }` starts one ground ASOC with every other field defaulted.
  2. Without the registration the resolver would report `asoc` as unknown; with it, it does not.
- **Source:** [UB]; [MC]
- **Depends:** –
- **Design:** DE-CFG

## 18. Mission lifecycle (LCY)

### ASOC-FR-LCY-001 · Controllers are mission resetters
- **Priority:** MVP · **Status:** decided
- **EARS:** Every ASOC controller shall implement `world.MissionResetter` (`OnMissionRestart`, `OnMissionReady`) and be reset through the Loop.
- **Acceptance:**
  1. `TestControllersAreMissionResetters` (cmd/overwatch) passes with the ASOC type in the tree and not in `lifecycleExempt`.
- **Source:** [ML] (Adding a component)
- **Depends:** –
- **Design:** DE-LCY

## 19. Removability (RMV)

### ASOC-FR-RMV-001 · Nothing imports the ASOC
- **Priority:** MVP · **Status:** decided
- **EARS:** No package outside `internal/asoc` shall import `internal/asoc/...`, except the wiring files `cmd/overwatch/asoc*.go` and `cmd/textsim/asoc*.go`.
- **Acceptance:**
  1. `TestNothingImportsASOC` (internal/asoc/imports_test.go), modelled on `TestNothingImportsATC`, checks at least 100 files.
- **Source:** [UB]; ATC `imports_test.go`
- **Depends:** –
- **Design:** DE-RMV

## 20. Voices and phrase cache (VOX)

### ASOC-FR-VOX-001 · Voice selection
- **Priority:** MVP · **Status:** derived
- **EARS:** The ASOC shall speak with the agency's `voice` alias (server voices map), else a voice from the shared voice-pool allocator under role `asoc` (falling back to the ATC controller pool, then the server default).
- **Acceptance:**
  1. `voice = "calm"` → the alias's voice; no alias → the allocation is logged (`voicealloc: asoc Overlord -> <id>`).
- **Source:** [MC] (voice aliases); `internal/voicepool`
- **Depends:** –
- **Design:** DE-VOX

## 21. F10 map layer (MAP)

### ASOC-FR-MAP-001 · The `aco` layer
- **Priority:** P2 · **Status:** derived
- **EARS:** The F10 map layer shall have a layer `aco` (on by default) produced by the ASOC role from the coalition's ACO, scoped to that coalition.
- **Acceptance:**
  1. `f10map.LayerACO = "aco"` (additive constant, tagged `// asoc hook`); `map.layers.aco = false` turns it off.
- **Source:** [UB]; [MAP §4]
- **Depends:** ASOC-FR-ACO-009
- **Design:** DE-MAP

## 22. Platforms and coverage (COV)

### ASOC-FR-COV-001 · Ground site coverage
- **Priority:** P2 · **Status:** decided
- **EARS:** While the ASOC's platform is a ground site, a flight shall be in coverage when it is within `range_nm` (default 200), above the radar horizon of the antenna (`antenna_ft`, default 50 ft AGL) and in terrain line of sight from it.
- **Acceptance:**
  1. A flight at 1,000 ft AGL behind a ridge 40 NM from CRC-1 is out of coverage; at 15,000 ft it is in.
  2. Terrain LOS uses `internal/terrain` and the sensor package's LOS cache, not a copy.
- **Source:** [UB] (fixed ground site with terrain-limited low coverage); F2
- **Depends:** ASOC-FR-ROL-005
- **Design:** DE-COV

## 23. Performance (PERF)

### ASOC-NFR-PERF-001 · Reply latency
- **Priority:** MVP · **Status:** derived
- **EARS:** The ASOC shall decide a reply within 50 ms of receiving the parsed call (excluding TTS) and start transmitting a check-in reply within 1.5 s of the end of the call when its segments are cached.
- **Acceptance:**
  1. Benchmark `BenchmarkCheckIn` < 50 ms at 20 sessions.
- **Source:** [spec]
- **Depends:** –
- **Design:** DE-AGY

## 24. Observability (OBS)

### ASOC-NFR-OBS-001 · State-change log lines
- **Priority:** MVP · **Status:** derived
- **EARS:** The ASOC shall log every session and clearance transition at Info with the flight, from, to and why.
- **Acceptance:**
  1. `asoc: session flight="Viper 1" from=tasked to=in_box why="report entering 88AY"`.
- **Source:** [SM §3] (events logged with from / to / why)
- **Depends:** –
- **Design:** DE-OBS

## 25. Testing (TST)

### ASOC-NFR-TST-001 · Unit tests
- **Priority:** MVP · **Status:** decided
- **EARS:** Every ASOC package shall have table-driven unit tests with no network, passing `make check`.
- **Acceptance:**
  1. `go test -race -tags nolibopusfile ./internal/asoc/...` passes offline.
- **Source:** AGENTS.md
- **Depends:** –
- **Design:** DE-RMV

### ASOC-NFR-TST-002 · textsim scenarios
- **Priority:** MVP · **Status:** derived
- **EARS:** Each scenario of `scenarios.md` shall be a textsim script `cmd/textsim/testdata/asoc-<name>.txt` run by `go test`.
- **Acceptance:**
  1. S-01…S-05 pass at M1.
- **Source:** [ATC] ATC-NFR-TST-004
- **Depends:** –
- **Design:** DE-OBS

## 26. Degraded modes (DEG)

### ASOC-NFR-DEG-001 · DCS-gRPC down
- **Priority:** MVP · **Status:** derived
- **EARS:** If the world feed is down, then the ASOC shall keep its sessions, make no monitoring call, say "not radar contact" at check-in, and resume when the feed syncs.
- **Acceptance:**
  1. Fake feed down: a check-in gets "not radar contact"; no MON call is made.
- **Source:** [ML] (feed pending)
- **Depends:** –
- **Design:** DE-AGY
