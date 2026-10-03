# ATC role: requirements

Spec-driven-development requirements for the Overwatch ATC role. Read
`README.md` first for the ID scheme, the source tags and the status
legend. Every requirement below has the same fields:

- **Priority**: `MVP` (milestone M1: tower + ground at one field with the
  overhead), `P2` (the full role, milestones M2–M7), `P3` (optional,
  polish or off by default).
- **Status**: `decided` (stated in the design docs), `derived` (this spec
  fills in detail consistent with the docs), `open → OD-nnn` (depends on an
  open decision in `open-decisions.md`; the text shows the recommended
  option).
- **EARS**: the requirement in EARS form ("When …", "While …", "If …,
  then …", "Where …", or "The … shall …").
- **Acceptance**: numbered, testable criteria with exact example
  phraseology. Unless a criterion says otherwise, examples use the FAA
  military variant at Nellis (runways 03L/21R and 03R/21L, runway in use
  21L, tower 327.0 / 132.55, ground 275.8, field elevation 1,870 ft), the
  flight "Viper 1-1" (an F-16C 2-ship, "Viper 1" flight) and wind 210° at
  8 kt.
- **Source**: document and paragraph (tags in `README.md` §3).
- **Depends**: other requirements (and open decisions) it needs.
- **Design**: the design elements (`DE-*`, `design.md` §2) that implement
  it.

Facility words in the EARS text: *the facility* is any staffed ATC
position; *the tower*, *ground*, *clearance delivery*, *approach*,
*departure*, *marshal*, *carrier tower*, *carrier approach*, *carrier
departure* and *ATIS* are the positions of `atc.AgencyKind`. When
positions are combined (Decision 3), "the tower" means the combined
facility working that position.


## Area index

| Area | Code | MVP | P2 | P3 | Total |
|---|---|---|---|---|---|
| Identity and sessions | IDN | 9 | 2 | 1 | 12 |
| Facilities, staffing, frequencies | FAC | 10 | 10 | 1 | 21 |
| Airspace and jurisdiction | ASP | 11 | 6 | 1 | 18 |
| Clearance delivery | CLR | 1 | 9 | 1 | 11 |
| Ground and taxi | GND | 12 | 2 | 1 | 15 |
| Tower: runway operations | TWR | 18 | 3 | 3 | 24 |

---

## 1. Identity and sessions (IDN)

### ATC-FR-IDN-001 · Unidentified caller
- **Priority:** MVP · **Status:** decided
- **EARS:** When a voice that has not stated a callsign to this facility calls without a callsign, the facility shall ask for the callsign and take no other action on the call.
- **Acceptance:**
  1. "Nellis Tower, request taxi" from an unknown voice gets "Station calling Nellis Tower, say callsign."
  2. No flight record, taxi route, sequence slot or ledger assertion is created for that call.
  3. The SRS client name, sender GUID and bound DCS unit are not used to guess the callsign (log shows no binding).
- **Source:** voice identity rule [UR]; [D §4, §10]; `controller/ident` package doc
- **Depends:** –
- **Design:** DE-IDN, DE-FAC

### ATC-FR-IDN-002 · Voice binding on the first stated callsign
- **Priority:** MVP · **Status:** decided
- **EARS:** When an unidentified voice states a callsign, the facility shall bind that voice to the callsign in its voice book and treat later calls from the voice without a callsign as that callsign's.
- **Acceptance:**
  1. "Nellis Tower, Viper 1-1, radio check" binds the voice; a later "request taxi" from the same voice is handled as Viper 1-1's.
  2. Callsigns are normalized "Viper 1-1" internally and spoken "Viper one one" (via `phraseology.Callsign`).
  3. Combined positions of one facility share one voice book (the same person); separately staffed positions and other facilities keep their own books: identifying to Nellis Tower does not identify the voice to Nellis Approach.
- **Source:** [UR]; `controller/ident`; [P §8.1] ("combined positions share one ledger")
- **Depends:** ATC-FR-IDN-001
- **Design:** DE-IDN

### ATC-FR-IDN-003 · Callsign correction window and lock
- **Priority:** MVP · **Status:** decided
- **EARS:** While a voice's binding is younger than the correction window (30 s), the facility shall accept an explicit callsign correction; after the window, the facility shall change the label only on "amend callsign, new callsign <X>".
- **Acceptance:**
  1. "Correction, Viper 1-2" at +10 s relabels the voice (log `ident: relabeled`).
  2. "Viper 1-2, …" at +45 s from the voice labelled Viper 1-1 is handled as Viper 1-1's; the log shows `Ignored`.
  3. "Amend callsign, new callsign Hawg 2-1" relabels at any time and moves the flight's session.
- **Source:** [UR] (callsign lock, 2026-09-27); `controller/ident`
- **Depends:** ATC-FR-IDN-002
- **Design:** DE-IDN

### ATC-FR-IDN-004 · Position only after identification
- **Priority:** MVP · **Status:** decided
- **EARS:** The facility shall use a transmission's bound DCS unit (position, altitude, speed, type) only after the voice has identified itself, and only as the unit of the identified flight.
- **Acceptance:**
  1. Before identification the facility computes no pose, procedure step, sequence or jurisdiction check for the voice.
  2. After "Nellis Tower, Viper 1-1, …" with SRS `UnitID` 1234, unit 1234 becomes the position source of Viper 1-1's member 1 (log `atc: bound unit`).
  3. A voice with `UnitID` 0 is served in pilot-only mode (ATC-FR-IDN-006).
- **Source:** [D §4] ("only after the pilot identified"); [UR]
- **Depends:** ATC-FR-IDN-002
- **Design:** DE-IDN, DE-WORLD

### ATC-FR-IDN-005 · Flight and member records
- **Priority:** MVP · **Status:** decided
- **EARS:** When a voice binds to a callsign whose bound unit belongs to a DCS group, the facility shall create or reuse one flight for that group, with members ordered by DCS unit number, and attach the voice to the member whose dash matches the callsign.
- **Acceptance:**
  1. Viper 1-1 (unit 1 of group "Viper-1") creates flight "Viper 1" with members [Viper 1-1, Viper 1-2].
  2. A second voice saying "Viper 1-2" attaches to member 2 of the same flight; no second flight is created.
  3. Two voices bound to the same callsign (pilot and WSO of one F-15E) both map to the same member.
- **Source:** [P §4.1, §5]
- **Depends:** ATC-FR-IDN-004
- **Design:** DE-IDN, DE-FRM

### ATC-FR-IDN-006 · Pilot-only mode
- **Priority:** MVP · **Status:** decided
- **EARS:** While an identified flight has no valid pose (unit not bound, or no DCS sample for 5 s), the facility shall drive its procedure state from the pilot's reports only and mark it `ConfReported`.
- **Acceptance:**
  1. With `UnitID` 0, "Viper 1-1, initial" moves the flight to `initial` with `conf=reported`; no "verify position" is issued.
  2. Geometry-only exits never fire for that flight.
  3. When a pose becomes valid, the flight is re-localised (ATC-FR-PAT-028).
- **Source:** [P §4.3 (Unknown row), §4.4]
- **Depends:** ATC-FR-IDN-004
- **Design:** DE-PROC, DE-IDN

### ATC-FR-IDN-007 · Separate DCS groups flying as one flight
- **Priority:** P2 · **Status:** decided (OD-028, recommendation accepted 2026-10-01)
- **EARS:** When a lead says "flight of two" (or four) and a unit whose callsign root matches ("Viper 1-2") stays within the standard formation of the lead for 10 s, the facility shall adopt that unit as a member of the lead's flight.
- **Acceptance:**
  1. Two single-ship groups "Viper 1-1" and "Viper 1-2" within 1 NM and 100 ft for 10 s after "Viper 1-1, flight of two" become one flight of two.
  2. Different callsign roots ("Viper 1-1" + "Hawg 2-1") are never merged automatically.
  3. The adoption is logged with both unit IDs.
- **Source:** [P §5 (Mapping), §12 Q8]
- **Depends:** ATC-FR-IDN-005, ATC-FR-FRM-001
- **Design:** DE-FRM

### ATC-FR-IDN-008 · Mission-listed flights
- **Priority:** P3 · **Status:** decided
- **EARS:** Where the mission lists flights in `OVERWATCH.atc.flights`, the facility shall use that membership instead of DCS groups for those units.
- **Acceptance:**
  1. `flights = { { callsign = "Viper 1", units = { "Viper-1-1", "Viper-2-1" } } }` makes the two units one flight even though they are in different groups.
  2. An unknown unit name is a validation issue (`atc.flights[1].units[2]: unknown unit`).
- **Source:** [P §5]
- **Depends:** ATC-FR-CFG-004
- **Design:** DE-FRM, DE-CFG

### ATC-FR-IDN-009 · Flight record lifetime
- **Priority:** MVP · **Status:** derived
- **EARS:** The facility shall keep a flight's record while it is in the facility's jurisdiction, on its frequency or handed to it, and drop it 30 s after it left the jurisdiction outside any `outside_ok` step (after the release call of ATC-FR-ASP-010 for a flight it was working), or after 10 min parked and silent.
- **Acceptance:**
  1. A flight departing VFR to the north gets "frequency change approved" at the Class D ring and is dropped 30 s later.
  2. A flight parked at its stand for 10 min with no call is dropped; its sequence slots are released.
  3. The voice binding survives the drop for the rest of the mission (a later call is still "Viper 1-1").
- **Source:** [P §9.2]; Decision 8 [D §1]
- **Depends:** ATC-FR-ASP-010
- **Design:** DE-IDN, DE-JUR

### ATC-FR-IDN-010 · Expected check-in after a handoff
- **Priority:** P2 · **Status:** decided
- **EARS:** When a handed-off flight's voice calls the receiving facility with the expected callsign, the receiving facility shall accept the check-in without asking "say callsign", and it shall not address the flight before that call.
- **Acceptance:**
  1. After Nellis Approach hands Viper 1-1 to Nellis Tower, the tower transmits nothing to Viper 1-1 until "Nellis Tower, Viper 1-1, …".
  2. That first call binds the voice in the tower's book without a "say callsign".
  3. A different voice claiming "Viper 1-1" is bound normally (no special trust).
- **Source:** [D §10 step 3]; [P §9.1]
- **Depends:** ATC-FR-HND-004
- **Design:** DE-HND, DE-IDN

### ATC-FR-IDN-011 · Blocked transmissions
- **Priority:** MVP · **Status:** decided
- **EARS:** When the radio reports a blocked transmission on a facility frequency, the facility shall answer "Blocked, say again." once and act on nothing.
- **Acceptance:**
  1. A `Blocked` inbound gets exactly "Blocked, say again." (no callsign).
  2. Two blocked transmissions within 5 s get one reply (the existing debounce).
- **Source:** `controller.Inbound.Blocked`; `controller/ident` (debounce)
- **Depends:** –
- **Design:** DE-FAC

### ATC-FR-IDN-012 · Calls to other stations are not ours
- **Priority:** MVP · **Status:** derived
- **EARS:** If a transmission on a facility frequency addresses another station or a flightmate ("Viper 1-2, go trail"), then the facility shall not reply to it.
- **Acceptance:**
  1. "Viper 1-2, Viper 1-1, push button three" on 327.0 gets no reply and changes no state.
  2. The transmission is still recorded as a turn in no ledger (it was not addressed to ATC); it is logged at DEBUG.
- **Source:** `controller.SpeakerLabeler` (air-to-air calls); [D §1 Decision 1]
- **Depends:** ATC-FR-IDN-002
- **Design:** DE-FAC, DE-NLU

---

## 2. Facilities, staffing and frequencies (FAC)

### ATC-FR-FAC-001 · Facility registry
- **Priority:** MVP · **Status:** decided
- **EARS:** When a mission starts (or the mission configuration changes), the ATC role shall build a registry of airbases and facilities from DCS airbases, the map pack airfield, the terrain's `radio.lua` tower frequencies and the `OVERWATCH.atc` table.
- **Acceptance:**
  1. On the Nevada test mission the registry lists Nellis with runways 03L/21R and 03R/21L from the map pack and tower frequencies 327.0 AM and 132.55 AM from `radio.lua`.
  2. Carriers appear as airbases with `Carrier = true` keyed by unit name.
  3. Build time is under 2 s (ATC-NFR-PERF-009).
- **Source:** [D §11, §15 step 2]
- **Depends:** ATC-FR-CFG-001; `mappack.ParseRadio` frequencies (OD-004)
- **Design:** DE-REG

### ATC-FR-FAC-018 · DCS's own ATC
- **Priority:** P3 · **Status:** decided (OD-002, recommendation accepted 2026-10-01)
- **EARS:** Where the mission sets `atc.silence_dcs_atc = true`, the ATC role shall put each staffed airbase's DCS AI ATC in radio-silent mode, if the scripting API supports it.
- **Acceptance:**
  1. With the flag set, the DCS comms-menu ATC at Nellis does not transmit; without it both coexist.
  2. If the API is missing, a WARN `atc: cannot silence DCS ATC` is logged once and nothing else changes.
- **Source:** [D §2 (Non-goals), §16 Q2]
- **Depends:** ATC-FR-FAC-001
- **Design:** DE-WORLD

## 3. Airspace and jurisdiction (ASP)

### ATC-FR-ASP-001 · Airspace sources and precedence
- **Priority:** MVP · **Status:** decided
- **EARS:** The airspace model shall take each airbase's airspace from the highest available source: the mission table, then ME drawings and trigger zones by naming convention, then real-world data shipped with the map (FAA NASR class airspace for the US maps, `Source` `faa`), then synthesized FAA-default shapes (`Source` `synthesized`, drawn `(est.)`).
- **Acceptance:**
  1. `airbases.Nellis.class = "C"` beats the heuristic class.
  2. A drawing `CLASS C Nellis 0-60` replaces the synthesized core (P2, ATC-FR-ASP-004).
  3. Each `atc.Airspace` records its `Source`; the announcement names it ("Class D Nellis (synthesized)", "Class B McCarran International (faa)").
- **Source:** Decision 2 [D §1, §5.2, §13.1]
- **Depends:** ATC-FR-CFG-001
- **Design:** DE-ASP

## 4. Clearance delivery (CLR)

### ATC-FR-CLR-001 · IFR clearance in CRAFT order
- **Priority:** P2 · **Status:** decided
- **EARS:** When a parked IFR flight requests its clearance, clearance delivery shall issue an IFR clearance with, in order, the clearance limit, the SID or vectors, the route, the altitude, the departure frequency and the beacon code.
- **Acceptance:**
  1. "Nellis Clearance, Viper 1-1, request IFR clearance to Nellis, with information Bravo" → "Viper 1-1, cleared to Nellis airport, fly runway heading, radar vectors, climb and maintain seven thousand, expect one seven thousand ten minutes after departure, departure frequency one two four point niner five, squawk four one zero one."
  2. The clearance is stored in `Flight.Clearance` with `Issued` set, and a readback question opens (ATC-FR-RBK-001).
  3. Combined facility: the tower answers as "Nellis Tower" with the same content.
- **Source:** [D §7.1]; [F1 §4-2-1, §4-3-2]
- **Depends:** ATC-FR-RBK-001, ATC-FR-CLR-004
- **Design:** DE-CLR, DE-PHR

## 5. Ground and taxi (GND)

### ATC-FR-GND-001 · Taxi clearance to the departure runway
- **Priority:** MVP · **Status:** decided
- **EARS:** When a parked flight requests taxi, ground shall plan a route to the hold-short of the runway in use and issue "Runway <rwy>, taxi via <taxiways>", ending with "hold short of runway <X>" for the first runway the route crosses.
- **Acceptance:**
  1. No crossing: "Viper 1-1, runway two one left, taxi via Alpha, Charlie."
  2. With a crossing of 21R: "Viper 1-1, runway two one left, taxi via Alpha, Charlie, hold short of runway two one right."
  3. The route, the runway and each hold-short are ledger assertions (`clr:taxi_route`, `clr:runway`, `clr:hold_short:21R`) and a readback question opens (ATC-FR-RBK-002).
- **Source:** [D §7.3]; [F1 §3-7-2b]
- **Depends:** ATC-FR-GND-002, ATC-FR-GND-005, ATC-FR-TWR-012
- **Design:** DE-GND, DE-TAXI, DE-PHR

## 6. Tower: runway operations (TWR)

### ATC-FR-TWR-001 · Takeoff clearance
- **Priority:** MVP · **Status:** decided
- **EARS:** When a flight at the hold-short of the runway in use (or lined up) calls ready and the runway sequencer releases it, the tower shall clear it for takeoff.
- **Acceptance:**
  1. FAA: "Viper 1-1, runway two one left, cleared for takeoff." The wind is added before "cleared" when not grounded in the last 2 min ("Viper 1-1, wind two one zero at eight, runway two one left, cleared for takeoff").
  2. ICAO: "Viper 1-1, wind two one zero degrees eight knots, runway two one left, cleared for take-off."
  3. A readback question opens with the runway as a required item (ATC-FR-RBK-002).
- **Source:** [D §7.4, §8]; [F1 §3-9-10]
- **Depends:** ATC-FR-SEQ-015, ATC-FR-RBK-002
- **Design:** DE-TWR, DE-SEQ

## 7. Tower: pattern and overhead (PAT)

### ATC-FR-PAT-001 · Overhead entry
- **Priority:** MVP · **Status:** decided
- **EARS:** When an inbound flight requests the overhead, the tower shall reply with the runway, the wind (unless grounded) and "report initial", adding the pattern altitude and break direction only when they are nonstandard.
- **Acceptance:**
  1. "Nellis Tower, Viper 1-1, one five miles south, two thousand five hundred, inbound for the overhead, information Bravo" → "Viper 1-1, Nellis Tower, runway two one left, report initial."
  2. Nonstandard (mission profile right-hand pattern): "…, runway two one left, right break, report initial."
  3. A sequence request is made (`seq.request`) and the flight enters the `overhead` procedure at `inbound`.
- **Source:** [D §7.4]; [F1 §3-10-12b]; [P §3.3]
- **Depends:** ATC-FR-PAT-013, ATC-FR-SEQ-001
- **Design:** DE-TWR, DE-PROC, DE-YAML

## 8. Tower: sequencing and runway separation (SEQ)

### ATC-FR-SEQ-001 · One sequencer per runway end
- **Priority:** MVP · **Status:** decided
- **EARS:** The tower shall run one sequencer per runway end in use, holding arrivals, option aircraft, departures and crossings, with AI and outside traffic as obstacles that get no instructions.
- **Acceptance:**
  1. `/seq 21L` lists entries with kind, priority, ETA, slot, number and follow.
  2. An AI F-16 on final is an entry with `AI=true`; it is never addressed.
- **Source:** [P §1 Decision 4, §6.1]
- **Depends:** –
- **Design:** DE-SEQ

## 9. Tower: formations (FRM)

### ATC-FR-FRM-001 · A DCS group is one flight
- **Priority:** MVP · **Status:** decided
- **EARS:** The facility shall treat one DCS group as one flight by default, with members ordered by DCS unit number, one shared procedure state and the lead's voice talking for the flight.
- **Acceptance:**
  1. `/proc Viper1` shows one state for both members until a split.
  2. A silent wingman never needs to identify to be tracked as a member.
- **Source:** [P §1 Decision 3, §5]
- **Depends:** ATC-FR-IDN-005
- **Design:** DE-FRM

## 10. Tower: arresting gear (GEAR)

### ATC-FR-GEAR-001 · Barrier / cable call
- **Priority:** P2 · **Status:** decided
- **EARS:** When a flight calls "barrier, barrier, barrier" or "cable, cable, cable" at a field with a departure-end arresting system, the tower shall reply with the gear status and the landing clearance.
- **Acceptance:**
  1. "Viper 1-1, barrier indicates up, runway two one left, cleared to land."
  2. The simulated raise takes 20 s; before that: "Viper 1-1, barrier coming up."
- **Source:** [G gap 6]; [F1 §3-3-6]
- **Depends:** ATC-FR-GEAR-002
- **Design:** DE-GEAR

## 11. Departure and approach radar (RAD)

### ATC-FR-RAD-001 · Radar contact
- **Priority:** P2 · **Status:** decided (OD-006, recommendation accepted 2026-10-01)
- **EARS:** When an identified flight first calls a radar facility and its bound unit has a DCS track (and, in `realistic` sensor mode, a coalition radar sees it), the facility shall say "radar contact" with the flight's position.
- **Acceptance:**
  1. "Nellis Approach, Viper 1-1, one five miles northeast, one two thousand, information Bravo" → "Viper 1-1, Nellis Approach, radar contact, one five miles northeast of Nellis, altimeter two niner niner two."
  2. Position is relative to the field or to a navaid radial/DME ("Las Vegas zero four five radial one five DME").
  3. The altimeter is omitted when the current ATIS letter was reported.
- **Source:** [D §7.5]; [F1 §5-3]
- **Depends:** ATC-FR-IDN-004
- **Design:** DE-RAD

## 12. Holding (HLD)

### ATC-FR-HLD-001 · When to hold
- **Priority:** P2 · **Status:** decided
- **EARS:** The facility shall issue holding only when the runway sequencer cannot give an approach slot within 5 min; otherwise it shall use vectors.
- **Acceptance:**
  1. Three arrivals with a closed runway for 10 min: holding is issued; with a 3 min delay: vectors for spacing.
- **Source:** [T §5.4]
- **Depends:** ATC-FR-RAD-016
- **Design:** DE-HOLD

## 13. Approaches (APP)

### ATC-FR-APP-001 · ILS approach
- **Priority:** P2 · **Status:** decided
- **EARS:** Where DCS has a localizer and glideslope for the runway end, the facility shall offer the ILS with the DCS localizer course, a 3.0° glideslope and DA = threshold + 200 ft (raised for terrain).
- **Acceptance:**
  1. Nellis 21L ILS uses the `IDIQ` course 220.1° true; clearance "cleared ILS runway two one left approach".
- **Source:** [T §4.6.3 (ILS), §4.3 item 5]
- **Depends:** ATC-FR-TPD-008
- **Design:** DE-TERM

## 13a. VFR routes (VFR)

Fighter VFR arrival, departure and transition routes with reporting
points, owned by ATC ([VR]). The examples use Al Dhafra (OMAM, Persian
Gulf, ICAO military variant, runway 31L in use) with the user's Steel
Opera drawings, under the generated names of [VR §2.2]: the labels are
the drawings' texts (`HI`, `LO` and the altitude blocks), and every route
is usable both ways. The section is
numbered 13a so that the section numbers of this file stay stable.

### ATC-FR-VFR-001 · VFR route sources and precedence
- **Priority:** P2 · **Status:** decided
- **EARS:** The registry shall build each towered field's VFR routes from four layers, highest first: the mission table `OVERWATCH.atc.vfr_routes`, mission drawings, the map's VFR pack, and generated defaults. A higher layer shall replace a route of the same name whole or disable it (`false` / `_OFF`), and the field's preferred route per kind and runway shall come from the highest layer that sets it.
- **Acceptance:**
  1. Nellis without mission routes: the generated routes exist (`/atc vfr KLSV` lists them with `source=generated`).
  2. A `VFR_OMAM_HI_Route_Northeast_120-UNL` drawing and a pack route "HI Route Northeast": the drawing wins; `source=drawing`.
  3. `routes["Arrival 13"] = false` removes that generated route.
  4. `preferred.arrival["31L"] = "HI Route Northeast"`: the tower offers the high route northeast first for 31L arrivals.
- **Source:** [VR §1, §3.3]; user review 2026-09-29
- **Depends:** ATC-FR-CFG-001
- **Design:** DE-VFR, DE-REG

## 14. PAR, ASR and no-gyro (GCA)

### ATC-FR-GCA-001 · PAR set-up
- **Priority:** P2 · **Status:** decided
- **EARS:** When a flight is assigned a PAR, the facility shall announce it and, in the civil variant or on mission request, give the lost-communication instructions.
- **Acceptance:**
  1. "Viper 1-1, this will be a PAR approach to runway two one left."
  2. Civil / mission: "If no transmissions are received for one minute in the pattern or five seconds on final, climb to five thousand, proceed to the Nellis TACAN."
- **Source:** [T §4.6.3 (PAR), §5.2]; [F1 §5-10-4]; [G gap 11]
- **Depends:** ATC-FR-RAD-009
- **Design:** DE-GCA

## 15. Conformance monitoring (CNF)

### ATC-FR-CNF-001 · Scope
- **Priority:** P2 · **Status:** decided
- **EARS:** The facility shall monitor conformance only for flights inside its airspace and working it, never on a PAR/ASR final, and never for aircraft outside, even on its procedures.
- **Acceptance:**
  1. A flight flying the published missed 1 NM outside Class D gets no conformance call (textsim scenario 32).
- **Source:** [T §5.6, Decision 7]; Decision 8
- **Depends:** ATC-FR-ASP-008
- **Design:** DE-CNF, DE-JUR

## 16. Safety alerts (SAF)

### ATC-FR-SAF-001 · Low altitude alert
- **Priority:** P2 · **Status:** decided
- **EARS:** If a flight in the facility's jurisdiction is below MVA − 100 ft while vectored, below a charted minimum by > 300 ft, below the glideslope by > 0.7° inside the FAF, or projected (30 s) below the MVA by > 300 ft, then the facility shall issue a low altitude alert immediately, except on a published final above the step-down or within 2 NM of the threshold below 500 ft AGL.
- **Acceptance:**
  1. "Viper 1-1, low altitude alert, check your altitude immediately. The minimum altitude in your area is six thousand five hundred."
  2. First priority on the transmit queue; bypasses conformance caps; repeated once after 10 s if worse.
- **Source:** [T §5.6 (Low altitude alert)]; [F1 §2-1-6]; [G gap 3]
- **Depends:** ATC-FR-RAD-006
- **Design:** DE-CNF, DE-TXQ

## 17. AI traffic (AIT)

AI here means any aircraft the facility cannot instruct: not flown by a
player, whether mission-scripted or driven by another script. The rules
decide which of them are called as traffic, how they are described, and
how the facility separates the flights it works from them. Overwatch-owned
civil traffic follows these rules too, and is also sequenced and controlled
as in ATC-FR-AIT-015 and the CIVT area (§32a).

### ATC-FR-AIT-001 · AI aircraft as traffic
- **Priority:** P2 · **Status:** decided
- **EARS:** The facility shall include in the traffic advisories and traffic information of ATC-FR-RAD-015, ATC-FR-FFL-002 and ATC-FR-TWR-014 every neutral AI aircraft (coalition neutral, including scripted civil "white air") and every friendly AI aircraft that meets their proximity rule and is visible under the server sensor mode (ATC-FR-RAD-002, OD-006).
- **Acceptance:**
  1. Hawg 2-1 on flight following; a neutral scripted 737 projected within 3 NM and 1,000 ft in 90 s: "Hawg 2-1, traffic, two o'clock, six miles, westbound, a 737, one thousand feet above you."
  2. A friendly AI KC-135 meeting the rule is called; one that stays 5 NM away is not.
  3. In `realistic` sensor mode, an AI aircraft no coalition radar sees is not called; in `omniscient` mode it is.
  4. Player aircraft are called exactly as before (ATC-FR-RAD-015, ATC-FR-TWR-014).
- **Source:** [D §16 Q6] (OD-008, resolved); [F1 §2-1-21]; user review 2026-09-29
- **Depends:** ATC-FR-RAD-015, ATC-FR-TWR-014
- **Design:** DE-AIT, DE-RAD

## 18. Flight following and pop-up IFR (FFL)

### ATC-FR-FFL-001 · Flight following
- **Priority:** P2 · **Status:** decided
- **EARS:** When a VFR flight requests flight following, the facility shall assign a code, establish radar contact and give the altimeter.
- **Acceptance:**
  1. "Nellis Approach, Viper 1-1, two zero miles north, eight thousand five hundred, request flight following to Creech" → "Viper 1-1, squawk four one zero three." → (on code) "Viper 1-1, radar contact, two zero miles north of Nellis, altimeter two niner niner two."
  2. `Flight.Following = true`.
- **Source:** [D §7.6]; [F1 ch. 7-6]
- **Depends:** ATC-FR-CLR-004, ATC-FR-RAD-001
- **Design:** DE-RAD

## 19. Class B, C and D entry (ENT)

### ATC-FR-ENT-001 · Class B clearance
- **Priority:** P2 · **Status:** decided
- **EARS:** When a flight requests entry into Class B and entry is allowed, the facility shall say "cleared into Class Bravo airspace" (or "through" / "out of") and record the clearance.
- **Acceptance:**
  1. "Viper 1-1, cleared into Nellis Class Bravo airspace, maintain eight thousand five hundred."
  2. Without that phrase the flight is not cleared, even if radar contact was given.
- **Source:** [D §5.1, §7.7]; [F1 §7-9-2]; [F3 3-2-3]
- **Depends:** ATC-FR-RAD-001
- **Design:** DE-RAD, DE-ASP

## 20. Guard outreach (GUA)

### ATC-FR-GUA-001 · Outreach candidates
- **Priority:** P2 · **Status:** decided
- **EARS:** The facility shall consider for outreach only a player aircraft whose DCS unit is inside the facility's non-hostile airspace, with no voice identified on the facility's frequency for that unit, and that is not a silent member of a known flight.
- **Acceptance:**
  1. An unidentified player F-16 inside Nellis Class D is a candidate; an AI aircraft is not; a silent Viper 1-2 is not (ATC-FR-FRM-007).
  2. An aircraft outside the airspace is never a candidate (ATC-FR-GUA-011).
- **Source:** [D §7.8]; [UR] (guard only to unidentified aircraft in our airspace)
- **Depends:** ATC-FR-ASP-008, ATC-FR-FRM-007
- **Design:** DE-GUARD

## 21. Weather (WX)

### ATC-FR-WX-001 · Weather at the field
- **Priority:** MVP · **Status:** decided
- **EARS:** The facility shall take the field's weather from the mission weather (clouds, visibility, fog, dust, QNH, temperature) and DCS-gRPC `GetWind` / `GetTemperatureAndPressure` at the field, refreshed at least every 60 s.
- **Acceptance:**
  1. `atc.Weather` for Nellis has wind, gust, visibility, ceiling, cover, temperature, dew point and QNH.
  2. Without DCS-gRPC the last known weather is used and flagged stale (ATC-NFR-DEG-001).
- **Source:** [D §7.9 (Weather source)]
- **Depends:** –
- **Design:** DE-WX

## 22. ATIS (ATI)

### ATC-FR-ATI-001 · ATIS content and order
- **Priority:** P2 · **Status:** decided
- **EARS:** The ATIS shall broadcast, in order: airport, information letter and observation time (Z); wind, visibility, sky condition, temperature/dew point, altimeter; approach and runways in use; threat (MANPADS) alerts; active field NOTAMs (short form); configurable remarks; and "advise on initial contact you have information <letter>".
- **Acceptance:**
  1. Golden text: "Nellis information Bravo, one four five five Zulu. Wind two one zero at eight. Visibility one zero. Few clouds at eight thousand. Temperature two four, dew point one. Altimeter two niner niner two. ILS runway two one left approach in use, landing and departing runway two one left. Notices to airmen: runway zero three left two one right closed. Hot pits open on the east ramp. Advise on initial contact you have information Bravo."
- **Source:** [D §7.9]; [F1 §2-9-3]
- **Depends:** ATC-FR-WX-001
- **Design:** DE-ATIS

## 23. NOTAMs and ATIS remarks (NTM)

### ATC-FR-NTM-001 · NOTAM sources
- **Priority:** P2 · **Status:** decided
- **EARS:** The ATC role shall read mission NOTAMs from `OVERWATCH.atc.notams`, from mission scripts at runtime (`OVERWATCH.notam({...})`, `OVERWATCH.notam_cancel("N1")`), and standing per-theatre NOTAMs from the server config, with the usual precedence and replacement by `id`.
- **Acceptance:**
  1. A NOTAM added by script with an existing `id` replaces it.
  2. A cancel removes it; both are seen within one probe period (ATC-FR-CFG-014).
- **Source:** [D §13.6]
- **Depends:** ATC-FR-CFG-014
- **Design:** DE-NOTAM, DE-CFG

## 24. Carrier: case, control areas and EMCON (CVC)

Examples in this chapter use CVN-71 "Rough Rider" (its ship callsign, the
CVN-71 default of the name table; another ship speaks its own: CVN-75
"Lone Warrior", ATC-FR-CVC-008), marshal
285.0, BRC 360, final bearing 351, the section led by side number 205
holding hands with 211 (F/A-18C 2-ship), singles 301, 302, 401, 402, and CV
NATOPS 2009 phraseology (NAVAIR 00-80T-105, Jul 2009; "[8 §x p.y]" cites
section and page). Carrier pilots name themselves by side number (modex)
and give the lineup by the other side numbers ("holding hands with two one
one"); marshal answers a section by the lead's side number. Examples
corrected per CV NATOPS §6.1.2 p.6-1, §4.4.2 p.4-5 and §6.4.13.1
p.6-20/21 (user correction 2026-10-01). The check-in wording is the
user's (CV NATOPS §6.1.2 lists the items only): "[Rough Rider] Marshal,
two zero five, [holding hands with two one one,] marking mom's two seven
zero for four five, angels two zero, state six point two" (or "low
state"); the pilot does not request a case, marshal assigns it (user
correction 2026-10-01).

### ATC-FR-CVC-001 · Case determination
- **Priority:** P2 · **Status:** decided
- **EARS:** The carrier facilities shall compute the recovery case from the DCS weather at the ship (cloud base minus deck height, visibility / fog) and the sun with `atc.DetermineCase`, apply `atc.CaseTracker` hysteresis, and honour a mission pin (`case = 1|2|3`).
- **Acceptance:**
  1. Day, ceiling 3,500 ft, visibility 10 NM → Case I; ceiling 2,000 → Case II; ceiling 800 → Case III.
  2. A ceiling oscillating around 3,000 ft does not flip the case within the hold time (default 5 min).
- **Source:** [D §7.10 (Case determination)]; [8 4.2, 4.1.3]
- **Depends:** ATC-FR-WX-001
- **Design:** DE-CV

## 25. Carrier: marshal (MSH)

### ATC-FR-MSH-001 · Check-in
- **Priority:** P2 · **Status:** decided
- **EARS:** When a flight entering the carrier control area checks in, marshal shall read the lead's side number, the lineup (the side numbers holding hands with him), the position (the bearing from the ship, "marking mom's" radial, then DME), the altitude (angels) and the fuel state or low state (plus hung or unexpended ordnance and NAVAID status when given), and work the section as one flight keyed by the lead's side number (`atc.FlightTable.HoldHands`). The pilot does not request a case: marshal decides it (ATC-FR-CVC-001) and assigns it in the reply (ATC-FR-MSH-002/003); a case or approach said is tolerated, the case is not used.
- **Acceptance:**
  1. "Rough Rider Marshal, two zero five, holding hands with two one one, marking mom's two six zero for four five, angels one eight, state six point two" → lead 205, wingman 211 (lineup 2), 260° / 45 DME, angels 18, state 6.2, the case marshal assigns recorded (`/proc 205`); a later call from 211 binds to the section's second member (corrected per CV NATOPS §6.1.2 p.6-1 (user correction 2026-10-01)).
  2. Forms read alike: "Marshal, 205, marking mom's 270 for 45, angels 20, state 6.2"; "marking moms two seven zero, four five"; "mother's"; DME "45", "four five" or "forty-five"; "low state six point two" (low-state flag) (user correction 2026-10-01).
  3. Missing items are asked once ("Two zero five, say state").
- **Source:** [D §7.10 (Marshal check-in)]; [8 §6.1.2 p.6-1 (1. position, 2. altitude, 3. fuel state (low state in flight), 4. total number of aircraft in flight (lineup), 5. type approach requested, 6. other pertinent information, 7. COD load report), §4.4.2 p.4-5 (ship's call sign on initial contact)]; wording and "no case request" per the user (2026-10-01): item 5 is not expected, marshal assigns the case and approach
- **Depends:** ATC-FR-CVC-008
- **Design:** DE-MSH, DE-NLU

## 26. Carrier: Case I tower (CV1)

### ATC-FR-CV1-001 · Port holding
- **Priority:** P2 · **Status:** decided
- **EARS:** The carrier facilities shall expect Case I flights in port holding (left-hand circle tangent to the BRC, ship at 3 o'clock, ≤ 5 NM diameter, ≥ 2,000 ft, 1,000 ft apart) and place them there by geometry.
- **Acceptance:**
  1. A flight at 4,000 ft, 5 NM on the port side circling left is `case1:port_holding`.
- **Source:** [D §7.10 (Case I)]; [8 6.2.1]; [P §3.4]
- **Depends:** ATC-FR-CVC-005
- **Design:** DE-YAML, DE-CV

## 27. Carrier: Case III approach and CCA (CV3)

### ATC-FR-CV3-001 · Case III descent expectations
- **Priority:** P2 · **Status:** decided
- **EARS:** The carrier approach procedure shall expect 250 KIAS and 4,000 fpm to the platform (5,000 ft), then 2,000 fpm, dirty at 8 NM unless directed, the 6 NM fix at 1,200 ft and 3 NM, and place the flight by geometry.
- **Acceptance:**
  1. A nominal Case III trajectory passes `commencing → platform → ten_miles → eight → six → three` with `conf=observed`.
- **Source:** [D §7.10 (Case III approach)]; [8 6.4.7.2]; [P §3.5 `case3`]
- **Depends:** ATC-FR-CVC-005
- **Design:** DE-YAML, DE-CV

## 28. Carrier: departures and CATCC (CVD)

### ATC-FR-CVD-001 · Case I departure
- **Priority:** P2 · **Status:** decided
- **EARS:** The carrier facilities shall expect Case I departures to make a clearing turn, stay at 500 ft to 7 NM and then climb VMC.
- **Acceptance:**
  1. `case1_departure.yaml` follows a nominal launch with no transmissions.
- **Source:** [D §7.10 (Departures)]; [8 5.13.3]
- **Depends:** ATC-FR-CVC-005
- **Design:** DE-YAML

## 29. Emergencies (EMG)

### ATC-FR-EMG-001 · Emergency priority
- **Priority:** MVP · **Status:** decided
- **EARS:** When a flight declares "mayday", "pan-pan" or (USAF) "emergency fuel" / "mayday fuel", the facility shall give it `PriEmergency` in every sequencer, ahead of spin re-entries and everyone else.
- **Acceptance:**
  1. A mayday on initial becomes number one; the others are re-numbered with update phrases.
  2. The emergency flag travels with handoffs (ATC-FR-HND-013).
- **Source:** [D §7.11]; [P §1 Decision 5, §6.2]; [AF1 glossary (Emergency Fuel)]
- **Depends:** ATC-FR-SEQ-002
- **Design:** DE-EMG, DE-SEQ

## 30. Handoffs (HND)

### ATC-FR-HND-001 · Handoff contract
- **Priority:** P2 · **Status:** decided
- **EARS:** The ATC role shall transfer flights with an `atc.Handoff` carrying flight, voice, unit, from, to, frequency, reason, squawk, remarks and, between ATC facilities, a `ProcSnapshot` (procedure, step, intent, members, sequence entry, told items, flags).
- **Acceptance:**
  1. A tower→departure handoff has `Proc.Step = "departed"` and the assigned altitude in `Told`.
- **Source:** [D §10]; [P §9.1]
- **Depends:** –
- **Design:** DE-HND

## 30a. FIR and ARTCC area control (FIR)

Area control is the en-route agency: "<name> Control" for an ICAO FIR,
"<name> Center" for an FAA ARTCC ([AC]). It serves player flights between
the war zone and the terminal areas, calls civil traffic to them, and
hands them to approach. It is kept minimal and trivial to run. The
facility kind is the scaffold's `atc.AgencyCenter`. Unless stated
otherwise, the examples use the Persian Gulf map with the Emirates FIR
(OMAE, "Emirates Control", 132.1, ICAO military variant) and Al Dhafra
Approach (126.5). The section is numbered 30a so that the section numbers
of this file stay stable.

### ATC-FR-FIR-001 · FIR and sector model
- **Priority:** P2 · **Status:** derived
- **EARS:** The registry shall hold each FIR or ARTCC with an ID, a spoken base name, a country, a lateral boundary polygon, vertical limits, frequencies and at least one sector. Each sector shall have a boundary (default: the FIR's), an altitude block and an optional frequency. A low/high split shall be two sectors over the same boundary with blocks meeting at the split level (default FL245).
- **Acceptance:**
  1. A FIR with no sectors given gets one sector equal to the FIR.
  2. `SECTOR_OMAE_East_0-245` and `SECTOR_OMAE_East_245-UNL` make a low/high split of the East sector.
  3. `Registry.FIRAt(pos, alt)` returns the FIR and sector at a point in O(1) (grid index).
- **Source:** [AC §2, §3]; user review 2026-09-29
- **Depends:** ATC-FR-FAC-001
- **Design:** DE-FIR, DE-REG

## 30b. IFR to and from operating areas (OPS)

A fighter takes an IFR clearance to a military operating area (a MOA,
restricted area or ATCAA), departs as usual, is handed toward the area,
released into it ("cleared into …, maintain block …") and works it under
MARSA with the range or the AWACS, then asks for an IFR clearance back
([AC §13]). Examples: Nellis, the Reveille MOA drawn `MOA Reveille
180-280 251.0` (FL180 to FL280, range control on 251.0), Los Angeles
Center (ZLA) and Darkstar (the blue AWACS). The section is numbered 30b
so that the section numbers of this file stay stable.

### ATC-FR-OPS-001 · An operating area as a clearance limit
- **Priority:** P2 · **Status:** derived
- **EARS:** When a flight asks for an IFR clearance to a name that matches an operating area of the area model (drawings, openAIP, server config; matched like a transit request, ATC-FR-ENT-007), clearance delivery shall make the area (or its entry fix, when the data gives one) the clearance limit, and the flight's clearance shall hold the area as its destination and the departure field as its origin.
- **Acceptance:**
  1. "Nellis Clearance, Viper 1-1, request IFR clearance to Reveille." → "Viper one one, Nellis Tower, cleared to the Reveille MOA, fly runway heading, radar vectors, climb and maintain seven thousand, expect flight level one eight zero ten minutes after departure, remain this frequency, squawk four one zero one."
  2. `Flight.Clearance.Area` = `moa:reveille`, `Origin` = "Nellis"; the clearance travels with the handoffs (ATC-FR-HND-005).
  3. An area with `entry: the Reveille entry` (server config): "cleared to the Reveille entry, …".
- **Source:** FAA JO 7110.65 §4-2-1, §4-3-2; USAF practice (MOA / ATCAA clearances); user task 2026-10-02
- **Depends:** ATC-FR-CLR-001, ATC-FR-CLR-002, ATC-FR-ASP-013
- **Design:** DE-OPS (atc opsarea.go, facility areaops.go, clearance.go)

## 31. Hostile airspace (HOS)

### ATC-FR-HOS-001 · Hostile sources
- **Priority:** P2 · **Status:** decided
- **EARS:** The ATC role shall build the hostile set as the union of `HOSTILE …` and `MEZ …` zones and drawings, areas of ROE rules with `declare = "hostile"`, the hostile side of `FLOT` / `FEBA` lines, the synthesized terminal airspace of enemy-owned airbases, and `AOR` areas (outbound handoff), each source disableable in `atc.hostile`.
- **Acceptance:**
  1. A `FLOT` line with more enemy airbases to the north makes the north side hostile; `hostile_side = "south"` overrides.
  2. `enemy_airbases = false` removes enemy fields from the set.
- **Source:** [D §6]
- **Depends:** ATC-FR-CFG-008
- **Design:** DE-HOS

## 32. Civil traffic (CIV)

### ATC-FR-CIV-001 · Civil detection
- **Priority:** P3 · **Status:** decided (OD-069, recommendation accepted 2026-10-01)
- **EARS:** The facility shall classify a flight as civil when its bound unit's type is in the civil/warbird-trainer list, its callsign is a registration or airline form, or the mission marks its coalition neutral/civil.
- **Acceptance:**
  1. "November one two three alpha bravo" in a Yak-52 → civil; "Southwest one two three" → civil; "Viper 1-1" in an F-16 → military.
- **Source:** [D §9]
- **Depends:** ATC-FR-IDN-004
- **Design:** DE-CIV

## 32a. Civil traffic: Overwatch-owned (CIVT)

Civil "white air" airliners that Overwatch itself generates, flies and
deletes, replacing the user's mission script (Gulf civil traffic v6.22,
analysed in [CC]). The design is [CT], adopted by the resolution of the
AIT open decision (ATC-FR-AIT-015).

**Guiding principle (user, 2026-09-29).** Civil AI aircraft are set
dressing: actors and background scenery that make the mission feel real.
Players always take priority. When civil traffic would get in the way, or
misbehaves, it quietly gets out of the way: it is slowed, flies over or is
deleted, out of the players' view where possible. Don't overthink it.

What matters, in order (the product vision, [D §0]):

1. **Predictable positions.** Civil flights follow their SIDs and STARs at
   their planned levels and speeds, so where they are is deterministic and
   traffic calls are right (ATC-FR-CIVT-031).
2. **Players first.** Slots, spawns and deletes always give way to players.
3. **Landing realism is secondary.** A real landing is the default where
   configured, but fly-over-and-delete is an acceptable result, and nothing
   more is built around landing fidelity.

The core is P2: spawning with callsigns, types and liveries; routes and
levels; real landings with the fallback; slots and players first; the
tower check-in only when a player listens; traffic calls; the cap; and
cleanup. Anything else is P3, "later / only if needed" (M8).

Unless stated otherwise, the examples use the `gulf-steel-opera` preset
on the Persian Gulf map. The facility there speaks ICAO civil at Dubai
Intl (OMDB), with runway 30L in use and Dubai Tower on 118.75. The
section is numbered 32a so that the section numbers of this file stay
stable.

### ATC-FR-CIVT-001 · Overwatch owns civil traffic
- **Priority:** P2 · **Status:** decided
- **EARS:** Where civil traffic is enabled for a mission (a preset or `OVERWATCH.civil.airports`), Overwatch shall generate, control and delete all civil airliner traffic itself, spawning single-aircraft groups in a neutral country and treating as owned only the groups recorded in its Lua registry for the current mission run and instance.
- **Acceptance:**
  1. With `OVERWATCH.civil = { preset = "gulf-steel-opera" }`, owned flights appear at the five preset airports; `/civil` lists them with state and slot.
  2. A neutral AI airliner placed in the mission editor is not owned: it is never commanded or deleted, and is called as ordinary AI traffic (ATC-FR-AIT-001).
  3. If the mission still runs the Lua script (`_G.GulfCivilTraffic` exists), Overwatch civil traffic does not start and announces "civil traffic: mission script GulfCivilTraffic active; Overwatch civil traffic disabled", unless `civil.override_script = true`.
  4. No preset and no airports: civil traffic is off and nothing is logged above DEBUG.
- **Source:** [CT §1 C1, §2, §8.4]; [CC]; user review 2026-09-29
- **Depends:** ATC-FR-CFG-001
- **Design:** DE-CIVT, DE-CFG

## 33. Phraseology variants (PHR)

### ATC-FR-PHR-001 · Variant per facility
- **Priority:** MVP · **Status:** decided (OD-049, recommendation accepted 2026-10-01)
- **EARS:** Each facility shall speak one `atc.Variant{Standard, Military, Service}`: the standard from the theatre default (Nevada FAA, every other theatre ICAO) unless the mission sets `standard` (global or per airbase); military for military fields; the service (USAF by default at FAA military fields; USN for carriers) from the mission.
- **Acceptance:**
  1. Nevada, Nellis: FAA, military, USAF.
  2. Caucasus, Batumi: ICAO, civil; Senaki: ICAO, military.
  3. `airbases.Incirlik.standard = "faa"` overrides the theatre.
- **Source:** [D §8]; [UR] (FAA 7110.65 phraseology, ICAO variant, USAF base DAFMAN 13-204v3 + ACC, NATOPS for the carrier)
- **Depends:** ATC-FR-CFG-004
- **Design:** DE-PHR, DE-REG

## 34. Comm discipline (COM)

### ATC-FR-COM-001 · Replies are never held; unsolicited calls are arbitrated
- **Priority:** MVP · **Status:** decided
- **EARS:** The facility shall send replies to pilot calls at once and route every unsolicited call (traffic, sequence updates, prompts, broadcasts, conformance) through the arbiter with ATC class rules.
- **Acceptance:**
  1. A reply is never delayed by the arbiter (existing Loop behaviour).
  2. ATC class rules exist for `atc.traffic`, `atc.seq`, `atc.prompt`, `atc.broadcast`, `atc.conformance`, `atc.safety`, `atc.pace` (PAR/CCA) with Quiet/Repeat values in design §6.
- **Source:** `controller/arbiter.go`; [P §10.3]
- **Depends:** –
- **Design:** DE-TXQ

## 35. Readbacks (RBK)

### ATC-FR-RBK-001 · Readback question
- **Priority:** MVP · **Status:** decided
- **EARS:** When the facility issues a clearance, it shall open one readback question per flight whose `About` lists the assertions that need reading back.
- **Acceptance:**
  1. A taxi clearance with a hold-short opens `readback` with About = [clr:runway, clr:hold_short:21R].
- **Source:** [D §7]; [P §8.1]
- **Depends:** ATC-FR-COM-009
- **Design:** DE-RBK, DE-LEDG

## 36. Mission configuration and validation (CFG)

### ATC-FR-CFG-001 · Layers and precedence
- **Priority:** MVP · **Status:** decided
- **EARS:** The ATC role shall combine configuration layers in this precedence: the `OVERWATCH.atc` table > ME drawings and trigger zones > real-world map pack data > DCS defaults, with the server config supplying server-wide defaults only (never mission content) and a later runtime admin layer.
- **Acceptance:**
  1. A mission `runway = "21L"` beats the DCS default; a server `atc.staffing_cap` is used when the mission sets none; the server never defines airbases.
- **Source:** Decision 2 [D §1, §13.1]
- **Depends:** –
- **Design:** DE-CFG

## 37. Terminal-procedure data (TPD)

### ATC-FR-TPD-001 · One leg model
- **Priority:** P2 · **Status:** decided
- **EARS:** The terminal-procedure package shall represent every SID, STAR, approach and hold from every source as procedure → transitions → ARINC 424 legs (IF, TF, CF, DF, FA, FC, FD, FM, CA, CD, CI, CR, VA, VD, VI, VM, VR, AF, RF, HA, HF, HM, PI) with fixes, altitude and speed constraints, and reject any other path terminator at load.
- **Acceptance:**
  1. Every PT in [T §3.3] parses from YAML and compiles; `TR` is rejected with a report line.
- **Source:** Decision 10 [D §1]; [T §3.1–3.5]
- **Depends:** –
- **Design:** DE-TERM

## 37a. Procedure encoding (ENC)

The Overwatch Procedure Encoding (OPE): the canonical, DCS-specific,
on-disk form of every terminal procedure, and the pipeline that derives it
from ARINC 424, AIXM and reviewed eAIP transcriptions ([T §10], Decision
11). Examples use the Nevada pack, KLAS from CIFP and the fictional Nellis
T21L of [T §4.5.1]. The section is numbered 37a so that the section
numbers of this file stay stable.

### ATC-FR-ENC-001 · Canonical encoding format
- **Priority:** P2 · **Status:** decided
- **EARS:** The procedure data shall be stored, per airport, in the Overwatch Procedure Encoding: a YAML document with `format: ope` and `format_version: 1`, keys in the fixed order of the `ope` types, charted units (feet, NM, knots, degrees magnetic and true) and DCS-frame positions (lat/lon on the DCS projection and map x/z); the reader shall reject another format version and any unknown key.
- **Acceptance:**
  1. `data/procpacks/nevada/ope/Nellis.ope.yaml` decodes; an added key `foo: 1` fails with `ope: Nellis.ope.yaml:12: unknown key "foo"`.
  2. `format_version: 2` fails with `ope: unsupported format_version 2` and the airport falls back to the lower layers.
  3. Every position carries `lat`, `lon` (7 decimals), `x`, `z` (0.1 m).
- **Source:** Decision 11 [D §1]; [T §10.1 E3, §10.4]
- **Depends:** ATC-FR-TPD-001
- **Design:** DE-OPE

## 37b. F10 map layer (MAP)

Overwatch draws what it believes exists on the DCS F10 map, per
coalition, for ATC and AWACS ([MAP], Decision 12). Examples use Nevada
with Nellis (blue, Class C synthesized by the mission class override; its default is an estimated D) and a blue AWACS with lanes North
and South, CAP Lion, and `HOSTILE North`; the VFR examples use Al Dhafra
([VR §2]). The section is numbered 37b so that the section numbers of this
file stay stable.

### ATC-FR-MAP-001 · Draw Overwatch's model
- **Priority:** P2 · **Status:** decided
- **EARS:** Where the F10 map layer is enabled, Overwatch shall draw on the F10 map, per coalition, the features its own model holds, through one producer per layer (airspace, hostile, awacs, notams, vfr, fixes, procedures, civil, legend) that reads the owners' published snapshots and never calls DCS.
- **Acceptance:**
  1. With `OVERWATCH.map = { enabled = true }`, blue sees the Nellis Class C (mission override), `HOSTILE North`, the lanes, CAP Lion and the fixes; `/map status` lists the counts per layer.
  2. A change in the ATC model (a runtime NOTAM) appears on the map within the debounce plus one batch (≤ 5 s).
- **Source:** Decision 12 [D §1]; [MAP §1 M1, §4]; user brief 2026-09-29
- **Depends:** ATC-FR-CFG-001
- **Design:** DE-MAP

## 38. Performance (PERF)

### ATC-NFR-PERF-001 · Hot path
- **Priority:** MVP · **Status:** decided
- **EARS:** The facility's `Handle` shall complete parse-to-reply-text in under 5 ms (p99) and shall never call DCS; it shall read the last world snapshot, treating a pose older than 2 s as Unknown.
- **Acceptance:**
  1. Benchmark `BenchmarkHandle` with 20 flights: p99 < 5 ms.
  2. A race-test asserts no DCS client call on the Handle path (fake sim panics if called).
- **Source:** [P §10.3]
- **Depends:** –
- **Design:** DE-FAC, DE-WORLD

## 39. Observability and logging (OBS)

### ATC-NFR-OBS-001 · Structured logs
- **Priority:** MVP · **Status:** decided
- **EARS:** The ATC role shall log with `log/slog` only, with the fields `role=atc`, `facility`, `flight`, `voice`, `unit` where applicable.
- **Acceptance:**
  1. A lint test finds no `fmt.Print*` or `log.*` in `internal/atc`.
- **Source:** AGENTS.md
- **Depends:** –
- **Design:** DE-OBS

## 40. Testing (TST)

### ATC-NFR-TST-001 · Unit tests
- **Priority:** MVP · **Status:** decided
- **EARS:** Each ATC package shall have table-driven unit tests with no network, using fakes and fixtures.
- **Acceptance:**
  1. `make check` (`-tags nolibopusfile`, `-race`) passes.
- **Source:** AGENTS.md
- **Depends:** –
- **Design:** DE-TXT

## 41. Degraded modes (DEG)

### ATC-NFR-DEG-001 · DCS-gRPC unavailable
- **Priority:** MVP · **Status:** derived
- **EARS:** While DCS-gRPC is unavailable, the ATC role shall keep existing facilities on the air in pilot-only mode, refuse radar services ("unable radar service"), use the last known weather (or omit weather items), and not start new facilities that need the registry.
- **Acceptance:**
  1. With the fake sim disconnected, "request taxi" still gets a clearance (no route: ATC-FR-GND-015); "request vectors" → "Viper 1-1, unable radar service, say intentions."
- **Source:** [spec]; [P §4.4]
- **Depends:** ATC-FR-IDN-006
- **Design:** DE-DEG
