# ASOC role: open decisions

The decisions the user must make. Each has options, a recommendation
(what the requirements assume today), the requirements it **blocks**
(their status is `open → OD-nnn`) and those it only **affects**. "Brief"
marks a question the user's brief asked to be decided; "new" a question
raised by this spec.

---

### OD-001 · CAS check-in: JTAC, ASOC or both
- **Source:** brief
- **Question:** Today the JTAC (Landshark) takes the full CAS check-in (J1 Fig V-12). With an ASOC, the flight checks in with the ASOC first. Does the JTAC's check-in move to the ASOC, stay dual, or become abbreviated?
- **Options:** (a) dual, unchanged: the flight gives the full check-in twice (ASOC, then JTAC); no JTAC change; (b) the ASOC passes the check-in brief to the JTAC in the hand-off (`controller.CheckIn` in `Handoff.State`, a small JTAC hook: `Offer` + use of the brief), and the JTAC accepts "checking in as fragged" and goes straight to its situation update; the full check-in still works, and without an ASOC the JTAC is unchanged; (c) moved: with an ASOC configured, the JTAC no longer takes check-ins and sends a flight that calls it first to the ASOC.
- **Recommendation:** (b). It matches doctrine (the aircraft checks in with each agency, abbreviated "as fragged" when the information was already passed: J1 p.V-34), saves the pilot a long repeat, keeps every mission without an ASOC exactly as today, and (c) would break flights that go straight to the JTAC.
- **Blocks:** ASOC-FR-JTC-002, ASOC-FR-JTC-003
- **Affects:** ASOC-FR-RTE-004, ASOC-FR-JTC-004.

### OD-002 · Default callsigns
- **Source:** brief
- **Question:** What does the ASOC call itself when the mission gives no `callsign`?
- **Options:** (a) "Overlord" for a ground or airborne ASOC (a DCS AWACS callsign, already in `nlu/norm.DefaultStations`, and the doc's example of a higher authority), falling back to "Chalice" when a persona already uses Overlord; (b) a fixed TACS-flavoured name ("Kingpin", "Warlord"), which needs adding to the station list; (c) always the coalition AWACS's callsign (co-located); (d) require `callsign`.
- **Recommendation:** (a) for `ground` / `airborne`, and the AWACS persona's callsign for `platform = "awacs"` (the same callsign on a second frequency, as on a real AWACS). The JTAC and AIC keep their mission callsigns (Landshark, Darkstar).
- **Blocks:** –
- **Affects:** ASOC-FR-CFG-002, ASOC-FR-ROL-006.

### OD-003 · One agency on two frequencies vs two agencies
- **Source:** brief
- **Question:** When the ASOC and the AIC share a platform (an E-3), are they one agency on two frequencies (same callsign) or two agencies (two callsigns)? And does the co-located ASOC share Darkstar's voice and voice book?
- **Options:** (a) mission config decides: `platform = "awacs"` gives the same callsign on a separate frequency, a separate controller, a separate voice book and a different voice (another crew member); a ground or airborne ASOC is its own agency with its own callsign; (b) always two callsigns; (c) the same callsign and the same voice and voice book (identify once for both) — this needs a shared book between the AWACS controller and the ASOC, i.e. an AWACS change, which the constraint forbids.
- **Recommendation:** (a). Separate books keep the AWACS untouched and match the ATC rule that separately staffed positions keep their own books; a pilot checks in on each frequency anyway. The mission may set the same `voice` alias to make it sound like one person.
- **Blocks:** ASOC-FR-ROL-006, ASOC-FR-VOX-003
- **Affects:** ASOC-FR-AIC-008, ASOC-FR-COV-003.

### OD-004 · How much ATO realism to require
- **Source:** brief
- **Question:** How much must a mission maker define (mission numbers, ATO lines), and what happens when the mission defines nothing?
- **Options:** (a) nothing required: with no `ato` table the ASOC infers one line per player-capable group from its DCS task (design §5.3) with generated mission numbers that it never volunteers; unfragged flights are accepted and tasked from their ordnance; `ato` lines add realism; `require_mission_number = true` makes it strict; (b) an `ato` table required for the ASOC to task anyone; (c) no ATO concept at all: always task from the check-in.
- **Recommendation:** (a). Zero-config first (the same principle as ATC), realism for those who want it, strict mode as an option.
- **Blocks:** ASOC-FR-ATO-002, ASOC-FR-ATO-004, ASOC-FR-ATO-008
- **Affects:** ASOC-FR-CHK-002, ASOC-FR-AIF-006.

### OD-017 · specview for the ASOC spec (resolved)
- **Source:** new (tooling)
- **Question:** `cmd/specview` only recognises `ATC-FR-` / `ATC-NFR-` IDs and titles its pages "ATC", so it cannot load this folder. It needs a code change.
- **Options:** (a) generalise the ID regexes in `internal/specview/{spec.go,link.go,review.go}` to any `<PREFIX>-(FR|NFR)-<AREA>-nnn`, take the default short-ID prefix and the page titles from the first requirement ID and the README title (task T-M0-06); (b) a `-prefix ASOC` flag; (c) review the Markdown without specview.
- **Recommendation:** (a): small, keeps both specs working, no flag to remember.
- **Blocks:** –
- **Affects:** review of every item in this folder.
- **Resolution:** Resolved 2026-10-02: (a) approved and implemented (T-M0-06); a README may also declare the prefix.
