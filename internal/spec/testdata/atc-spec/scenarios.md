# ATC role: acceptance scenarios

Each scenario is a scripted radio exchange that becomes a textsim script
(`cmd/textsim/testdata/atc-<name>.txt`, ATC-NFR-TST-004). Lines give the
time (T+mm:ss on the fake clock), the position of the aircraft that
matters, the speaker and the exact transmission, and the checks the script
asserts (`@expect`, `@expect-none`, `@must-not`, `@proc`, `@seq`,
`@ground`). Controller lines are the expected output; where a line is
optional it is marked *(opt)*.

Common setup unless stated: Nevada, Nellis AFB (field elevation 1,870 ft),
zero-config combined **Nellis Tower** on 327.0 / 132.55 (FAA, military,
USAF), runway in use **21L** (wind 210° at 8 kt, altimeter 29.92, ceiling
none, visibility 10), Class C synthesized (5 NM core surface–4,000 AGL).
"Viper 1" = F-16C 2-ship (group Viper-1, units Viper 1-1, Viper 1-2),
"Hawg 2" = A-10C 2-ship, "Viper 3" = F-16C single. Taxiway names come from
the Nellis ground test fixture (`facility.NellisGroundFixture`; its letters
A, B, C, D are fixture names, not the real field's). Positions: "NE 8 NM" = 8 NM
northeast of the field on or near the 21L extended centreline (arrivals to
21L come from the northeast); altitudes MSL.

The coverage matrix at the end lists every MVP requirement and the
scenarios that cover it. S-01 … S-21 are the MVP set (milestone M1);
S-22 onward cover the later milestones and the tricky cases.

---

## MVP scenarios (M1)

### S-01 · Identity, radio check and station basics
Script `atc-identity.txt`. Setup: one client tunes 327.0 at T−00:05 (the
tower is staffed lazily).

| T | Position | Speaker | Transmission / event | Checks |
|---|---|---|---|---|
| −00:05 | stand 12 | (roster) | client tunes 327.0 | `@ground "staffed nellis tower"` (log); SRS client "Nellis Tower" exists |
| 00:00 | stand 12 | voice A | "Nellis Tower, radio check." | |
| 00:01 | | Tower | "Station calling Nellis Tower, say callsign." | `@expect "say callsign"`; no flight created |
| 00:10 | stand 12 | voice A | "Nellis Tower, Viper 1-1, radio check." | |
| 00:11 | | Tower | "Viper 1-1, Nellis Tower, read you loud and clear." | `@proc Viper11 proc=none`; unit bound (log `bound unit`) |
| 00:20 | stand 12 | voice A | "Say altimeter." | reply on 327.0 (call came on 327.0) |
| 00:21 | | Tower | "Viper 1-1, altimeter two niner niner two." | `@expect "altimeter two niner niner two"` |
| 00:30 | stand 12 | voice A (on 132.55) | "Say active." | |
| 00:31 | | Tower (132.55) | "Viper 1-1, Nellis Tower, runway two one left in use." | reply on 132.55; station name after the frequency change (ATC-FR-PHR-010) |
| 00:35 | stand 12 | voice A | "Correction, Viper 1-2." | within 30 s of binding |
| 00:36 | | Tower | *(none: relabel only)* | `@expect-none`; label Viper 1-2 |
| 00:40 | stand 12 | voice A | "Amend callsign, new callsign Viper 1-1." | |
| 00:41 | | Tower | "Viper 1-1." | label Viper 1-1 |
| 01:30 | stand 12 | voice A | "Viper 1-2, …" (a similar callsign, after the window) | handled as Viper 1-1's (log `Ignored`) |
| 01:40 | stand 13 | voice B | "Viper 1-2, Viper 1-1, go button four." (air-to-air) | `@expect-none` |
| 01:50 | | (radio) | blocked transmission ×2 within 5 s | `@expect "blocked, say again"` once |
| 02:00 | stand 12 | voice A | (STT returns empty) | `@expect "Viper 1-1, say again"` |
| 02:05 | stand 12 | voice A | (STT returns empty again) | no second identical "say again"; ladder step 2 |

**Covers:** ATC-FR-IDN-001, IDN-002, IDN-003, IDN-004, IDN-005, IDN-011, IDN-012, FAC-001, FAC-002, FAC-006, FAC-010, FAC-011, FAC-014, FAC-020, TWR-016, WX-003, PHR-001, PHR-002, PHR-010, COM-010, CFG-003, ATC-NFR-DEG-005.

### S-06 · Go-around for an occupied runway, but not for a flightmate
Script `atc-goaround-flightmate.txt`. Viper 1 (2-ship) and Hawg 2-1
(single A-10C) in the overhead; Viper 1-2 rolls out slowly.

| T | Position | Speaker | Transmission / event | Checks |
|---|---|---|---|---|
| 00:00 | Viper 1 initial | Viper 1-1 | "Viper 1-1, initial." | `@expect-none` |
| 00:30 | Hawg initial | Hawg 2-1 | "Hawg 2-1, initial." | |
| 00:31 | | Tower | "Hawg 2-1, number two, follow the F-16s on the break." | |
| 01:20 | 1-1 on runway | – | Viper 1-1 lands | |
| 01:32 | 1-2 short final, 1-1 rolling at 4,000 ft | – | tt(1-2) = 10 s with a flightmate on the runway | `@must-not "Viper 1-2, go around"` |
| 01:42 | 1-2 lands, rolls slowly (40 kt at the 6,000 ft mark) | – | – | occupancy: two Vipers |
| 02:05 | Hawg short final (tt 20 s) | (sequencer) | projection: 1-2 still on the runway at Hawg's threshold crossing, not improving | |
| 02:13 | Hawg tt 12 s | Tower | "Hawg 2-1, go around, runway occupied." | Critical; `@proc Hawg21 intent=go_around` |
| 02:15 | Hawg climbing | Hawg 2-1 | "Hawg 2-1, going around." | |
| 02:30 | upwind | Hawg 2-1 | "Hawg 2-1, request closed." | |
| 02:31 | | Tower | "Hawg 2-1, left closed traffic approved." | never "climb runway heading, maintain" (`@must-not "maintain two thousand"`) |
| 02:40 | – | Viper 1-2 | "Viper 1-2, clear." | |
| 03:30 | Hawg downwind | – | new arrival Viper 3-1 at initial with an equal ETA | Hawg keeps number one (`PriSpinReentry`): `@seq 21L Hawg21,Viper31` |

**Covers:** ATC-FR-SEQ-005, SEQ-006, SEQ-012, SEQ-013, SEQ-020, SEQ-023, PAT-011, PAT-009, TWR-007, COM-002, SEQ-018.

## Later milestones and tricky cases (M2–M8)

Setup additions: split positions where stated (mission `freqs`), **Nellis
Approach** on 124.95 (Nellis + Creech), synthesized **Class B** at Nellis
for S-22 (mission `class = "B"`), AWACS **Darkstar** on 251.0 with an
`AOR North` area and a `FLOT` line 60 NM north.

### S-22 · Class B entry denied
Script `atc-classb-denied.txt`. Mission: `airbases.Nellis.class = "B"`,
Class B saturation policy (OD-052): more than 6 VFR flights inside
→ deny; the scenario starts with 7.

| T | Position | Speaker | Transmission / event | Checks |
|---|---|---|---|---|
| 00:00 | 25 NM NE, 8,500, outside the B | Viper 3-1 | "Nellis Approach, Viper 3-1, two five miles northeast, eight thousand five hundred, VFR, request Class Bravo entry for the overhead." | |
| 00:01 | | Approach | "Viper 3-1, Nellis Approach, squawk four one zero three." | code assigned |
| 00:10 | (squawk 4103 seen) | Approach | "Viper 3-1, radar contact, two five miles northeast of Nellis, remain clear of Class Bravo airspace, traffic saturation, expect five minutes delay, altimeter two niner niner two." | `@must-not "cleared into class bravo"`; ledger refusal assertion |
| 00:40 | 22 NM NE, heading for the 20 NM shelf | Viper 3-1 | "Viper 3-1, request Bravo entry." | |
| 00:41 | | Approach | "Viper 3-1, remain clear of Class Bravo airspace." | same answer (ledger) |
| 05:30 | saturation cleared | Approach | "Viper 3-1, cleared into Nellis Class Bravo airspace, maintain eight thousand five hundred, proceed direct Nellis, report initial with Tower one three two point five five." | explicit phrase |
| 12:00 | leaving the B northbound after landing and departing | Approach | "Viper 3-1, leaving Nellis Bravo airspace, radar service terminated, squawk one two zero zero." | |

**Covers:** ATC-FR-ENT-001, ENT-002, ENT-006, RAD-001, CLR-004.

### S-26 · Spin versus break (Case I)
Script `atc-case1-spin.txt`. CVN-71, Case I, BRC 360, deck clear,
`nmax = 6`, interval 50 s. Three aircraft already in the pattern with
touchdowns at +100, +150, +200 s. The 4-ship led by side number 205
(holding hands with 211, 212 and 213 since its check-in, [8 §6.1.2 p.6-1])
commences from port holding. Pilots and the tower use side numbers
(corrected per CV NATOPS §6.1.2 p.6-1 (user correction 2026-10-01)). Its
check-in (wording per the user, 2026-10-01; no case requested, marshal
assigns Case I by day, [8 §6.2 p.6-2]): "Rough Rider Marshal, two zero
five, holding hands with two one one and two one two and two one three,
marking mom's two seven zero for eight, angels four, state six point two."
→ "Two zero five, Rough Rider Marshal, Case one recovery, expected BRC
three six zero."

| T | Position | Speaker | Transmission / event | Checks |
|---|---|---|---|---|
| 00:00 | 3 NM astern, 800 ft | 205 | "Two zero five, initial." | TBow = +34 s; E(0) = +112; first free slot +250 ≤ L = +255 |
| 00:00 | | (solver) | δ = 138 s, break 4 NM ahead + extension; spin compared: re-break +124, slot 250 ≤ 260 | decision SpinAll (P §7.5 #2) |
| 00:01 | | Tower (Boss) | *(silent: lead decides)* | the lead may spin on their own |
| 00:29 | bow − 5 s, lead has not spun | Tower | "Two zero five, spin." | before `TBow − 5 s`; committed |
| 00:40 | past the bow, climbing left through 1,000 ft | 205 | "Two zero five, spinning." | `@proc 205 step=spin`; `PriSpinReentry` |
| 01:10 | 301 (new) initial | – | – | 205's re-break has priority over 301 |
| 02:10 | re-entering the break at 800 ft | (geometry) | spin → break | 4 members split; landing slots 250/300/350/400 |
| (variant B) | Nmax = 4, 2 in the pattern | Tower | "Two one three, spin." | SpinPortion(1) (P §7.5 #3): the portion by its side number |
| (variant C) | empty pattern | – | BreakNow, silent | `@expect-none` (P §7.5 #1) |
| (variant D) | 212 climbs in the spin aft of the beam | – | logged only | `@must-not` any correction |

**Covers:** ATC-FR-CV1-003, CV1-004, CV1-005, CV1-006, CV1-007, CV1-008, CV1-014.

### S-27 · Case III marshal with EATs
Script `atc-case3-marshal.txt`. Night (Case III), final bearing 351,
marshal 285.0 combined with approach (the ship's callsign "Rough Rider" is
the CVN-71 example: another ship speaks its own, ATC-FR-CVC-008), four singles (side numbers 301, 302,
401, 402) check in within a minute. Check-in, marshal instructions and
reports in CV NATOPS form: check-in [8 §6.1.2 p.6-1], ship's call sign on
initial contact [8 §4.4.2 p.4-5], marshal instructions [8 §6.4.2 p.6-7/6-8],
before commencing [8 §6.4.3 p.6-8], compulsory reports [8 §6.4.13.1
p.6-20/21], needles [8 §6.4.7.4 p.6-15]; corrected per CV NATOPS §6.1.2 p.6-1, §6.4.2 p.6-7, §6.4.13.1 p.6-21 (user correction 2026-10-01). Check-in wording per the user (2026-10-01; NATOPS lists the items only): position "marking mom's <radial> for <DME>", "state" or "low state", no case requested: marshal assigns Case three (night) and the CV-1 approach.

| T | Position | Speaker | Transmission / event | Checks |
|---|---|---|---|---|
| 00:00 | 45 NM, angels 18 | 301 | "Rough Rider Marshal, three zero one, marking mom's two six zero for four five, angels one eight, state six point two." | no case requested |
| 00:02 | | Marshal | "Three zero one, Rough Rider Marshal, Case three recovery, CV one approach, expected final bearing three five one, altimeter two niner niner two." "Three zero one, marshal on the one seven one radial, two one DME, angels six." "Three zero one, expected approach time zero five, approach button one five, time check one four zero two." | slot angels 6 / DME 21; `atc.MarshalFix(351,6)` |
| 00:10 | | 301 | "Three zero one, one seven one, two one, angels six, zero five, button one five." | readback correct (silent) |
| 00:20 | | 302 | check-in | "…, angels seven, two two DME, expected approach time zero six." |
| 00:35 | | 401 | check-in "…, marking moms two six zero, four seven, angels two zero, low state five point niner." | angels 8 / 23 DME / zero seven; low state read |
| 00:50 | | 402 | check-in | angels 9 / 24 DME / zero eight |
| 00:55 | | 402 | "…, one seven one, two four, angels nine, zero seven." | wrong EAT → "Four zero two, negative, expected approach time zero eight." |
| 01:30 | entering marshal | 301 | "Three zero one, established, angels six, state five point eight." | compulsory report |
| 02:30 | – | Marshal | "Three zero one, Mother's weather eight hundred overcast, two miles, deck is green, divert Batumi, two six five at one one zero, bingo three point two, time one four zero four." | before commencing |
| 05:00 | push | 301 | "Three zero one, commencing, state five point four, altimeter two niner niner two." | on time |
| 05:40 | 5,000 ft | (geometry) | platform, no call | placed by geometry, no prompt |
| 06:20 | 10 NM | 301 | "Three zero one, ten miles." | |
| 06:50 | 6 NM, lock-on | Approach | "Three zero one, lock on, six miles, say needles." | CCA |
| 06:53 | | 301 | "Three zero one, needles up and right." | [8 §6.4.7.4 item 3 p.6-15] |
| 06:54 | | Approach | "Three zero one, concur." | |
| 07:30 | ball | 301 | "Three zero one, Hornet ball, four point six." | CATCC talk ends; `@must-not "roger ball"` |
| 06:10 | 302 late push | 302 | "Three zero two, commencing, one minute late, state five point one." | following EATs re-checked; unchanged (spacing kept) |
| (variant) | 401 and 402 as a section | 401 | "Rough Rider Marshal, four zero one, holding hands with four zero two, marking mom's two six zero for four five, angels one eight, state six point two." | one flight 401 with member 402 (`/proc 401`); at night each aircraft gets its own slot and EAT, the wingman by his side number: "Four zero two, marshal on the one seven one radial, two four DME, angels nine." ([8 §6.4 p.6-5], [8 §6.1.2 p.6-1]) |

**Covers:** ATC-FR-MSH-001, MSH-003, MSH-004, MSH-006, MSH-007, MSH-010, MSH-011, MSH-012, RBK-004, CV3-001, CV3-002, CV3-003, CVC-001, CVC-002, CVC-008.

### S-28 · A bolter (Case III, then Case I)
Script `atc-bolter.txt`. Continues S-27 with 301 boltering (the same
check-ins as S-27). The Case I part's singles check in "Rough Rider
Marshal, two one one, marking mom's two seven zero for five, angels four,
state five point two." → "Two one one, Rough Rider Marshal, Case one
recovery, expected BRC three six zero." (no case requested; marshal
assigns it, [8 §6.2 p.6-2]; wording per the user, 2026-10-01).

| T | Position | Speaker | Transmission / event | Checks |
|---|---|---|---|---|
| 07:45 | touches down, airborne past the angled-deck end at 130 kt | (geometry) | `bolter` predicate | `@proc 301 step=bolter` |
| 07:46 | | Approach | "Three zero one, bolter, climb to one thousand two hundred, turn right heading one five zero." | hole solver: next 2-min gap at the 6 NM fix after projected re-arrival (~+5.5 min) |
| 07:50 | | 301 | "Three zero one, climbing one point two, one five zero." | |
| 08:00 | – | Marshal | "Four zero two, new expected approach time one zero." | no gap → every holding EAT +1 min, top-down (4-2 first) |
| 08:05 | – | Marshal | "Four zero one, new expected approach time zero niner." | each acknowledged (qud per slot) |
| 09:30 | abeam | 301 | "Three zero one, abeam, state four point one." | compulsory |
| 13:20 | – | 301 traps | – | slot released |
| (Case I part) 00:00 | Case I, 211 bolters | (geometry) | parallel BRC, climb to 600 | a launch in progress from cat 3 |
| 00:02 | | Tower | "Two one one, traffic launching off the waist." | composed; else silent |
| 00:40 | downwind at 600 | – | re-sequenced before new arrivals from the initial | `@seq CVN-71 211,…` |

**Covers:** ATC-FR-CV3-004, CV1-009, MSH-006, RBK-004.

### S-29 · Case III Delta
Script `atc-case3-delta.txt`. Deck fouled by a crash at T+00:00; 301
passing 11,300 ft after commencing; 302 in marshal at angels 7;
401 at 6,500 ft on final. All three checked in as in S-27 ("Rough Rider
Marshal, three zero one, marking mom's two six zero for four five, angels
one eight, state six point two."; no case requested).

| T | Position | Speaker | Transmission / event | Checks |
|---|---|---|---|---|
| 00:00 | – | (world) | crash in the landing area | deck foul, clear ETA unknown |
| 00:02 | – | Marshal | "Ninety-nine, Delta four." | [8 §6.4.9 p.6-17] |
| 00:05 | 11,300 ft descending | Marshal | "Three zero one, level at one one thousand, hold on the three five one bearing, two six DME, report established." | next lower odd altitude |
| 00:06 | 6,500 ft | – | 401 continues (≤ 7,000) | no instruction |
| 00:30 | – | Marshal | "Three zero two, new expected approach time two two." then "Three zero one, new expected approach time two one." | top-down order |
| 00:40 | – | 302 | "Three zero two, two two." | acked |
| 01:40 | – | (301 NORDO, no ack) | – | in Delta holding without ack: commences immediately at the next opportunity; slot reserved |

**Covers:** ATC-FR-CV3-006, CV1-011, MSH-008.

## Coverage matrix

Every MVP requirement and the scenarios that cover it (generated from the **Covers** lines). Later-priority requirements covered by scenarios are listed after it.

| Requirement | Title | Scenarios |
|---|---|---|
| ATC-FR-IDN-001 | Unidentified caller | S-01 |
| ATC-FR-IDN-002 | Voice binding on the first stated callsign | S-01, S-40 |
| ATC-FR-IDN-003 | Callsign correction window and lock | S-01 |
| ATC-FR-IDN-004 | Position only after identification | S-01 |
| ATC-FR-IDN-005 | Flight and member records | S-01, S-05 |
| ATC-FR-IDN-006 | Pilot-only mode | S-16 |

### P2 / P3 requirements with scenario coverage

| Requirement | Pri | Scenarios |
|---|---|---|
| ATC-FR-IDN-010 | P2 | S-33 |
| ATC-FR-FAC-003 | P2 | S-33, S-40 |
| ATC-FR-FAC-005 | P2 | S-33, S-40 |
| ATC-FR-FAC-021 | P2 | S-46 |
| ATC-FR-ASP-016 | P2 | S-36 |
| ATC-FR-CLR-001 | P2 | S-33 |

Scenario coverage: MVP 200/200; P2/P3 249/427 (the rest are covered by the unit tests named in `tasks.md`).
